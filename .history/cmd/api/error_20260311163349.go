package main

import "net/http"

//logError Print Standard Dev log of error
func (app *application) logError(r *http.Request, err error, nameofFunction string) {
	app.log.Printf("[ %s ] %v ", nameofFunction, err)
}

// errorResponse generic error for wrappers
func (app *application) errorResponse(w http.ResponseWriter, r *http.Request, status int, message interface{}) {
	data := envelope{"Error": message}
	err := app.writeJSON(w, status, data, nil)
	if err != nil {
		app.logError(r, err,"[error/errorResponse]")
		w.WriteHeader(int(status))
	}

}


//notFoundErrorResponse wrapper
func (app *application)notFoundErrorResponse(w http.ResponseWriter, r *http.Request)  {
	
}