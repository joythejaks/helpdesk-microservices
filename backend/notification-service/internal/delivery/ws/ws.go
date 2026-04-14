package ws

import (
	"log"
	"net/http"
	"os"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan string)

// ambil secret dari env
var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// 👉 boleh true untuk dev, nanti bisa dibatasi domain
		return true
	},
}

// =======================
// 🔥 HANDLE CONNECTION (AUTH)
// =======================
func HandleConnections(w http.ResponseWriter, r *http.Request) {

	// ambil token dari query
	tokenString := r.URL.Query().Get("token")

	if tokenString == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	// parse JWT
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	// upgrade connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	clients[conn] = true
	log.Println("🔐 WebSocket client connected (authenticated)")
}

// =======================
// 🔥 HANDLE BROADCAST
// =======================
func HandleMessages() {
	for {
		msg := <-broadcast

		for client := range clients {
			err := client.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				client.Close()
				delete(clients, client)
			}
		}
	}
}

// =======================
// 🔥 SEND MESSAGE
// =======================
func Send(msg string) {
	select {
	case broadcast <- msg:
	default:
		log.Println("⚠️ No websocket listeners")
	}
}
