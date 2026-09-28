package migrations

import "embed"

// Files contains the canonical Atlas migration directory embedded in the API
// binary. atlas.sum is embedded with the SQL files so startup validates the
// exact same migration history used by the Atlas CLI.
//
//go:embed *.sql atlas.sum
var Files embed.FS
