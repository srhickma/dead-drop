package main

import (
	"dead-drop/lib"
	"encoding/json"
	"github.com/google/logger"
	"github.com/gorilla/mux"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
)

type Handler struct {
	db   *Database
	auth *Authenticator
}

var keyNameRegex = regexp.MustCompile(lib.KeyNameRegex)

func (handler *Handler) handlePull(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	oid := params["oid"]

	data, err := handler.db.pull(oid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else if data == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	_, err = w.Write(data)
	if err != nil {
		logger.Errorf("Failed to write object response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (handler *Handler) handleDrop(w http.ResponseWriter, req *http.Request) {
	bytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logger.Errorf("Failed to read object body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	oid := handler.db.drop(bytes)

	_, err = io.WriteString(w, oid)
	if err != nil {
		logger.Errorf("Failed to write object response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (handler *Handler) handleAddKey(w http.ResponseWriter, req *http.Request) {
	var payload lib.AddKeyPayload
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		logger.Errorf("Failed to decode payload: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !keyNameRegex.Match([]byte(payload.KeyName)) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	logger.Infof("Adding public key %s", payload.KeyName)

	if err := handler.auth.addAuthorizedKey(payload.Key, payload.KeyName); err != nil {
		logger.Errorf("Failed to add authorized key: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (handler *Handler) handleToken(w http.ResponseWriter, req *http.Request) {
	var payload lib.TokenRequestPayload
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		logger.Errorf("Failed to decode authentication payload: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !keyNameRegex.Match([]byte(payload.KeyName)) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	storedKey, err := handler.auth.getAuthorizedKey(payload.KeyName)
	if err != nil {
		logger.Errorf("Failed to load authorized key: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	token, err := handler.auth.generateToken(storedKey)
	if err == UnauthorizedErr {
		w.WriteHeader(http.StatusUnauthorized)
		return
	} else if err != nil {
		logger.Errorf("Failed to generate authorization token: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = io.WriteString(w, token)
	if err != nil {
		logger.Errorf("Failed to write authorization token response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (handler *Handler) authenticate(h http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		token := req.Header.Get("Authorization")

		if !handler.auth.validateToken(token) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		h.ServeHTTP(w, req)
	})
}
