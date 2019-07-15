package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"dead-drop/lib"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

const remoteFlag = "remote"
const privKeyFlag = "private-key"
const keyNameFlag = "key-name"

var confFile string
var keyNameRegex = regexp.MustCompile(lib.KeyNameRegex)

func main() {
	cobra.OnInitialize(loadConfig)

	var rootCmd = &cobra.Command{Use: "dead"}
	rootCmd.AddCommand(setupDropCmd(), setupPullCmd(), setupAddKeyCmd(), setupKeyGenCmd())

	rootCmd.PersistentFlags().StringVar(&confFile, "config", "",
		"config file (default is "+filepath.Join("$HOME", lib.DefaultConfigDir, lib.DefaultConfigName)+".yml)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("FATAL: Failed to execute command: %v\n", err)
		os.Exit(1)
	}
}

func loadConfig() {
	if confFile != "" {
		viper.SetConfigFile(confFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			fmt.Printf("Failed to load default configuration: %v\n", err)
			return
		}

		filepath.Join(home)

		confDir, err := homedir.Expand(filepath.Join("~", lib.DefaultConfigDir))
		if err != nil {
			fmt.Printf("Failed to load default configuration: %v\n", err)
			return
		}

		viper.AddConfigPath(confDir)
		viper.SetConfigName(lib.DefaultConfigName)
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading config file: %v\n", err)
		os.Exit(1)
	}
}

func getStringFlag(flag string) (string, error) {
	value := viper.GetString(flag)
	if value == "" {
		return "", fmt.Errorf("flag '%s' not specified or empty", flag)
	}

	return value, nil
}

func bindPFlag(cmd *cobra.Command, flag string) {
	if err := viper.BindPFlag(flag, cmd.PersistentFlags().Lookup(flag)); err != nil {
		fmt.Printf("Error binding %s flag for the %s command: %v\n", flag, cmd.Name(), err)
	}
}

func setupRemoteCmdFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String(remoteFlag, "", "Remote dead-drop host")
	cmd.PersistentFlags().String(privKeyFlag, "",
		"Private key to use for authentication (e.g. generated by keygen)")
	cmd.PersistentFlags().String(keyNameFlag, "", "Key name to use for authentication")
}

func bindRemoteCmdFlags(cmd *cobra.Command) {
	bindPFlag(cmd, remoteFlag)
	bindPFlag(cmd, privKeyFlag)
	bindPFlag(cmd, keyNameFlag)
}

func setupDropCmd() *cobra.Command {
	cmdDrop := &cobra.Command{
		Use:   "drop <file path>",
		Short: "Drop a file to remote",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			filePath := args[0]

			bindRemoteCmdFlags(cmd)

			oid, err := drop(filePath)
			if err != nil {
				fmt.Printf("ERROR: Failed to drop file '%s': %v\n", filePath, err)
				os.Exit(1)
			}

			fmt.Printf("Dropped %s -> %s\n", filePath, oid)
		},
	}

	setupRemoteCmdFlags(cmdDrop)

	return cmdDrop
}

func setupPullCmd() *cobra.Command {
	cmdPull := &cobra.Command{
		Use:   "pull <oid> <destination path>",
		Short: "Pull a dropped object from remote",
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			oid := args[0]
			destPath := args[1]

			bindRemoteCmdFlags(cmd)

			if err := pull(oid, destPath); err != nil {
				fmt.Printf("ERROR: Failed to pull object '%s': %v\n", oid, err)
				os.Exit(1)
			}

			fmt.Printf("Pulled %s <- %s\n", destPath, oid)
		},
	}

	setupRemoteCmdFlags(cmdPull)

	return cmdPull
}

func setupAddKeyCmd() *cobra.Command {
	cmdAddKey := &cobra.Command{
		Use:   "add-key <public key path> <key name>",
		Short: "Add a public key as an authorized key on remote",
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			pubKeyPath := args[0]
			keyName := args[1]

			bindRemoteCmdFlags(cmd)

			if err := addKey(pubKeyPath, keyName); err != nil {
				fmt.Printf("ERROR: Failed to add authorized key '%s': %v\n", pubKeyPath, err)
				os.Exit(1)
			}

			fmt.Printf("Added %s -> %s\n", pubKeyPath, keyName)
		},
	}

	setupRemoteCmdFlags(cmdAddKey)

	return cmdAddKey
}

func setupKeyGenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "gen-key <private key path> <public key path>",
		Short: "Generates an RSA key-pair, for use authenticating requests",
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			privPath := args[0]
			pubPath := args[1]

			if err := keyGen(privPath, pubPath); err != nil {
				fmt.Printf("ERROR: Failed to generate key-pair: %v\n", err)
				os.Exit(1)
			}
		},
	}
}

func drop(filePath string) (string, error) {
	remote, err := getStringFlag(remoteFlag)
	if err != nil {
		return "", err
	}

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

	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := makeAuthenticatedRequest(client, req, remote)
	if err != nil {
		return "", err
	}

	oid, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	return string(oid), nil
}

func pull(oid string, destPath string) error {
	remote, err := getStringFlag(remoteFlag)
	if err != nil {
		return err
	}

	remoteUrl := fmt.Sprintf("%s/d/%s", remote, oid)

	client := &http.Client{}

	req, err := http.NewRequest("GET", remoteUrl, nil)
	if err != nil {
		return fmt.Errorf("error building request: %v", err)
	}

	resp, err := makeAuthenticatedRequest(client, req, remote)
	if err != nil {
		return err
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}

	if err = ioutil.WriteFile(destPath, data, lib.ObjectPerms); err != nil {
		return fmt.Errorf("error writing object to '%s': %v", destPath, err)
	}

	return nil
}

func addKey(pubKeyPath string, keyName string) error {
	remote, err := getStringFlag(remoteFlag)
	if err != nil {
		return err
	}

	remoteUrl := fmt.Sprintf("%s/add-key", remote)

	client := &http.Client{}

	pubKeyBytes, err := ioutil.ReadFile(pubKeyPath)
	if err != nil {
		return fmt.Errorf("error reading public key '%s': %v", pubKeyPath, err)
	}

	payload := lib.AddKeyPayload{
		Key:     pubKeyBytes,
		KeyName: keyName,
	}

	body := new(bytes.Buffer)
	if err := json.NewEncoder(body).Encode(payload); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", remoteUrl, body)
	if err != nil {
		return fmt.Errorf("error building request: %v", err)
	}

	_, err = makeAuthenticatedRequest(client, req, remote)
	return err
}

func keyGen(privPath string, pubPath string) error {
	privKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("failed generating private key: %v", err)
	}

	privKeyBytes := pem.EncodeToMemory(&pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   x509.MarshalPKCS1PrivateKey(privKey),
	})

	if err := ioutil.WriteFile(privPath, privKeyBytes, lib.PrivateKeyPerms); err != nil {
		return fmt.Errorf("failed to write private key: %v", err)
	}
	fmt.Printf("Wrote private key to %s\n", privPath)

	pubKeyBytes := pem.EncodeToMemory(&pem.Block{
		Type:    "RSA PUBLIC KEY",
		Headers: nil,
		Bytes:   x509.MarshalPKCS1PublicKey(&privKey.PublicKey),
	})

	if err := ioutil.WriteFile(pubPath, pubKeyBytes, lib.PublicKeyPerms); err != nil {
		return fmt.Errorf("failed to write public key: %v", err)
	}
	fmt.Printf("Wrote public key to %s\n", pubPath)

	return nil
}

func makeAuthenticatedRequest(client *http.Client, req *http.Request, remote string) (*http.Response, error) {
	resp, err := makeAuthenticatedRequestInternal(client, req, remote)
	if err != nil {
		return resp, fmt.Errorf("request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		return resp, fmt.Errorf("request failed with status: %s", resp.Status)
	}

	return resp, nil
}

func makeAuthenticatedRequestInternal(client *http.Client, req *http.Request, remote string) (*http.Response, error) {
	keyName, err := getStringFlag(keyNameFlag)
	if err != nil {
		return nil, err
	}

	if !keyNameRegex.Match([]byte(keyName)) {
		return nil, fmt.Errorf("invalid key name")
	}

	for i := 0; true; i++ {
		token, err := authenticate(remote, keyName)
		if err != nil {
			return nil, fmt.Errorf("authentication failed: %v", err)
		}

		req.Header.Set("Authorization", token)

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusUnauthorized && i < 1 {
			// If we get here it is because the JWT secret rotated between the two requests.
			// This happens infrequently, so retrying will succeed.
			continue
		}

		return resp, nil
	}

	// Unreachable.
	return nil, nil
}

func authenticate(remote string, keyName string) (string, error) {
	rawPrivKeyPath, err := getStringFlag(privKeyFlag)
	if err != nil {
		return "", err
	}
	privKeyPath, err := homedir.Expand(rawPrivKeyPath)
	if err != nil {
		return "", fmt.Errorf("error locating private key: %v\n", err)
	}

	remoteUrl := fmt.Sprintf("%s/token", remote)

	payload := lib.TokenRequestPayload{
		KeyName: keyName,
	}

	body := new(bytes.Buffer)
	if err := json.NewEncoder(body).Encode(payload); err != nil {
		return "", err
	}

	resp, err := http.Post(remoteUrl, "application/json", body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("response status: %s\n", resp.Status)
	}

	cipher, err := ioutil.ReadAll(resp.Body)

	privKeyBytes, err := ioutil.ReadFile(privKeyPath)
	if err != nil {
		return "", fmt.Errorf("error reading private key '%s': %v", privKeyPath, err)
	}

	privKeyDer, _ := pem.Decode(privKeyBytes)
	if privKeyDer == nil {
		return "", fmt.Errorf("failed to decode pem bytes\n")
	}
	privKey, err := x509.ParsePKCS1PrivateKey(privKeyDer.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %v\n", err)
	}

	token, err := rsa.DecryptOAEP(sha512.New(), rand.Reader, privKey, cipher, []byte(lib.TokenCipherLabel))
	if err != nil {
		return "", fmt.Errorf("failed to decrypt authorization token: %v\n", err)
	}

	return string(token), nil
}
