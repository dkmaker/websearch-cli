package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "websearch [flags] <query>",
	Short: "Web search CLI for AI agents",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("websearch stub:", args[0])
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}
