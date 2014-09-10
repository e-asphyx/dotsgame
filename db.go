package main

import (
	"os"
	"log"
	"time"
	"database/sql"
	"encoding/json"
	_ "github.com/lib/pq"
)

type DBProxy interface {
	RoomId(uid string) (uint64, error)
	NewRoom(uid string) (uint64, error)
	NewPlayer(roomId, cid uint64, scheme string) (uint64, error)
	GetPlayer(roomId, cid uint64) (uint64, error)

	NewUser(token string) (uint64, error)
	VerifyToken(token string) (uint64, error)

	PostHistory(msg *GameMessage) error
	LoadHistory(id uint64) (*GameMessage, error)

	LoadSession(sid, name string) (string, error)
	SaveSession(sid, name string, data string) error

	NewInvitation(roomId uint64, token string) (uint64, error)

	SyncUser(cid uint64, name, picture, token string, expires time.Time) error
	GetUserProfile(cid uint64) (*UserProfile, error)
	GetPlayerProfile(cid, roomId uint64) (*UserProfile, error)
	GetPlayers(roomId uint64) ([]UserProfile, error)
}

/* PostgreSQL proxy */
type PQProxy struct {
	*sql.DB
}

func NewPQProxy() (*PQProxy, error) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "user=asphyx dbname=dotsgame sslmode=disable"
	}

	db, err := sql.Open("postgres", url)
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

	if err != nil {
		log.Println("NewRoom: ", err)
	}

	return roomId, err
}

func (db *PQProxy) PostHistory(msg *GameMessage) error {
	/* Add or modify player */
	for cid, scheme := range msg.Players {
		res, err := db.Exec("UPDATE player SET color_scheme = $1 WHERE room_id = $2 AND client_id = $3", scheme, msg.roomId, cid)
		if err != nil {return err}

		if affected, _ := res.RowsAffected(); affected == 0 {
			_, err = db.Exec("INSERT INTO player (room_id, client_id, color_scheme) " +
								"VALUES ($1, $2, $3) RETURNING id", msg.roomId, cid, scheme)
			if err != nil {return err}
		}
	}

	/* TODO leave */

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
			_, err = db.Exec("INSERT INTO area (room_id, cid, area) VALUES ($1, $2, $3)", msg.roomId, cid, jsondata)
			if err != nil {return err}
		}
	}

	return nil
}

func (db *PQProxy) LoadHistory(id uint64) (*GameMessage, error) {
	msg := GameMessage {
		Points: make(map[string][]Point),
		Areas: make(map[string][][]Point),
		Players: make(map[string]string),
		roomId: id,
	}

	/* Load players */
	rows, err := db.Query("SELECT client.id, player.color_scheme FROM client LEFT JOIN player ON client.id = player.client_id " +
						"WHERE player.room_id = $1 ORDER BY timestamp", id)

	if err != nil {return nil, err}
	defer rows.Close()

	for rows.Next() {
		var (
			scheme sql.NullString
			cid string
		)

		err = rows.Scan(&cid, &scheme)
		if err != nil {return nil, err}

		msg.Players[cid] = scheme.String
	}
	err = rows.Err()
	if err != nil {return nil, err}

	/* Load points */
	rows, err = db.Query("SELECT cid, x, y FROM point WHERE room_id=$1", id)
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
			log.Println("LoadHistory: ",err)
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
		log.Println("NewUser: ", err)
	}

	return cid, err
}

func (db *PQProxy) VerifyToken(token string) (uint64, error) {
	var cid uint64
	err := db.QueryRow("SELECT id FROM client WHERE auth_token = $1", token).Scan(&cid)

	if err != nil && err != sql.ErrNoRows {
		log.Println("VerifyToken: ", err)
	}

	return cid, err
}

func (db *PQProxy) NewPlayer(roomId uint64, cid uint64, scheme string) (uint64, error) {
	var pid uint64
	err := db.QueryRow("INSERT INTO player (room_id, client_id, color_scheme) " +
						"VALUES ($1, $2, $3) RETURNING id", roomId, cid, scheme).Scan(&pid)

	if err != nil {
		log.Println("NewPlayer", err)
	}

	return pid, err
}

func (db *PQProxy) GetPlayer(roomId uint64, cid uint64) (uint64, error) {
	var pid uint64
	err := db.QueryRow("SELECT id FROM player WHERE room_id = $1 AND client_id = $2", roomId, cid).Scan(&pid)

	if err != nil && err != sql.ErrNoRows {
		log.Println("GetPlayer: ", err)
	}

	return pid, err
}

func (db *PQProxy) NewInvitation(roomId uint64, token string) (uint64, error) {
	var id uint64
	err := db.QueryRow("INSERT INTO invitation (room_id, code) VALUES ($1, $2) RETURNING id", roomId, token).Scan(&id)

	if err != nil {
		log.Println("NewInvitation: ", err)
	}

	return id, err
}

func (db *PQProxy) LoadSession(sid string, name string) (string, error) {
	var data string
	/*err := db.QueryRow("SELECT data FROM session WHERE sid = $1 AND name = $2 AND CURRENT_TIMESTAMP - timestamp < ttl", sid, name).Scan(&data)*/
	err := db.QueryRow("SELECT data FROM session WHERE sid = $1 AND name = $2", sid, name).Scan(&data)

	if err != nil && err != sql.ErrNoRows {
		log.Println("LoadSession: ", err)
	}

	return data, err
}

func (db *PQProxy) SaveSession(sid string, name string, data string) error {
	res, err := db.Exec("UPDATE session SET data = $1, timestamp = DEFAULT WHERE sid = $2 AND name = $3", data, sid, name)
	if err != nil {
		log.Println("SaveSession: ", err)
		return err
	}

	if affected, _ := res.RowsAffected(); affected == 0 {
		_, err = db.Exec("INSERT INTO session (sid, name, data) VALUES ($1, $2, $3)", sid, name, data)

		if err != nil {
			log.Println("SaveSession: ", err)
		}
	}

	return err
}

func (db *PQProxy) SyncUser(cid uint64, name, picture, token string, expires time.Time) error {
	res, err := db.Exec("UPDATE client SET name = $1, picture = $2, access_token = $3, expires = $4 WHERE id = $5",
						name, picture, token, expires, cid)

	if err != nil {
		log.Println("SyncUser: ", err)
		return err
	}

	if affected, _ := res.RowsAffected(); affected == 0 {
		_, err = db.Exec("INSERT INTO client (id, name, picture, access_token, expires) VALUES ($1, $2, $3, $4, $5)",
						cid, name, picture, token, expires)

		if err != nil {
			log.Println("SyncUser: ", err)
		}
	}

	return err
}

func (db *PQProxy) GetUserProfile(cid uint64) (*UserProfile, error) {
	var name, picture sql.NullString

	err := db.QueryRow("SELECT name, picture FROM client WHERE id = $1", cid).Scan(&name, &picture)

	if err != nil && err != sql.ErrNoRows {
		log.Println("GetProfile: ", err)
	}

	profile := UserProfile {
		ID: cid,
		Name: name.String,
		Picture: picture.String,
	}

	return &profile, err
}

func (db *PQProxy) GetPlayerProfile(cid, roomId uint64) (*UserProfile, error) {
	var (
		name, picture, scheme sql.NullString
		pid uint64
		ts time.Time
	)

	err := db.QueryRow("SELECT name, picture, player.id, color_scheme, timestamp FROM client LEFT JOIN player ON client.id = player.client_id " +
						"WHERE client.id = $1 AND player.room_id = $2", cid, roomId).Scan(&name, &picture, &pid, &scheme, &ts)

	if err != nil && err != sql.ErrNoRows {
		log.Println("GetProfileRoom: ", err)
	}

	profile := UserProfile {
		ID: cid,
		Name: name.String,
		Picture: picture.String,
		Player: pid,
		Scheme: scheme.String,
		Timestamp: ts,
	}

	return &profile, err
}

func (db *PQProxy) GetPlayers(roomId uint64) ([]UserProfile, error) {
	var result []UserProfile

	rows, err := db.Query("SELECT client.id, name, picture, player.id, color_scheme, timestamp FROM client LEFT JOIN player ON client.id = player.client_id " +
						"WHERE player.room_id = $1 ORDER BY timestamp", roomId)

	if err != nil {return nil, err}
	defer rows.Close()

	for rows.Next() {
		var (
			name, picture, scheme sql.NullString
			cid, pid uint64
			ts time.Time
		)

		err = rows.Scan(&cid, &name, &picture, &pid, &scheme, &ts)
		if err != nil {return nil, err}

		result = append(result, UserProfile {
			ID: cid,
			Name: name.String,
			Picture: picture.String,
			Player: pid,
			Scheme: scheme.String,
			Timestamp: ts,
		})
	}

	err = rows.Err()
	if err != nil {return nil, err}

	return result, err
}
