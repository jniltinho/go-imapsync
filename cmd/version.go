package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the go-imapsync version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("go-imapsync", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
