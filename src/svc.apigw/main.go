package main

import (
	discovery "common/discovery/registry"
	"svc.apigw/handlers"
	"svc.apigw/proxy"

	"log"
	"net/http"
)

func main() {
	registry := discovery.NewServiceRegistry("localhost", 8500).(*discovery.ServiceRegistry)
	apiManager := handlers.NewAPIManager(registry)
	proxyManaer := proxy.NewProxyManager(registry)

	svc := apiManager.GetWebservice()
	mux := proxyManaer.GetServerMux()
	svc.Handle("/", mux)

	err := http.ListenAndServe(":8080", svc)
	if err != nil {
		log.Fatalf("Error starting server: %+v", err)
	}
}
