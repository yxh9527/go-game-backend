package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:  "lottery server",
	Long: "开奖逻辑服务器",
}

func Execute() error {
	return rootCmd.Execute()
}
