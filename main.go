package main

import (
	"os"
	"log"
	"net/http"
	"io"
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

type invitationReply struct {
	Room string `json:"room"`
	Code string `json:"code"`
}

/*-------------------------------------------------------------------------------*/

func NewRoom(w http.ResponseWriter, req *http.Request) {
	cid := GetInjectedValueUint(req, "cid")
	/* new room */
	newUid := randStr(6)

	roomId, err := db.NewRoom(newUid)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	/* First player */
	/* TODO color scheme */
	pid, err := db.NewPlayer(roomId, cid, "");
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("New room: %s (%d), first player %d (pid %d)\n", newUid, roomId, cid, pid)

	http.Redirect(w, req, "/" + newUid + "/", http.StatusTemporaryRedirect)
}

func Login(w http.ResponseWriter, req *http.Request) {
	token := req.FormValue("token")
	cid, err := db.VerifyToken(token)

	if err != nil {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	err = AuthAuthenticate(w, cid)

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

func RoomInvitation(req *http.Request) (interface{}, error) {
	roomId := GetInjectedValueUint(req, "room_id")
	roomUid := mux.Vars(req)["room_id"]

	token := randStr(20)

	id, err := db.NewInvitation(roomId, token)
	if err != nil {return nil, err}

	log.Printf("New invitation issued: %d\n", id)

	reply := invitationReply {
		Room: roomUid,
		Code: token,
	}

	return &reply, nil
}

func RoomServer(w http.ResponseWriter, req *http.Request) {
	err := templates.ExecuteTemplate(w, templateMain, nil)
    if err != nil {
		log.Println(err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

func WebSocketServer(ws *websocket.Conn) {
	cid := GetInjectedValueUint(ws.Request(), "cid")
	roomId := GetInjectedValueUint(ws.Request(), "room_id")
	pid := GetInjectedValueUint(ws.Request(), "pid")

	if cid == 0 || roomId == 0 || pid == 0 {return}

	log.Printf("Connected cid %d to room %d as pid %d\n", cid, roomId, pid)

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

/*-------------------------------------------------------------------------------*/

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
	router.Handle("/", NewAuthWrapper(http.HandlerFunc(NewRoom)))

	/* Game room */
	router.Handle("/{room_id}/", NewAuthWrapper(NewRoomWrapper(http.HandlerFunc(RoomServer))))

	/* Room API */
	router.Path("/{room_id}/api/invitation").Methods("GET").Handler(NewAuthWrapper(NewRoomWrapper(JSONHandlerFunc(RoomInvitation))))
	/*
	router.Path("/{room_id}/api/player").Methods("POST").Handler(&AuthWrapper{handler: http.HandlerFunc(PlayerNew)})
	router.Path("/{room_id}/api/player").Methods("GET").Handler(&AuthWrapper{handler: http.HandlerFunc(PlayersList)})

	router.Path("/{room_id}/api/player/{id}").Methods("GET").Handler(&AuthWrapper{handler: http.HandlerFunc(PlayerShow)})
	router.Path("/{room_id}/api/player/{id}").Methods("PUT", "PATCH").Handler(&AuthWrapper{handler: http.HandlerFunc(PlayerUpdate)})
	*/

	/* Serve WebSocket */
	router.Handle("/{room_id}/websocket", NewAuthWrapper(NewRoomWrapper(websocket.Handler(WebSocketServer))))

	http.Handle("/", router)
	/* Start server */

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(http.ListenAndServe(":" + port, nil))
}
