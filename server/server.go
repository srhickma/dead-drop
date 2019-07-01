package main

import (
	"github.com/google/logger"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"github.com/urfave/negroni"
	"io/ioutil"
	"net/http"
)

const defaultAddress = ":4444"
const defaultDataDir = "~/dead-drop"

func main() {
	log := logger.Init("Logger", true, true, ioutil.Discard)
	defer log.Close()

	loadConfig()

	db := initDatabase(viper.GetString("data_dir"))

	handler := &Handler {db}

	router := mux.NewRouter()

	router.Handle("/d/{oid}", authenticate(handler.handlePull)).Methods("GET")
	router.Handle("/d", authenticate(handler.handleDrop)).Methods("POST")
	router.HandleFunc("/token", handler.handleToken).Methods("GET")

	n := negroni.Classic()
	n.UseHandler(router)

	addr := viper.GetString("addr")
	_ = http.ListenAndServe(addr, n)
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
