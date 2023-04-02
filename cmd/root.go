package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "az-spec-validator",
	Short: "Validate Resource Manager API specifications",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func GetRootCmd() *cobra.Command {
	return rootCmd
}

func init() {
	rootCmd.SetOutput(os.Stdout)
	rootCmd.PersistentFlags().StringP("source", "s", "./azure-rest-api-specs", "Source directory containing API specifications")
	viper.BindPFlag("source", rootCmd.PersistentFlags().Lookup("source"))
	viper.SetEnvPrefix("AZ_SPEC_VALIDATOR")
	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()
}
