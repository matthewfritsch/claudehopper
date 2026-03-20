package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// TitleCache maps session IDs to generated titles.
type TitleCache struct {
	Titles map[string]CachedTitle `json:"titles"`
	mu     sync.Mutex
	path   string
	dirty  bool
}

// CachedTitle holds a generated title and when it was created.
type CachedTitle struct {
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
}

// LoadTitleCache loads the cache from disk. The cache is stored at
// configDir/title-cache.json where configDir is the claudehopper config dir.
func LoadTitleCache(configDir string) *TitleCache {
	p := filepath.Join(configDir, "title-cache.json")
	tc := &TitleCache{
		Titles: make(map[string]CachedTitle),
		path:   p,
	}

	data, err := os.ReadFile(p)
	if err != nil {
		return tc
	}

	json.Unmarshal(data, tc)
	if tc.Titles == nil {
		tc.Titles = make(map[string]CachedTitle)
	}
	return tc
}

// Save writes the cache to disk.
func (tc *TitleCache) Save() error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if !tc.dirty {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(tc.path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(tc, "", "  ")
	if err != nil {
		return err
	}

	tc.dirty = false
	return os.WriteFile(tc.path, data, 0o644)
}

// Get returns a cached title if it exists.
func (tc *TitleCache) Get(sessionID string) (string, bool) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	ct, ok := tc.Titles[sessionID]
	return ct.Title, ok
}

// Set stores a title in the cache.
func (tc *TitleCache) Set(sessionID, title string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.Titles[sessionID] = CachedTitle{
		Title:     title,
		CreatedAt: time.Now().Format(time.RFC3339),
	}
	tc.dirty = true
}

// Count returns the number of cached titles.
func (tc *TitleCache) Count() int {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	return len(tc.Titles)
}

// ClearTitles resets the titles map and saves.
func (tc *TitleCache) ClearTitles() error {
	tc.mu.Lock()
	tc.Titles = make(map[string]CachedTitle)
	tc.dirty = true
	tc.mu.Unlock()
	return tc.Save()
}

func extractSessionContext(sess *Session, maxMessages int) (string, error) {
	f, err := os.Open(sess.FilePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	var userMessages []string

	for scanner.Scan() {
		if len(userMessages) >= maxMessages {
			break
		}
		line := scanner.Text()
		if line == "" {
			continue
		}
		var ev Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		if len(ev.Message) == 0 {
			continue
		}
		var msg MessageContent
		if err := json.Unmarshal(ev.Message, &msg); err != nil {
			continue
		}
		if msg.Role == "user" {
			text := extractTextContent(msg.Content)
			text = strings.TrimSpace(text)
			if text != "" && !strings.HasPrefix(text, "<") {
				userMessages = append(userMessages, text)
			}
		}
	}

	if len(userMessages) == 0 {
		return "", fmt.Errorf("no user messages found")
	}
	return strings.Join(userMessages, "\n---\n"), nil
}

// GenerateTitle calls claude to summarize a session into a short title.
func GenerateTitle(sess *Session) (string, error) {
	context, err := extractSessionContext(sess, 5)
	if err != nil {
		return "", err
	}
	if len(context) > 2000 {
		context = context[:2000]
	}

	prompt := fmt.Sprintf(
		"Based on these user messages from a coding session, generate a very short title (max 8 words, no quotes, no punctuation at the end). Just output the title, nothing else.\n\nProject directory: %s\n\nMessages:\n%s",
		sess.Project,
		context,
	)

	cmd := exec.Command("claude", "-p", prompt, "--model", "haiku")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("claude command failed: %w", err)
	}

	title := strings.TrimSpace(string(out))
	title = strings.Trim(title, "\"'`")
	return title, nil
}

// GenerateTitles generates titles for sessions that don't have cached ones.
func GenerateTitles(sessions []Session, cache *TitleCache, maxConcurrent int) (int, error) {
	type work struct {
		idx  int
		sess *Session
	}

	var toGenerate []work
	for i := range sessions {
		if _, ok := cache.Get(sessions[i].ID); !ok {
			toGenerate = append(toGenerate, work{i, &sessions[i]})
		}
	}

	if len(toGenerate) == 0 {
		return 0, nil
	}

	sem := make(chan struct{}, maxConcurrent)
	var mu sync.Mutex
	var generated int
	var errs []error

	var wg sync.WaitGroup
	for _, w := range toGenerate {
		wg.Add(1)
		go func(w work) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			title, err := GenerateTitle(w.sess)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("%s: %w", w.sess.ID[:8], err))
				mu.Unlock()
				return
			}

			cache.Set(w.sess.ID, title)

			mu.Lock()
			generated++
			mu.Unlock()

			fmt.Printf("  generated: %s -> %s\n", w.sess.ID[:8], title)
		}(w)
	}

	wg.Wait()

	if err := cache.Save(); err != nil {
		return generated, fmt.Errorf("saving cache: %w", err)
	}

	if len(errs) > 0 {
		return generated, fmt.Errorf("%d errors (first: %w)", len(errs), errs[0])
	}

	return generated, nil
}
