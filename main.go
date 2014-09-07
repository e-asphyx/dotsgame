package main

import (
	"fmt"
	"os"
	"log"
	"net/http"
	"io"
	"time"
	"html/template"
	"strconv"

	"code.google.com/p/go.net/websocket"
	"github.com/gorilla/mux"
	"github.com/gorilla/context"
	"github.com/e-asphyx/goauth2/oauth"
)

/*-------------------------------------------------------------------------------*/

const (
	templatesRoot = "templates/"
	templateMain = "index.html"
	keepAliveInterval = 30 /* sec */

	FlagKeepAlive = 0x1

	GraphAPIProfile = "https://graph.facebook.com/v2.1/me"
	GraphAPIPicture = "https://graph.facebook.com/v2.1/me/picture?type=normal&redirect=false"
)

var (
	templates = template.Must(template.ParseFiles(templatesRoot + "head.html",
													templatesRoot + "templates.html",
													templatesRoot + templateMain))

	oauthConfig = &oauth.Config {
		ClientId:     os.Getenv("FB_ID"),
		ClientSecret: os.Getenv("FB_SECRET"), /* Come from Heroku app config */

		AuthURL:      "https://www.facebook.com/dialog/oauth",
		TokenURL:     "https://graph.facebook.com/oauth/access_token",
		RedirectURL:  "http://dotsgame.herokuapp.com/login",
	}

	db DBProxy
	store *DBSessionStore
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
	session, _ := store.Get(req, "session")

	cid, _ := getUint64(session.Values["cid"])

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
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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

		/* Authenticate */
		session, _ := store.New(req, "session")
		session.Values["cid"] = cid

		err = session.Save(req, w)
		if err != nil {
			log.Println(err)
		}

		log.Printf("User %d logged in\n", cid)
		return
	}

	/* OAuth2 login */
	if code := req.FormValue("code"); code != "" {
		var state string
		if state = req.FormValue("state"); state == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		session, _ := store.Get(req, "session")

		newstate, ok := session.Values["state"].(string)
		if !ok || state != newstate {
			log.Printf("%s != %s\n", state, newstate);

			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		transport := &oauth.Transport{Config: oauthConfig}
		tok, err := transport.Exchange(code)

		if err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		/* Get profile */
		var profile struct {
			ID string `json:"id"`
			Name string `json:"name"`
		}

		err = OAuthCall(transport, GraphAPIProfile, &profile)
		if err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		/* Get picture */
		var picture struct {
			Data struct {
				Url string `json:"url"`
			} `json:"data"`
		}

		err = OAuthCall(transport, GraphAPIPicture, &picture)
		if err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		cid, _ := strconv.ParseUint(profile.ID, 10, 64)

		err = db.SyncUser(cid, profile.Name, picture.Data.Url, tok.AccessToken, tok.Expiry)
		if err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		/* Authenticate */
		session.Values["cid"] = cid
		err = session.Save(req, w)
		if err != nil {
			log.Println(err)
		}

		log.Printf("Logged in Facebook user id %d\n", cid)

		fmt.Fprintln(w, "Done!")

		return
	}

	http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
}

/*
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
*/

type UserProfile struct {
	ID uint64 `json:"id,omitempty"`
	Name string `json:"name"`
	Picture string `json:"picture"`
	Player uint64 `json:"player,omitempty"`
}

func GetUser(req *http.Request) (interface{}, error) {
	session, _ := store.Get(req, "session")
	cid, _ := getUint64(session.Values["cid"])
	reqId, _ := strconv.ParseUint(mux.Vars(req)["user_id"], 10, 64)

	/* Only self */
	if cid != reqId {
		return nil, HTTPError(http.StatusForbidden)
	}

	reply, err := db.GetUserProfile(cid)
	if err != nil {
		return nil, HTTPError(http.StatusNotFound)
	}

	return reply, nil
}

func GetPlayer(req *http.Request) (interface{}, error) {
	session, _ := store.Get(req, "session")
	cid, _ := getUint64(session.Values["cid"])
	roomId, _ := context.Get(req, "room_id").(uint64)

	reply, err := db.GetPlayerProfile(cid, roomId)
	if err != nil {
		return nil, HTTPError(http.StatusNotFound)
	}

	return reply, nil
}

func GetPlayers(req *http.Request) (interface{}, error) {
	roomId, _ := context.Get(req, "room_id").(uint64)

	reply, err := db.GetPlayers(roomId)
	if err != nil {
		return nil, HTTPError(http.StatusNotFound)
	}

	return reply, nil
}

func RoomInvitation(req *http.Request) (interface{}, error) {
	roomId, _ := context.Get(req, "room_id").(uint64)
	token := randStr(20)

	id, err := db.NewInvitation(roomId, token)
	if err != nil {return nil, err}

	log.Printf("New invitation issued: %d\n", id)

	reply := invitationReply {
		Room: mux.Vars(req)["room_id"],
		Code: token,
	}

	return &reply, nil
}

type AuthData struct {
	ID uint64
}

func RoomServer(w http.ResponseWriter, req *http.Request) {
	session, _ := store.Get(req, "session")
	cid, _ := getUint64(session.Values["cid"])

	data := AuthData {
		ID: cid,
	}

	err := templates.ExecuteTemplate(w, templateMain, &data)
    if err != nil {
		log.Println(err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

func WebSocketServer(ws *websocket.Conn) {
	session, _ := store.Get(ws.Request(), "session")

	cid, _ := getUint64(session.Values["cid"])
	roomId, _ := context.Get(ws.Request(), "room_id").(uint64)
	pid, _ := context.Get(ws.Request(), "player_id").(uint64)

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

func (redirect *OAuthRedirect) Redirect(w http.ResponseWriter, r *http.Request) error {
	config := (*oauth.Config)(redirect)
	state := randStr(6)

	session, _ := store.Get(r, "session")
	session.Values["state"] = state
	err := session.Save(r, w)

	if err != nil {
		log.Println(err)
	}

	http.Redirect(w, r, config.AuthCodeURL(state), http.StatusTemporaryRedirect)

	return nil
}

/*-------------------------------------------------------------------------------*/

func main() {
	log.Println("Start")

	var err error
	db, err = NewPQProxy()
	if err != nil {
		log.Fatal(err)
	}

	store = NewDBSessionStore(db)

	router := mux.NewRouter()
	router.StrictSlash(true)

	/* Serve static */
	router.PathPrefix("/static/").Handler(http.FileServer(http.Dir("")))

	/* router.Handle("/newuser", JSONHandlerFunc(NewUser)) */

	/* token login */
	router.HandleFunc("/login", Login)

	/* Main page */
	router.Handle("/", NewAuthWrapper(http.HandlerFunc(NewRoom), (*OAuthRedirect)(oauthConfig)))

	/* Main API */
	router.Path("/api/users/{user_id}").Methods("GET").Handler(NewAuthWrapper(JSONHandlerFunc(GetUser),
																					(*OAuthRedirect)(oauthConfig)))
	/* Game room */
	router.Handle("/{room_id}/", NewAuthWrapper(http.HandlerFunc(RoomServer), (*OAuthRedirect)(oauthConfig)))

	/* Room API */
	router.Path("/{room_id}/api/invitation").Methods("POST").Handler(NewAuthWrapper(JSONHandlerFunc(RoomInvitation),
																					(*OAuthRedirect)(oauthConfig)))

	router.Path("/{room_id}/api/users").Methods("GET").Handler(NewAuthWrapper(JSONHandlerFunc(GetPlayers),
																					(*OAuthRedirect)(oauthConfig)))

	router.Path("/{room_id}/api/users/{user_id}").Methods("GET").Handler(NewAuthWrapper(JSONHandlerFunc(GetPlayer),
																					(*OAuthRedirect)(oauthConfig)))

	/* Serve WebSocket */
	router.Handle("/{room_id}/websocket", NewAuthWrapper(websocket.Handler(WebSocketServer), (*OAuthRedirect)(oauthConfig)))

	http.Handle("/", router)
	/* Start server */

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(http.ListenAndServe(":" + port, nil))
}
