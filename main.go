package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dustin/go-humanize"

	"github.com/lagren/tlsguard/tls"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/lagren/tlsguard/persistence"
	"github.com/lagren/tlsguard/slack"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		panic("failed to connect database")
	}

	if err := db.AutoMigrate(&persistence.Host{}, &persistence.NotificationChannel{}); err != nil {
		panic(err)
	}

	checker := &tlsChecker{
		db: db,
	}

	go func(checker *tlsChecker) {
		for {
			time.Sleep(24 * time.Hour)

			ctx := context.Background()

			if err := checker.DailyRun(ctx); err != nil {
				logrus.Errorf("Could not run daily run: %s", err)
			}
		}
	}(checker)

	r := mux.NewRouter()
	r.Use(slack.AuthCheck(os.Getenv("SLACK_SIGNING_KEY")))

	r.HandleFunc("/", dispatchHandler(checker))

	if err := http.ListenAndServe("127.0.0.1:8080", handlers.LoggingHandler(os.Stdout, r)); err != nil {
		panic(err)
	}
}

func dispatchHandler(checker *tlsChecker) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		err := r.ParseForm()
		if err != nil {
			panic(err)
		}

		values := r.Form

		channelID := values.Get("channel_id")

		text := values.Get("text")
		tokens := strings.Split(text, " ")

		action := tokens[0]
		args := tokens[1:]

		switch action {
		case "add":
			if len(args) > 0 {
				err = checker.AddAndSubscribe(ctx, channelID, args[0])
			} else {
				err = fmt.Errorf("missing hostname")
			}
		case "remove":
			if len(args) > 0 {
				err = checker.Remove(ctx, channelID, args[0])
			} else {
				err = fmt.Errorf("missing hostname")
			}
		case "check":
			hostname := args[0]

			expires, issuer, err := tls.Check(hostname)

			if err == nil {
				sendSlackResponse(w, fmt.Sprintf("%s's certificate is issued by %s and expires %s (%s)", hostname, issuer, humanize.Time(expires), expires.Format(time.RFC3339)))
			}
		case "run":
			err = checker.Run(ctx)
		default:
			// TODO Unknown/unsupported action. Send help message to user
		}

		if err != nil {
			logrus.Warnf("Could not execute request: %s", err)
			sendSlackResponse(w, "Could not send process request: "+err.Error())

			return
		}
	}
}

func sendSlackResponse(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf("{\"response_type\": \"in_channel\",\"text\": \"%s\"}", message)))
}
