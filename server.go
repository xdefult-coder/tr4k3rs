package main

// server.go
// - HTTP API: POST /report (expects Authorization: Bearer <token>)
// - GET  /get/{phone} -> returns recent locations (JSON)
// - Serves viewer.html at / and static files at /static/
// - WebSocket /ws broadcasts live Location to connected viewers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Location struct {
	Phone string  `json:"phone"`
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
	When  string  `json:"when"`
	IP    string  `json:"ip,omitempty"`
}

var (
	// in-memory stores
	store   = map[string][]Location{}
	stMutex = sync.RWMutex{}

	// simple token store (phone -> token)
	tokens   = map[string]string{}
	tokensMu = sync.RWMutex{}

	// websockets
	clients   = make(map[*websocket.Conn]bool)
	clientsMu = sync.Mutex{}
	upgrader  = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

func main() {
	// OPTIONAL: initialize a demo token for "kali-device"
	tokensMu.Lock()
	tokens["kali-device"] = "mytoken123" // change in production
	tokensMu.Unlock()

	r := mux.NewRouter()
	r.HandleFunc("/report", reportHandler).Methods("POST")
	r.HandleFunc("/get/{phone}", getHandler).Methods("GET")
	r.HandleFunc("/ws", wsHandler)
	// Serve static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	// viewer
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "./viewer.html") })

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}
	addr := fmt.Sprintf(":%s", port)
	log.Printf("Starting server on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

func validToken(phone, token string) bool {
	tokensMu.RLock()
	defer tokensMu.RUnlock()
	if t, ok := tokens[phone]; ok {
		return t == token
	}
	return false
}

func reportHandler(w http.ResponseWriter, r *http.Request) {
	// Expect JSON body and Authorization header
	auth := r.Header.Get("Authorization")
	var token string
	if len(auth) > 7 && auth[:7] == "Bearer " {
		token = auth[7:]
	}
	var p struct {
		Phone string  `json:"phone"`
		Lat   float64 `json:"lat"`
		Lon   float64 `json:"lon"`
		IP    string  `json:"ip,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if p.Phone == "" {
		http.Error(w, "phone required", http.StatusBadRequest)
		return
	}
	if !validToken(p.Phone, token) {
		http.Error(w, "invalid token", http.StatusForbidden)
		return
	}
	loc := Location{
		Phone: p.Phone,
		Lat:   p.Lat,
		Lon:   p.Lon,
		When:  time.Now().UTC().Format(time.RFC3339),
		IP:    p.IP,
	}
	// store
	stMutex.Lock()
	store[p.Phone] = append(store[p.Phone], loc)
	if len(store[p.Phone]) > 500 {
		store[p.Phone] = store[p.Phone][len(store[p.Phone])-500:]
	}
	stMutex.Unlock()

	// broadcast
	broadcast(loc)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	phone := vars["phone"]
	stMutex.RLock()
	locs := store[phone]
	stMutex.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"phone": phone, "locations": locs})
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("ws upgrade:", err)
		return
	}
	clientsMu.Lock()
	clients[conn] = true
	clientsMu.Unlock()

	// keep connection until error
	for {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			clientsMu.Lock()
			delete(clients, conn)
			clientsMu.Unlock()
			conn.Close()
			break
		}
	}
}

func broadcast(loc Location) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	for c := range clients {
		if err := c.WriteJSON(loc); err != nil {
			// remove bad client
			c.Close()
			delete(clients, c)
		}
	}
}
