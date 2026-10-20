package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:  "data center",
	Long: "数据中心服务器",
}

func Execute() error {
	return rootCmd.Execute()
}
