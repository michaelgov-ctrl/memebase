package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)

	router.HandlerFunc(http.MethodGet, "/v1/memes", app.listMemesHandler)
	router.HandlerFunc(http.MethodPost, "/v1/memes", app.createMemeHandler)
	router.HandlerFunc(http.MethodGet, "/v1/memes/:id", app.showMemeHandler)
	router.HandlerFunc(http.MethodPatch, "/v1/memes/:id", app.updateMemeHandler)
	router.HandlerFunc(http.MethodDelete, "/v1/memes/:id", app.deleteMemeHandler)

	router.HandlerFunc(http.MethodGet, "/v1/rand", app.showRandMemeHandler)

	router.Handler(http.MethodGet, "/metrics", promhttp.Handler())

	return app.metrics(app.recoverPanic(app.enableCORS(app.rateLimit(router))))
}
