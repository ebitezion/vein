package main

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// envelope JSON responses behind a wrapper
type envelope map[string]interface{}

// Read ID param
func readIDParam(r *http.Request) string {
	//get the request context
	params := httprouter.ParamsFromContext(r.Context())
	return params.ByName("ID")
}

// writeJSON writes a formats our response to a json
func (app *application) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {

	//Receive input and conv to map format
	resp, err := json.Marshal(data)

	//set header to accept application/json
	for k, v := range headers {
		w.Header()[k] = v
	}

	resp = append(resp, '\n')

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(resp)

	return err

}


func (app *application)readJSON(w http.ResponseWriter, r *http.Request, data interface)error  {
	
}