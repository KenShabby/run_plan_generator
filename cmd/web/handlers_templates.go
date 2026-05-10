package main

import (
	"log"
	"net/http"

	"github.com/KenShabby/run_plan_generator/internal/templates/pages"
	"github.com/go-chi/chi/v5"
)

func (app *application) registerTemplateRoutes(r chi.Router) {
	r.Get("/templates", app.handleGetTemplates)
}

func (app *application) handleGetTemplates(w http.ResponseWriter, r *http.Request) {
	username := app.username(r)
	plans, err := app.queries.ListTemplatePlansWithCounts(r.Context())
	if err != nil {
		log.Printf("error fetching templates: %v", err)
		http.Error(w, "failed to load templates", http.StatusInternalServerError)
		return
	}
	if r.Header.Get("HX-Request") == "true" {
		pages.TemplatesContent(plans).Render(r.Context(), w)
	} else {
		pages.Templates(plans, username).Render(r.Context(), w)
	}
}
