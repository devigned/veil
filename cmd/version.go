package cmd

import (
	"fmt"

	"github.com/devigned/veil/core"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Veil",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Veil v%s\n", core.Version)
	},
}
