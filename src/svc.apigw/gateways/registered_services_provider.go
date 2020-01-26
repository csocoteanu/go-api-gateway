package gateways

import "common/discovery/domain"

type registeredServicesProvider struct {
	registrants []domain.RegistrantInfo
	handlers    []domain.RegisterHandler
}

func NewRegisteredServicesProvider() domain.ServiceRegistry {
	sp := registeredServicesProvider{
		registrants: getRegistrants(),
	}

	return &sp
}

func (sp *registeredServicesProvider) RegisterHandler(h domain.RegisterHandler) {
	sp.handlers = append(sp.handlers, h)
}

func (sp *registeredServicesProvider) Start() {
	for _, h := range sp.handlers {
		for _, ri := range sp.registrants {
			h.OnServiceRegistered() <- ri
		}
	}
}

func getRegistrants() []domain.RegistrantInfo {
	return []domain.RegistrantInfo{
		{
			ServiceName:            domain.EchoServiceName,
			ServiceBalancerAddress: "localhost:10010",
		},
		{
			ServiceName:            domain.PingPongServiceName,
			ServiceBalancerAddress: "localhost:9090",
		},
	}
}
