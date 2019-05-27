package main

import (
	"io/ioutil"
	"net/http"

	"github.com/google/logger"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"github.com/urfave/negroni"
)

func main() {
	loadConfig()

	objectMap := make(map[string][]byte)

	router := mux.NewRouter()

	router.HandleFunc("/d/{oid}", func(w http.ResponseWriter, req *http.Request) {
		params := mux.Vars(req)
		oid := params["oid"]

		_, err := w.Write(objectMap[oid])
		if err != nil {
			logger.Errorf("Failed to write object response: %v \n", err)
			panic(err)
		}
	}).Methods("GET")

	router.HandleFunc("/d/{oid}", func(w http.ResponseWriter, req *http.Request) {
		params := mux.Vars(req)
		oid := params["oid"]

		bytes, err := ioutil.ReadAll(req.Body)
		if err != nil {
			logger.Warningf("Failed to read object body: %v \n", err)
			panic(err)
		}

		objectMap[oid] = bytes
	}).Methods("POST")

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
