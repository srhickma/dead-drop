package main

import (
	"github.com/google/logger"
	"github.com/gorilla/mux"
	"io"
	"io/ioutil"
	"net/http"
)

type Handler struct {
	db *Database
}

func (handler *Handler) handlePull(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	oid := params["oid"]

	data, err := handler.db.pull(oid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = w.Write(data)
	if err != nil {
		logger.Errorf("Failed to write object response: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (handler *Handler) handleDrop(w http.ResponseWriter, req *http.Request) {
	bytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logger.Errorf("Failed to read object body: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	oid := handler.db.drop(bytes)

	_, err = io.WriteString(w, oid)
	if err != nil {
		logger.Errorf("Failed to write object response: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (handler *Handler) handleToken(w http.ResponseWriter, req *http.Request) {
	// TODO(shane) check that public key exists

	token, err := generateToken()
	if err != nil {
		logger.Errorf("Failed to generate authorization token: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// TODO(shane) encrypt token with public key

	logger.Infof("Returning token: %s\n", token)

	_, err = io.WriteString(w, token)
	if err != nil {
		logger.Errorf("Failed to write authorization token response: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func authenticate(h http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		token := req.Header.Get("Authorization")

		logger.Infof("Accepting token: %s\n", token)

		if !validateToken(token) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		h.ServeHTTP(w, req)
	})
}