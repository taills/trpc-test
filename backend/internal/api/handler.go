package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/taills/trpc-test/backend/internal/dsl"
	"github.com/taills/trpc-test/backend/internal/engine"
	"github.com/taills/trpc-test/backend/internal/registry"
	"github.com/taills/trpc-test/backend/internal/store"
	"github.com/taills/trpc-test/backend/internal/toolreg"
)

type Handler struct {
	mu              sync.RWMutex
	workflows       map[string]*dsl.Workflow
	runStore        *store.Store
	execRegistry    *engine.ExecutorRegistry
	toolRegistry    *toolreg.Registry
	dynamicRegistry *registry.DynamicRegistry
	frontendDir     string
}

func NewHandler(runStore *store.Store, execRegistry *engine.ExecutorRegistry, toolRegistry *toolreg.Registry, dynamicRegistry *registry.DynamicRegistry, frontendDir string) *Handler {
	return &Handler{
		workflows:       make(map[string]*dsl.Workflow),
		runStore:        runStore,
		execRegistry:    execRegistry,
		toolRegistry:    toolRegistry,
		dynamicRegistry: dynamicRegistry,
		frontendDir:     frontendDir,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case path == "/api/workflows" && r.Method == "POST":
		h.createWorkflow(w, r)
	case path == "/api/workflows" && r.Method == "GET":
		h.listWorkflows(w, r)
	case strings.HasPrefix(path, "/api/workflows/") && strings.HasSuffix(path, "/run") && r.Method == "POST":
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/api/workflows/"), "/run")
		h.runWorkflow(w, r, id)
	case strings.HasPrefix(path, "/api/workflows/") && r.Method == "GET":
		id := strings.TrimPrefix(path, "/api/workflows/")
		h.getWorkflow(w, r, id)
	case path == "/api/runs" && r.Method == "GET":
		h.listRuns(w, r)
	case strings.HasPrefix(path, "/api/runs/") && r.Method == "GET":
		id := strings.TrimPrefix(path, "/api/runs/")
		h.getRun(w, r, id)
	case path == "/" || path == "/index.html":
		http.ServeFile(w, r, h.frontendDir+"/index.html")
	default:
		http.FileServer(http.Dir(h.frontendDir)).ServeHTTP(w, r)
	}
}

func (h *Handler) createWorkflow(w http.ResponseWriter, r *http.Request) {
	var wf dsl.Workflow
	if err := json.NewDecoder(r.Body).Decode(&wf); err != nil {
		jsonError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if wf.ID == "" {
		wf.ID = uuid.New().String()
	}
	if err := dsl.Validate(&wf); err != nil {
		jsonError(w, "validation error: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Dynamically register required tools and executors
	if err := h.dynamicRegistry.RegisterFromWorkflow(&wf, h.toolRegistry, h.execRegistry); err != nil {
		jsonError(w, "registration error: "+err.Error(), http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	h.workflows[wf.ID] = &wf
	h.mu.Unlock()
	jsonResponse(w, map[string]string{"id": wf.ID, "name": wf.Name}, http.StatusCreated)
}

func (h *Handler) listWorkflows(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	list := make([]*dsl.Workflow, 0, len(h.workflows))
	for _, wf := range h.workflows {
		list = append(list, wf)
	}
	h.mu.RUnlock()
	jsonResponse(w, list, http.StatusOK)
}

func (h *Handler) getWorkflow(w http.ResponseWriter, r *http.Request, id string) {
	h.mu.RLock()
	wf, ok := h.workflows[id]
	h.mu.RUnlock()
	if !ok {
		jsonError(w, fmt.Sprintf("workflow %q not found", id), http.StatusNotFound)
		return
	}
	jsonResponse(w, wf, http.StatusOK)
}

func (h *Handler) runWorkflow(w http.ResponseWriter, r *http.Request, id string) {
	h.mu.RLock()
	wf, ok := h.workflows[id]
	h.mu.RUnlock()
	if !ok {
		jsonError(w, fmt.Sprintf("workflow %q not found", id), http.StatusNotFound)
		return
	}
	result, err := engine.Run(r.Context(), wf, h.execRegistry)
	if err != nil {
		if saveErr := h.runStore.Save(result); saveErr != nil {
			fmt.Printf("warning: failed to save run result: %v\n", saveErr)
		}
		jsonResponse(w, map[string]string{"run_id": result.RunID, "status": "error"}, http.StatusOK)
		return
	}
	if saveErr := h.runStore.Save(result); saveErr != nil {
		fmt.Printf("warning: failed to save run result: %v\n", saveErr)
	}
	jsonResponse(w, map[string]string{"run_id": result.RunID, "status": result.Status}, http.StatusOK)
}

func (h *Handler) listRuns(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, h.runStore.List(), http.StatusOK)
}

func (h *Handler) getRun(w http.ResponseWriter, r *http.Request, id string) {
	run, err := h.runStore.Get(id)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}
	jsonResponse(w, run, http.StatusOK)
}

func jsonResponse(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	jsonResponse(w, map[string]string{"error": msg}, status)
}
