package main

import (
	"github.com/gorilla/sessions"
	"net/http"
	"encoding/json"
	"log"
)

type DBSessionStore struct {
	Options *sessions.Options
	db DBProxy
}

func (s *DBSessionStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(s, name)
}

func (s *DBSessionStore) New(r *http.Request, name string) (*sessions.Session, error) {
	session := sessions.NewSession(s, name)

	opts := *s.Options
	session.Options = &opts

	session.IsNew = true

	var err error
	cookie, _ := r.Cookie(name)
	if cookie != nil {
		session.ID = cookie.Value
		err = s.load(session)
		session.IsNew = (err != nil)
	}

	return session, err
}

func (s *DBSessionStore) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	if session.ID == "" {
		session.ID = randStr(20)
	}

	if err := s.save(session); err != nil {
		return err
	}

	http.SetCookie(w, sessions.NewCookie(session.Name(), session.ID, session.Options))

	return nil
}

func (s *DBSessionStore) Refresh(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	http.SetCookie(w, sessions.NewCookie(session.Name(), session.ID, session.Options))

	return nil
}

func (s *DBSessionStore) load(session *sessions.Session) error {
	data, err := s.db.LoadSession(session.ID, session.Name())
	if err == nil {
		/* JSON accepts only string keys */
		tmp := make(map[string]interface{})

		if err = json.Unmarshal([]byte(data), &tmp); err == nil {
			for key, value := range tmp {
				session.Values[key] = value
			}
		} else {
			log.Println(err)
		}
	}

	return err
}

func (s *DBSessionStore) save(session *sessions.Session) error {
	/* JSON accepts only string keys */
	tmp := make(map[string]interface{})

	for key, value := range session.Values {
		strkey, ok := key.(string)
		if ok {
			tmp[strkey] = value
		}
	}

	data, _ := json.Marshal(tmp)

	return s.db.SaveSession(session.ID, session.Name(), string(data))
}

func NewDBSessionStore(db DBProxy) *DBSessionStore {
	return &DBSessionStore {
		Options: &sessions.Options {
			Path:   "/",
			MaxAge: 60 * 60 * 24 * 60, /* sec */
		},
		db: db,
	}
}
