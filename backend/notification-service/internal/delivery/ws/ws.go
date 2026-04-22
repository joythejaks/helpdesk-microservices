package ws

import (
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var (
	clients   = make(map[*websocket.Conn]bool)
	clientsMu sync.RWMutex
	broadcast = make(chan string, 256)
)

func getSecret() []byte {
	return []byte(os.Getenv("JWT_SECRET"))
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func HandleConnections(w http.ResponseWriter, r *http.Request) {
	tokenString := r.URL.Query().Get("token")

	if tokenString == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return getSecret(), nil
	})

	if err != nil || !token.Valid {
		log.Println("❌ Invalid WS token:", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	clientsMu.Lock()
	clients[conn] = true
	clientsMu.Unlock()

	log.Println("🔐 WebSocket connected (authorized)")
}

func HandleMessages() {
	for msg := range broadcast {
		clientsMu.Lock()
		for client := range clients {
			err := client.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				client.Close()
				delete(clients, client)
			}
		}
		clientsMu.Unlock()
	}
}

func Send(msg string) {
	select {
	case broadcast <- msg:
	default:
		log.Println("⚠️ No websocket listeners")
	}
}
