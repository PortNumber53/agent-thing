package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os/exec"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

// handleDockerShellWS opens an interactive shell in the managed docker container
// and bridges stdin/stdout over a WebSocket.
func handleDockerShellWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("shell websocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	dm := NewDockerManager()
	if err := dm.startContainer(r.Context()); err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("Failed to start container: "+err.Error()+"\n"))
		return
	}

	// Try bash first, then fallback to sh.
	cmd := exec.Command("docker", "exec", "-it", dm.containerName, "/bin/bash")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		cmd = exec.Command("docker", "exec", "-it", dm.containerName, "/bin/sh")
		ptmx, err = pty.Start(cmd)
		if err != nil {
			_ = conn.WriteMessage(websocket.TextMessage, []byte("Failed to start shell: "+err.Error()+"\n"))
			return
		}
	}
	defer func() {
		_ = ptmx.Close()
		_ = cmd.Process.Kill()
	}()

	// Stream PTY -> WS
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 4096)
		for {
			n, readErr := ptmx.Read(buf)
			if n > 0 {
				// Send raw bytes to the client.
				if writeErr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); writeErr != nil {
					return
				}
			}
			if readErr != nil {
				if readErr != io.EOF {
					_ = conn.WriteMessage(websocket.TextMessage, []byte("\n[pty closed]\n"))
				}
				return
			}
		}
	}()

	type resizeMsg struct {
		Type string `json:"type"`
		Rows int    `json:"rows"`
		Cols int    `json:"cols"`
	}

	// WS -> PTY (also accepts resize control messages)
	for {
		messageType, msg, wsErr := conn.ReadMessage()
		if wsErr != nil {
			<-done
			return
		}
		// Accept both text and binary frames; forward bytes as-is.
		switch messageType {
		case websocket.TextMessage:
			// Try to parse resize control message.
			if len(msg) > 0 && msg[0] == '{' {
				var rm resizeMsg
				if err := json.Unmarshal(msg, &rm); err == nil && rm.Type == "resize" && rm.Rows > 0 && rm.Cols > 0 {
					_ = pty.Setsize(ptmx, &pty.Winsize{
						Rows: uint16(rm.Rows),
						Cols: uint16(rm.Cols),
					})
					continue
				}
			}
			_, _ = ptmx.Write(msg)
		case websocket.BinaryMessage:
			_, _ = ptmx.Write(msg)
		}
	}
}
