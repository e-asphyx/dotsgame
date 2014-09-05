package main

import (
	"fmt"
	"os"
	"log"
	"net/http"
	"io"
	"time"
	"html/template"

	"code.google.com/p/go.net/websocket"
	"github.com/gorilla/mux"
	"code.google.com/p/goauth2/oauth"
)

/*-------------------------------------------------------------------------------*/

const (
	templatesRoot = "templates/"
	templateMain = "index.html"
	keepAliveInterval = 30 /* sec */

	FlagKeepAlive = 0x1
)

var (
	templates = template.Must(template.ParseFiles(templatesRoot + templateMain))
	oauthConfig = &oauth.Config {
		ClientId:     os.Getenv("FB_ID"),
		ClientSecret: os.Getenv("FB_SECRET"), /* Come from Heroku app config */

		AuthURL:      "https://www.facebook.com/dialog/oauth",
		TokenURL:     "https://graph.facebook.com/oauth/access_token",
		RedirectURL:  "http://dotsgame.herokuapp.com/login",
	}

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
	if token := req.FormValue("token"); token != "" {
		/* Test user login */
		cid, err := db.VerifyToken(token)

		if err != nil {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		err = AuthAuthenticate(w, cid)

		if err != nil {
			log.Println(err)
		}

		log.Printf("User %d logged in\n", cid)
		return
	}

	/* OAuth2 login */
	if code := req.FormValue("code"); code != "" {
		transport := &oauth.Transport{Config: oauthConfig}
		tok, err := transport.Exchange(code)

		if err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		fmt.Fprintln(w, "Done!")
		log.Println(tok)

		return
	}

	http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
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

	timer := time.NewTimer(time.Second * keepAliveInterval)

	keepalive := GameMessage {
		Flags: FlagKeepAlive,
	}
	/* main loop */
	for {
		select {
		case msg, ok := <-incoming:
			if !ok {return}
			msg.roomId = roomId
			room.Post(msg)
			timer.Reset(time.Second * keepAliveInterval)

		case msg := <-client.msg:
			err := websocket.JSON.Send(ws, msg)
			if err != nil {return}
			timer.Reset(time.Second * keepAliveInterval)

		case <-timer.C:
			err := websocket.JSON.Send(ws, &keepalive)
			if err != nil {return}
			timer.Reset(time.Second * keepAliveInterval)
		}
	}
}

/*-------------------------------------------------------------------------------*/
type OAuthRedirect oauth.Config

func (redirect *OAuthRedirect) URL() string {
	config := (*oauth.Config)(redirect)
	state := randStr(6)

	/* TODO save in db */

	return config.AuthCodeURL(state)
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
	router.HandleFunc("/login", Login)

	/* Main page */
	router.Handle("/", NewAuthWrapper(http.HandlerFunc(NewRoom), (*OAuthRedirect)(oauthConfig)))

	/* Game room */
	router.Handle("/{room_id}/", NewAuthWrapper(NewRoomWrapper(http.HandlerFunc(RoomServer)), (*OAuthRedirect)(oauthConfig)))

	/* Room API */
	router.Path("/{room_id}/api/invitation").Methods("GET").Handler(NewAuthWrapper(NewRoomWrapper(JSONHandlerFunc(RoomInvitation)),
																					(*OAuthRedirect)(oauthConfig)))

	/* Serve WebSocket */
	router.Handle("/{room_id}/websocket", NewAuthWrapper(NewRoomWrapper(websocket.Handler(WebSocketServer)), (*OAuthRedirect)(oauthConfig)))

	http.Handle("/", router)
	/* Start server */

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(http.ListenAndServe(":" + port, nil))
}
