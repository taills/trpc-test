package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/taills/trpc-test/backend/internal/api"
	"github.com/taills/trpc-test/backend/internal/engine"
	"github.com/taills/trpc-test/backend/internal/registry"
	"github.com/taills/trpc-test/backend/internal/store"
	"github.com/taills/trpc-test/backend/internal/toolreg"
)

func main() {
	// Create empty registries (will be populated dynamically based on workflows)
	toolReg := toolreg.New()
	execReg := engine.NewExecutorRegistry()

	// Create dynamic registry with all available tools and executors
	dynamicReg := registry.NewDynamicRegistry()

	// Run store
	runStore := store.New("")

	// Frontend dir
	frontendDir := "./frontend"
	if dir := os.Getenv("FRONTEND_DIR"); dir != "" {
		frontendDir = dir
	}

	// API handler with dynamic registry
	handler := api.NewHandler(runStore, execReg, toolReg, dynamicReg, frontendDir)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Starting server on :%s\n", port)
	fmt.Println("Tools and executors will be registered dynamically based on workflow definitions")
	log.Fatal(http.ListenAndServe(":"+port, handler))
}
