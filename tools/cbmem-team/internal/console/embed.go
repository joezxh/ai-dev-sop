// Package console · static-file server for the M2 console UI.
//
// We embed the single-file HTML at compile time so the binary remains
// self-contained — no need to ship /static/ alongside cbmem-team.exe.
package console

import (
	"embed"
	"io"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed static/*
var staticFS embed.FS

// StaticUIHandler serves the embedded M2 console at /ui/m2 and /ui/m2/*.
// Root path redirects to /ui/m2/.
func StaticUIHandler() gin.HandlerFunc {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic("console: static fs sub: " + err.Error())
	}
	return func(c *gin.Context) {
		path := c.Param("filepath")
		if path == "" || path == "/" {
			path = "/m2-console.html"
		}
		f, err := sub.Open(path[1:]) // strip leading /
		if err != nil {
			c.String(http.StatusNotFound, "ui asset not found: %s", path)
			return
		}
		defer f.Close()
		// Reasonable content-type defaults.
		switch {
		case endsWith(path, ".html"):
			c.Header("Content-Type", "text/html; charset=utf-8")
		case endsWith(path, ".js"):
			c.Header("Content-Type", "application/javascript")
		case endsWith(path, ".css"):
			c.Header("Content-Type", "text/css")
		}
		c.Header("Cache-Control", "no-cache")
		if _, err := io.Copy(c.Writer, f); err != nil {
			// best-effort; client probably disconnected
			return
		}
	}
}

func endsWith(s, suffix string) bool {
	if len(s) < len(suffix) {
		return false
	}
	return s[len(s)-len(suffix):] == suffix
}