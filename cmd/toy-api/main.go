package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/matt-hoiland/toy-api/internal/app"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func init() {
	// viper initialization
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/toy-api/")
	viper.AddConfigPath("$HOME/.toy-api")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			panic(fmt.Errorf("fatal error config file: %v", err))
		}
	}

	// Defaults
	viper.SetDefault("log-json", false)
	viper.SetDefault("log-level", "INFO")

	// Environment Variables
	viper.SetEnvPrefix("")
	viper.BindEnv("log-json", "LOG_JSON")
	viper.BindEnv("log-level", "LOG_LEVEL")

	// logrus initialization
	level, err := log.ParseLevel(viper.GetString("log-level"))
	if err != nil {
		panic(fmt.Errorf("fatal error logging: %v", err))
	}
	log.SetLevel(level)
	if viper.GetBool("log-json") {
		log.SetFormatter(&log.JSONFormatter{})
	}
}

func main() {
	if err := run(); err != nil {
		log.WithError(err).Error("(╯°□°）╯︵ ┻━┻ error on startup")
		os.Exit(1)
	}
}

func run() (err error) {
	log.Info("(づ｡◕‿‿◕｡)づ Hello! Starting up!")

	log.WithFields(log.Fields{
		"log-json":  viper.GetBool("log-json"),
		"log-level": viper.GetString("log-level"),
	}).Debug("logging configuration")

	app := app.NewServer(http.NewServeMux())
	srv := &http.Server{
		Addr:    "localhost:8080",
		Handler: app,
	}

	log.WithFields(log.Fields{
		"address": srv.Addr,
	}).Info("(づ￣ ³￣)づ Here we go! Serving!")
	log.Fatal(srv.ListenAndServe())

	return nil
}
