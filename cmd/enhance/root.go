package enhance

import (
	"context"
	_ "embed"
	"fmt"
	slog "log"
	"net/url"
	"os"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/log/v2"
	goversion "github.com/caarlos0/go-version"
	"github.com/charmbracelet/colorprofile"
	"github.com/cli/go-gh"
	"github.com/spf13/cobra"

	"github.com/charmbracelet/fang"
	"github.com/dlvhdr/gh-enhance/internal/api"
	"github.com/dlvhdr/gh-enhance/internal/tui"
)

//go:embed logo.txt
var asciiArt string

var rootCmd = &cobra.Command{
	Use:   "gh enhance [<PR URL> | <PR number>] [flags]",
	Long:  lipgloss.NewStyle().Foreground(lipgloss.Green).Render(asciiArt),
	Short: "A Blazingly Fast Terminal UI for GitHub Actions",
	Args:  cobra.MaximumNArgs(1),
	Example: `# view all workflow runs for the current repo
 gh enhance

 # view workflow runs for a specific repo
 gh enhance -R dlvhdr/gh-dash

 # view workflow runs filtered by branch
 gh enhance --branch main

 # look up via a full URL to a GitHub PR
 gh enhance https://github.com/dlvhdr/gh-dash/pull/767

 # look up via a PR number
 gh enhance 767`,
}

func Execute(version goversion.Info) error {
	rootCmd.Version = version.String()
	return fang.Execute(context.Background(), rootCmd, fang.WithColorSchemeFunc(func(
		ld lipgloss.LightDarkFunc,
	) fang.ColorScheme {
		def := fang.DefaultColorScheme(ld)
		def.DimmedArgument = ld(lipgloss.Black, lipgloss.White)
		def.Codeblock = ld(lipgloss.Color("#F1EFEF"), lipgloss.Color("#141417"))
		def.Title = lipgloss.Green
		def.Command = lipgloss.Green
		def.Program = lipgloss.Green
		return def
	}))
}

func init() {
	var loggerFile *os.File
	_, debug := os.LookupEnv("DEBUG")

	if debug {
		var fileErr error
		newConfigFile, fileErr := os.OpenFile("debug.log",
			os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666)
		if fileErr == nil {
			log.SetColorProfile(colorprofile.TrueColor)
			log.SetOutput(newConfigFile)
			log.SetTimeFormat("15:04:05.000")
			log.SetReportCaller(true)
			setDebugLogLevel()
			log.Debug("Logging to debug.log")
		} else {
			loggerFile, _ = tea.LogToFile("debug.log", "debug")
			slog.Print("Failed setting up logging", fileErr)
		}
	} else {
		log.SetOutput(os.Stderr)
		log.SetLevel(log.FatalLevel)
	}

	if loggerFile != nil {
		defer loggerFile.Close()
	}

	var repo string
	var number string

	rootCmd.PersistentFlags().StringVarP(
		&repo,
		"repo",
		"R",
		"",
		`[HOST/]OWNER/REPO   Select another repository using the [HOST/]OWNER/REPO format`,
	)

	rootCmd.SetVersionTemplate(`gh-enhance {{printf "version %s\n" .Version}}`)

	rootCmd.Flags().Bool(
		"flat",
		false,
		"passing this flag will present checks as a flat list",
	)

	rootCmd.Flags().Bool(
		"debug",
		false,
		"passing this flag will allow writing debug output to debug.log",
	)

	rootCmd.Flags().BoolP(
		"help",
		"h",
		false,
		"help for gh-enhance",
	)

	rootCmd.Flags().String("branch", "", "Filter workflow runs by branch name")
	rootCmd.Flags().String("workflow", "", "Filter workflow runs by workflow filename (e.g. ci.yml) or ID")
	rootCmd.Flags().String("event", "", "Filter workflow runs by event type (push, pull_request, schedule, etc.)")
	rootCmd.Flags().String("status", "", "Filter workflow runs by status (queued, in_progress, completed)")

	rootCmd.Run = func(_ *cobra.Command, args []string) {
		mode := tui.ModeRepo

		if len(args) > 0 {
			u, err := url.Parse(args[0])
			if err == nil && u.Hostname() == "github.com" {
				parts := strings.Split(u.Path, "/")
				if len(parts) < 5 {
					exitWithUsage()
				}
				repo = parts[1] + "/" + parts[2]
				number = parts[4]
			}

			if number == "" {
				if _, err := strconv.Atoi(args[0]); err != nil {
					fmt.Printf("Error: %q is not a valid PR number or GitHub URL.\n", args[0])
					os.Exit(1)
				}
				number = args[0]
			}
			mode = tui.ModePR
		}

		if repo == "" {
			r, err := gh.CurrentRepository()
			if err == nil {
				repo = r.Owner() + "/" + r.Name()
			}
		}

		if repo == "" {
			fmt.Println("Error: could not determine repository. Use -R flag or run from within a repo.")
			os.Exit(1)
		}

		flat, err := rootCmd.Flags().GetBool("flat")
		if err != nil {
			log.Fatal("Cannot parse the flat flag", err)
		}

		opts := tui.ModelOpts{Flat: flat, Mode: mode}
		if mode == tui.ModeRepo {
			branch, err := rootCmd.Flags().GetString("branch")
			if err != nil {
				log.Fatal("Cannot parse the branch flag", err)
			}
			workflow, err := rootCmd.Flags().GetString("workflow")
			if err != nil {
				log.Fatal("Cannot parse the workflow flag", err)
			}
			event, err := rootCmd.Flags().GetString("event")
			if err != nil {
				log.Fatal("Cannot parse the event flag", err)
			}
			status, err := rootCmd.Flags().GetString("status")
			if err != nil {
				log.Fatal("Cannot parse the status flag", err)
			}
			opts.RepoRunsFilter = api.RepoRunsFilter{
				Branch:   branch,
				Workflow: workflow,
				Event:    event,
				Status:   status,
				PerPage:  30,
			}
		}

		p := tea.NewProgram(tui.NewModel(repo, number, opts))
		if _, err := p.Run(); err != nil {
			log.Error("failed starting program", "err", err)
			fmt.Println(err)
			os.Exit(1)
		}
	}
}

func exitWithUsage() {
	fmt.Println("Usage: -R owner/repo 15623 or URL to a PR")
	os.Exit(1)
}

func setDebugLogLevel() {
	switch os.Getenv("LOG_LEVEL") {
	case "debug", "":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	}

	log.Debug("log level set", "level", log.GetLevel())
}
