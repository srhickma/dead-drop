package main

import (
	"github.com/google/logger"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"github.com/urfave/negroni"
	"io/ioutil"
	"net/http"
)

type Error string

func (e Error) Error() string {
	return string(e)
}

func main() {
	log := logger.Init("Logger", true, true, ioutil.Discard)
	defer log.Close()

	loadConfig()

	db := initDatabase(viper.GetString("data_dir"))
	auth := newAuthenticator(viper.GetString("keys_dir"))
	handler := &Handler {db, auth}

	router := mux.NewRouter()

	router.Handle("/d/{oid}", handler.authenticate(handler.handlePull)).Methods("GET")
	router.Handle("/d", handler.authenticate(handler.handleDrop)).Methods("POST")
	router.HandleFunc("/token", handler.handleToken).Methods("POST")

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

	viper.SetDefault("addr", ":4444")
	viper.SetDefault("data_dir", "~/dead-drop")
	viper.SetDefault("keys_dir", "~/.dead-drop/keys")

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
