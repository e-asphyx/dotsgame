package main

import (
	"log"
	"database/sql"
	"encoding/json"
	_ "github.com/lib/pq"
)

type DBProxy interface {
	RoomId(uid string) (uint64, error)
	NewRoom(uid string) (uint64, error)

	NewUser(token string) (uint64, error)
	VerifyToken(token string) (uint64, error)

	PostHistory(msg *GameMessage) error
	LoadHistory(id uint64) (*GameMessage, error)

	SetAuthToken(cid uint64, token string) error
	VerifyAuthToken(token string) (uint64, error)
}

/* PostgreSQL proxy */
type PQProxy struct {
	*sql.DB
}

func NewPQProxy() (*PQProxy, error) {
	db, err := sql.Open("postgres", "user=asphyx dbname=dotsgame sslmode=disable")
	if err != nil {return nil, err}

	proxy := &PQProxy {
		DB: db,
	}

	return proxy, err
}

func (db *PQProxy) RoomId(uid string) (uint64, error) {
	var roomId uint64
	err := db.QueryRow("SELECT id FROM room WHERE uid=$1", uid).Scan(&roomId)
	return roomId, err
}

func (db *PQProxy) NewRoom(uid string) (uint64, error) {
	var roomId uint64
	err := db.QueryRow("INSERT INTO room (uid) VALUES ($1) RETURNING id", uid).Scan(&roomId)
	return roomId, err
}

func (db *PQProxy) PostHistory(msg *GameMessage) error {
	/* Insert point(s) */
	for cid, points := range msg.Points {
		for _, p := range points {
			_, err := db.Exec("INSERT INTO point (room_id, cid, x, y) VALUES ($1, $2, $3, $4)", msg.roomId, cid, p.X, p.Y)
			if err != nil {return err}
		}
	}

	/* Update area as single record */
	for cid, area := range msg.Areas {
		jsondata, _ := json.Marshal(area)
		res, err := db.Exec("UPDATE area SET area = $1 WHERE room_id = $2 AND cid = $3", jsondata, msg.roomId, cid)
		if err != nil {return err}

		if affected, _ := res.RowsAffected(); affected == 0 {
			_, err := db.Exec("INSERT INTO area (room_id, cid, area) VALUES ($1, $2, $3)", msg.roomId, cid, jsondata)
			if err != nil {return err}
		}
	}

	return nil
}

func (db *PQProxy) LoadHistory(id uint64) (*GameMessage, error) {
	msg := GameMessage {
		Points: make(map[string][]Point),
		Areas: make(map[string][][]Point),
		roomId: id,
	}

	/* Load points */
	rows, err := db.Query("SELECT cid, x, y FROM point WHERE room_id=$1", id)
	if err != nil {return nil, err}
	defer rows.Close()

	for rows.Next() {
		var (
			cid string
			x, y uint
		)

		err = rows.Scan(&cid, &x, &y)
		if err != nil {return nil, err}

		msg.Points[cid] = append(msg.Points[cid], Point{x, y})
	}
	err = rows.Err()
	if err != nil {return nil, err}

	/* Load area */
	rows, err = db.Query("SELECT cid, area FROM area WHERE room_id=$1", id)
	if err != nil {return nil, err}
	defer rows.Close()

	for rows.Next() {
		var (
			cid string
			area []byte
			points [][]Point
		)

		err = rows.Scan(&cid, &area)
		if err != nil {return nil, err}

		err = json.Unmarshal(area, &points)
		if err != nil {
			log.Println(err)
		} else {
			msg.Areas[cid] = points
		}
	}
	err = rows.Err()
	if err != nil {return nil, err}

	return &msg, nil
}

/* login secret */
func (db *PQProxy) NewUser(token string) (uint64, error) {
	var cid uint64
	err := db.QueryRow("INSERT INTO client (auth_token) VALUES ($1) RETURNING id", token).Scan(&cid)

	if err != nil {
		log.Println(err)
	}

	return cid, err
}

func (db *PQProxy) VerifyToken(token string) (uint64, error) {
	var cid uint64
	err := db.QueryRow("SELECT id FROM client WHERE auth_token = $1", token).Scan(&cid)

	if err != nil && err != sql.ErrNoRows {
		log.Println(err)
	}

	return cid, err
}

/* session secret (cookie) */
func (db *PQProxy) VerifyAuthToken(token string) (uint64, error) {
	var cid uint64
	err := db.QueryRow("SELECT cid FROM auth WHERE token = $1 AND CURRENT_TIMESTAMP - timestamp < ttl", token).Scan(&cid)

	if err != nil && err != sql.ErrNoRows {
		log.Println(err)
	}

	return cid, err
}

func (db *PQProxy) SetAuthToken(cid uint64, token string) error {
	_, err := db.Exec("INSERT INTO auth (cid, token) VALUES ($1, $2)", cid, token)

	if err != nil {
		log.Println(err)
	}

	return err
}
