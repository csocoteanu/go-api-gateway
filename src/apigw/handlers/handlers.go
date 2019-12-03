package handlers

import (
	"github.com/golang/glog"
	"net/http"
	"path"
	"strings"
)

var dir = "./pb/"

func SwaggerHandler(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.URL.Path, ".swagger.json") {
		glog.Errorf("Not Found: %s", r.URL.Path)
		http.NotFound(w, r)
		return
	}

	p := strings.TrimPrefix(r.URL.Path, "/swagger/")
	p = path.Join(dir, p)
	glog.Infof("Serving %s [%s]", r.URL.Path, p)

	http.ServeFile(w, r, p)
}
