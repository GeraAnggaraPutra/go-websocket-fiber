package main

import (
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

type Message struct {
	Name  string
	Price string
}

type hub struct {
	clients               map[*websocket.Conn]bool
	clientRegisterChannel chan *websocket.Conn
	clientRemovalChannel  chan *websocket.Conn
	broadcastMessage      chan Message
}

func (h *hub) run() {
	for {
		select {
		case conn := <-h.clientRegisterChannel:
			h.clients[conn] = true
		case conn := <-h.clientRemovalChannel:
			delete(h.clients, conn)
		case msg := <-h.broadcastMessage:
			for conn := range h.clients {
				_ = conn.WriteJSON(msg)
			}
		}
	}
}

func main() {
	h := &hub{
		clients:               make(map[*websocket.Conn]bool),
		clientRegisterChannel: make(chan *websocket.Conn),
		clientRemovalChannel:  make(chan *websocket.Conn),
		broadcastMessage:      make(chan Message),
	}

	go h.run()

	app := fiber.New()
	app.Use("/ws", AllowUpgrade)
	app.Use("/ws/bid", websocket.New(BidPrice(h)))

	_ = app.Listen(":3000")
}

func AllowUpgrade(ctx *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(ctx) {
		return ctx.Next()
	}

	return fiber.ErrUpgradeRequired
}

func BidPrice(h *hub) func(*websocket.Conn) {
	return func(conn *websocket.Conn) {
		defer func() {
			h.clientRemovalChannel <- conn
			_ = conn.Close()
		}()

		name := conn.Query("name", "anonymous")
		h.clientRegisterChannel <- conn

		for {
			messageType, price, err := conn.ReadMessage()
			if err != nil {
				return
			}

			if messageType == websocket.TextMessage {
				h.broadcastMessage <- Message{
					Name:  name,
					Price: string(price),
				}
			}
		}
	}
}
