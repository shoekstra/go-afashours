package csv

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/gocarina/gocsv"
)

type Line struct {
	Date        string    `csv:"date"`
	Start       time.Time `csv:"-"`
	Stop        time.Time `csv:"-"`
	StartTime   string    `csv:"start_time"`
	StopTime    string    `csv:"stop_time"`
	Project     string    `csv:"project"`
	Description string    `csv:"description,omitempty"`
}

func LoadFromFile(csvFile string) ([]*Line, error) {
	// Read CSV file.
	fc, err := os.Open(csvFile)
	if err != nil {
		return nil, err
	}
	defer fc.Close()

	lines := []*Line{}

	if err := gocsv.UnmarshalFile(fc, &lines); err != nil {
		return nil, err
	}

	for _, v := range lines {
		start, err := time.Parse("2006-01-02 15:04", fmt.Sprintf("%s %s", v.Date, v.StartTime))
		if err != nil {
			return nil, err
		}
		v.Start = start

		end, err := time.Parse("2006-01-02 15:04", fmt.Sprintf("%s %s", v.Date, v.StopTime))
		if err != nil {
			return nil, err
		}
		v.Stop = end
	}

	sort.Slice(lines, func(i, j int) bool {
		return lines[i].Date < lines[j].Date
	})

	return lines, nil
}
