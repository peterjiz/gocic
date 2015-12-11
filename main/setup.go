package main

import (
	"github.com/carbocation/interpose"
	"github.com/carbocation/interpose/adaptors"
	interpose_middlewares "github.com/carbocation/interpose/middleware"
	"github.com/codegangsta/negroni"
	gorilla_mux "github.com/gorilla/mux"
	"github.com/peterjiz/gocic/retriever"
	"net/http"
	"time"
	"encoding/json"
	"fmt"
	"os"
)

// Application is the application object that runs HTTP server.
type Application struct {
	cicRequests []retriever.CICRequest
}

// NewApplication is the constructor for Application struct.
func NewApplication() (*Application, error) {
	app := &Application{}

	err := app.LoadCICApplications()
	if err != nil { // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
		return nil, err
	}

	return app, nil
}

//Load CIC Applications from "cicapplications.json" config file
func (app *Application) LoadCICApplications() error {
	fh, err := os.Open("cicapplications.json")
	if err != nil {
		return err
	}
	//cicRequests := []retriever.CICRequest{}
	dec := json.NewDecoder(fh)
	err = dec.Decode(&(app.cicRequests))
	if err != nil {
		return err
	}
	return nil
}

func (app *Application) startTimedRefresher() {
	t := time.NewTicker(15 * time.Minute)
	go func(ticker *time.Ticker) {
		for _ = range t.C {
			//retrieve different cic requests
			for _, cicApplication := range app.cicRequests {
				//run job
				cicApplication.TimedRefresh()
			}
		}
	}(t)

}

//Middlewares
func (app *Application) interposedMiddlewares() (*interpose.Middleware, error) {
	middle := interpose.New()

	//Recovery (Negroni)
	middle.Use(adaptors.FromNegroni(negroni.NewRecovery()))

	//Logging
	middle.Use(interpose_middlewares.GorillaLog())

	//Routing
	middle.UseHandler(app.muxRouter())

	return middle, nil
}

func (app *Application) muxRouter() *gorilla_mux.Router {
	mainRouter := gorilla_mux.NewRouter()
	mainRouter = app.wwwRouter(mainRouter)
	return mainRouter
}

func (app *Application) wwwRouter(mainRouter *gorilla_mux.Router) *gorilla_mux.Router {

	router := mainRouter.Host("www.example.com").Subrouter()
	router.HandleFunc("/refresh", app.refreshAllApplications).Methods("GET", "POST")

	//Static File Serving
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("static")))

	return router
}

func (app *Application) refreshAllApplications(w http.ResponseWriter, r *http.Request) {
	//retrieve different cic requests
	for _, cicApplication := range app.cicRequests {
		//refresh all applications
		cicApplication.ForcedRefresh()
	}
}
