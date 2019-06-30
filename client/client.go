package main

import (
	"bytes"
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"net/http"
	"os"
)

func main() {
	var cmdDrop = &cobra.Command {
		Use:   "drop [file path] [remote]",
		Short: "Drop a file to remote",
		Args: cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			filePath := args[0]
			remote := args[1]

			data, err := ioutil.ReadFile(filePath)
			if err != nil {
				fmt.Printf("ERROR: Failed to read file '%s': %v\n", filePath, err)
				os.Exit(1)
			}

			remoteUrl := fmt.Sprintf("%s/d", remote)

			resp, err := http.Post(remoteUrl, "application/octet-stream", bytes.NewReader(data))
			if err != nil {
				fmt.Printf("ERROR: Failed to drop file '%s': %v\n", filePath, err)
				return
			}
			if resp.StatusCode != 200 {
				fmt.Printf("ERROR: Failed to drop file '%s': %s\n", filePath, resp.Status)
				return
			}

			oid, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("ERROR: Failed to read drop response: %v\n", err)
				return
			}

			fmt.Printf("Dropped %s -> %s\n", filePath, oid)
		},
	}

	var cmdPull = &cobra.Command {
		Use:   "pull [remote] [oid] [destination path]",
		Short: "Pull a dropped object from remote",
		Args: cobra.MinimumNArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			remote := args[0]
			oid := args[1]
			destPath := args[2]

			remoteUrl := fmt.Sprintf("%s/d/%s", remote, oid)

			resp, err := http.Get(remoteUrl)
			if err != nil {
				fmt.Printf("ERROR: Failed to pull object '%s': %v\n", oid, err)
				return
			}
			if resp.StatusCode != 200 {
				fmt.Printf("ERROR: Failed to pull object '%s': %s\n", oid, resp.Status)
				return
			}

			data, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("ERROR: Failed to read pull response: %v\n", err)
				return
			}

			if err = ioutil.WriteFile(destPath, data, 0660); err != nil {
				fmt.Printf("ERROR: Failed to write object %s to %s: %v\n", oid, destPath, err)
				return
			}

			fmt.Printf("Pulled %s <- %s\n", destPath, oid)
		},
	}

	var rootCmd = &cobra.Command{Use: "dead"}
	rootCmd.AddCommand(cmdDrop, cmdPull)

	_ = rootCmd.Execute()
}