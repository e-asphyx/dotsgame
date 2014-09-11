package main

import (
	"log"
	"container/list"
)

type Point struct {
	X uint `json:"x"`
	Y uint `json:"y"`
}

type GameMessage struct {
	roomId uint64 `json:"-"`
	sender *Client `json:"-"`

	CID uint64 `json:"cid"`
	Flags uint `json:"fl"`

	Points map[string][]Point `json:"p,omitempty"`
	Areas map[string][][]Point `json:"a,omitempty"`

	Players map[string]string `json:"players,omitempty"`
	Leave []uint64 `json:"leave,omitempty"`

	sync chan<- bool `json:"-"`
}

type Client struct {
	cid uint64
	roomId uint64
	server *GameServer
	msg chan *GameMessage
}

type GameServer struct {
	add chan *Client
	remove chan *Client
	msg chan *GameMessage
	roomId uint64
	pool *GamePool

	ref uint
}

type gamePoolMsg struct {
	roomId uint64
	reply chan<- *GameServer
}

type GamePool struct {
	req chan *gamePoolMsg
	get chan uint64
	put chan uint64
}

func (srv *GameServer) gameServer() {
	clients := list.New()

	/* main loop */
	for {
		select {
		case cl, ok := <-srv.add:
			if !ok {return}
			clients.PushBack(cl)

			/* load history */
			hist, err := db.LoadHistory(srv.roomId)
			if err != nil {
				log.Printf("db.LoadHistory: %s\n", err.Error())
			} else if len(hist.Points) != 0 || len(hist.Areas) != 0 || len(hist.Players) != 0 {
				cl.msg <- hist
			}

		case cl := <-srv.remove:
			for e := clients.Front(); e != nil; e = e.Next() {
				if e.Value.(*Client) == cl {
					clients.Remove(e)
					break
				}
			}

		case msg := <-srv.msg:
			/* post history */
			if err := db.PostHistory(msg); err != nil {
				log.Printf("db.PostHistory: %s\n", err.Error())
			}
			/* TODO leave */

			if msg.sync != nil {
				msg.sync <- true
			}

			for e := clients.Front(); e != nil; e = e.Next() {
				client := e.Value.(*Client)

				if msg.sender != client {
					client.msg <- msg
				}
			}
		}
	}
}

func (srv *GameServer) NewClient(cid uint64) *Client {
	client := Client {
		cid: cid,
		roomId: srv.roomId,
		msg: make(chan *GameMessage, 32),
		server: srv,
	}
	srv.add <- &client

	return &client
}

func (srv *GameServer) Post(msg *GameMessage) {
	srv.msg <- msg
}

func (srv *GameServer) Get() *GameServer {
	srv.pool.get <- srv.roomId
	return srv
}

func (srv *GameServer) Put() *GameServer {
	srv.pool.put <- srv.roomId
	return srv
}

func (srv *GameServer) cancel() {
	close(srv.add)
}

func newGameServer(pool *GamePool, roomId uint64) *GameServer {
	srv := GameServer {
		add: make(chan *Client),
		remove: make(chan *Client),
		msg: make(chan *GameMessage, 32),
		roomId: roomId,
		pool: pool,
		ref: 1,
	}

	go srv.gameServer()

	return &srv
}

func (client *Client) Cancel() {
	client.server.remove <- client
}

func (srv *GamePool) Get(roomId uint64) *GameServer {
	reply := make(chan *GameServer)
	srv.req <- &gamePoolMsg{roomId, reply}
	return <-reply
}

func NewGamePool() *GamePool {
	pool := GamePool {
		req: make(chan *gamePoolMsg),
		get: make(chan uint64),
		put: make(chan uint64),
	}
	go pool.gamePool()
	return &pool
}

func (pool *GamePool) gamePool() {
	servers := make(map[uint64]*GameServer)

	for {
		select {
		case req := <-pool.req:
			srv, ok := servers[req.roomId]
			if ok {
				srv.ref++
			} else {
				srv = newGameServer(pool, req.roomId)
				servers[req.roomId] = srv
			}
			req.reply <- srv

		case id := <-pool.get:
			srv, ok := servers[id]
			if ok {
				srv.ref++
			}

		case id := <-pool.put:
			srv, ok := servers[id]
			if ok {
				if srv.ref != 0 {srv.ref--}
				if srv.ref == 0 {
					srv.cancel()
					delete(servers, id)
				}
			}
		}
	}
}

var Pool = NewGamePool()
