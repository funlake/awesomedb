package cmd

import (
	"fmt"
	"github.com/funlake/awesomedb/cmd/server"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{Use: "awesome"}

func init() {
	rootCmd.AddCommand(server.Command)
}

//Execute : Cobra entrance
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
