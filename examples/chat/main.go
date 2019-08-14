package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/ccbrown/keyvaluestore/memorystore"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/ccbrown/api-fu/examples/chat/api"
	"github.com/ccbrown/api-fu/examples/chat/app"
	"github.com/ccbrown/api-fu/examples/chat/store"
	"github.com/ccbrown/api-fu/examples/chat/ui"
)

func main() {
	api := &api.API{
		App: &app.App{
			Store: &store.Store{
				Backend: memorystore.NewBackend(),
			},
		},
	}

	router := mux.NewRouter()
	router.HandleFunc("/graphql", api.ServeGraphQL)
	router.NotFoundHandler = http.HandlerFunc(ui.ServeHTTP)

	cors := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "HEAD", "PATCH", "POST", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)

	server := &http.Server{
		Addr:        ":8080",
		Handler:     cors(router),
		ReadTimeout: 2 * time.Minute,
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt)
		<-ch
		logrus.Info("signal caught. shutting down...")
		cancel()
	}()

	go func() {
		<-ctx.Done()
		if err := server.Shutdown(context.Background()); err != nil {
			logrus.Error(err)
		}
	}()

	logrus.Info("listening at http://127.0.0.1:8080")
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		logrus.Error(err)
	}
}
