package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/ccbrown/keyvaluestore"
	"github.com/ccbrown/keyvaluestore/memorystore"
	"github.com/ccbrown/keyvaluestore/redisstore"
	"github.com/go-redis/redis"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/ccbrown/api-fu/examples/chat/api"
	"github.com/ccbrown/api-fu/examples/chat/app"
	"github.com/ccbrown/api-fu/examples/chat/store"
	"github.com/ccbrown/api-fu/examples/chat/ui"
)

func main() {
	redisAddress := flag.String("redis-address", "", "can be used to run with a redis database")
	flag.Parse()

	var backend keyvaluestore.Backend
	if *redisAddress == "" {
		logrus.Info("using a temporary database. if you would like data to be persistent, provide --redis-address")
		backend = memorystore.NewBackend()
	} else {
		backend = &redisstore.Backend{
			Client: redis.NewClient(&redis.Options{
				Addr: *redisAddress,
			}),
		}
	}

	api := &api.API{
		App: &app.App{
			Store: &store.Store{
				Backend: backend,
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
