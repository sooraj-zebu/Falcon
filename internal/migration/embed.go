package migration

import "embed"

//go:embed sql/*.sql
var MigrationFiles embed.FS
