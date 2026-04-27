package main

import (
	"log"
	"net/http"
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

func (app *application) getSessionUserID(r *http.Request) (int32, bool) {
	session, err := app.store.Get(r, sessionName)
	if err != nil {
		return 0, false
	}
	id, ok := session.Values["user_id"].(int32)
	return id, ok
}

func (app *application) setSessionUserID(w http.ResponseWriter, r *http.Request, userID int32) error {
	session, err := app.store.Get(r, sessionName)
	if err != nil {
		return err
	}
	session.Values["user_id"] = userID
	return session.Save(r, w)
}

func (app *application) clearSession(w http.ResponseWriter, r *http.Request) error {
	session, err := app.store.Get(r, sessionName)
	if err != nil {
		return err
	}
	session.Options.MaxAge = -1
	return session.Save(r, w)
}
