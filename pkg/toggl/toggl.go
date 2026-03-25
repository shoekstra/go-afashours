package toggl

import (
	"context"
	"fmt"
	"time"

	togglapi "github.com/shoekstra/go-toggl"
)

type Client struct {
	togglClient *togglapi.Client
	Projects    Projects
	WorkSpaceID int
}

// TimeEntry holds the fields from a Toggl time entry needed by the rest of the
// application. Description and Stop are unwrapped from the pointers returned by
// the go-toggl client, and Project is enriched from the workspace project list.
type TimeEntry struct {
	ProjectID   *int
	Project     *togglapi.Project
	Start       time.Time
	Stop        time.Time
	Description string
}

type Projects []*togglapi.Project

// GetByID returns the project with the given ID, or nil if not found.
func (ps Projects) GetByID(id int) *togglapi.Project {
	for _, v := range ps {
		if v.ID == id {
			return v
		}
	}
	return nil
}

func (c *Client) GetProjects() error {
	projects, _, err := c.togglClient.Projects.ListProjects(context.Background(), c.WorkSpaceID, nil)
	if err != nil {
		return err
	}
	c.Projects = Projects(projects)

	return nil
}

func (c *Client) GetTimeEntries(start, end time.Time) ([]*TimeEntry, error) {
	opts := &togglapi.ListTimeEntriesOptions{
		StartDate: togglapi.String(start.Format("2006-01-02")),
		EndDate:   togglapi.String(end.Format("2006-01-02")),
	}

	result, _, err := c.togglClient.TimeEntries.ListTimeEntries(context.Background(), opts)
	if err != nil {
		return nil, err
	}

	entries := make([]*TimeEntry, 0, len(result))

	for _, v := range result {
		te := &TimeEntry{
			ProjectID: v.ProjectID,
			Start:     v.Start,
		}
		if v.Stop != nil {
			te.Stop = *v.Stop
		}
		if v.Description != nil {
			te.Description = *v.Description
		}
		if v.ProjectID != nil {
			te.Project = c.Projects.GetByID(*v.ProjectID)
		}
		entries = append(entries, te)
	}

	return entries, nil
}

func NewClient(token string) (*Client, error) {
	client, err := togglapi.NewClient(token)
	if err != nil {
		return nil, err
	}

	wid, err := workspaceID(client)
	if err != nil {
		return nil, err
	}

	return &Client{togglClient: client, WorkSpaceID: wid}, nil
}

// workspaceID returns the workspace ID. For now only a single workspace is supported.
func workspaceID(tc *togglapi.Client) (int, error) {
	ws, _, err := tc.Workspaces.ListWorkspaces(context.Background())
	if err != nil {
		return 0, err
	}

	switch {
	case len(ws) == 0:
		return 0, fmt.Errorf("no workspaces found")
	case len(ws) > 1:
		return 0, fmt.Errorf("more than one workspace is not yet supported")
	default:
		return ws[0].ID, nil
	}
}
