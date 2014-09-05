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

type Redirector interface {
	Redirect(w http.ResponseWriter, r *http.Request) error
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
	Redirect Redirector
}

func (wrapper *AuthWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")

	log.Println(session.Values)

	cid, ok := getUint64(session.Values["cid"])
	if !ok {
		/* redirect to login dialog */
		if wrapper.Redirect != nil {
			wrapper.Redirect.Redirect(w, r)
			return
		}

		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	uid := mux.Vars(r)["room_id"]
	if uid != "" {
		/* Authenticate user in given room */
		roomId, err := db.RoomId(uid)

		if err != nil {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		pid, err := db.GetPlayer(roomId, cid)

		if err != nil {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		context.Set(r, "room_id", roomId)
		context.Set(r, "player_id", pid)
	}

	/* handle */
	/* TODO update cookie expiration date */

	wrapper.Handler.ServeHTTP(w, r)

}

func NewAuthWrapper(handler http.Handler, redirect Redirector) *AuthWrapper {
	return &AuthWrapper{Handler: handler, Redirect: redirect}
}

/*-------------------------------------------------------------------------------*/

func randStr(n uint) string {
	buf := make([]byte, n)
	rand.Read(buf)
    return base64.URLEncoding.EncodeToString(buf)
}

func getUint64 (v interface{}) (uint64, bool) {
	var ret uint64

	switch val := v.(type) {
	default:
		return 0, false

	case int:
		ret = uint64(val)

	case int64:
		ret = uint64(val)

	case uint:
		ret = uint64(val)

	case uint64:
		ret = uint64(val)

	case float64:
		ret = uint64(val)
	}

	return ret, true
}
