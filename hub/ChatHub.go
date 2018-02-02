package hub

type ChatHub struct {
	register chan *Client
	unregister chan *Client
	broadcasts chan []byte
	clients map[*Client]bool
}

func NewChatHub() *ChatHub {
	return &ChatHub{
		register: make(chan *Client),
		unregister: make(chan *Client),
		broadcasts: make(chan []byte),
		clients: make(map[*Client]bool),
	}
}

func (h *ChatHub) Run()  {
	for {
		select {
			case client := <- h.register:
				h.clients[client] = true
			case client := <- h.unregister:
				if _,ok := h.clients[client]; ok {
					delete(h.clients, client)
					close(client.Send)
				}
			case message := <- h.broadcasts:
				for client := range h.clients {
					select {
						case client.Send <- message:
						default:
							delete(h.clients, client)
							close(client.Send)
					}
				}
		}
	}
}