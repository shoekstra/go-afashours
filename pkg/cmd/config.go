package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type config struct {
	AfasAccount    *string      `mapstructure:"afas_account"`
	AfasToken      *string      `mapstructure:"afas_token"`
	EmployeeNumber *string      `mapstructure:"employee_number"`
	Projects       *projects    `mapstructure:"projects"`
	Toggl          *togglConfig `mapstructure:"toggl"`
}

func (c *config) validate() error {
	if c.AfasAccount == nil {
		if afasAccount := os.Getenv("AFAS_ACCOUNT"); afasAccount == "" {
			return fmt.Errorf("AFAS account number not found in config file %s, set it using the afas_account key", viper.ConfigFileUsed())
		} else {
			c.AfasAccount = &afasAccount
		}
	}

	if c.AfasToken == nil {
		if afasToken := os.Getenv("AFAS_TOKEN"); afasToken == "" {
			return fmt.Errorf("AFAS API token not found in config file %s, set it using the afas_token key", viper.ConfigFileUsed())
		} else {
			c.AfasToken = &afasToken
		}
	}

	if c.EmployeeNumber == nil {
		return fmt.Errorf("Employee number not found in config file %s, set it using the employee_number key", viper.ConfigFileUsed())
	}

	if c.Projects == nil {
		return fmt.Errorf("No projects found in config file %s, set them using the projects key", viper.ConfigFileUsed())
	}

	return nil
}

type project struct {
	Code string `mapstructure:"project"`
	Type string `mapstructure:"type"`
}

type projects map[string]*project

func (ps projects) GetByName(name string) (*project, error) {
	for k, v := range ps {
		if strings.EqualFold(k, name) {
			return v, nil
		}
	}
	return nil, fmt.Errorf("Could not find project with name \"%s\"\n", name)
}

func (ps projects) Names() []string {
	keys := make([]string, 0, len(ps))
	for k := range ps {
		keys = append(keys, k)
	}
	return keys
}

type togglConfig struct {
	Token *string `mapstructure:"token"`
}

func newConfig() (*config, error) {
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	cfg := &config{}
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}
