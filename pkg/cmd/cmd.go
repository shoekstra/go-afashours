package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
)

// cmd represents the "afashours-cli" command when called without any subcommands.
func cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "afashours-cli",
		Short:         "afashours-cli registers time in AFAS using the Hours API",
		SilenceErrors: true,
	}

	cmd.Flags().BoolP("help", "h", false, "Help for "+cmd.Name())

	cmd.AddCommand(initCmd())
	cmd.AddCommand(syncCmd())

	return cmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the cmd.
func Execute() {
	if err := cmd().Execute(); err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".afashours-cli" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".afashours-cli")
	}

	viper.AutomaticEnv() // read in environment variables that match
}

func askForConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/n]: ", s)

		response, _ := reader.ReadString('\n')
		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}
	}
}
