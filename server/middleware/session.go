// Copyright 2017 Frédéric Guillot. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package middleware

import (
	"context"
	"github.com/miniflux/miniflux2/model"
	"github.com/miniflux/miniflux2/server/route"
	"github.com/miniflux/miniflux2/storage"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type SessionMiddleware struct {
	store  *storage.Storage
	router *mux.Router
}

func (s *SessionMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := s.getSessionFromCookie(r)

		if session == nil {
			log.Println("[Middleware:Session] Session not found")
			if s.isPublicRoute(r) {
				next.ServeHTTP(w, r)
			} else {
				http.Redirect(w, r, route.GetRoute(s.router, "login"), http.StatusFound)
			}
		} else {
			log.Println("[Middleware:Session]", session)
			ctx := r.Context()
			ctx = context.WithValue(ctx, "UserId", session.UserID)
			ctx = context.WithValue(ctx, "IsAuthenticated", true)

			next.ServeHTTP(w, r.WithContext(ctx))
		}
	})
}

func (s *SessionMiddleware) isPublicRoute(r *http.Request) bool {
	route := mux.CurrentRoute(r)
	switch route.GetName() {
	case "login", "checkLogin", "stylesheet", "javascript":
		return true
	default:
		return false
	}
}

func (s *SessionMiddleware) getSessionFromCookie(r *http.Request) *model.Session {
	sessionCookie, err := r.Cookie("sessionID")
	if err == http.ErrNoCookie {
		return nil
	}

	session, err := s.store.GetSessionByToken(sessionCookie.Value)
	if err != nil {
		log.Println(err)
		return nil
	}

	return session
}

func NewSessionMiddleware(s *storage.Storage, r *mux.Router) *SessionMiddleware {
	return &SessionMiddleware{store: s, router: r}
}