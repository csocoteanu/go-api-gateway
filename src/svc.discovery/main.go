package main

import (
	"flag"
	"svc.discovery/usecases"
)

var useTCPLoadBalacing = false

func parseArgs() {
	flag.BoolVar(&useTCPLoadBalacing, "use-tcp-load-balancing", false, "Use TCP load balancing")
	flag.Parse()
}

func main() {
	parseArgs()

	registry := usecases.NewServiceRegistryServer("localhost", 8500)
	registryInteractor := usecases.NewServiceRegistryInteractor(registry)

	if useTCPLoadBalacing {
		registry.RegisterHandler(registryInteractor)
	}

	registry.Start()
}
