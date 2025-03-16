package main

import (
	"database/sql"
	"fmt"
	"log"
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