package main

import (
	"log"
	"net/http"
	"encoding/json"
	"strconv"
	"crypto/rand"
	"encoding/base64"
	"time"
)

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

type TokenAuthWrapper struct {
	handler http.Handler
}

func (wrapper *TokenAuthWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie("session_token")

	if cookie != nil {
		cid, err := db.VerifyAuthToken(cookie.Value);

		if err == nil {
			_ = r.ParseForm()
			if r.Form != nil {
				/* store CID in fake form value */
				r.Form["_auth_wrapper_cid_"] = []string{strconv.FormatUint(cid, 10)}
			}
			updateTokenCookie(w, cookie.Value)

			/* handle */
			wrapper.handler.ServeHTTP(w, r)
			return
		}
	}

	http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
}

func updateTokenCookie(w http.ResponseWriter, token string) {
	cookie := http.Cookie {
		Name: "session_token",
		Value: token,
		Path: "/",
		Expires: time.Now().Add(time.Hour * 24 * 30),
	}
	http.SetCookie(w, &cookie)
}

func TokenAuthAuthenticate(w http.ResponseWriter, cid uint64) error {
	token := randStr(20)

	err := db.SetAuthToken(cid, token)
	if err == nil {
		updateTokenCookie(w, token)
	}
	return err
}

func randStr(n uint) string {
	buf := make([]byte, n)
	rand.Read(buf)
    return base64.URLEncoding.EncodeToString(buf)
}

