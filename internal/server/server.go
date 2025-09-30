package server

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// Server manages the web server and WebSocket connections.
type Server struct {
	router          *mux.Router
	upgrader        websocket.Upgrader
	AgentLogic      func(*websocket.Conn)
	allowedOrigins  map[string]struct{}
	allowAllOrigins bool
}

// NewServer creates a new web server.
func NewServer(agentLogic func(*websocket.Conn)) *Server {
	srv := &Server{
		router: mux.NewRouter(),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all connections for development.
				return true
			},
		},
		AgentLogic:      agentLogic,
		allowedOrigins:  make(map[string]struct{}),
		allowAllOrigins: true,
	}

	if originsEnv := strings.TrimSpace(os.Getenv("AGENT_THING_ALLOWED_ORIGINS")); originsEnv != "" {
		srv.allowAllOrigins = false
		for _, origin := range strings.Split(originsEnv, ",") {
			trimmed := strings.TrimSpace(origin)
			if trimmed != "" {
				srv.allowedOrigins[trimmed] = struct{}{}
			}
		}
	}

	return srv
}

// setupRoutes configures the server's routes.
func (s *Server) setupRoutes() {
	s.router.Use(s.corsMiddleware)
	// Handle WebSocket connections first, as PathPrefix is a catch-all.
	s.router.HandleFunc("/ws", s.handleWebSocket)

	// Lightweight health probe
	s.router.HandleFunc("/health", s.handleHealth).Methods(http.MethodGet)

	// Serve the frontend files.
	s.router.PathPrefix("/").Handler(http.FileServer(http.Dir("./public/")))
}

// handleWebSocket upgrades an HTTP connection to a WebSocket and starts the agent logic.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// Start a new goroutine to handle the agent logic for this connection.
	go s.AgentLogic(conn)
}

// handleHealth responds to health probes
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
		log.Printf("Error writing health check response: %v", err)
	}
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := origin != "" && s.isOriginAllowed(origin)
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		}

		if r.Method == http.MethodOptions {
			if allowed {
				w.WriteHeader(http.StatusNoContent)
			} else {
				w.WriteHeader(http.StatusForbidden)
			}
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) isOriginAllowed(origin string) bool {
	if s.allowAllOrigins {
		return true
	}
	_, ok := s.allowedOrigins[origin]
	return ok
}

// Start launches the web server.
func (s *Server) Start(addr string) {
	s.setupRoutes()
	log.Printf("Server starting on %s", addr)
	if err := http.ListenAndServe(addr, s.router); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
