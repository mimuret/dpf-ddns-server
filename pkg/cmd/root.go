/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/mimuret/dpf-ddns-server/pkg/server"
	"github.com/mimuret/dpf-ddns-server/pkg/zone"
	"github.com/mimuret/golang-iij-dpf/pkg/api"
	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var configFile string

type rootOption struct {
	configFile string
	listen     string
	endpoint   string
	DebugLevel int
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ddns",
	Short: "Dynamic DNS Server for IIJ DNS Platform Service",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := &api.StdLogger{LogLevel: viper.GetInt("debug")}
		cl := api.NewClient(viper.GetString("token"), viper.GetString("endpoint"), logger)
		reader := zone.NewDpfZoneReader(cl, logger)
		server := server.New(viper.GetString("listen"), reader, logger)
		return server.Run(context.Background())
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is $aHOME/.ddns.yml)")

	rootCmd.PersistentFlags().String("listen", ":53", "config file (default is :53)")
	viper.BindPFlag("listen", rootCmd.Flags().Lookup("listen"))
	rootCmd.PersistentFlags().String("endpoint", "https://api.dns-platform.jp/dpf/v1", "API Endpoint (default is https://api.dns-platform.jp/dpf/v1)")
	viper.BindPFlag("endpoint", rootCmd.Flags().Lookup("endpoint"))
	rootCmd.PersistentFlags().Int("debug", 2, "debug level 0=trace,1=debug,2=info,3=error (default is 2)")
	viper.BindPFlag("debug", rootCmd.Flags().Lookup("debug"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if configFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(configFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".ddns" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".ddns")
	}

	viper.SetEnvPrefix("dpf")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
