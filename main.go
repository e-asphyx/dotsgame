package main

import (
	"log"
	"net/http"
	"io"
	"strconv"
	"crypto/rand"
	"encoding/base64"
	"html/template"

	"code.google.com/p/go.net/websocket"
	"github.com/gorilla/mux"
)

/*-------------------------------------------------------------------------------*/

const (
	templatesRoot = "templates/"
	templateMain = "index.html"
)

var (
	templates = template.Must(template.ParseFiles(templatesRoot + templateMain))
	db DBProxy
)

func randStr(n uint) string {
	buf := make([]byte, n)
	rand.Read(buf)
    return base64.URLEncoding.EncodeToString(buf)
}

func NewRoom(w http.ResponseWriter, req *http.Request) {
	/* new room */
	newUid := randStr(6)

	roomId, err := db.NewRoom(newUid)

	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		log.Printf("New room: %s (%d)\n", newUid, roomId)
		http.Redirect(w, req, "/" + newUid + "/", http.StatusTemporaryRedirect)
	}
	return
}

func MainServer(w http.ResponseWriter, req *http.Request) {
	uid := mux.Vars(req)["uid"]

	_, err := db.RoomId(uid)
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

func WebSocketServer(ws *websocket.Conn) {
	uid := mux.Vars(ws.Request())["uid"]

	roomId, err := db.RoomId(uid)
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

	room := Pool.Get(roomId)
	defer room.Put()

	client := room.NewClient(cid)
	defer client.Cancel()

	/* main loop */
	for {
		select {
			case msg, ok := <-incoming:
				if !ok {return}
				msg.roomId = roomId
				room.Post(msg)

			case msg := <-client.msg:
				err := websocket.JSON.Send(ws, msg)
				if err != nil {return}
		}
	}
}

func main() {
	log.Println("Start")

	var err error
	db, err = NewPQProxy()
	if err != nil {
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
