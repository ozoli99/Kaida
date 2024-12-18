package api

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type WebSocketServer struct {
	clients map[*websocket.Conn]bool
	broadcast chan []byte
	mutex sync.Mutex
	upgrader websocket.Upgrader
}

func NewWebSocketServer() *WebSocketServer {
	return &WebSocketServer{
		clients: make(map[*websocket.Conn]bool),
		broadcast: make(chan []byte),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (ws *WebSocketServer) HandleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	ws.mutex.Lock()
	ws.clients[conn] = true
	ws.mutex.Unlock()

	defer func() {
		ws.mutex.Lock()
		delete(ws.clients, conn)
		ws.mutex.Unlock()
		conn.Close()
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket connection error: %v", err)
			break
		}
	}
}

func (ws *WebSocketServer) Broadcast(message []byte) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	for client := range ws.clients {
		if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("WebSocket write error: %v", err)
			client.Close()
			delete(ws.clients, client)
		}
	}
}

func StartWebSocketServer(server *WebSocketServer, port string) {
	http.HandleFunc("/ws", server.HandleConnections)
	go func() {
		log.Printf("WebSocket server running on ws://localhost:%s/ws", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatalf("WebSocket server failed: %v", err)
		}
	}()
}