package main

import (
	"log"
	"os"

	"github.com/KenShabby/run_plan_generator/internal/db"
	"github.com/gorilla/sessions"
)

const sessionName = "rpg-session"

type application struct {
	logger  *log.Logger
	queries *db.Queries
	store   *sessions.CookieStore
}

func newApplication(queries *db.Queries, sessionSecret string) *application {
	return &application{
		logger:  log.New(os.Stdout, "", log.Ldate|log.Ltime),
		queries: queries,
		store:   sessions.NewCookieStore([]byte(sessionSecret)),
	}
}
