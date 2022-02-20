package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	dryRun  bool
	test    bool
	verbose bool
)

// syncCmd represents the "sync" sub-command.
func syncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "sync",
		Short:         "Sync hours to AFAS",
		Long:          runSyncCmdDescription(),
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.Flags().BoolP("help", "h", false, "Help for "+cmd.Name())

	cmd.AddCommand(syncCsvCmd())
	cmd.AddCommand(syncTogglCmd())

	return cmd
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
