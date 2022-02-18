package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/shoekstra/go-afashours/pkg/afas"
	"github.com/shoekstra/go-afashours/pkg/csv"
	"github.com/spf13/cobra"
)

var (
	csvFile string
)

// syncCsvCmd represents the "sync csv" sub-command.
func syncCsvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "csv",
		Short:         "Sync hours from a CSV file",
		Long:          runSyncCsvCmdDescription(),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if csvFile == "" {
				_ = cmd.Help()
				os.Exit(0)
			}
			if err := runSyncCsvCmd(args); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.afashours-cli.yaml)")
	cmd.Flags().StringVarP(&csvFile, "csv-file", "f", "", "path to CSV file")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "perform a dry run")
	cmd.Flags().BoolVarP(&test, "test", "t", false, "use the test REST endpoint")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose")

	_ = cmd.Flags().MarkHidden("dry-run")
	_ = cmd.Flags().MarkHidden("test")

	return cmd
}

func runSyncCsvCmd(args []string) error {
	cfg, err := newConfig()
	if err != nil {
		return err
	}

	fmt.Printf("Getting time entries from CSV file %s... ", csvFile)

	data, err := csv.LoadFromFile(csvFile)
	if err != nil {
		return err
	}

	// Get start and end days of the month to report to AFAS.
	startDate := data[0].Date
	endDate := data[len(data)-1].Date

	fmt.Printf("found %d entries in CSV file between %v and %v: \n",
		len(data),
		startDate,
		endDate)

	// Get a list of time entries with no project set or a project not found in our config file.
	goodEntries, badEntries, missingEntries := validateCsvTimeEntries(data, cfg.Projects.Names())
	if err != nil {
		return err
	}

	printCsvProjectMatchReport(goodEntries, badEntries, missingEntries)

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
		p, err := cfg.Projects.GetByName(v.Project)
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

func runSyncCsvCmdDescription() string {
	return strings.TrimSpace(`
The sync command is used to synchronise hour registrations from supported
sources to AFAS using the hours API.`)
}

func printCsvProjectMatchReport(good, bad, missing []*csv.Line) {
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
					e.Project)
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
					e.Project)
			}
		}
	}
}

func validateCsvTimeEntries(te []*csv.Line, projects []string) (good, bad, empty []*csv.Line) {
	for _, v := range te {
		if v.Project == "" {
			empty = append(empty, v)
			continue
		}

		if !contains(projects, v.Project) {
			bad = append(bad, v)
			continue
		}

		good = append(good, v)
	}

	return good, empty, bad
}
