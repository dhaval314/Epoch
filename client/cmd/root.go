package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var caCert string
var cert string
var key string
var target string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "client <command> <flags>",
	Short: "Connect to the server",
	Long: `Connect to the server by providing target ip and required certificates (check help for default paths)`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVarP(&target, "target", "t", "127.0.0.1:50051", "Server IP")
	rootCmd.PersistentFlags().StringVarP(&caCert, "ca-cert", "r", "certs/ca-cert.pem", "Root cert.pem file path")
	rootCmd.PersistentFlags().StringVarP(&cert, "cert", "e", "certs/client-cert.pem", "Client cert.pem file path")
	rootCmd.PersistentFlags().StringVarP(&key, "key", "k", "certs/client-key.pem", "Client key.pem file path")
	
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}


