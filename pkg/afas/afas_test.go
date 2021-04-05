package afas

import (
	"fmt"
	"os"
	"reflect"
	"testing"
)

var (
	afasAccount = os.Getenv("AFAS_ACCOUNT")
	afasToken   = os.Getenv("AFAS_TOKEN")
	employeeID  = os.Getenv("EMPLOYEE_ID")
)

func checkTestReqs(t *testing.T) {
	if afasAccount == "" || afasToken == "" {
		t.Skip("This test requires an AFAS account number and token, set them by exporting AFAS_ACCOUNT and AFAS_TOKEN")
	}

	if employeeID == "" {
		t.Skip("This test requires an employee ID, set it by exporting EMPLOYEE_ID")
	}
}

func TestClient_DeleteHours(t *testing.T) {
	checkTestReqs(t)

	ac := NewClient(afasToken, fmt.Sprintf("%s.resttest.afas.online", afasAccount))

	wes := []*WorkEntry{{
		DateTime:         "2021-02-01",
		Description:      "Random meeting",
		EmployeeNumber:   employeeID,
		ExAp:             false,
		ID:               0,
		InPu:             false,
		ItemCode:         "SBPINT",
		ProjectID:        "000000000",
		StID:             "1",
		StartTime:        "09:00:00",
		EndTime:          "10:00:00",
		UnknownCandidate: false,
		VaIt:             "1",
	}}

	if err := ac.PostHours(wes); err != nil {
		t.Fatalf("Error posting test data: %s", err)
	}

	dwe, err := ac.DayWorkEntries("2021-02-01", employeeID)
	if err != nil {
		t.Fatalf("Error fetching work entries: %s", err)
	}

	if err := ac.DeleteHours(dwe); err != nil {
		t.Fatalf("Error deleting work entries: %s", err)
	}

	got, err := ac.DayWorkEntries("2021-02-01", employeeID)
	if err != nil {
		t.Fatalf("Error fetching work entries: %s", err)
	}

	if len(got) != 0 {
		t.Errorf("Unexpected number of work entries, got: %d, want: 0", len(got))
	}
}

func TestClient_PostHours(t *testing.T) {
	checkTestReqs(t)

	ac := NewClient(afasToken, fmt.Sprintf("%s.resttest.afas.online", afasAccount))

	dwe, err := ac.DayWorkEntries("2021-01-01", employeeID)
	if err != nil {
		t.Fatalf("Error fetching work entries: %s", err)
	}
	if len(dwe) > 0 {
		if err := ac.DeleteHours(dwe); err != nil {
			t.Fatalf("Error deleting work entries: %s", err)
		}
	}

	wes := []*WorkEntry{{
		DateTime:         "2021-01-01",
		Description:      "Random meeting",
		EmployeeNumber:   employeeID,
		ExAp:             false,
		ID:               0,
		InPu:             false,
		ItemCode:         "SBPINT",
		ProjectID:        "000000000",
		StID:             "1",
		StartTime:        "09:00:00",
		EndTime:          "10:00:00",
		UnknownCandidate: false,
		VaIt:             "1",
	}}

	if err := ac.PostHours(wes); err != nil {
		t.Fatalf("Error posting test data: %s", err)
	}

	resp, err := ac.DayWorkEntries("2021-01-01", employeeID)
	if err != nil {
		t.Fatalf("%s", err)
	}

	testCases := []struct {
		desc string
		got  string
		want string
	}{
		{
			desc: "number of records",
			got:  fmt.Sprint(len(resp)),
			want: "1",
		},
		{
			desc: "project code",
			got:  resp[0].ProjectCode,
			want: wes[0].ProjectID,
		},
		{
			desc: "item code",
			got:  resp[0].Activity,
			want: wes[0].ItemCode,
		},
		{
			desc: "date",
			got:  resp[0].Period,
			want: fmt.Sprintf("%sT00:00:00Z", wes[0].DateTime),
		},
		{
			desc: "start time",
			got:  resp[0].StartTime,
			want: wes[0].StartTime,
		},
		{
			desc: "end time",
			got:  resp[0].EndTime,
			want: wes[0].EndTime,
		},
	}

	for _, tC := range testCases {
		if tC.got != tC.want {
			t.Errorf("Unexpected %s, got: %v, want: %v", tC.desc, tC.got, tC.want)
		}
	}
}

func TestBatchHoursByDate(t *testing.T) {
	have := []*WorkEntry{
		{
			DateTime:         "2021-01-04",
			Description:      "Random meeting",
			EmployeeNumber:   "12345",
			ExAp:             false,
			ID:               0,
			InPu:             false,
			ItemCode:         "INTERNAL",
			ProjectID:        "PROJECT123",
			StID:             "1",
			StartTime:        "09:32:52",
			EndTime:          "09:51:57",
			UnknownCandidate: false,
			VaIt:             "1",
		},
		{
			DateTime:         "2021-01-18",
			Description:      "Random meeting",
			EmployeeNumber:   "12345",
			ExAp:             false,
			ID:               0,
			InPu:             false,
			ItemCode:         "INTERNAL",
			ProjectID:        "PROJECT123",
			StID:             "1",
			StartTime:        "08:35:55",
			EndTime:          "08:59:56",
			UnknownCandidate: false,
			VaIt:             "1",
		},
		{
			DateTime:         "2021-01-18",
			Description:      "Random meeting",
			EmployeeNumber:   "12345",
			ExAp:             false,
			ID:               0,
			InPu:             false,
			ItemCode:         "INTERNAL",
			ProjectID:        "PROJECT123",
			StID:             "1",
			StartTime:        "12:42:07",
			EndTime:          "13:04:31",
			UnknownCandidate: false,
			VaIt:             "1",
		},
		{
			DateTime:         "2021-01-20",
			Description:      "",
			EmployeeNumber:   "12345",
			ExAp:             false,
			ID:               0,
			InPu:             false,
			ItemCode:         "INTERNAL",
			ProjectID:        "PROJECT123",
			StID:             "1",
			StartTime:        "13:30:11",
			EndTime:          "15:00:14",
			UnknownCandidate: false,
			VaIt:             "1",
		},
	}
	want := map[string][]*WorkEntry{
		"2021-01-04": {
			{
				DateTime:         "2021-01-04",
				Description:      "Random meeting",
				EmployeeNumber:   "12345",
				ExAp:             false,
				ID:               0,
				InPu:             false,
				ItemCode:         "INTERNAL",
				ProjectID:        "PROJECT123",
				StID:             "1",
				StartTime:        "09:32:52",
				EndTime:          "09:51:57",
				UnknownCandidate: false,
				VaIt:             "1",
			},
		},
		"2021-01-18": {
			{
				DateTime:         "2021-01-18",
				Description:      "Random meeting",
				EmployeeNumber:   "12345",
				ExAp:             false,
				ID:               0,
				InPu:             false,
				ItemCode:         "INTERNAL",
				ProjectID:        "PROJECT123",
				StID:             "1",
				StartTime:        "12:42:07",
				EndTime:          "13:04:31",
				UnknownCandidate: false,
				VaIt:             "1",
			},
			{
				DateTime:         "2021-01-18",
				Description:      "Random meeting",
				EmployeeNumber:   "12345",
				ExAp:             false,
				ID:               0,
				InPu:             false,
				ItemCode:         "INTERNAL",
				ProjectID:        "PROJECT123",
				StID:             "1",
				StartTime:        "08:35:55",
				EndTime:          "08:59:56",
				UnknownCandidate: false,
				VaIt:             "1",
			},
		},
		"2021-01-20": {
			{
				DateTime:         "2021-01-20",
				Description:      "",
				EmployeeNumber:   "12345",
				ExAp:             false,
				ID:               0,
				InPu:             false,
				ItemCode:         "INTERNAL",
				ProjectID:        "PROJECT123",
				StID:             "1",
				StartTime:        "13:30:11",
				EndTime:          "15:00:14",
				UnknownCandidate: false,
				VaIt:             "1",
			},
		},
	}
	got := batchHoursByDate(have)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Hours not batched together as expected, want: %v, got: %v", want, got)
	}
}
