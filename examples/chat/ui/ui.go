//go:generate sh -c "GOBIN=`pwd`/bin go install github.com/gobuffalo/packr/v2/packr2"
//go:generate sh -c "./bin/packr2"
package ui

import (
	"net/http"

	"github.com/gobuffalo/packr/v2"
)

var box = packr.New("static", "./static")
var server = http.FileServer(box)

func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	server.ServeHTTP(w, r)
}
