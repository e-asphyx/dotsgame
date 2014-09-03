package main

import (
	"log"
	"net/http"
	"encoding/json"
)

/* custom error interface */
type HTTPError int

type JSONHandlerFunc func(*http.Request) (interface{}, error)

func (err HTTPError) Error() string {
	return http.StatusText(int(err))
}

/* wrap function into http.Handler iface */
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
