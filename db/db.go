package db

import (
	"database/sql"
	
	_ "github.com/mattn/go-sqlite3"
	"github.com/sebatt90/discord-soundboard/models"
)

var DB *sql.DB

func Init(path string) error {
	var err error
	DB, err = sql.Open("sqlite3", path)
	if err != nil {
		return err
	}

	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS tracks (
			id   INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			data BLOB NOT NULL
		)
	`)
	return err
}

func GetTrack(query string) (track models.Track, err error) {
	stmt, err := DB.Prepare(`SELECT name, data FROM tracks WHERE name LIKE ? LIMIT 1`)
	if err != nil {
		return track, err
	}
	defer stmt.Close()

	err = stmt.QueryRow("%" + query + "%").Scan(&track.Name, &track.Data)
	return track, err
}

func Close() {
	DB.Close()
}
