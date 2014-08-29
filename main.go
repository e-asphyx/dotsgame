package main

import (
	"log"
	"net/http"
	"io"
	"strconv"
	"container/list"
	"code.google.com/p/go.net/websocket"
)

type Point struct {
	X uint `json:"x"`
	Y uint `json:"y"`
}

type GameMessage struct {
	CID uint `json:"cid"`
	Points []Point `json:"points,omitempty"`
	UpdArea uint `json:"updarea"`
	Area [][]Point `json:"area,omitempty"`
}

type Client struct {
	cid uint
	server *GameServer
	msg chan *GameMessage
}

type GameServer struct {
	add chan *Client
	remove chan *Client
	msg chan *GameMessage
}

func (srv *GameServer) gameServer() {
	clients := list.New()
	/* main loop */
	for {
		select {
			case cl := <-srv.add:
				clients.PushBack(cl)

			case cl := <-srv.remove:
				for e := clients.Front(); e != nil; e = e.Next() {
					if e.Value.(*Client) == cl {
						clients.Remove(e)
						break
					}
				}

			case msg := <-srv.msg:
				for e := clients.Front(); e != nil; e = e.Next() {
					client := e.Value.(*Client)

					if client.cid != msg.CID {
						client.msg <- msg
					}
				}
		}
	}
}

func (srv *GameServer) NewClient(cid uint) *Client {
	client := Client {
		cid: cid,
		msg: make(chan *GameMessage, 32),
		server: srv,
	}
	srv.add <- &client

	return &client
}

func (srv *GameServer) Post(msg *GameMessage) {
	srv.msg <- msg
}

func NewGameServer() *GameServer {
	srv := GameServer {
		add: make(chan *Client),
		remove: make(chan *Client),
		msg: make(chan *GameMessage, 32),
	}
	go srv.gameServer()
	return &srv
}

func (client *Client) Cancel() {
	client.server.remove <- client
}

/*-------------------------------------------------------------------------------*/

var gameserver *GameServer

func WebSocketServer(ws *websocket.Conn) {
	tmp := ws.Request().FormValue("cid")
	if tmp == "" {
		return
	}
	cid, _ := strconv.ParseUint(tmp, 10, 0)

	log.Printf("Connected CID: %d\n", cid)

	/* WebSocket reading wrapper */
	incoming := make(chan *GameMessage)
	go func() {
		for {
			msg := new(GameMessage)
			err := websocket.JSON.Receive(ws, msg)
			if err != nil {
				if err == io.EOF {
					close(incoming)
					return
				}
				/* skip unmarshalling errors */
			} else if msg.CID == uint(cid) {
				incoming <- msg
			}
		}
	}()

	client := gameserver.NewClient(uint(cid))
	defer client.Cancel()

	/* main loop */
	for {
		select {
			case msg, ok := <-incoming:
				if !ok {return}
				gameserver.Post(msg)

			case msg := <-client.msg:
				err := websocket.JSON.Send(ws, msg);
				if err != nil {return}
		}
	}
}

func main() {
	log.Println("Start")

	gameserver = NewGameServer()

	/* Serve WebSocket */
	http.Handle("/websocket", websocket.Handler(WebSocketServer))

	/* Serve static */
	http.Handle("/static/", http.FileServer(http.Dir("")))

	/* Start server */
	log.Fatal(http.ListenAndServe(":8080", nil))
}
