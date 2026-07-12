package console

import "time"

// nowUTC returns the current UTC time formatted for our DATETIME columns.
// Both SQLite and MySQL accept this layout (`2006-01-02 15:04:05`) so the
// helper works on both backends without conversion.
func nowUTC() string {
	return time.Now().UTC().Format("2006-01-02 15:04:05")
}
