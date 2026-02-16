package store

import (
	"database/sql"
	_ "embed"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

func Open(path string) (*Queries, *sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, nil, err
	}
	return New(db), db, nil
}
