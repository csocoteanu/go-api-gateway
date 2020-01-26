package handlers

import (
	"common/clients/discovery"
	"common/discovery/domain"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"
	"strings"
)

const (
	swaggerDirectory = "../common/svcprotos/swagger"
	swaggerDistro    = "../swagger-ui/dist"
)

type ServiceInfo struct {
	BalancerAddress string   `json:"balancer"`
	LocalAddresses  []string `json:"locals"`
}

type APIManager struct {
	handler domain.RegisterHandler
	client  discovery.Client
}

func NewAPIManager(handler domain.RegisterHandler, client discovery.Client) *APIManager {
	m := APIManager{
		handler: handler,
		client:  client,
	}

	return &m
}

func (m *APIManager) ServicesHandler(w http.ResponseWriter, r *http.Request) {
	services, err := m.client.GetServices()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte(fmt.Sprintf("Error: %s", err.Error())))
		if err != nil {
			log.Printf("Unable to service /services request! err=%s", err.Error())
		}
		return
	}

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

	for _, svc := range services {
		m.handler.OnServiceRegistered() <- *svc
	}
}

func (m *APIManager) SwaggerHandler(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.URL.Path, ".swagger.json") {
		log.Printf("Not Found: %s", r.URL.Path)
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	p := strings.TrimPrefix(r.URL.Path, "/swagger/")
	p = path.Join(swaggerDirectory, p)
	log.Printf("Serving %s [%s]", r.URL.Path, p)

	http.ServeFile(w, r, p)
}

func (m *APIManager) GetWebservice() *http.ServeMux {
	svc := http.NewServeMux()
	fs := http.FileServer(http.Dir(swaggerDistro))

	svc.HandleFunc("/services", m.ServicesHandler)
	svc.HandleFunc("/swagger/", m.SwaggerHandler)
	svc.Handle("/swagger-dist/", http.StripPrefix("/swagger-dist/", fs))

	return svc
}
