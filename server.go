package main

import (
	"crypto/rand"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/google/logger"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"github.com/urfave/negroni"
)

const oidLen = 16
const maxOidAttempts = 16

func main() {
	log := initLogger()
	defer log.Close()

	loadConfig()

	objectMap := make(map[string][]byte)

	router := mux.NewRouter()

	router.HandleFunc("/d/{oid}", func(w http.ResponseWriter, req *http.Request) {
		params := mux.Vars(req)
		oid := params["oid"]

		_, err := w.Write(objectMap[oid])
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

		objectMap[oid] = bytes

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

	viper.SetDefault("addr", ":4444")

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
