package main

import (
	"crypto/rand"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/logger"
	"github.com/gorilla/mux"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"github.com/urfave/negroni"
)

const oidLen = 16
const maxOidAttempts = 16
const defaultAddress = ":4444"
const defaultDataDir = "~/dead-drop"

func main() {
	log := initLogger()
	defer log.Close()

	loadConfig()

	dataDir, err := createDataDir()
	if err != nil {
		logger.Fatalf("Failed to create data directory: %v \n", err)
		os.Exit(1)
	}

	// TODO(shane) build object map from persisted files
	objectMap := make(map[string]bool)

	router := mux.NewRouter()

	router.HandleFunc("/d/{oid}", func(w http.ResponseWriter, req *http.Request) {
		params := mux.Vars(req)
		oid := params["oid"]

		if _, ok := objectMap[oid]; !ok {
			return
		}

		data, err := readObject(oid, dataDir)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		_, err = w.Write(data)
		if err != nil {
			logger.Errorf("Failed to write object response: %v \n", err)
			w.WriteHeader(500)
			return
		}
	}).Methods("GET")

	router.HandleFunc("/d", func(w http.ResponseWriter, req *http.Request) {
		bytes, err := ioutil.ReadAll(req.Body)
		if err != nil {
			logger.Errorf("Failed to read object body: %v \n", err)
			w.WriteHeader(500)
			return
		}

		oid := ""
		attempt := 1
		for {
			oid = randomOid(oidLen)
			if _, ok := objectMap[oid]; !ok {
				break
			}
			attempt++
			if attempt > maxOidAttempts {
				logger.Error("Key-space is very full, data is being overwritten")
				break
			}
		}

		objectMap[oid] = true
		writeObject(oid, bytes, dataDir)

		_, err = io.WriteString(w, oid)
		if err != nil {
			logger.Errorf("Failed to write object response: %v \n", err)
			w.WriteHeader(500)
			return
		}
	}).Methods("POST")

	n := negroni.Classic()
	n.UseHandler(router)

	addr := viper.GetString("addr")
	_ = http.ListenAndServe(addr, n)
}

func initLogger() *logger.Logger {
	return logger.Init("Logger", true, true, ioutil.Discard)
}

func loadConfig() {
	viper.SetConfigName("conf")

	viper.AddConfigPath("/etc/dead-drop/")
	viper.AddConfigPath("$HOME/.dead-drop/")
	viper.AddConfigPath(".")

	viper.SetDefault("addr", defaultAddress)
	viper.SetDefault("data_dir", defaultDataDir)

	err := viper.ReadInConfig()
	if err != nil {
		switch err.(type) {
		case viper.ConfigFileNotFoundError:
			logger.Info("No config file found, using the default configuration")
			break
		default:
			logger.Warningf("Failed to load config file: %v \n", err)
		}
	}
}

func createDataDir() (string, error) {
	dataDir, err := homedir.Expand(viper.GetString("data_dir"))
	if err != nil {
		return "", err
	}
	return dataDir, os.MkdirAll(dataDir, 0770)
}

func randomOid(length int) string {
	const characters = "abcdefghijklmnopqrstuvwxyz"

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		logger.Fatalf("Failed to generate random oid: %v \n", err)
	}

	modulo := byte(len(characters))
	for i, b := range bytes {
		bytes[i] = characters[b%modulo]
	}
	return string(bytes)
}

func writeObject(oid string, data []byte, dataDir string) {
	if err := ioutil.WriteFile(objectPath(oid, dataDir), data, 0660); err != nil {
		logger.Errorf("Failed to write object %s to disk: %v \n", oid, err)
	}
}

func readObject(oid string, dataDir string) ([]byte, error) {
	data, err := ioutil.ReadFile(objectPath(oid, dataDir))
	if err != nil {
		logger.Errorf("Failed to read object %s from disk: %v \n", oid, err)
	}
	return data, err
}

func objectPath(oid string, dataDir string) string {
	return filepath.Join(dataDir, oid)
}
