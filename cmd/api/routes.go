package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (app *application) routes() http.Handler {

	router := chi.NewRouter()

	router.Use(app.recoverPanic)
	router.Use(app.rateLimiter)

	router.NotFound(app.notFoundResponse)
	router.MethodNotAllowed(app.methodNotAllowedResponse)

	router.Route("/v1", func(r chi.Router) {

		r.Get("/healthcheck", app.healthcheckHandler)

		r.Route("/movies", func(r chi.Router) {
			r.Get("/", app.listMoviesHandler)
			r.Post("/", app.createMovieHandler)
			r.Get("/{id}", app.showMovieHandler)
			r.Patch("/{id}", app.updateMovieHandler)
			r.Delete("/{id}", app.deleteMovieHandler)
		})
	})

	return router

}
