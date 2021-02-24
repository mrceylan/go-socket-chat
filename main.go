package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"

	gows "github.com/gorilla/websocket"
)

type client struct {
	id   string
	conn *gows.Conn
}

type message struct {
	sender  string
	message []byte
}

var clients map[string]*client

var messageChannel chan message

var upgrader = gows.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {
	clients = make(map[string]*client)
	messageChannel = make(chan message)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})
	http.HandleFunc("/socket", func(w http.ResponseWriter, r *http.Request) {
		startSocket(w, r)
	})
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func startSocket(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Println(err)
		return
	}

	cl := client{generateGUID(), conn}
	clients[cl.id] = &cl

	go cl.readMessages()
	go sendMessages()
}

func (cl *client) readMessages() {
	defer func() {
		delete(clients, cl.id)
	}()
	for {
		_, msg, err := cl.conn.ReadMessage()
		if err != nil {
			if gows.IsUnexpectedCloseError(err, gows.CloseGoingAway, gows.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		messageChannel <- message{cl.id, msg}
	}
}

func sendMessages() {
	for msg := range messageChannel {
		for id, cl := range clients {
			if msg.sender != id {
				cl.conn.WriteMessage(gows.TextMessage, msg.message)
			}
		}
	}
}

func generateGUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal(err)
	}
	uuid := fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return uuid
}
