package hub

import (
	"github.com/gorilla/websocket"
	"net/http"
	"log"
	"time"
	"bytes"
	"math/rand"
	"encoding/json"
)

const (
	maxMessageSize = 512
	writeWait = 10 * time.Second
	pongWait = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	mapX = 300
	mapY = 200
)
type Client struct {
	ID int
	Conn *websocket.Conn
	Send chan []byte
	ChatHub *ChatHub
	Position *Position
}

type Position struct {
	PositionX,
	PositionY int
}

type Message struct {
	Type string
	Value string
}

var upgrader = websocket.Upgrader{
	ReadBufferSize: 1024,
	WriteBufferSize: 1024,
}

var (
	newLine = []byte{'\n'}
	space = []byte{' '}
	ID = 0
)

func ServeChat(hub *ChatHub, w http.ResponseWriter, r *http.Request)  {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
		return
	}
	ID++
	client := &Client{
		ID: ID,
		Conn: conn,
		Send: make(chan []byte, 512),
		ChatHub: hub,
		Position: NewPosition(),
	}
	client.ChatHub.register <- client
	res := client.ReturnResponse(client, "init")
	client.Send <- res
	go client.readPump()
	go client.writePump()

}

func NewPosition() *Position {
	return &Position{
		PositionX: generateRandom(300),
		PositionY: generateRandom(200),
	}
}

func generateRandom(n int) int {
	rand.Seed(time.Now().Unix() * rand.Int63n(1000))
	num := rand.Intn(n)
	if num % 9 == 1 || num % 5 == 1 {
		num = -num
	}
	return num
}

func (c *Client) writePump()  {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
			case message, ok := <- c.Send:
				c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
				if !ok {
					c.Conn.WriteMessage(websocket.CloseMessage, nil)
					return
				}
				w, err := c.Conn.NextWriter(websocket.TextMessage)
				if err != nil {
					return
				}
				w.Write(message)
				n := len(c.Send)
				for i := 0; i < n; i++ {
					w.Write(newLine)
					w.Write(<- c.Send)
				}
				if ok := w.Close(); ok != nil {
					return
				}
			case <- ticker.C:
				c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
		}
	}

}

func (c *Client) readPump()  {
	defer func() {
		c.ChatHub.unregister <- c
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(appData string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Fatalf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newLine, space, -1))
		response := c.handleMessage(message)
		c.ChatHub.broadcasts <- response
	}
}

func (c *Client) handleMessage(message []byte) []byte {
	var msg Message
	err := json.Unmarshal(message, &msg)
	if err != nil {
		return nil
	}
	var response []byte
	switch msg.Type {
		case "move" :
			c.changePosition(msg.Value)
			response = c.ReturnResponse(c, "move")
		case "chat" :
			response = c.ReturnResponse(msg.Value, "chat")
	}
	return response
}

func (c *Client) changePosition(pos string)  {
	var position Position
	err := json.Unmarshal([]byte(pos), &position)
	if err != nil {
		return
	}
	if c.Position.PositionX + position.PositionX < mapX && c.Position.PositionX + position.PositionX > -mapX {
		c.Position.PositionX += position.PositionX
	}
	if c.Position.PositionY + position.PositionY < mapY && c.Position.PositionY + position.PositionY > -mapY {
		c.Position.PositionY += position.PositionY
	}
}

type Response struct {
	Code int
	Response *Data
}

type Data struct {
	ID int
	Type string
	Data interface{}
}

func (c *Client) ReturnResponse(data interface{}, t string) []byte {
	switch client := data.(type) {
		case *Client:
			d := &Data{
				ID: client.ID,
				Type: t,
				Data: client.Position,
			}
			response := &Response{
				Code: 200,
				Response: d,
			}
			res, err := json.Marshal(response)
			if err != nil {
				return nil
			}
			return res
		case string :
			d := &Data{
				ID: c.ID,
				Type: t,
				Data: string(client),
			}
			response := &Response{
				Code: 200,
				Response: d,
			}
			res, err := json.Marshal(response)
			if err != nil {
				return nil
			}
			return res
	}
	return nil
}
