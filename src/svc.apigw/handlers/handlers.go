package handlers

import (
	"common/discovery/registry"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"log"
	"net/http"
	"path"
	"strings"
)

var dir = "./pb/"

type APIManager struct {
	provider registry.ActiveServicesProvider
}

func NewAPIManager(provider registry.ActiveServicesProvider) *APIManager {
	m := APIManager{
		provider: provider,
	}

	return &m
}

func (m *APIManager) ServicesHandler(w http.ResponseWriter, r *http.Request) {
	services := m.provider.GetActiveServices()
	servicesBytes, err := json.Marshal(services)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte(fmt.Sprintf("Error: %s", err.Error())))
		if err != nil {
			log.Printf("Unable to service /services request! err=%s", err.Error())
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(servicesBytes)
	if err != nil {
		log.Printf("Unable to service /services request! err=%s", err.Error())
	}
}

func (m *APIManager) SwaggerHandler(w http.ResponseWriter, r *http.Request) {
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

func (m *APIManager) GetWebservice() *http.ServeMux {
	svc := http.NewServeMux()
	svc.HandleFunc("/services", m.ServicesHandler)
	svc.HandleFunc("/swagger", m.SwaggerHandler)

	return svc
}
