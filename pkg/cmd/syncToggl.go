package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/shoekstra/go-afashours/pkg/afas"
	"github.com/shoekstra/go-afashours/pkg/toggl"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	reportMonth string
)

// syncTogglCmd represents the "sync toggl" sub-command.
func syncTogglCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "toggl",
		Short:         "Sync hours from Toggl",
		Long:          runSyncTogglCmdDescription(),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if reportMonth == "" {
				_ = cmd.Help()
				os.Exit(0)
			}
			if err := runSyncTogglCmd(args); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&cfgFile, "config", "c", "", "Config file (default is $HOME/.afashours-cli.yaml)")
	cmd.Flags().StringVarP(&reportMonth, "month", "m", "", "Month to sync (in YYYY-MM format)")
	cmd.Flags().BoolP("help", "h", false, "Help for "+cmd.Name())
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Perform a dry run")
	cmd.Flags().BoolVarP(&test, "test", "t", false, "Use the test REST endpoint")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose")

	_ = cmd.Flags().MarkHidden("dry-run")
	_ = cmd.Flags().MarkHidden("test")

	return cmd
}

func runSyncTogglCmd(args []string) error {
	if err := validReportMonth(); err != nil {
		return nil
	}

	cfg, err := newConfig()
	if err != nil {
		return err
	}

	if cfg.Toggl == nil || cfg.Toggl.Token == nil {
		if togglToken := os.Getenv("TOGGL_TOKEN"); togglToken == "" {
			return fmt.Errorf("Toggl token not found in config file %s", viper.ConfigFileUsed())
		} else {
			cfg.Toggl = &togglConfig{Token: &togglToken}
		}
	}

	// Setup Toggl Client.
	tc, err := toggl.NewClient(*cfg.Toggl.Token)
	if err != nil {
		return err
	}

	// Fetch projects in Toggl workspace.
	if err := tc.GetProjects(); err != nil {
		return err
	}
	if verbose {
		fmt.Printf("Found %d Toggl workspace projects\n", len(tc.Projects))
	}

	// Get start and end days of the month to report to AFAS.
	startDate, endDate, err := reportRange(reportMonth)
	if err != nil {
		return err
	}
	fmt.Printf("Getting time entries from Toggl between %v and %v... ",
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"))

	// Get Toggl time entries.
	te, err := tc.GetTimeEntries(startDate, endDate)
	if err != nil {
		return err
	}
	fmt.Printf("found %d entries:\n", len(te))

	// Get a list of time entries with no project set or a project not found in our config file.
	goodEntries, badEntries, missingEntries := validateTogglTimeEntries(te, cfg.Projects.Names())
	if err != nil {
		return err
	}

	printTogglProjectMatchReport(goodEntries, badEntries, missingEntries)

	fmt.Println()

	c := askForConfirmation("Do you want to continue?")
	if !c {
		os.Exit(0)
	}

	fmt.Println()

	// Setup AFAS client.
	afasHostname := fmt.Sprintf("%s.rest.afas.online", *cfg.AfasAccount)
	if test {
		afasHostname = fmt.Sprintf("%s.resttest.afas.online", *cfg.AfasAccount)
	}
	ac := afas.NewClient(*cfg.AfasToken, afasHostname)

	// Create list of work entries.
	wes := []*afas.WorkEntry{}
	for _, v := range goodEntries {
		p, err := cfg.Projects.GetByName(v.Project.Name)
		if err != nil {
			return err
		}

		we, err := afas.NewWorkEntry(v, *cfg.EmployeeNumber, p.Code, p.Type)
		if err != nil {
			return err
		}

		wes = append(wes, we)
	}

	// Post work entries to AFAS.
	if err := ac.PostHours(wes); err != nil {
		return err
	}

	return nil
}

func runSyncTogglCmdDescription() string {
	return strings.TrimSpace(`
The sync command is used to synchronise hour registrations from supported
sources to AFAS using the hours API.`)
}

func printTogglProjectMatchReport(good, bad, missing []*toggl.TimeEntry) {
	if len(bad) > 0 {
		fmt.Printf("  * %2d time entries with no project set\n", len(bad))
		if verbose {
			for _, e := range bad {
				fmt.Printf("Time entry on %s starting at %s and ending at %s has no project set\n",
					e.Start.Format("2006-01-02"),
					e.Start.Format("15:04:05"),
					e.Stop.Format("15:04:05"))
			}
		}
	}
	if len(missing) > 0 {
		fmt.Printf("  * %2d time entries with a project name not found in provided config file\n", len(missing))
		if verbose {
			for _, e := range missing {
				fmt.Printf("Time entry on %s starting at %s and ending at %s has project name %s\n",
					e.Start.Format("2006-01-02"),
					e.Start.Format("15:04:05"),
					e.Stop.Format("15:04:05"),
					e.Project.Name)
			}
		}
	}
	if len(good) > 0 {
		fmt.Printf("  * %2d time entries ready to post\n", len(good))
		if verbose {
			for _, e := range good {
				fmt.Printf("Time entry on %s starting at %s and ending at %s has project name %s and is ready to post\n",
					e.Start.Format("2006-01-02"),
					e.Start.Format("15:04:05"),
					e.Stop.Format("15:04:05"),
					e.Project.Name)
			}
		}
	}
}

func validReportMonth() error {
	re := regexp.MustCompile(`\d{4}-\d{2}`)
	if !re.MatchString(reportMonth) {
		return fmt.Errorf("Invalid date form: %s, --report-month should be in format YYYY-MM", reportMonth)
	}
	return nil
}

func validateTogglTimeEntries(te []*toggl.TimeEntry, projects []string) (good, bad, empty []*toggl.TimeEntry) {
	for _, v := range te {
		if v.ProjectID == nil {
			empty = append(empty, v)
			continue
		}

		if v.Project == nil || !contains(projects, v.Project.Name) {
			bad = append(bad, v)
			continue
		}

		good = append(good, v)
	}

	return good, empty, bad
}
