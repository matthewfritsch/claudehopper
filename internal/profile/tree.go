package profile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/matthewfritsch/claudehopper/internal/config"
)

// Tree connector constants — match the Python version exactly.
const (
	connectorMid  = "├── "
	connectorLast = "└── "
	prefixMid     = "│   "
	prefixLast    = "    "
)

// TreeNode holds the display information for a single profile in the lineage tree.
type TreeNode struct {
	Name         string
	Active       bool
	Description  string
	ManagedCount int
	SharedCount  int
	SharedPaths  map[string]string // path -> source profile name
	ManagedPaths []string
	CreatedFrom  string
	Children     []*TreeNode
}

// BuildTree reads all profiles from profilesDir, builds parent-child
// relationships from the created_from manifest field, and returns the root
// nodes (profiles with no parent, or whose parent no longer exists).
// Cycle-safe: profiles whose created_from creates a cycle are handled gracefully.
func BuildTree(profilesDir, configPath string) ([]*TreeNode, error) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*TreeNode{}, nil
		}
		return nil, fmt.Errorf("read profiles dir: %w", err)
	}

	// Build a map of name -> TreeNode for all valid profiles.
	nodes := make(map[string]*TreeNode)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(profilesDir, entry.Name(), ".hop-manifest.json")
		m, err := config.LoadManifest(manifestPath)
		if err != nil {
			continue // Not a profile directory
		}
		nodes[entry.Name()] = &TreeNode{
			Name:         entry.Name(),
			Active:       entry.Name() == cfg.Active,
			Description:  m.Description,
			ManagedCount: len(m.ManagedPaths),
			SharedCount:  len(m.SharedPaths),
			SharedPaths:  m.SharedPaths,
			ManagedPaths: m.ManagedPaths,
			CreatedFrom:  m.CreatedFrom,
			Children:     []*TreeNode{},
		}
	}

	// Build childrenOf map: parent name -> list of child nodes.
	childrenOf := make(map[string][]*TreeNode)
	for _, node := range nodes {
		if node.CreatedFrom != "" {
			if _, parentExists := nodes[node.CreatedFrom]; parentExists {
				childrenOf[node.CreatedFrom] = append(childrenOf[node.CreatedFrom], node)
			}
		}
	}

	// Sort children alphabetically within each parent.
	for parentName := range childrenOf {
		sort.Slice(childrenOf[parentName], func(i, j int) bool {
			return childrenOf[parentName][i].Name < childrenOf[parentName][j].Name
		})
		nodes[parentName].Children = childrenOf[parentName]
	}

	// Build set of nodes that appear as someone else's child.
	isChild := make(map[string]bool)
	for _, node := range nodes {
		if node.CreatedFrom != "" {
			if _, parentExists := nodes[node.CreatedFrom]; parentExists {
				isChild[node.Name] = true
			}
		}
	}

	// Root nodes: profiles that are not a child of any existing profile.
	// This naturally handles cycles: in a cycle, neither node has been placed
	// as a child in the non-cyclic sense... wait, both ARE children. So we
	// need a different approach for full cycles.
	var roots []*TreeNode
	for _, node := range nodes {
		if node.CreatedFrom == "" {
			roots = append(roots, node)
		} else if _, parentExists := nodes[node.CreatedFrom]; !parentExists {
			roots = append(roots, node)
		}
	}

	// If no roots were found (complete cycle or all nodes are children of each other),
	// pick the alphabetically first node as a root to break the cycle.
	if len(roots) == 0 && len(nodes) > 0 {
		var names []string
		for name := range nodes {
			names = append(names, name)
		}
		sort.Strings(names)
		roots = append(roots, nodes[names[0]])
	}

	// Sort roots alphabetically.
	sort.Slice(roots, func(i, j int) bool {
		return roots[i].Name < roots[j].Name
	})

	return roots, nil
}

// RenderTree produces an ASCII tree of profiles with box-drawing connectors,
// active markers, shared file annotations, and managed/shared counts.
// Cycle detection uses a visited set — profiles seen twice are skipped.
func RenderTree(roots []*TreeNode) string {
	var buf strings.Builder
	visited := make(map[string]bool)
	for i, root := range roots {
		isLast := i == len(roots)-1
		renderNode(root, "", isLast, visited, &buf)
	}
	return buf.String()
}

// renderNode recursively renders a single tree node and its children.
func renderNode(node *TreeNode, prefix string, isLast bool, visited map[string]bool, buf *strings.Builder) {
	if visited[node.Name] {
		return
	}
	visited[node.Name] = true

	connector := connectorMid
	if isLast {
		connector = connectorLast
	}

	// Build the profile line: connector + name + count info + optional active marker
	activeMarker := ""
	if node.Active {
		activeMarker = " (active)"
	}

	countInfo := fmt.Sprintf("%d managed", node.ManagedCount)
	if node.SharedCount > 0 {
		countInfo = fmt.Sprintf("%d managed, %d shared", node.ManagedCount, node.SharedCount)
	}

	fmt.Fprintf(buf, "%s%s%s (%s)%s\n", prefix, connector, node.Name, countInfo, activeMarker)

	// Child prefix depends on whether this node is the last sibling.
	childPrefix := prefix + prefixMid
	if isLast {
		childPrefix = prefix + prefixLast
	}

	// Render shared paths as annotations beneath the profile line.
	sharedKeys := make([]string, 0, len(node.SharedPaths))
	for k := range node.SharedPaths {
		sharedKeys = append(sharedKeys, k)
	}
	sort.Strings(sharedKeys)

	for i, path := range sharedKeys {
		source := node.SharedPaths[path]
		isLastShared := i == len(sharedKeys)-1 && len(node.Children) == 0
		sharedConnector := connectorMid
		if isLastShared {
			sharedConnector = connectorLast
		}
		fmt.Fprintf(buf, "%s%s%s (shared from %s)\n", childPrefix, sharedConnector, path, source)
	}

	// Render children recursively.
	for i, child := range node.Children {
		isLastChild := i == len(node.Children)-1
		renderNode(child, childPrefix, isLastChild, visited, buf)
	}
}

// treeJSONProfile is the JSON representation of a single profile in the tree output.
type treeJSONProfile struct {
	Name         string            `json:"name"`
	Active       bool              `json:"active"`
	Description  string            `json:"description"`
	CreatedFrom  *string           `json:"created_from"`
	ManagedPaths []string          `json:"managed_paths"`
	ManagedCount int               `json:"managed_count"`
	SharedPaths  map[string]string `json:"shared_paths"`
	SharedCount  int               `json:"shared_count"`
	Children     []string          `json:"children"`
}

// treeJSONOutput is the top-level JSON structure for TreeJSON.
type treeJSONOutput struct {
	Active   string            `json:"active"`
	Profiles []treeJSONProfile `json:"profiles"`
}

// TreeJSON returns a rich JSON representation of the profile tree.
// The schema includes managed/shared counts, shared_paths map, and children
// names. Uses 2-space indentation.
func TreeJSON(roots []*TreeNode, activeName string) ([]byte, error) {
	var profiles []treeJSONProfile
	collectProfiles(roots, &profiles)

	out := treeJSONOutput{
		Active:   activeName,
		Profiles: profiles,
	}

	if out.Profiles == nil {
		out.Profiles = []treeJSONProfile{}
	}

	return json.MarshalIndent(out, "", "  ")
}

// collectProfiles does a depth-first traversal of roots, appending each node
// to profiles in the order encountered.
func collectProfiles(nodes []*TreeNode, profiles *[]treeJSONProfile) {
	for _, node := range nodes {
		var createdFrom *string
		if node.CreatedFrom != "" {
			cf := node.CreatedFrom
			createdFrom = &cf
		}

		children := make([]string, 0, len(node.Children))
		for _, c := range node.Children {
			children = append(children, c.Name)
		}

		managedPaths := node.ManagedPaths
		if managedPaths == nil {
			managedPaths = []string{}
		}

		sharedPaths := node.SharedPaths
		if sharedPaths == nil {
			sharedPaths = map[string]string{}
		}

		*profiles = append(*profiles, treeJSONProfile{
			Name:         node.Name,
			Active:       node.Active,
			Description:  node.Description,
			CreatedFrom:  createdFrom,
			ManagedPaths: managedPaths,
			ManagedCount: node.ManagedCount,
			SharedPaths:  sharedPaths,
			SharedCount:  node.SharedCount,
			Children:     children,
		})

		// Recurse into children
		collectProfiles(node.Children, profiles)
	}
}

// bytesEqual returns true if the files at pathA and pathB have identical content.
// Returns false if either file cannot be read.
func bytesEqual(pathA, pathB string) bool {
	a, err := os.ReadFile(pathA)
	if err != nil {
		return false
	}
	b, err := os.ReadFile(pathB)
	if err != nil {
		return false
	}
	return bytes.Equal(a, b)
}
