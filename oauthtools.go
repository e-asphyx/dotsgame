package main

import (
	"log"
	"encoding/json"
	"github.com/e-asphyx/goauth2/oauth"
)

func OAuthCall(transport *oauth.Transport, url string, result interface{}) error {
	res, err := transport.Client().Get(url)
	if err != nil {
		log.Println("OAuthCall: ", err)
		return err
    }
    defer res.Body.Close()

	dec := json.NewDecoder(res.Body)
	err = dec.Decode(result)
	if err != nil {
		log.Println("OAuthCall (json.Decoder.Decode): ", err)
    }

	return err
}
