package main

import (
	"github.com/Sirupsen/logrus"
	_ "github.com/lib/pq"
	"net/http"
)

func main() {

	//Create Application Struct (Config)
	app, err := NewApplication()
	if err != nil {
		logrus.Fatalf("Failed with Error: %v\n", err)
	}

	//Create Tick Refresh Job - will check for updates every 15 mins
	app.startTimedRefresher()

	//Create Middlewares + Handlers
	interposedMiddlewares, err := app.interposedMiddlewares()
	if err != nil {
		logrus.Fatal(err.Error())
	}

	//Create Server
	logrus.Infoln("Running HTTP server on " + "port 80")
	err = http.ListenAndServe(":80", interposedMiddlewares)
	if err != nil {
		logrus.Fatal(err.Error())
	}

}
