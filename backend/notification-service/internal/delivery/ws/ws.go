package ws

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan string)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func HandleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	clients[conn] = true
	log.Println("🔌 Client connected")
}

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

func Send(msg string) {
	select {
	case broadcast <- msg:
	default:
		log.Println("⚠️ No websocket listeners")
	}
}
