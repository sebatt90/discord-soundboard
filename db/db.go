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
BEGIN TRANSACTION;
CREATE TABLE IF NOT EXISTS "guilds" (
	"id"	TEXT,
	"name"	TEXT NOT NULL,
	"iconhash" TEXT NOT NULL,
	PRIMARY KEY("id")
);
CREATE TABLE IF NOT EXISTS "tracks" (
	"id"	INTEGER,
	"name"	TEXT NOT NULL,
	"data"	BLOB NOT NULL,
	"guildid"	TEXT,
	PRIMARY KEY("id" AUTOINCREMENT),
	CONSTRAINT "guildid" FOREIGN KEY("guildid") REFERENCES "guilds"("id")
);
COMMIT;
	`)
	return err
}

func GetTrack(query string, guildid string) (track models.Track, err error) {
	stmt, err := DB.Prepare(`SELECT id, name, data FROM tracks WHERE name LIKE ? AND guildid LIKE ? LIMIT 1`)
	if err != nil {
		return track, err
	}
	defer stmt.Close()

	err = stmt.QueryRow("%" + query + "%",guildid).Scan(&track.ID, &track.Name, &track.Data)
	return track, err
}

func GetGuildByID(id string) (guild models.Guild, err error){
	stmt, err := DB.Prepare(`SELECT id, name, iconhash FROM guilds WHERE id LIKE ? LIMIT 1`)

	if err != nil {
		return guild, err
	}
	defer stmt.Close()

	err = stmt.QueryRow(id).Scan(&guild.ID, &guild.Name, &guild.IconHash)
	return guild,err
}

func GetGuilds() (guilds []models.Guild, err error){
	rows, err := DB.Query(`SELECT id, name, iconhash FROM guilds`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var g models.Guild
		if err := rows.Scan(&g.ID, &g.Name, &g.IconHash); err != nil {
			return nil, err
		}
		guilds = append(guilds, g)
	}
	return guilds, rows.Err()
}

func GetTracksPerGuild(guildid string) (tracks []models.Track, err error) {
	stmt, err := DB.Prepare(`SELECT id, name, data FROM tracks WHERE guildid LIKE ?;`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var rows *sql.Rows
	rows, err = stmt.Query(guildid)

	for rows.Next() {
		var t models.Track
		if err := rows.Scan(&t.ID, &t.Name, &t.Data); err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}

func DeleteTrack(guildID string, trackID int) error {
	stmt, err := DB.Prepare(`DELETE FROM tracks WHERE id = ? AND guildid = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(trackID, guildID)
	return err
}

func InsertTrack(guildID string, name string, data []byte) error {
    stmt, err := DB.Prepare(`INSERT INTO tracks (guildid, name, data) VALUES (?, ?, ?)`)
    if err != nil {
        return err
    }
    defer stmt.Close()

    _, err = stmt.Exec(guildID, name, data)
    return err
}

func UpdateTrack(id int, name string, data []byte) error {
    stmt, err := DB.Prepare(`UPDATE tracks SET name = ?, data = ? WHERE id = ?`)
    if err != nil {
        return err
    }
    defer stmt.Close()

    _, err = stmt.Exec(name, data, id)
    return err
}

func InsertGuild(id string, name string, iconhash string) (err error) {
	stmt, err := DB.Prepare(`INSERT OR REPLACE INTO guilds(id, name, iconhash) VALUES (?,?,?);`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(id,name, iconhash)
	return err
}

func Close() {
	DB.Close()
}
