package main

import (
	"common/discovery/gateways"
	"common/discovery/usecases"
	"svc.apigw/usecases/handlers"
	"svc.apigw/usecases/proxy"

	"log"
	"net/http"
)

func main() {
	registry := gateways.NewServiceRegistryServer("localhost", 8500)
	registryInteractor := usecases.NewServiceRegistryInteractor(registry)
	apiManager := handlers.NewAPIManager(registry)
	proxyManager := proxy.NewProxyManager()

	registry.RegisterHandler(proxyManager)
	registry.RegisterHandler(registryInteractor)

	svc := apiManager.GetWebservice()
	mux := proxyManager.GetServerMux()
	svc.Handle("/", mux)

	err := http.ListenAndServe(":8080", svc)
	if err != nil {
		log.Fatalf("Error starting server: %+v", err)
	}
}
