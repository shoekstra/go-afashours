package toggl

import (
	"fmt"
	"time"

	"github.com/dougEfresh/gtoggl-api/gthttp"
	"github.com/dougEfresh/gtoggl-api/gtproject"
	gttimeentry "github.com/dougEfresh/gtoggl-api/gttimentry"
	"github.com/dougEfresh/gtoggl-api/gtworkspace"
)

type Client struct {
	*gthttp.TogglHttpClient
	Projects    Projects
	WorkSpaceID int
}

func (c *Client) GetProjects() error {
	projects, err := gtproject.NewClient(c.TogglHttpClient).List(c.WorkSpaceID)
	if err != nil {
		return err
	}
	c.Projects = Projects(projects)

	return nil
}

func (c *Client) GetTimeEntries(start, end time.Time) ([]*gttimeentry.TimeEntry, error) {
	result, err := gttimeentry.NewClient(c.TogglHttpClient).GetRange(start, end)
	if err != nil {
		return nil, err
	}

	entries := []*gttimeentry.TimeEntry{}

	for _, v := range result {
		p := c.Projects.GetByPID(v.Pid)
		v.Project = &p
		e := v
		entries = append(entries, &e)
	}

	return entries, nil
}

type Projects []gtproject.Project

func (ps Projects) GetByPID(pid uint64) gtproject.Project {
	for _, v := range ps {
		if v.Id == pid {
			return v
		}
	}
	return gtproject.Project{}
}

func NewClient(token string) (*Client, error) {
	client, err := gthttp.NewClient(token)
	if err != nil {
		return nil, err
	}

	wid, err := workspaceID(client)
	if err != nil {
		return nil, err
	}

	return &Client{TogglHttpClient: client, WorkSpaceID: wid}, nil
}

// workspaceID returns the workspace ID. For now only a single workspace is supported.
func workspaceID(tc *gthttp.TogglHttpClient) (int, error) {
	client := gtworkspace.NewClient(tc)

	ws, err := client.List()
	if err != nil {
		return 0, err
	}

	switch {
	case len(ws) == 0:
		return 0, fmt.Errorf("no workspaces found")
	case len(ws) > 1:
		return 0, fmt.Errorf("more than one workspace is not yet supported")
	default:
		return int(ws[0].Id), nil
	}
}
