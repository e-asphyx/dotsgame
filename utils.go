package main

import (
	"log"
	"strconv"
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
	Redirect string
}

func (wrapper *AuthWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")

	cid, ok := getUint64(session.Values["cid"])
	if !ok || cid == 0 {
		/* redirect to login dialog */
		if wrapper.Redirect != "" {
			http.Redirect(w, r, wrapper.Redirect, http.StatusTemporaryRedirect)
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

	/* update cookie expiration date */
	store.Refresh(r, w, session)

	/* handle */
	wrapper.Handler.ServeHTTP(w, r)

}

func NewAuthWrapper(handler http.Handler, redirect string) *AuthWrapper {
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

	case string:
		ret, _ = strconv.ParseUint(val, 10, 64)
	}

	return ret, true
}
