package main

import (
	"dead-drop/lib"
	"github.com/google/logger"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/urfave/negroni"
	"io/ioutil"
	"net/http"
	"path/filepath"
)

var confFile string

type Error string

func (e Error) Error() string {
	return string(e)
}

func main() {
	showGreeting()
	log := logger.Init("Logger", true, true, ioutil.Discard)
	defer log.Close()

	cobra.OnInitialize(loadConfig)

	var rootCmd = &cobra.Command{
		Use: "deadd",
		Run: func(cmd *cobra.Command, args []string) {
			startServer()
		},
	}
	rootCmd.PersistentFlags().StringVar(&confFile, "config", "",
		"config file (default is "+filepath.Join("~", lib.DefaultConfigDir, lib.DefaultConfigName)+".yml)")

	if err := rootCmd.Execute(); err != nil {
		logger.Fatalf("Failed to execute command: %v\n", err)
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
	if confFile != "" {
		viper.SetConfigFile(confFile)
		logger.Infof("Loading configuration from %s\n", confFile)
	} else {
		dir := filepath.Join("$HOME", lib.DefaultConfigDir)
		viper.AddConfigPath(dir)
		viper.SetConfigName(lib.DefaultConfigName)
		viper.SetConfigType(lib.DefaultConfigType)
		logger.Infof(
			"Searching for configuration at %s.%s\n",
			filepath.Join(dir, lib.DefaultConfigName),
			lib.DefaultConfigType,
		)
	}

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
	} else {
		logger.Infof("Successfully loaded configuration\n")
	}
}

func startServer() {
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
