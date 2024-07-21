package main

import (
	"bytes"
	"log"
	"sync"
	"text/template"
)

type Message struct {
	ClientId string
	Text     string
}

type WsMessage struct {
	Headers interface{} `json:"HEADERS"`
	Text    string      `json:"text"`
}

type Hub struct {
	sync.Mutex
	clients  map[*Client]bool
	messages []*Message

	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan *Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.Lock()
			h.clients[client] = true

			log.Printf("Client %s connected", client.id)

			for _, message := range h.messages {
				client.send <- getMessageTemplate(message)
			}

			h.Unlock()

		case client := <-h.unregister:
			h.Lock()
			if _, ok := h.clients[client]; ok {
				close(client.send)
				delete(h.clients, client)
				log.Printf("Client %s disconnected", client.id)
			}
			h.Unlock()
		case message := <-h.broadcast:
			h.Lock()
			h.messages = append(h.messages, message)
			h.Unlock()

			for client := range h.clients {
				select {
				case client.send <- getMessageTemplate(message):
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}

		}
	}
}

func getMessageTemplate(message *Message) []byte {
	tmpl, err := template.ParseFiles("templates/message.html")
	if err != nil {
		log.Println(err)
	}

	var renderedMessage bytes.Buffer
	err = tmpl.Execute(&renderedMessage, message)
	if err != nil {
		log.Println(err)
	}

	return renderedMessage.Bytes()
}
