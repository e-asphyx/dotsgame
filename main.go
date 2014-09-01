package main

import (
	"log"
	"net/http"
	"io"
	"strconv"
	"container/list"
	"database/sql"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"html/template"

	"code.google.com/p/go.net/websocket"
	_ "github.com/lib/pq"
	"github.com/gorilla/mux"
)

type Point struct {
	X uint `json:"x"`
	Y uint `json:"y"`
}

type GameMessage struct {
	roomId uint64 `json:"-"`

	CID uint64 `json:"cid"`
	Flags uint `json:"fl"`

	Points map[string][]Point `json:"p,omitempty"`
	Areas map[string][][]Point `json:"a,omitempty"`
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

					if client.roomId == msg.roomId && client.cid != msg.CID {
						client.msg <- msg
					}
				}
		}
	}
}

func (srv *GameServer) NewClient(cid uint64, roomId uint64) *Client {
	client := Client {
		cid: cid,
		roomId: roomId,
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

const (
	templatesRoot = "templates/"
	templateMain = "index.html"
)

var (
	gameserver  = NewGameServer()
	templates = template.Must(template.ParseFiles(templatesRoot + templateMain))

	db *sql.DB
)

func randStr(n uint) string {
	buf := make([]byte, n)
	rand.Read(buf)
    return base64.URLEncoding.EncodeToString(buf)
}

func NewRoom(w http.ResponseWriter, req *http.Request) {
	/* new room */
	newUid := randStr(6)

	var roomId uint64
	err := db.QueryRow("INSERT INTO room(uid) VALUES ($1) RETURNING id", newUid).Scan(&roomId)

	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		log.Printf("New room: %s (%d)\n", newUid, roomId)
		http.Redirect(w, req, "/" + newUid + "/", http.StatusTemporaryRedirect)
	}
	return
}

func getRoomId(uid string) (uint64, error) {
	var roomId uint64
	err := db.QueryRow("SELECT id FROM room WHERE uid=$1", uid).Scan(&roomId)
	return roomId, err
}

func MainServer(w http.ResponseWriter, req *http.Request) {
	uid := mux.Vars(req)["uid"]

	_, err := getRoomId(uid)
	if err != nil {
		log.Println(err)
		http.NotFound(w, req)
		return
	}

	err = templates.ExecuteTemplate(w, templateMain, nil)
    if err != nil {
		log.Println(err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

func postHistory(msg *GameMessage) error {
	tx, err := db.Begin()
	if err != nil {return err}
	defer tx.Rollback()

	for cid, points := range msg.Points {
		for _, p := range points {
			_, err := tx.Exec("INSERT INTO point(room_id, cid, x, y) VALUES ($1, $2, $3, $4)", msg.roomId, cid, p.X, p.Y)
			if err != nil {return err}
		}
	}

	for cid, area := range msg.Areas {
		jsondata, _ := json.Marshal(area)
		res, err := tx.Exec("UPDATE area SET area = $1 WHERE room_id = $2 AND cid = $3", jsondata, msg.roomId, cid)
		if err != nil {return err}

		if affected, _ := res.RowsAffected(); affected == 0 {
			_, err := tx.Exec("INSERT INTO area(room_id, cid, area) VALUES ($1, $2, $3)", msg.roomId, cid, jsondata)
			if err != nil {return err}
		}
	}

	return tx.Commit()
}

func loadHistory(roomId uint64) (*GameMessage, error) {
	tx, err := db.Begin()
	if err != nil {return nil, err}
	defer tx.Rollback()

	msg := GameMessage {
		Points: make(map[string][]Point),
		Areas: make(map[string][][]Point),
	}

	rows, err := tx.Query("SELECT cid, x, y FROM point WHERE room_id=$1", roomId)
	if err != nil {return nil, err}
	defer rows.Close()

	for rows.Next() {
		var (
			cid string
			x, y uint
		)

		err = rows.Scan(&cid, &x, &y)
		if err != nil {return nil, err}

		msg.Points[cid] = append(msg.Points[cid], Point{x, y})
	}
	err = rows.Err()
	if err != nil {return nil, err}

	rows, err = tx.Query("SELECT cid, area FROM area WHERE room_id=$1", roomId)
	if err != nil {return nil, err}
	defer rows.Close()

	for rows.Next() {
		var (
			cid string
			area []byte
			points [][]Point
		)

		err = rows.Scan(&cid, &area)
		if err != nil {return nil, err}

		err = json.Unmarshal(area, &points)
		if err != nil {
			log.Println(err)
		} else {
			msg.Areas[cid] = points
		}
	}
	err = rows.Err()
	if err != nil {return nil, err}

	return &msg, nil
}

func WebSocketServer(ws *websocket.Conn) {
	uid := mux.Vars(ws.Request())["uid"]

	roomId, err := getRoomId(uid)
	if err != nil {
		log.Println(err)
		return
	}

	tmp := ws.Request().FormValue("cid")
	if tmp == "" {
		return
	}
	cid, _ := strconv.ParseUint(tmp, 10, 0)

	log.Printf("Connected cid %d to room %d (%s)\n", cid, roomId, uid)

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
			} else if msg.CID == cid {
				incoming <- msg
			}
		}
	}()

	client := gameserver.NewClient(cid, roomId)
	defer client.Cancel()

	/* load history */
	hist, err := loadHistory(roomId)
	if err != nil {
		log.Println(err)
		return
	}

	if len(hist.Points) != 0 || len(hist.Areas) != 0 {
		err := websocket.JSON.Send(ws, hist)
		if err != nil {return}
	}

	/* main loop */
	for {
		select {
			case msg, ok := <-incoming:
				if !ok {return}
				msg.roomId = roomId

				err := postHistory(msg)
				if err != nil {
					log.Println(err)
					return
				}
				gameserver.Post(msg)

			case msg := <-client.msg:
				err := websocket.JSON.Send(ws, msg)
				if err != nil {return}
		}
	}
}

func main() {
	log.Println("Start")

	var err error
	db, err = sql.Open("postgres", "user=asphyx dbname=dotsgame sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	router := mux.NewRouter()
	router.StrictSlash(true)

	/* Main page */
	router.HandleFunc("/", NewRoom)
	router.HandleFunc("/{uid}/", MainServer)

	/* Serve WebSocket */
	router.Handle("/{uid}/websocket", websocket.Handler(WebSocketServer))

	/* Serve static */
	router.PathPrefix("/static/").Handler(http.FileServer(http.Dir("")))

	http.Handle("/", router)
	/* Start server */
	log.Fatal(http.ListenAndServe(":8080", nil))
}
