package api

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type WebSocketServer struct {
	connectedClients   map[*websocket.Conn]bool
	messageChannel     chan []byte
	clientMutex        sync.Mutex
	connectionUpgrader websocket.Upgrader
}

func NewWebSocketServer() *WebSocketServer {
	return &WebSocketServer{
		connectedClients: make(map[*websocket.Conn]bool),
		messageChannel: make(chan []byte),
		connectionUpgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (wsServer *WebSocketServer) HandleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := wsServer.connectionUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	wsServer.clientMutex.Lock()
	wsServer.connectedClients[conn] = true
	wsServer.clientMutex.Unlock()

	defer func() {
		wsServer.clientMutex.Lock()
		delete(wsServer.connectedClients, conn)
		wsServer.clientMutex.Unlock()
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

func (wsServer *WebSocketServer) Broadcast(message []byte) {
	wsServer.clientMutex.Lock()
	defer wsServer.clientMutex.Unlock()

	for client := range wsServer.connectedClients {
		if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("WebSocket write error: %v", err)
			client.Close()
			delete(wsServer.connectedClients, client)
		}
	}
}

func StartWebSocketServer(wsServer *WebSocketServer, port string) {
	http.HandleFunc("/ws", wsServer.HandleConnections)
	go func() {
		log.Printf("WebSocket server running on ws://localhost:%s/ws", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatalf("WebSocket server failed: %v", err)
		}
	}()
}