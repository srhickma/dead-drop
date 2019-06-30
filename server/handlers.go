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
		w.WriteHeader(500)
		return
	}

	_, err = w.Write(data)
	if err != nil {
		logger.Errorf("Failed to write object response: %v\n", err)
		w.WriteHeader(500)
		return
	}
}

func (handler *Handler) handleDrop(w http.ResponseWriter, req *http.Request) {
	bytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logger.Errorf("Failed to read object body: %v\n", err)
		w.WriteHeader(500)
		return
	}

	oid := handler.db.drop(bytes)

	_, err = io.WriteString(w, oid)
	if err != nil {
		logger.Errorf("Failed to write object response: %v\n", err)
		w.WriteHeader(500)
		return
	}
}
