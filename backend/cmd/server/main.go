package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/taills/trpc-test/backend/internal/api"
	"github.com/taills/trpc-test/backend/internal/engine"
	"github.com/taills/trpc-test/backend/internal/executors"
	"github.com/taills/trpc-test/backend/internal/store"
	"github.com/taills/trpc-test/backend/internal/toolreg"
)

func main() {
	// Tool registry
	toolReg := toolreg.New()
	toolreg.RegisterEchoTool(toolReg)
	toolreg.RegisterHTTPGetTool(toolReg)

	// Executor registry
	execReg := engine.ExecutorRegistry{
		"start": &executors.StartExecutor{},
		"end":   &executors.EndExecutor{},
		"agent": &executors.AgentExecutor{ToolRegistry: toolReg},
		"http":  &executors.HTTPNodeExecutor{},
		"echo":  &executors.StartExecutor{}, // echo node just passes through
	}

	// Run store
	runStore := store.New("")

	// Frontend dir
	frontendDir := "./frontend"
	if dir := os.Getenv("FRONTEND_DIR"); dir != "" {
		frontendDir = dir
	}

	// API handler
	handler := api.NewHandler(runStore, execReg, toolReg, frontendDir)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Starting server on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}
