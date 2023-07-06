package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/shoekstra/go-afashours/pkg/afas"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

var (
	afasAccount    string
	afasToken      string
	employeeNumber string
	togglToken     string
)

// initCmd represents the "sync init" sub-command.
func initCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "init",
		Short:         "Initialise local config file",
		Long:          runInitCmdDescription(),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runInitCmd(args); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&cfgFile, "config", "c", "", "Config file (default is $HOME/.afashours-cli.yaml)")
	cmd.Flags().StringVarP(&afasAccount, "afas-account", "", "", "AFAS account number")
	cmd.Flags().StringVarP(&afasToken, "afas-token", "", "", "AFAS token")
	cmd.Flags().StringVarP(&employeeNumber, "employee-number", "", "", "Employee number")
	cmd.Flags().StringVarP(&togglToken, "toggl-token", "", "", "Toggl token")

	return cmd
}

func runInitCmd(args []string) error {
	cfg, err := newConfig(withMode("init"))
	if err != nil {
		return err
	}

	if *cfg == (config{}) {
		fmt.Println("Creating new config file...")
	} else {
		fmt.Println("Updating config file:", viper.ConfigFileUsed())
	}

	validateInt := func(input string) error {
		_, err := strconv.ParseInt(input, 0, 64)
		if err != nil {
			return errors.New("Invalid number")
		}
		return nil
	}

	if afasAccount == "" {
		afasAccount = prompt(promptui.Prompt{
			Label:    "AFAS instance",
			Default:  initDefault(cfg.AfasAccount),
			Validate: validateInt,
		})
	}
	cfg.AfasAccount = &afasAccount

	if afasToken == "" {
		afasToken = prompt(promptui.Prompt{
			Label:   "AFAS API token",
			Default: initDefault(cfg.AfasToken),
			Mask:    '*',
		})
	}
	cfg.AfasToken = &afasToken

	if employeeNumber == "" {
		employeeNumber = prompt(promptui.Prompt{
			Label:    "Employee number",
			Default:  initDefault(cfg.EmployeeNumber),
			Validate: validateInt,
		})
	}
	cfg.EmployeeNumber = &employeeNumber

	if togglToken != "" {
		cfg.Toggl = &togglConfig{Token: &togglToken}
	} else {
		togglUser := prompt(promptui.Prompt{
			Label:       "Do you use Toggl",
			HideEntered: true,
			IsConfirm:   true,
		})
		if togglUser == "y" {
			togglToken := prompt(promptui.Prompt{
				Label:   "Toggl API token",
				Default: initDefault(cfg.Toggl.Token),
				Mask:    '*',
			})
			cfg.Toggl = &togglConfig{Token: &togglToken}
		}
	}

	fmt.Println()

	addProject := prompt(promptui.Prompt{
		Label:       "Do you want add a project",
		HideEntered: true,
		IsConfirm:   true,
	})

	if addProject == "y" {
		// Setup AFAS client.
		afasHostname := fmt.Sprintf("%s.rest.afas.online", *cfg.AfasAccount)
		if test {
			afasHostname = fmt.Sprintf("%s.resttest.afas.online", *cfg.AfasAccount)
		}
		ac := afas.NewClient(*cfg.AfasToken, afasHostname)

		fmt.Println("Fetching AFAS projects and project types...")

		// Fetch AFAS projects.
		if err := ac.GetProjects(); err != nil {
			return err
		}
		fmt.Printf("Found %d AFAS projects\n", len(ac.Projects))

		// Fetch AFAS project types.
		if err := ac.GetProjectTypes(); err != nil {
			return err
		}
		fmt.Printf("Found %d AFAS project types\n\n", len(ac.ProjectTypes))

		projects := projects{}

		// Add any pre-existing projects to our new variable.
		if cfg.Projects != nil {
			for k, v := range *cfg.Projects {
				projects[k] = v
			}
		}

		for {
			p, pn, err := addProjectConfig(ac)
			if err != nil {
				return err
			}
			projects[pn] = &p

			addAnotherProject := prompt(promptui.Prompt{
				Label:       "Do you want add another project",
				IsConfirm:   true,
				HideEntered: true,
			})

			if addAnotherProject != "y" {
				break
			}
		}

		cfg.Projects = &projects
	}

	home, err := homedir.Dir()
	if err != nil {
		return err
	}

	writeFile := prompt(promptui.Prompt{
		Label:     "Do you want to save your config",
		IsConfirm: true,
	})

	if writeFile != "y" {
		return nil
	}

	file := prompt(promptui.Prompt{
		Label:   "Enter path to write config file",
		Default: fmt.Sprintf("%s/.afashours-cli.yaml", home),
	})

	if err := writeConfigFile(cfg, file); err != nil {
		return err
	}

	return nil
}

func runInitCmdDescription() string {
	return strings.TrimSpace(`
The init command is used to create a new config file.`)
}

func addProjectConfig(client *afas.Client) (project, string, error) {
	p, err := chooseProject(client)
	if err != nil {
		return project{}, "", err
	}

	pt, err := chooseProjectType(client, p.Group)
	if err != nil {
		return project{}, "", err
	}

	pn := prompt(promptui.Prompt{Label: "Name of entry"})

	return project{Code: p.ID, Type: pt.ItemCode}, pn, nil
}

func chooseProject(client *afas.Client) (*afas.Project, error) {
	funcMap := promptui.FuncMap
	funcMap["date"] = func(input string) string {
		if len(input) == 0 {
			return ""
		}
		t, _ := time.Parse("2006-01-02T00:00:00Z", input)
		date := t.Format("2006-01-02")
		return date
	}

	searcher := func(input string, index int) bool {
		project := client.Projects[index]
		name := strings.Replace(strings.ToLower(project.Name), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)
		if strings.Contains(name, input) {
			return true
		}
		if strings.Contains(project.ID, input) {
			return true
		}
		return false
	}

	templates := promptui.SelectTemplates{
		Active:   `{{ "\\" | blue | bold }} {{ .Name | cyan | bold }}`,
		Inactive: `   {{ .Name | cyan }}`,
		Selected: `{{ "✔" | green | bold }} {{ "Project" | bold }}: {{ .Name | cyan }}`,
		Details: `
Details:
  Project ID: {{ .ID | cyan }}
  Start Date: {{ .StartDate | date | cyan }}
  End Date:   {{ .EndDate | date | cyan }}`,
	}

	prompt := promptui.Select{
		Label:             "Select project",
		Items:             client.Projects,
		Searcher:          searcher,
		StartInSearchMode: true,
		Templates:         &templates,
	}

	index, _, err := prompt.Run()
	if err != nil {
		return nil, err
	}

	return client.Projects[index], nil
}

func chooseProjectType(client *afas.Client, group string) (*afas.ProjectType, error) {
	projectTypes := client.ProjectTypes.FilterByGroup(group)

	searcher := func(input string, index int) bool {
		projectType := projectTypes[index]
		description := strings.Replace(strings.ToLower(projectType.Description), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)
		return strings.Contains(description, input)
	}

	templates := promptui.SelectTemplates{
		Active:   `{{ "\\" | blue | bold }} {{ .Description | cyan | bold }}`,
		Inactive: `   {{ .Description | cyan }}`,
		Selected: `{{ "✔" | green | bold }} {{ "Project Type" | bold }}: {{ .Description | cyan }}`,
		Details: `
Details:
  Item Code: {{ .ItemCode | cyan }}`,
	}

	prompt := promptui.Select{
		Label:             "Select project type",
		Items:             projectTypes,
		Searcher:          searcher,
		StartInSearchMode: true,
		Templates:         &templates,
	}

	index, _, err := prompt.Run()
	if err != nil {
		return nil, err
	}

	return projectTypes[index], nil
}

func initDefault(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func prompt(prompt promptui.Prompt) string {
	result, err := prompt.Run()
	if err != nil {
		// ErrAbort is the error returned when confirm prompts are supplied "n" or "N" as an answer.
		// If another error occurred, exit with status 1.
		if err != promptui.ErrAbort {
			os.Exit(1)
		}
	}
	return result
}

func writeConfigFile(cfg *config, file string) error {
	y, err := yaml.Marshal(cfg)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		return err
	}

	viper.SetConfigType("yaml")

	if err := viper.ReadConfig(bytes.NewBuffer(y)); err != nil {
		return err
	}

	if err := viper.WriteConfigAs(file); err != nil {
		return err
	}

	return nil
}
