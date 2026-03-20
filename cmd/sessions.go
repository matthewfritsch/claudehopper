package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/matthewfritsch/claudehopper/internal/config"
	"github.com/matthewfritsch/claudehopper/internal/session"
	"github.com/spf13/cobra"
)

var (
	sessListProject string
	sessListLimit   int
	sessListFlat    bool
	sessResumeExec  bool
	sessPruneOlder  string
	sessPruneDry    bool
	sessPruneProj   string
	sessTitleConc   int
	sessTitleProj   string
)

var sessionsCmd = &cobra.Command{
	Use:     "sessions",
	Short:   "Manage Claude Code sessions",
	Long:    "List, inspect, resume, and prune Claude Code sessions across projects.",
	Aliases: []string{"sesh"},
}

var sessionsListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List sessions grouped by project",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		cDir := claudeDir()
		sessions, err := session.ScanFilter(cDir, sessListProject)
		if err != nil {
			return err
		}
		if len(sessions) == 0 {
			fmt.Println("No sessions found.")
			return nil
		}

		var cache *session.TitleCache
		cfgDir, err := config.ConfigDir()
		if err == nil {
			cache = session.LoadTitleCache(cfgDir)
		}

		if sessListFlat {
			return sessListPrintFlat(sessions, cache)
		}
		return sessListPrintGrouped(sessions, cache)
	},
}

var sessionsInfoCmd = &cobra.Command{
	Use:               "info <session-id>",
	Short:             "Show detailed information about a session",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: sessCompleteID,
	RunE: func(cmd *cobra.Command, args []string) error {
		sess, err := session.FindByID(claudeDir(), args[0])
		if err != nil {
			return err
		}

		fmt.Printf("Session:      %s\n", sess.ID)
		fmt.Printf("Project:      %s\n", sess.Project)
		fmt.Printf("Started:      %s (%s)\n", sess.StartedAt.Format("2006-01-02 15:04:05"), session.FormatAge(sess.StartedAt))
		fmt.Printf("Last active:  %s (%s)\n", sess.LastActivity.Format("2006-01-02 15:04:05"), session.FormatAge(sess.LastActivity))
		fmt.Printf("Messages:     %d events\n", sess.MessageCount)
		fmt.Printf("Size:         %s\n", session.FormatSize(sess.FileSize))

		if sess.Version != "" {
			fmt.Printf("Version:      %s\n", sess.Version)
		}
		if sess.GitBranch != "" {
			fmt.Printf("Git branch:   %s\n", sess.GitBranch)
		}

		fmt.Printf("File:         %s\n", sess.FilePath)

		cfgDir, err := config.ConfigDir()
		if err == nil {
			cache := session.LoadTitleCache(cfgDir)
			if title, ok := cache.Get(sess.ID); ok {
				fmt.Printf("Title:        %s\n", title)
			}
		}

		fmt.Printf("\nTopic:\n  %s\n", sess.FirstMessage)
		fmt.Printf("\nResume with:\n  claude resume %s\n", sess.ID)

		return nil
	},
}

var sessionsResumeCmd = &cobra.Command{
	Use:               "resume <session-id>",
	Short:             "Resume a Claude session",
	Long:              "Print or execute the 'claude resume' command for a session.",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: sessCompleteID,
	RunE: func(cmd *cobra.Command, args []string) error {
		sess, err := session.FindByID(claudeDir(), args[0])
		if err != nil {
			return err
		}

		if !sessResumeExec {
			fmt.Printf("claude resume %s\n", sess.ID)
			return nil
		}

		claudePath, err := exec.LookPath("claude")
		if err != nil {
			return fmt.Errorf("claude not found in PATH: %w", err)
		}
		return syscall.Exec(claudePath, []string{"claude", "resume", sess.ID}, os.Environ())
	},
}

var sessionsPruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove old sessions",
	Long:  "Remove sessions older than a specified duration. Use --dry-run to preview.",
	RunE: func(cmd *cobra.Command, args []string) error {
		duration, err := sessParseDuration(sessPruneOlder)
		if err != nil {
			return fmt.Errorf("invalid duration %q: %w\nExamples: 7d, 30d, 2w, 3m", sessPruneOlder, err)
		}

		if sessPruneDry {
			fmt.Println("DRY RUN — no files will be deleted")
			fmt.Println()
		}

		result, err := session.Prune(claudeDir(), duration, sessPruneDry)
		if err != nil {
			return err
		}

		if len(result.Deleted) == 0 {
			fmt.Printf("No sessions older than %s found.\n", sessPruneOlder)
			return nil
		}

		verb := "Deleted"
		if sessPruneDry {
			verb = "Would delete"
		}

		for _, s := range result.Deleted {
			project := filepath.Base(s.Project)
			fmt.Printf("  %s %s (%s, %s, %s)\n",
				verb, s.ID[:8], project,
				session.FormatAge(s.LastActivity),
				session.FormatSize(s.FileSize),
			)
		}

		fmt.Printf("\n%s %d session(s), freeing %s\n",
			verb, len(result.Deleted), session.FormatSize(result.BytesFreed))

		for _, err := range result.Errors {
			fmt.Printf("  error: %s\n", err)
		}

		return nil
	},
}

var sessionsTitlesCmd = &cobra.Command{
	Use:   "titles",
	Short: "Generate short AI titles for sessions",
	Long: `Use Claude to generate short summary titles for each session.
Titles are cached so each session is only summarized once.
Uses claude -p with haiku for minimal token usage.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cDir := claudeDir()
		sessions, err := session.ScanFilter(cDir, sessTitleProj)
		if err != nil {
			return err
		}
		if len(sessions) == 0 {
			fmt.Println("No sessions found.")
			return nil
		}

		cfgDir, err := config.ConfigDir()
		if err != nil {
			return fmt.Errorf("config dir: %w", err)
		}
		cache := session.LoadTitleCache(cfgDir)

		var needGen int
		for _, s := range sessions {
			if _, ok := cache.Get(s.ID); !ok {
				needGen++
			}
		}

		if needGen == 0 {
			fmt.Printf("All %d sessions already have cached titles.\n", len(sessions))
			return nil
		}

		fmt.Printf("Generating titles for %d session(s) (%d already cached)...\n\n", needGen, len(sessions)-needGen)

		generated, err := session.GenerateTitles(sessions, cache, sessTitleConc)

		fmt.Printf("\nGenerated %d new title(s). Total cached: %d\n", generated, cache.Count())

		if err != nil {
			fmt.Printf("Warning: %s\n", err)
		}
		return nil
	},
}

var sessionsTitlesClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the title cache",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgDir, err := config.ConfigDir()
		if err != nil {
			return err
		}
		cache := session.LoadTitleCache(cfgDir)
		count := cache.Count()
		if err := cache.ClearTitles(); err != nil {
			return err
		}
		fmt.Printf("Cleared %d cached title(s).\n", count)
		return nil
	},
}

var sessionsStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show session statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		sessions, err := session.ScanAll(claudeDir())
		if err != nil {
			return err
		}
		if len(sessions) == 0 {
			fmt.Println("No sessions found.")
			return nil
		}

		groups := session.GroupByProject(sessions)

		var totalSize int64
		var totalMessages int
		for _, s := range sessions {
			totalSize += s.FileSize
			totalMessages += s.MessageCount
		}

		fmt.Printf("Sessions:   %d\n", len(sessions))
		fmt.Printf("Projects:   %d\n", len(groups))
		fmt.Printf("Events:     %d\n", totalMessages)
		fmt.Printf("Disk usage: %s\n", session.FormatSize(totalSize))

		if len(sessions) > 0 {
			oldest := sessions[len(sessions)-1]
			newest := sessions[0]
			fmt.Printf("Oldest:     %s (%s)\n", session.FormatAge(oldest.LastActivity), filepath.Base(oldest.Project))
			fmt.Printf("Newest:     %s (%s)\n", session.FormatAge(newest.LastActivity), filepath.Base(newest.Project))
		}

		fmt.Println("\nTop projects by session count:")
		type projectCount struct {
			path  string
			count int
			size  int64
		}
		var pcs []projectCount
		for p, ss := range groups {
			var sz int64
			for _, s := range ss {
				sz += s.FileSize
			}
			pcs = append(pcs, projectCount{p, len(ss), sz})
		}
		sort.Slice(pcs, func(i, j int) bool {
			return pcs[i].count > pcs[j].count
		})
		limit := 10
		if limit > len(pcs) {
			limit = len(pcs)
		}
		for _, pc := range pcs[:limit] {
			fmt.Printf("  %-30s %3d sessions  %s\n", filepath.Base(pc.path), pc.count, session.FormatSize(pc.size))
		}

		return nil
	},
}

func init() {
	sessionsListCmd.Flags().StringVarP(&sessListProject, "project", "p", "", "filter by project path (substring match)")
	sessionsListCmd.Flags().IntVarP(&sessListLimit, "limit", "n", 0, "limit sessions per project (0 = all)")
	sessionsListCmd.Flags().BoolVar(&sessListFlat, "flat", false, "flat list instead of grouped by project")

	sessionsResumeCmd.Flags().BoolVarP(&sessResumeExec, "exec", "x", false, "exec into the session directly instead of printing the command")

	sessionsPruneCmd.Flags().StringVar(&sessPruneOlder, "older-than", "30d", "remove sessions older than this (e.g. 7d, 2w, 3m)")
	sessionsPruneCmd.Flags().BoolVar(&sessPruneDry, "dry-run", false, "preview what would be deleted without deleting")
	sessionsPruneCmd.Flags().StringVarP(&sessPruneProj, "project", "p", "", "only prune sessions for this project")

	sessionsTitlesCmd.Flags().IntVarP(&sessTitleConc, "concurrency", "c", 3, "max concurrent Claude calls")
	sessionsTitlesCmd.Flags().StringVarP(&sessTitleProj, "project", "p", "", "only generate titles for this project")

	sessionsTitlesCmd.AddCommand(sessionsTitlesClearCmd)

	sessionsCmd.AddCommand(sessionsListCmd)
	sessionsCmd.AddCommand(sessionsInfoCmd)
	sessionsCmd.AddCommand(sessionsResumeCmd)
	sessionsCmd.AddCommand(sessionsPruneCmd)
	sessionsCmd.AddCommand(sessionsTitlesCmd)
	sessionsCmd.AddCommand(sessionsStatsCmd)

	rootCmd.AddCommand(sessionsCmd)
}

// sessCompleteID provides shell completion for session IDs.
func sessCompleteID(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return session.SessionIDs(claudeDir()), cobra.ShellCompDirectiveNoFileComp
}

// sessDisplayTopic returns a cached title if available, otherwise the first message.
func sessDisplayTopic(s session.Session, cache *session.TitleCache) string {
	if cache != nil {
		if title, ok := cache.Get(s.ID); ok {
			return title
		}
	}
	if s.FirstMessage != "" {
		return s.FirstMessage
	}
	return "(no message)"
}

func sessBranchTag(branch string) string {
	if branch == "" {
		return ""
	}
	return "[" + branch + "]"
}

func sessListPrintGrouped(sessions []session.Session, cache *session.TitleCache) error {
	groups := session.GroupByProject(sessions)

	var projects []string
	for p := range groups {
		projects = append(projects, p)
	}
	sort.Slice(projects, func(i, j int) bool {
		gi, gj := groups[projects[i]], groups[projects[j]]
		return gi[0].LastActivity.After(gj[0].LastActivity)
	})

	for _, project := range projects {
		projectSessions := groups[project]
		displayName := filepath.Base(project)
		if displayName == "." || displayName == "/" {
			displayName = project
		}

		fmt.Printf("\n%s (%s) — %d session(s)\n", displayName, project, len(projectSessions))
		fmt.Println(strings.Repeat("─", 80))

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

		limit := len(projectSessions)
		if sessListLimit > 0 && sessListLimit < limit {
			limit = sessListLimit
		}

		for _, s := range projectSessions[:limit] {
			fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\n",
				s.ID[:8],
				session.FormatAge(s.LastActivity),
				session.FormatSize(s.FileSize),
				sessBranchTag(s.GitBranch),
				sessDisplayTopic(s, cache),
			)
		}
		w.Flush()

		if sessListLimit > 0 && sessListLimit < len(projectSessions) {
			fmt.Printf("  ... and %d more\n", len(projectSessions)-sessListLimit)
		}
	}

	fmt.Printf("\nTotal: %d sessions across %d projects\n", len(sessions), len(groups))
	fmt.Println("Use 'hop sessions info <id>' for details, 'hop sessions resume <id>' to resume.")
	return nil
}

func sessListPrintFlat(sessions []session.Session, cache *session.TitleCache) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tPROJECT\tAGE\tSIZE\tTOPIC\n")

	limit := len(sessions)
	if sessListLimit > 0 && sessListLimit < limit {
		limit = sessListLimit
	}

	for _, s := range sessions[:limit] {
		topic := sessDisplayTopic(s, cache)
		if len(topic) > 60 {
			topic = topic[:57] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			s.ID[:8],
			filepath.Base(s.Project),
			session.FormatAge(s.LastActivity),
			session.FormatSize(s.FileSize),
			topic,
		)
	}
	w.Flush()

	if sessListLimit > 0 && sessListLimit < len(sessions) {
		fmt.Printf("\n... and %d more (showing %d/%d)\n", len(sessions)-sessListLimit, sessListLimit, len(sessions))
	}

	return nil
}

func sessParseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("duration too short")
	}

	unit := s[len(s)-1]
	value := s[:len(s)-1]

	var n int
	if _, err := fmt.Sscanf(value, "%d", &n); err != nil {
		return 0, fmt.Errorf("invalid number: %s", value)
	}

	switch unit {
	case 'd':
		return time.Duration(n) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(n) * 7 * 24 * time.Hour, nil
	case 'm':
		return time.Duration(n) * 30 * 24 * time.Hour, nil
	default:
		return time.ParseDuration(s)
	}
}
