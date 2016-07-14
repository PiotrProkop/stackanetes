package main

import (
	"os"

	entry "github.com/stackanetes/docker-entrypoint/dependencies"

	"github.com/stackanetes/docker-entrypoint/logger"
	comm "github.com/stackanetes/docker-entrypoint/util/command"
	"github.com/stackanetes/docker-entrypoint/util/env"

	//Register resolvers
	_ "github.com/stackanetes/docker-entrypoint/dependencies/config"
	_ "github.com/stackanetes/docker-entrypoint/dependencies/daemonset"
	_ "github.com/stackanetes/docker-entrypoint/dependencies/job"
	_ "github.com/stackanetes/docker-entrypoint/dependencies/service"
)

func main() {
	entrypoint := entry.NewEntrypoint()
	err := entrypoint.Resolve()
	if err != nil {
		logger.Error.Printf("Failed to resolve dependecy: %v", err)
		os.Exit(1)
	}
	command := os.Getenv("COMMAND")
	if command == "" {
		logger.Error.Printf("COMMAND env is empty")
		os.Exit(1)
	}
	comm.ExecuteCommand(env.SplitEnvToList("COMMAND", " "))
}
