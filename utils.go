package main

import (
	"log"
	"net/http"
	"encoding/json"
	"crypto/rand"
	"encoding/base64"
	"github.com/gorilla/context"

	"github.com/gorilla/mux"
)

/*-------------------------------------------------------------------------------*/
/* custom error interface */
type HTTPError int

func (err HTTPError) Error() string {
	return http.StatusText(int(err))
}

/*-------------------------------------------------------------------------------*/

type String interface {
	String() string
}

type StringVal string

func (val StringVal) String() string {
	return string(val)
}

/*-------------------------------------------------------------------------------*/
/* wrap function into http.Handler iface */
type JSONHandlerFunc func(*http.Request) (interface{}, error)

func (f JSONHandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	/* handle here */
	data, err := f(r)

	if err != nil {
		log.Println(err)

		var httpcode int
		if code, iscode := err.(HTTPError); iscode {
			httpcode = int(code)
		} else {
			httpcode = http.StatusInternalServerError
		}

		http.Error(w, err.Error(), httpcode)
		return
	}

	enc := json.NewEncoder(w)
	w.Header().Set("Content-Type", "application/json")
	_ = enc.Encode(data)
}

/*-------------------------------------------------------------------------------*/
type AuthWrapper struct {
	Handler http.Handler
	Redirect String
}

func (wrapper *AuthWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")

	_, ok := session.Values["cid"].(uint64)

	if ok {
		/* handle */
		/* TODO update cookie expired time */

		wrapper.Handler.ServeHTTP(w, r)
		return
	}

	/* redirect to login dialog */
	if wrapper.Redirect != nil {
		http.Redirect(w, r, wrapper.Redirect.String(), http.StatusTemporaryRedirect)
		return
	}

	http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
}

func NewAuthWrapper(handler http.Handler, redirect String) *AuthWrapper {
	return &AuthWrapper{Handler: handler, Redirect: redirect}
}

/*-------------------------------------------------------------------------------*/

type RoomData struct {
	roomId uint64
	pid uint64
}

/* Check if user is in room and injects room id and player id */
type RoomWrapper struct {
	handler http.Handler
}

func (wrapper *RoomWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	cid, ok := session.Values["cid"].(uint64)

	if !ok {
		http.Error(w, http.StatusText(http.StatusForbidden) + " (bad cid)", http.StatusForbidden)
		return
	}

	uid := mux.Vars(r)["room_id"]
	if uid == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest) + " (bad room_id)", http.StatusBadRequest)
		return
	}

	roomId, err := db.RoomId(uid)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound) + " (bad room_id)", http.StatusNotFound)
		return
	}

	pid, err := db.GetPlayer(roomId, cid)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	rd := RoomData {
		roomId: roomId,
		pid: pid,
	}
	context.Set(r, contextKey("room"), &rd)

	// handle here
	wrapper.handler.ServeHTTP(w, r)
}

func NewRoomWrapper(handler http.Handler) *RoomWrapper {
	return &RoomWrapper{handler: handler}
}

/*-------------------------------------------------------------------------------*/

func randStr(n uint) string {
	buf := make([]byte, n)
	rand.Read(buf)
    return base64.URLEncoding.EncodeToString(buf)
}

