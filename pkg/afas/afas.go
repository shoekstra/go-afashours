package afas

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"

	gttimeentry "github.com/dougEfresh/gtoggl-api/gttimentry"
	"github.com/mitchellh/mapstructure"
	"github.com/tim-online/go-afas-profit-rest"
)

type Client struct {
	*afas.API
}

func (c *Client) DeleteHours(we []*WorkEntryResponse) error {
	req := c.Connector.NewDeleteRequest()
	req.SetMethod("POST")
	req.URLParams().ConnectorID = "PtRealization"

	body := &WorkEntryRequest{}

	for _, v := range we {
		body.PtRealization.Element.Fields = []*WorkEntryRequestField{
			{
				Action: "delete",
				WorkEntry: &WorkEntry{
					DateTime: v.Period,
					ID:       v.RecordNumber,
				},
			},
		}

		req.SetRequestBody(body)

		fmt.Printf("  * Deleting entry id %d... ", v.RecordNumber)
		if _, err := req.Do(); err != nil {
			return err
		}
		fmt.Println("Success")
	}

	return nil
}

// DayWorkEntries returns any work entries registered in AFAS for an employee
// on a given day.
func (c *Client) DayWorkEntries(date string, eid string) ([]*WorkEntryResponse, error) {
	req := c.Connector.NewListRequest()
	req.URLParams().ConnectorID = "_Hours_Entries"
	req.QueryParams().FilterFieldIDs = "resource_id"
	req.QueryParams().FilterValues = eid
	req.QueryParams().OrderByFieldIDs = "period"
	req.QueryParams().Take = 9999

	_, err := req.Do()
	if err != nil {
		return nil, err
	}

	entries := []*WorkEntryResponse{}

	for _, v := range req.ResponseBody().Rows.([]interface{}) {
		fdate := fmt.Sprintf("%sT00:00:00Z", date)
		if fdate == v.(map[string]interface{})["period"] {
			entry := &WorkEntryResponse{}

			decoder, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
				Result:  &entry,
				TagName: "json",
			})

			if err := decoder.Decode(v); err != nil {
				return nil, err
			}

			entries = append(entries, entry)
		}
	}

	return entries, nil
}

func (c *Client) PostHours(entries []*WorkEntry) error {
	batch := make(map[string][]*WorkEntry)
	for _, v := range entries {
		entries := []*WorkEntry{v}
		if existing, ok := batch[v.DateTime]; ok {
			entries = append(entries, existing...)
		}
		batch[v.DateTime] = entries
	}

	batchKeys := make([]string, 0, len(batch))
	for k := range batch {
		batchKeys = append(batchKeys, k)
	}
	sort.Strings(batchKeys)

	for i := range batchKeys {
		date := batchKeys[i]
		wes := batch[date]

		fmt.Printf("Updating hours for %s:\n", date)

		dwe, err := c.DayWorkEntries(date, wes[0].EmployeeNumber)
		if err != nil {
			return err
		}

		if len(dwe) > 0 {
			fmt.Printf("Found %d pre-existing entries to delete:\n", len(dwe))
			if err := c.DeleteHours(dwe); err != nil {
				return err
			}
		}

		fmt.Printf("Adding %d new entries:\n", len(wes))

		sort.Slice(wes, func(i, j int) bool {
			return wes[i].StartTime < wes[j].StartTime
		})

	WES:
		for _, we := range wes {
			fmt.Printf("  * Posting entry starting at %s and ending at %s... ", we.StartTime, we.EndTime)

			body := &WorkEntryRequest{}
			body.PtRealization.Element.Fields = []*WorkEntryRequestField{
				{
					Action:    "insert",
					WorkEntry: we,
				},
			}

			req := c.Connector.NewInsertRequest()
			req.SetRequestBody(body)
			req.URLParams().ConnectorID = "PtRealization"

			resp, err := req.Do()
			if err != nil {
				// Just print the error and continue onto next entry.
				fmt.Printf("%s\n", err)
				continue WES
			}

			r := &WorkEntryResponse{}
			if err := json.Unmarshal(resp.Results, &r); err != nil {
				return err
			}

			fmt.Printf("Success: Created entry with ID %s\n", r.PtRealization.ID)
		}

		fmt.Println()
	}

	return nil
}

type WorkEntry struct {
	DateTime         string `json:"DaTi,omitempty"`
	Description      string `json:"Ds,omitempty"`
	EmployeeNumber   string `json:"EmId,omitempty"`
	ExAp             bool   `json:"ExAp,omitempty"`
	ID               int    `json:"Id,omitempty"`
	InPu             bool   `json:"InPu,omitempty"`
	ItemCode         string `json:"ItCd,omitempty"`
	ProjectID        string `json:"PrId,omitempty"`
	StID             string `json:"StId,omitempty"`
	StartTime        string `json:"StTi,omitempty"`
	EndTime          string `json:"EnTi,omitempty"`
	UnknownCandidate bool   `json:"UnknownCandidate,omitempty"`
	VaIt             string `json:"VaIt,omitempty"`
}

type WorkEntryResponse struct {
	Activity     string  `json:"activity"`
	EndTime      string  `json:"end_time"`
	HoursType    string  `json:"hours_type"`
	Name         string  `json:"name"`
	Period       string  `json:"period"`
	Project      string  `json:"project"`
	ProjectCode  string  `json:"project_code"`
	RecordNumber int     `json:"record_number"`
	ResourceID   string  `json:"resource_id"`
	StartTime    string  `json:"start_time"`
	UsedHrs      float64 `json:"used_hrs"`

	PtRealization struct {
		ID         string `json:"Id"`
		XpRe       string `json:"XpRe"`
		Sessieguid string `json:"Sessieguid"`
	} `json:"PtRealization"`
}

type WorkEntryRequest struct {
	PtRealization struct {
		Element struct {
			Fields []*WorkEntryRequestField `json:"Fields"`
		} `json:"Element"`
	} `json:"PtRealization"`
}

type WorkEntryRequestField struct {
	Action string `json:"@Action"`
	*WorkEntry
}

func NewClient(token, host string) *Client {
	client := afas.NewAPI(nil, "", token)
	client.SetBaseURL(
		url.URL{
			Scheme: "https",
			Host:   host,
			Path:   "/ProfitRestServices/",
		},
	)
	return &Client{API: client}
}

func NewWorkEntry(te *gttimeentry.TimeEntry, employeeNumber, pcode, ptype string) (*WorkEntry, error) {
	return &WorkEntry{
		UnknownCandidate: false,
		DateTime:         te.Start.Format("2006-01-02"),
		VaIt:             "1",
		ItemCode:         ptype,
		EmployeeNumber:   employeeNumber,
		ProjectID:        pcode,
		Description:      te.Description,
		StID:             "1",
		StartTime:        te.Start.Format("15:04:05"),
		EndTime:          te.Stop.Format("15:04:05"),
		ExAp:             false,
		InPu:             false,
	}, nil
}
