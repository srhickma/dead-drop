package main

import (
	"github.com/google/logger"
	"github.com/spf13/cobra"
	"io/ioutil"
)

func main() {
	log := logger.Init("Logger", true, true, ioutil.Discard)
	defer log.Close()

	var cmdDrop = &cobra.Command {
		Use:   "drop [file path] [remote]",
		Short: "Drop a file to remote",
		Args: cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			// TODO(shane)
		},
	}

	var cmdPull = &cobra.Command {
		Use:   "pull [remote] [oid] [destination path]",
		Short: "Pull a dropped object from remote",
		Args: cobra.MinimumNArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			// TODO(shane)
		},
	}

	var rootCmd = &cobra.Command{Use: "dead"}
	rootCmd.AddCommand(cmdDrop, cmdPull)

	_ = rootCmd.Execute()
}