// Package migrations carries this service's SQL files as an embedded fs.FS,
// to be applied by pkg/dbmigrate at boot.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
