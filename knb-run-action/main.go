package main

/*
	This microservice is built to execute the action within a specific database.
	this microservice register itself into a consul service registry and
	retrieve the database connection parameters from a vault server.
	it exposes four main functionalities:
		- run action: execute an action and return the result;
		- fetch data: fetch data from the database and return the result;
		- check: check if a specific action can be executed or if a specific expression can be parsed on a specific table
		  whose the name is given;
		- ping: health check of the microservice.
	This service requires the following command line arguments:
		- mode: defines the running mode of the microservice. It can be either 'action' or 'fetch';
		- port: defines the port number where the microservice is listening	;
		- registry: defines the address and port of the consul service registry as well as the check interval, timeout interval
		  and quit interval;
		- vault: defines the address and port of the vault server;
		- config: defines the address of the configuration service and the default configuration path;
		- name: the name of the microservice to be registered into the consul service registry.
	This microservice requires also the following environment variables:
		- WEBAPI_VAULT_TOKEN: the token used to access the vault server;
		- WEBAPI_SRV_NAME: the name of the microservice to be registered into the consul service registry.
*/

import (
	"log/slog"
	"os"
	"strings"

	"github.com/akristianlopez/run-action/knb-run-action/providers"
)

// validateIP checks if the given string is a valid IPv4 or IPv6 address

func main() {
	setupLogger()

	var provider providers.ConfigProvider
	switch strings.ToLower(os.Getenv("SERVICE_KIND")) {
	case "swarm":
		provider = providers.NewSwarmProvider() // Logique avec fichiers /run/secrets
		provider.ReadConfig()
		provider.Launch()
	case "standalone":
		provider = providers.NewStandAloneProvider()
		// ip = providers.GetLocalIP().String()
		provider.ReadConfig()
		provider.Launch()
	default:
		slog.Error("Unknow mode")
		os.Exit(0)
		// log.Fatal("Unknow mode")
	}
}
func setupLogger() {
	// 1. Lecture de la variable d'env (Défaut : INFO)
	levelStr := os.Getenv("LOG_LEVEL")
	var level slog.Level

	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		level = slog.LevelDebug
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// 2. Configuration du logger (format JSON pour la prod/Docker)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))

	slog.SetDefault(logger)
}
