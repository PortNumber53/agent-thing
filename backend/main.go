package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const defaultListenAddr = ":18511"

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in dev.
		return true
	},
}

func main() {
	listenAddr := getListenAddress()

	// Load .env / backend/.env if present.
	loadDotEnv()

	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if maybeHandleMigrateSubcommand(cfg) {
		return
	}

	_, dbErr := ConnectDB(cfg)
	if dbErr != nil {
		log.Fatalf("failed to connect db: %v", dbErr)
	}

	dockerManager := NewDockerManager()
	googleAuth := NewGoogleAuthHandler(cfg)
	stripeHandler := NewStripeHandler(cfg)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/ws", handleWebSocketTimeStream)
	mux.HandleFunc("/docker/status", withCors(dockerManager.handleStatus))
	mux.HandleFunc("/docker/start", withCors(dockerManager.handleStart))
	mux.HandleFunc("/docker/stop", withCors(dockerManager.handleStop))
	mux.HandleFunc("/docker/rebuild", withCors(dockerManager.handleRebuild))
	mux.HandleFunc("/docker/shell", handleDockerShellWS)
	mux.HandleFunc("/auth/google/login", withCors(googleAuth.handleLogin))
	mux.HandleFunc("/callback/oauth/google", withCors(googleAuth.handleCallback))
	mux.HandleFunc("/billing/create-checkout-session", withCors(stripeHandler.handleCreateCheckoutSession))
	// Stripe webhooks (canonical path in prod):
	mux.HandleFunc("/webhook/stripe", withCors(stripeHandler.handleWebhook))
	// Backwards-compatible alias:
	mux.HandleFunc("/billing/webhook", withCors(stripeHandler.handleWebhook))

	log.Printf("Backend listening on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, corsHandler(mux)); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}

func getListenAddress() string {
	if envAddr := strings.TrimSpace(os.Getenv("AGENT_THING_LISTEN_ADDR")); envAddr != "" {
		return envAddr
	}

	if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
		if strings.HasPrefix(port, ":") {
			return port
		}
		return ":" + port
	}

	return defaultListenAddr
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func handleWebSocketTimeStream(w http.ResponseWriter, r *http.Request) {
	connection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}
	defer connection.Close()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case currentTime := <-ticker.C:
			message := currentTime.UTC().Format(time.RFC3339Nano)
			if err := connection.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
				log.Printf("websocket write failed: %v", err)
				return
			}
		case <-r.Context().Done():
			return
		}
	}
}

func withCors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func corsHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		} else {
			w.Header().Set("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeJson(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
