package storage

import (
	"database/sql"
	"os"

	_ "modernc.org/sqlite"
)

func Open(chemin string, fichierMigration string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", chemin)
	if err != nil {
		return nil, err
	}

	// evite les erreurs "database is locked"
	db.SetMaxOpenConns(1)

	// sqlite n'active pas les cles etrangeres par defaut
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		return nil, err
	}

	schema, err := os.ReadFile(fichierMigration)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(string(schema))
	if err != nil {
		return nil, err
	}

	return db, nil
}
