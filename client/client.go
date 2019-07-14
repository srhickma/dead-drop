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
	"github.com/spf13/cobra"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

const examplePrivateKey = "-----BEGIN RSA PRIVATE KEY-----\n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\n-----END RSA PRIVATE KEY-----\n"
const examplePublicKey = "-----BEGIN RSA PUBLIC KEY-----\nMIICCgKCAgEAvHs49QMMO9XytbxJsGdbdX/yQWIe0tpHFdfQzja1Zl4GHIE3yKWw\njNg074cNGzzfCj8ynOcIuK820GrXMCOi/d7tYOlpgNmYNP27/6n0e5/CCua3qTRE\nU3b2J9sPaZHOZhxAx80dvJZ5sF7S0lShuSQNgN4HSf28tr2ypUR4t9+tDN9Se7lE\nycc4nRIZH+KfyqOLxVMAD1sFYGZDTCSbNPGf0gS2BttqjnpPieRzZCyXxMIi98s0\nyOKztg8i6jHbI95nXRfvhdPEd67dXuq3M0y8TOsIya4GRrLL0Dbux42lLuStloaG\nXKjJZUmbyt0nfHlF6YTVyRdi4gAsd1Fheef5wDkj3ooIJMVGYTNcs/ffTnWbiWHE\nlBQrdu5/2hsotHvicQeWAYABNV8cjuhxNR/48FRdXxvBZ/wJjrmQJ6q/arkwXpsC\nrtWvI/ZZA6QaCxQRI3L5ldcpXEsUedyiXVW5MzQus/y/vR1ucADcgFUtRcgC4Oc5\n05bpqAFyZDNHUC/TG6YgX7p0/vj5kr6voVM7XhPlTlU2NJZivXCBxuZ7MyrHL3Cz\nkIIpC5M+OGW71uVXvxYvHoTG7APivnftUwspigFBStg0yOjWolfRE+OiDYvPu2ZL\nJVdOYEv3rzIee53GuKHD9P4P961tTijiIGcb6excr01nKEyqi9RRuoUCAwEAAQ==\n-----END RSA PUBLIC KEY-----\n"

func main() {
	var cmdDrop = &cobra.Command {
		Use:   "drop <file path> <remote>",
		Short: "Drop a file to remote",
		Args: cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			filePath := args[0]
			remote := strings.TrimSuffix(args[1], "/")

			oid, err := drop(filePath, remote)
			if err != nil {
				fmt.Printf("ERROR: Failed to drop file '%s': %v\n", filePath, err)
				os.Exit(1)
			}

			fmt.Printf("Dropped %s -> %s\n", filePath, oid)
		},
	}

	var cmdPull = &cobra.Command {
		Use:   "pull <remote> <oid> <destination path>",
		Short: "Pull a dropped object from remote",
		Args: cobra.MinimumNArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			remote := strings.TrimSuffix(args[0], "/")
			oid := args[1]
			destPath := args[2]

			if err := pull(remote, oid, destPath); err != nil {
				fmt.Printf("ERROR: Failed to pull object '%s': %v\n", oid, err)
				os.Exit(1)
			}

			fmt.Printf("Pulled %s <- %s\n", destPath, oid)
		},
	}

	var cmdKeyGen = &cobra.Command {
		Use:   "keygen <private key path> <public key path>",
		Short: "Generates an RSA key-pair, for use authenticating requests",
		Args: cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			privPath := args[0]
			pubPath := args[1]

			if err := keyGen(privPath, pubPath); err != nil {
				fmt.Printf("ERROR: Failed to generate key-pair: %v\n", err)
				os.Exit(1)
			}
		},
	}

	var rootCmd = &cobra.Command{Use: "dead"}
	rootCmd.AddCommand(cmdDrop, cmdPull, cmdKeyGen)

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

	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := makeAuthenticatedRequest(client, req, remote)
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

	resp, err := makeAuthenticatedRequest(client, req, remote)
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

	if err := ioutil.WriteFile(privPath, privKeyBytes, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %v", err)
	}
	fmt.Printf("Wrote private key to %s\n", privPath)

	pubKeyBytes := pem.EncodeToMemory(&pem.Block{
		Type:    "RSA PUBLIC KEY",
		Headers: nil,
		Bytes:   x509.MarshalPKCS1PublicKey(&privKey.PublicKey),
	})

	if err := ioutil.WriteFile(pubPath, pubKeyBytes, 0660); err != nil {
		return fmt.Errorf("failed to write public key: %v", err)
	}
	fmt.Printf("Wrote public key to %s\n", pubPath)

	return nil
}

func makeAuthenticatedRequest(client *http.Client, req *http.Request, remote string) (*http.Response, error) {
	for i := 0; true; i++ {
		token, err := authenticate(remote, []byte(examplePublicKey), "key-name")
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

	// Unreachable
	return nil, nil
}

func authenticate(remote string, key []byte, keyName string) (string, error) {
	remoteUrl := fmt.Sprintf("%s/token", remote)

	payload := lib.TokenRequestPayload {
		Key: key,
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

	privKeyDer, _ := pem.Decode([]byte(examplePrivateKey))
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