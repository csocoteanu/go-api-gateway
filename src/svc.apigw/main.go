package main

import (
	"common/clients/discovery"
	"github.com/pkg/errors"
	"svc.apigw/gateways"
	"svc.apigw/usecases/handlers"
	"svc.apigw/usecases/proxy"

	"log"
	"net/http"
)

func main() {
	discoveryClient, err := discovery.NewClient("localhost", 8500)
	if err != nil {
		panic(errors.Wrapf(err, "Failed starting api gw! err=%s", err.Error()))
	}

	proxyManager := proxy.NewProxyManager()
	apiManager := handlers.NewAPIManager(proxyManager, discoveryClient)

	svc := apiManager.GetWebservice()
	mux := proxyManager.GetServerMux()
	svc.Handle("/", mux)

	servicesProvider := gateways.NewRegisteredServicesProvider()
	servicesProvider.RegisterHandler(proxyManager)
	servicesProvider.Start()

	err = http.ListenAndServe(":8080", svc)
	if err != nil {
		log.Fatalf("Error starting server: %+v", err)
	}
}
