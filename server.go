package main

import (
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"

	"github.com/urfave/negroni"
)

func main() {
	router := mux.NewRouter()

	objectMap := make(map[string][]byte)

	router.HandleFunc("/d/{oid}", func(w http.ResponseWriter, req *http.Request) {
		params := mux.Vars(req)
		oid := params["oid"]

		_, err := w.Write(objectMap[oid])
		if err != nil {
			panic(err)
		}
	}).Methods("GET")

	router.HandleFunc("/d/{oid}", func(w http.ResponseWriter, req *http.Request) {
		params := mux.Vars(req)
		oid := params["oid"]

		bytes, err := ioutil.ReadAll(req.Body)
		if err != nil {
			panic(err)
		}

		objectMap[oid] = bytes
	}).Methods("POST")

	n := negroni.Classic()
	n.UseHandler(router)

	_ = http.ListenAndServe(":4444", n)
}