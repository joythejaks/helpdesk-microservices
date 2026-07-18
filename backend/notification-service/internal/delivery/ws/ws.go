package ws

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

const shutdownWriteTimeout = 2 * time.Second

type client struct {
	conn   *websocket.Conn
	role   string
	userID uint
}

// outbound is a queued message plus its routing target — exactly one of
// targetUserID/targetRoles should be set.
type outbound struct {
	payload      string
	targetUserID *uint
	targetRoles  []string
}

var (
	clients   = make(map[*websocket.Conn]*client)
	clientsMu sync.RWMutex
	broadcast = make(chan outbound, 256)

	jwtSecret      []byte
	allowedOrigins []string
	maxConnections int
)

// Init wires the runtime config the ws package needs. Must be called once
// from main before any HTTP handlers are registered.
func Init(secret []byte, origins []string, maxConns int) {
	jwtSecret = secret
	allowedOrigins = origins
	maxConnections = maxConns
}

var upgrader = websocket.Upgrader{
	CheckOrigin: checkOrigin,
}

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		// Non-browser clients (e.g. the Flutter app) typically don't send
		// an Origin header at all — nothing to check against.
		return true
	}
	for _, o := range allowedOrigins {
		if o == "*" || o == origin {
			return true
		}
	}
	return false
}

func HandleConnections(w http.ResponseWriter, r *http.Request) {
	tokenString := r.URL.Query().Get("token")
	if tokenString == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// cegah algorithm confusion attack
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		log.Println("❌ Invalid WS token:", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	role, _ := claims["role"].(string)
	var userID uint
	if idFloat, ok := claims["user_id"].(float64); ok {
		userID = uint(idFloat)
	}

	clientsMu.RLock()
	full := len(clients) >= maxConnections
	clientsMu.RUnlock()
	if full {
		http.Error(w, "too many connections", http.StatusServiceUnavailable)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	c := &client{conn: conn, role: role, userID: userID}
	clientsMu.Lock()
	clients[conn] = c
	clientsMu.Unlock()

	log.Println("🔐 WebSocket connected (authorized), role:", role, "user_id:", userID)

	// goroutine untuk detect disconnect dan cleanup
	go func() {
		defer func() {
			clientsMu.Lock()
			delete(clients, conn)
			clientsMu.Unlock()
			conn.Close()
			log.Println("🔌 WebSocket disconnected, client removed")
		}()

		for {
			// ReadMessage blocks — error berarti client disconnect
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

func HandleMessages() {
	for out := range broadcast {
		clientsMu.Lock()
		for conn, c := range clients {
			if !matches(c, out) {
				continue
			}
			if err := conn.WriteMessage(websocket.TextMessage, []byte(out.payload)); err != nil {
				conn.Close()
				delete(clients, conn)
			}
		}
		clientsMu.Unlock()
	}
}

func matches(c *client, out outbound) bool {
	if out.targetUserID != nil {
		return c.userID == *out.targetUserID
	}
	for _, r := range out.targetRoles {
		if c.role == r {
			return true
		}
	}
	return false
}

// SendToUser delivers payload to the single connected client (if any) whose
// JWT carried this user_id — e.g. "your ticket status changed".
func SendToUser(userID uint, payload string) {
	enqueue(outbound{payload: payload, targetUserID: &userID})
}

// SendToRoles delivers payload to every connected client whose JWT role is
// in roles — e.g. "new ticket" going out to admin+agent.
func SendToRoles(roles []string, payload string) {
	enqueue(outbound{payload: payload, targetRoles: roles})
}

func enqueue(out outbound) {
	select {
	case broadcast <- out:
	default:
		log.Println("⚠️ notification broadcast queue full, dropping message")
	}
}

// CloseAll closes every active WebSocket connection. Used during graceful
// shutdown — http.Server.Shutdown doesn't manage connections that have
// already been hijacked for a WS upgrade, so they need to be closed explicitly.
func CloseAll() {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	for conn := range clients {
		_ = conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseServiceRestart, "server shutting down"),
			time.Now().Add(shutdownWriteTimeout))
		conn.Close()
		delete(clients, conn)
	}
}
