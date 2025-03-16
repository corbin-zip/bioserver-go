package main

import (
	"database/sql"
	"fmt"
	"log"
	"io"
	"bytes"
	
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
	_ "github.com/go-sql-driver/mysql"
)

type Database struct {
	db *sql.DB
}

func NewDatabase(dbUser, dbPassword string) (*Database, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(localhost:3306)/bioserver?charset=utf8", dbUser, dbPassword)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	log.Println("Database connection established")
	database := &Database{db: db}
	if err = database.setupDBRestart(); err != nil {
		log.Printf("setupDBRestart warning: %v", err)
	}
	return database, nil
}

func (d *Database) setupDBRestart() error {
	_, err := d.db.Exec("UPDATE sessions SET area=-1, room=0, slot=0, gamesess=0, state=0")
	return err
}

func (d *Database) GetUserID(session string) (string, error) {
	var userID string
	err := d.db.QueryRow("SELECT userid FROM sessions WHERE sessid=?", session).Scan(&userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user id: %w", err)
	}
	return userID, nil
}

func (d *Database) CheckHandle(handle string) (bool, error) {
	var count int
	err := d.db.QueryRow("SELECT count(*) as cnt FROM hnpairs WHERE handle=?", handle).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to get count for handle %s: %w", handle, err)
	}
	if count == 0 {
		return true, nil
	} else {
		return false, nil
	}
}

func decodeSJIS(b []byte) (string, error) {
	r := transform.NewReader(bytes.NewReader(b), japanese.ShiftJIS.NewDecoder())
	decoded,err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func (d * Database) CreateNewHNPair(cl *Client) {
	uid := cl.userID
	handle := string(cl.hnPair.handle)
	var nickname string

	if dec, err := decodeSJIS(cl.hnPair.nickname); err != nil {
		nickname = "sjis"
		log.Printf("ShiftJIS decoding failed: %v", err)
	} else {
		nickname = dec
	}
	query := "INSERT INTO hnpairs (userid, handle, nickname) VALUES (?, ?, ?)"

	if _, err := d.db.Exec(query, uid, handle, nickname); err != nil {
		log.Printf("Failed to insert HNPair: %v", err)
	}

}

// what is the point of this ...?
func (d * Database) UpdateHNPair(cl *Client) {
	uid := cl.userID
	handle := string(cl.hnPair.handle)
	var nickname string

	if dec, err := decodeSJIS(cl.hnPair.nickname); err != nil {
		nickname = "sjis"
		log.Printf("ShiftJIS decoding failed: %v", err)
	} else {
		nickname = dec
	}
	query := "UPDATE hnpairs SET nickname=? WHERE userid=? AND handle=?"

	if _, err := d.db.Exec(query, nickname, uid, handle); err != nil {
		log.Printf("Failed to update HNPair: %v", err)
	}
}

func (d *Database) UpdateClientOrigin(userid string, state, area, room, slot int) error {
	_, err := d.db.Exec("UPDATE sessions SET state=?, area=?, room=?, slot=? WHERE userid=?", state, area, room, slot, userid)
	return err
}

func (d *Database) GetGameNumber(userid string) (int, error) {
	var gameNumber int
	err := d.db.QueryRow("SELECT gamesess FROM sessions WHERE userid=?", userid).Scan(&gameNumber)
	if err != nil {
		return 0, fmt.Errorf("failed to get game number: %w", err)
	}
	fmt.Println("Returning game number:", gameNumber)
	return gameNumber, nil
}

func (d *Database) GetHNPairs(userid string) *HNPairs {
	hnpairs := NewHNPairs()
	rows, err := d.db.Query("SELECT handle, nickname FROM hnpairs WHERE userid=?", userid)
	if err != nil {
		log.Printf("failed to get HN pairs: %v", err)
		return hnpairs
	}
	defer rows.Close()
	for rows.Next() {
		var handle, nickname string
		if err := rows.Scan(&handle, &nickname); err != nil {
			log.Printf("failed to scan HN pair: %v", err)
			continue
		}
		hnpairs.Add(NewHNPairFromStrings(handle, nickname))
	}
	return hnpairs
}

func (d *Database) GetMOTD() (string, error) {
	var motd string
	err := d.db.QueryRow("SELECT message FROM motd WHERE active=1 ORDER BY id DESC LIMIT 0,1").Scan(&motd)
	if err != nil {
		return "", fmt.Errorf("failed to get MOTD: %w", err)
	}
	fmt.Println("Returning MOTD:", motd)
	return motd, nil
}