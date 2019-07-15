package main

import (
	"dead-drop/lib"
	"github.com/google/logger"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"github.com/urfave/negroni"
	"io/ioutil"
	"net/http"
	"path/filepath"
)

type Error string

func (e Error) Error() string {
	return string(e)
}

func main() {
	showGreeting()
	log := logger.Init("Logger", true, true, ioutil.Discard)
	defer log.Close()

	loadConfig()

	db := initDatabase(viper.GetString("data_dir"))
	auth := newAuthenticator(viper.GetString("keys_dir"))
	handler := &Handler{db, auth}

	router := mux.NewRouter()

	router.Handle("/d/{oid}", handler.authenticate(handler.handlePull)).Methods("GET")
	router.Handle("/d", handler.authenticate(handler.handleDrop)).Methods("POST")
	router.Handle("/add-key", handler.authenticate(handler.handleAddKey)).Methods("POST")
	router.HandleFunc("/token", handler.handleToken).Methods("POST")

	negroniServer := negroni.Classic()
	negroniServer.UseHandler(router)

	addr := viper.GetString("addr")
	logger.Infof("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, negroniServer); err != nil {
		logger.Fatalf("Failed to start server: %v\n", err)
	}
}

func showGreeting() {
	data, err := Asset("data/greeting.txt")
	if err != nil {
		return
	}

	println(string(data))
}

func loadConfig() {
	// TODO(shane) make the configuration name and directory configurable.
	viper.SetConfigName(lib.DefaultConfigName)

	viper.AddConfigPath(filepath.Join("/etc", lib.DefaultConfigDir))
	viper.AddConfigPath(filepath.Join("$HOME", lib.DefaultConfigDir))
	viper.AddConfigPath(".")

	viper.SetDefault("addr", ":4444")
	viper.SetDefault("data_dir", "~/dead-drop")
	viper.SetDefault("keys_dir", filepath.Join("~", lib.DefaultConfigDir, "keys"))

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
