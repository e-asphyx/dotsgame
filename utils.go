package main

import (
	"log"
	"net/http"
	"encoding/json"
	"strconv"
	"crypto/rand"
	"encoding/base64"
	"time"
	"github.com/gorilla/mux"
)

/*-------------------------------------------------------------------------------*/
/* custom error interface */
type HTTPError int

func (err HTTPError) Error() string {
	return http.StatusText(int(err))
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
func InjectValue(r *http.Request, key string, value string) {
	_ = r.ParseForm()
	if r.Form != nil {
		/* store fake form value */
		r.Form["_injected_value_" + key] = []string{value}
	}
}

func GetInjectedValue(r *http.Request, key string) string {
	return r.FormValue("_injected_value_" + key)
}

func GetInjectedValueUint(r *http.Request, key string) uint64 {
	tmp := GetInjectedValue(r, key)
	val, _ := strconv.ParseUint(tmp, 10, 0)
	return val
}

/*-------------------------------------------------------------------------------*/
type AuthWrapper struct {
	handler http.Handler
}

func (wrapper *AuthWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie("session_token")

	if cookie != nil {
		cid, err := db.VerifyAuthToken(cookie.Value);

		if err == nil {
			InjectValue(r, "cid", strconv.FormatUint(cid, 10))
			updateTokenCookie(w, cookie.Value)

			/* handle */
			wrapper.handler.ServeHTTP(w, r)
			return
		}
	}

	http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
}

func NewAuthWrapper(handler http.Handler) *AuthWrapper {
	return &AuthWrapper{handler: handler}
}

func updateTokenCookie(w http.ResponseWriter, token string) {
	cookie := http.Cookie {
		Name: "session_token",
		Value: token,
		Path: "/",
		Expires: time.Now().Add(time.Hour * 24 * 60),
	}
	http.SetCookie(w, &cookie)
}

func AuthAuthenticate(w http.ResponseWriter, cid uint64) error {
	token := randStr(20)

	err := db.SetAuthToken(cid, token)
	if err == nil {
		updateTokenCookie(w, token)
	}
	return err
}

/*-------------------------------------------------------------------------------*/

/* Check if user is in room and injects room id and player id */
type RoomWrapper struct {
	handler http.Handler
}

func (wrapper *RoomWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	cid := GetInjectedValueUint(r, "cid")
	if cid == 0 {
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
		http.Error(w, http.StatusText(http.StatusNotFound) + " (bad pid)", http.StatusNotFound)
		return
	}
	InjectValue(r, "room_id", strconv.FormatUint(roomId, 10))

	pid, err := db.GetPlayer(roomId, cid)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	InjectValue(r, "pid", strconv.FormatUint(pid, 10))

	/* handle here */
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

