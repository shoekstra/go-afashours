package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	gttimeentry "github.com/dougEfresh/gtoggl-api/gttimentry"
	"github.com/shoekstra/go-afashours/pkg/afas"
	"github.com/shoekstra/go-afashours/pkg/toggl"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	dryRun  bool
	test    bool
	verbose bool

	reportMonth string
)

// syncCmd represents the "sync" sub-command.
var syncCmd = &cobra.Command{
	Use:           "sync",
	Short:         "Sync hours to AFAS",
	Long:          runSyncCmdDescription(),
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if reportMonth == "" {
			_ = cmd.Help()
			os.Exit(0)
		}
		if err := runSyncCmd(args); err != nil {
			return err
		}
		return nil
	},
}

func initSyncCmd() {
	cmd.AddCommand(syncCmd)

	syncCmd.Flags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.afashours-cli.yaml)")
	syncCmd.Flags().StringVarP(&reportMonth, "month", "m", "", "month to sync (in YYYY-MM format)")
	syncCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "perform a dry run")
	syncCmd.Flags().BoolVarP(&test, "test", "t", false, "use the test REST endpoint")
	syncCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose")

	_ = syncCmd.Flags().MarkHidden("dry-run")
	_ = syncCmd.Flags().MarkHidden("test")
}

func runSyncCmd(args []string) error {
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
	goodEntries, badEntries, missingEntries := validateTimeEntries(te, cfg.Projects.Names())
	if err != nil {
		return err
	}

	printProjectMatchReport(goodEntries, badEntries, missingEntries)

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

func runSyncCmdDescription() string {
	return strings.TrimSpace(`
The sync command is used to synchronise hour registrations from a supported
source to AFAS using the hours API. The source is treated as truth, so when
synchronising hours to AFAS, any existing hour registrations are replied
with those listed at the source.`)
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if strings.EqualFold(v, str) {
			return true
		}
	}
	return false
}

func formatDate(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
}

func printProjectMatchReport(good, bad, missing []*gttimeentry.TimeEntry) {
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

func reportRange(reportMonth string) (time.Time, time.Time, error) {
	if _, err := time.Parse("2006-01", reportMonth); err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("%s, --report-month should be in format YYYY-MM", err)
	}

	rangeMonth := strings.Split(reportMonth, "-")

	year, err := strconv.Atoi(rangeMonth[0])
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	month, err := strconv.Atoi(rangeMonth[1])
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	return formatDate(year, month, 1), formatDate(year, month+1, 0), nil
}

func validReportMonth() error {
	re := regexp.MustCompile(`\d{4}-\d{2}`)
	if !re.MatchString(reportMonth) {
		return fmt.Errorf("Invalid date form: %s, --report-month should be in format YYYY-MM", reportMonth)
	}
	return nil
}

func validateTimeEntries(te []*gttimeentry.TimeEntry, projects []string) (good, bad, empty []*gttimeentry.TimeEntry) {
	for _, v := range te {
		if v.Pid == 0 {
			empty = append(empty, v)
			continue
		}

		if !contains(projects, v.Project.Name) {
			bad = append(bad, v)
			continue
		}

		good = append(good, v)
	}

	return good, empty, bad
}
