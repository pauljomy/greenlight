package main

import "net/http"

func (app *application) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	payload := struct {
		Status      string `json:"status"`
		Environment string `json:"environment"`
		Version     string `json:"version"`
	}{
		Status:      "ok",
		Environment: app.config.env,
		Version:     version,
	}

	app.writeJSON(w, http.StatusOK, payload)
}
