package main

import (
	"common/clients/discovery"
	"github.com/pkg/errors"
	"svc.apigw/usecases/handlers"
	"svc.apigw/usecases/proxy"

	"log"
	"net/http"
)

func main() {
	proxyManager := proxy.NewProxyManager()
	discoveryClient, err := discovery.NewClient("localhost", 8500)
	if err != nil {
		panic(errors.Wrapf(err, "Failed starting api gw! err=%s", err.Error()))
	}

	apiManager := handlers.NewAPIManager(proxyManager, discoveryClient)

	svc := apiManager.GetWebservice()
	mux := proxyManager.GetServerMux()
	svc.Handle("/", mux)

	err = http.ListenAndServe(":8080", svc)
	if err != nil {
		log.Fatalf("Error starting server: %+v", err)
	}
}
