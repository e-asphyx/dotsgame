package main

import (
	"log"
	"net/http"
	"io"
	"strconv"
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

type newUserReply struct {
	ID uint64 `json:"id"`
	AuthToken string `json:"token"`
}

func NewRoom(w http.ResponseWriter, req *http.Request) {
	/* new room */
	newUid := randStr(6)

	roomId, err := db.NewRoom(newUid)

	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("New room: %s (%d)\n", newUid, roomId)
	http.Redirect(w, req, "/" + newUid + "/", http.StatusTemporaryRedirect)
}

func Login(w http.ResponseWriter, req *http.Request) {
	token := req.FormValue("token")
	cid, err := db.VerifyToken(token)

	if err != nil {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	err = TokenAuthAuthenticate(w, cid)

	if err != nil {
		log.Println(err)
		return
	}

	log.Printf("User %d logged in\n", cid)
}

func NewUser(req *http.Request) (interface{}, error) {
	token := randStr(20)

	cid, err := db.NewUser(token)
	if err != nil {return nil, err}

	log.Printf("New user: %d\n", cid)

	reply := newUserReply {
		ID: cid,
		AuthToken: token,
	}

	return &reply, nil
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

	/* Serve static */
	router.PathPrefix("/static/").Handler(http.FileServer(http.Dir("")))

	/* API */
	router.Handle("/api/newuser", JSONHandlerFunc(NewUser)) /* TODO delete this */

	/* token login */
	router.HandleFunc("/login", Login) /* TODO delete this */

	/* Main page */
	router.Handle("/", &TokenAuthWrapper{handler: http.HandlerFunc(NewRoom)})
	router.Handle("/{uid}/", &TokenAuthWrapper{handler: http.HandlerFunc(MainServer)})

	/* Serve WebSocket */
	router.Handle("/{uid}/websocket", &TokenAuthWrapper{handler: websocket.Handler(WebSocketServer)})

	http.Handle("/", router)
	/* Start server */
	log.Fatal(http.ListenAndServe(":8080", nil))
}
