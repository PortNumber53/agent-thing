package server

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// Server manages the web server and WebSocket connections.
type Server struct {
	router     *mux.Router
	upgrader   websocket.Upgrader
	AgentLogic func(*websocket.Conn)
}

// NewServer creates a new web server.
func NewServer(agentLogic func(*websocket.Conn)) *Server {
	return &Server{
		router: mux.NewRouter(),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all connections for development.
				return true
			},
		},
		AgentLogic: agentLogic,
	}
}

// setupRoutes configures the server's routes.
func (s *Server) setupRoutes() {
	// Handle WebSocket connections first, as PathPrefix is a catch-all.
	s.router.HandleFunc("/ws", s.handleWebSocket)

	// Lightweight health probe
	s.router.HandleFunc("/health", s.handleHealth).Methods(http.MethodGet)

	// Serve the frontend files.
	s.router.PathPrefix("/").Handler(http.FileServer(http.Dir("./frontend/")))
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
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// Start launches the web server.
func (s *Server) Start(addr string) {
	s.setupRoutes()
	log.Printf("Server starting on %s", addr)
	if err := http.ListenAndServe(addr, s.router); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
