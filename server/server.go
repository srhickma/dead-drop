package main

import (
	"crypto/tls"
	"dead-drop/lib"
	"github.com/google/logger"
	"github.com/gorilla/mux"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/urfave/negroni"
	"io/ioutil"
	"net/http"
	"path/filepath"
)

const ttlMinFlag = "ttl-min"
const dataDirFlag = "data-dir"
const keysDirFlag = "keys-dir"
const addrFlag = "addr"
const destructiveReadFlag = "destructive-read"
const tlsCertFlag = "tls-cert"
const tlsKeyFlag = "tls-key"

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
		logger.Fatalf("Failed to execute command: %v", err)
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
		logger.Infof("Loading configuration from %s", confFile)
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

	viper.SetDefault(addrFlag, ":4444")
	viper.SetDefault(dataDirFlag, "~/dead-drop")
	viper.SetDefault(keysDirFlag, filepath.Join("~", lib.DefaultConfigDir, "keys"))
	viper.SetDefault(ttlMinFlag, 1440)
	viper.SetDefault(destructiveReadFlag, true)

	err := viper.ReadInConfig()
	if err != nil {
		switch err.(type) {
		case viper.ConfigFileNotFoundError:
			logger.Info("No config file found, using the default configuration")
			break
		default:
			logger.Warningf("Failed to load config file: %v", err)
		}
	} else {
		logger.Infof("Successfully loaded configuration")
	}
}

func startServer() {
	db := initDatabase(viper.GetString(dataDirFlag), viper.GetUint(ttlMinFlag), viper.GetBool(destructiveReadFlag))
	auth := newAuthenticator(viper.GetString(keysDirFlag))
	handler := &Handler{db, auth}

	router := mux.NewRouter()

	router.Handle("/d/{oid}", handler.authenticate(handler.handlePull)).Methods("GET")
	router.Handle("/d", handler.authenticate(handler.handleDrop)).Methods("POST")
	router.Handle("/add-key", handler.authenticate(handler.handleAddKey)).Methods("POST")
	router.HandleFunc("/token", handler.handleToken).Methods("POST")

	negroniServer := negroni.Classic()
	negroniServer.UseHandler(router)

	tlsCert := viper.GetString(tlsCertFlag)
	if len(tlsCert) == 0 {
		logger.Fatalf("A tls certificate must be specified")
	}
	tlsCert, err := homedir.Expand(tlsCert)
	if err != nil {
		logger.Fatalf("Failed to load tls certificate: %v", err)
	}

	tlsKey := viper.GetString(tlsKeyFlag)
	if len(tlsKey) == 0 {
		logger.Fatalf("A tls key must be specified")
	}
	tlsKey, err = homedir.Expand(tlsKey)
	if err != nil {
		logger.Fatalf("Failed to load tls key: %v", err)
	}

	addr := viper.GetString(addrFlag)
	logger.Infof("Starting server on %s", addr)

	tlsConfig := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}

	server := &http.Server{
		Addr:         addr,
		Handler:      negroniServer,
		TLSConfig:    tlsConfig,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}

	if err := server.ListenAndServeTLS(tlsCert, tlsKey); err != nil {
		logger.Fatalf("Failed to start server: %v", err)
	}
}
