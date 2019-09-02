package cmd

import (
	"fmt"
	"github.com/funlake/awesomedb/cmd/dragonboat"
	//"github.com/funlake/awesomedb/cmd/etcdraft"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{Use: "awesomedb"}

func init() {
	//they are conflicts
	//这两货冲突,T_T,每次只能编译其中一个
	//rootCmd.AddCommand(etcdraft.Command)
	rootCmd.AddCommand(dragonboat.Command)
}

//Execute : Cobra entrance
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
