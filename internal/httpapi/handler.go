// Package httpapi contains HTTP routing and transport-level request handling.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/MAK890/task-manager-api/internal/model"
	"github.com/MAK890/task-manager-api/internal/service"
)

type TaskService interface {
	Create(ctx context.Context, task model.Task) (int64, error)
	List(ctx context.Context) ([]model.Task, error)
	GetByID(ctx context.Context, id string) (model.Task, error)
	Update(ctx context.Context, id string, task model.Task) error
	Delete(ctx context.Context, id string) error
}

type Handler struct {
	tasks  TaskService
	logger *log.Logger
}

type responseRecorder struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	if r.wroteHeader {
		return
	}
	r.ResponseWriter.WriteHeader(statusCode)
	r.status = statusCode
	r.wroteHeader = true
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	written, err := r.ResponseWriter.Write(data)
	r.bytes += written
	return written, err
}

func (r *responseRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func NewHandler(tasks TaskService, logger *log.Logger) *Handler {
	return &Handler{tasks: tasks, logger: logger}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.handleHealthCheck)       // health request nu direct karan layi
	mux.HandleFunc("POST /tasks", h.handleCreateTask)        // task create karan layi
	mux.HandleFunc("GET /tasks", h.handleGetTasks)           // task get karan layi
	mux.HandleFunc("GET /tasks/{id}", h.handleGetTaskByID)   // task get by ID karan layi
	mux.HandleFunc("PUT /tasks/{id}", h.handleUpdateTask)    // task update karan layi
	mux.HandleFunc("DELETE /tasks/{id}", h.handleDeleteTask) // task delete karan layi
	return h.logRequests(h.recoverPanics(mux))
}

func (h *Handler) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		recorder := &responseRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		h.logger.Printf(
			"HTTP request started: method=%s path=%s remote_address=%s user_agent=%q",
			r.Method,
			r.URL.Path,
			r.RemoteAddr,
			r.UserAgent(),
		)
		defer func() {
			h.logger.Printf(
				"HTTP request completed: method=%s path=%s status=%d response_bytes=%d duration=%s",
				r.Method,
				r.URL.Path,
				recorder.status,
				recorder.bytes,
				time.Since(startedAt),
			)
		}()

		next.ServeHTTP(recorder, r)
	})
}

func (h *Handler) recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				h.logger.Printf(
					"HTTP handler panic recovered: method=%s path=%s panic=%s stack=%s",
					r.Method,
					r.URL.Path,
					fmt.Sprint(recovered),
					debug.Stack(),
				)

				if recorder, ok := w.(*responseRecorder); ok && recorder.wroteHeader {
					h.logger.Printf("Could not replace response after panic because headers were already written: method=%s path=%s status=%d", r.Method, r.URL.Path, recorder.status)
					return
				}
				h.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// Ae helper function JSON response write karan layi use hunda.
func (h *Handler) writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Printf("Could not write JSON response: status=%d error=%v", statusCode, err)
	}
}

// Ae health check function server di health request nu respond karda te ok return karda.
func (h *Handler) handleHealthCheck(w http.ResponseWriter, _ *http.Request) {
	h.logger.Println("Health check handled successfully")
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Ae create task function request data decode karda te task service nu call karda.
func (h *Handler) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var task model.Task
	// NewDecoder te Decode JSON library de functions ne.
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		h.logger.Printf("Task creation request JSON decode failed: method=%s path=%s error=%v", r.Method, r.URL.Path, err)
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	id, err := h.tasks.Create(r.Context(), task)
	if h.writeServiceError(w, r, err) {
		return
	}

	h.writeJSON(w, http.StatusCreated, map[string]any{
		"message": "Task created successfully",
		"id":      id,
	})
	h.logger.Printf("Task creation response written: id=%d status=%d", id, http.StatusCreated)
}

// Ae get tasks function Redis/MySQL orchestration task service nu delegate karda.
func (h *Handler) handleGetTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.tasks.List(r.Context())
	if h.writeServiceError(w, r, err) {
		return
	}
	h.writeJSON(w, http.StatusOK, tasks)
}

// Ae get task by ID function request path chon ID retrieve karda.
func (h *Handler) handleGetTaskByID(w http.ResponseWriter, r *http.Request) {
	task, err := h.tasks.GetByID(r.Context(), r.PathValue("id"))
	if h.writeServiceError(w, r, err) {
		return
	}
	h.writeJSON(w, http.StatusOK, task)
}

// Ae update task function request data retrieve kar ke task service nu dinda.
func (h *Handler) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	var task model.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		h.logger.Printf("Task update request JSON decode failed: method=%s path=%s id=%s error=%v", r.Method, r.URL.Path, r.PathValue("id"), err)
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	if err := h.tasks.Update(r.Context(), r.PathValue("id"), task); h.writeServiceError(w, r, err) {
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Task updated successfully"})
}

// Ae delete task function request path chon ID retrieve kar ke task delete karda.
func (h *Handler) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	if err := h.tasks.Delete(r.Context(), r.PathValue("id")); h.writeServiceError(w, r, err) {
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Task deleted successfully"})
}

func (h *Handler) writeServiceError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}

	switch {
	case errors.Is(err, service.ErrTitleRequired):
		h.logger.Printf("Request validation failed: method=%s path=%s error=%v", r.Method, r.URL.Path, err)
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Title is required"})
	case errors.Is(err, service.ErrInvalidPriority):
		h.logger.Printf("Request validation failed: method=%s path=%s error=%v", r.Method, r.URL.Path, err)
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Priority must be low, medium, or high"})
	case errors.Is(err, service.ErrTaskNotFound):
		h.logger.Printf("Requested task was not found: method=%s path=%s id=%s", r.Method, r.URL.Path, r.PathValue("id"))
		h.writeJSON(w, http.StatusNotFound, map[string]string{"error": "Task not found"})
	default:
		h.logger.Printf("Request failed: method=%s path=%s error=%v", r.Method, r.URL.Path, err)
		h.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}
	return true
}
