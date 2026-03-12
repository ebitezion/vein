package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

func (app *application) readJSON(w http.ResponseWriter, r *http.Request, data interface{}) error {
	err := json.NewDecoder(r.Body).Decode(data)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidMarshalErr *json.InvalidUnmarshalError

		switch err {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")

		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)

		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

		case errors.As(err, &invalidUnmarshalError):
			panic(err)
		// For anything else, return the error message as-is.
		default:
			return err

		}

	}
}
