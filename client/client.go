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

			oid, err := drop(filePath, remote)
			if err != nil {
				fmt.Printf("ERROR: Failed to drop file '%s': %v\n", filePath, err)
				os.Exit(1)
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

			if err := pull(remote, oid, destPath); err != nil {
				fmt.Printf("ERROR: Failed to pull object '%s': %v\n", oid, err)
				os.Exit(1)
			}

			fmt.Printf("Pulled %s <- %s\n", destPath, oid)
		},
	}

	var rootCmd = &cobra.Command{Use: "dead"}
	rootCmd.AddCommand(cmdDrop, cmdPull)

	_ = rootCmd.Execute()
}

func drop(filePath string, remote string) (string, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("error reading file '%s': %v", filePath, err)
	}

	remoteUrl := fmt.Sprintf("%s/d", remote)

	client := &http.Client{}

	req, err := http.NewRequest("POST", remoteUrl, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("error building request: %v", err)
	}

	token, err := authenticate(remote)
	if err != nil {
		return "", fmt.Errorf("authentication failed: %v", err)
	}

	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("request failed with status: %s", resp.Status)
	}

	oid, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	return string(oid), nil
}

func pull(remote string, oid string, destPath string) error {
	remoteUrl := fmt.Sprintf("%s/d/%s", remote, oid)

	client := &http.Client{}

	req, err := http.NewRequest("GET", remoteUrl, nil)
	if err != nil {
		return fmt.Errorf("error building request: %v", err)
	}

	token, err := authenticate(remote)
	if err != nil {
		return fmt.Errorf("authentication failed: %v", err)
	}

	req.Header.Set("Authorization", token)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("request failed with status: %s", resp.Status)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}

	if err = ioutil.WriteFile(destPath, data, 0660); err != nil {
		return fmt.Errorf("error writing object to '%s': %v", destPath, err)
	}

	return nil
}

func authenticate(remote string) (string, error) {
	remoteUrl := fmt.Sprintf("%s/token", remote)

	resp, err := http.Get(remoteUrl)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("response status: %s\n", resp.Status)
	}

	data, err := ioutil.ReadAll(resp.Body)
	return string(data), err
}