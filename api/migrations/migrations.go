// Package migrations embeds the SQL migration files so the API and the migrate
// command ship as single binaries with their schema.
package migrations

import "embed"

// FS holds every *.sql migration file, named
// <version>_<name>.up.sql / .down.sql as expected by golang-migrate.
//
//go:embed *.sql
var FS embed.FS
