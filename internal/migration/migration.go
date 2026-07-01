package migration

import (
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
)

type Migrator struct {
	db *sql.DB
}

func New(db *sql.DB) *Migrator {
	return &Migrator{
		db: db,
	}
}

func (m *Migrator) Run() error {

	files, err := fs.Glob(MigrationFiles, "sql/*.sql")
	if err != nil {
		return err
	}

	sort.Strings(files)

	for _, file := range files {

		fmt.Println("Running migration:", file)

		data, err := MigrationFiles.ReadFile(file)
		if err != nil {
			return err
		}

		_, err = m.db.Exec(string(data))
		if err != nil {
			return err
		}
	}

	return nil
}
