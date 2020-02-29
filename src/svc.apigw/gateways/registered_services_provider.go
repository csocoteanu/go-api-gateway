package gateways

import "common"

type registeredServicesProvider struct {
	registrants []common.RegistrantInfo
	handlers    []common.RegisterHandler
}

func NewRegisteredServicesProvider() common.ServiceRegistry {
	sp := registeredServicesProvider{
		registrants: getRegistrants(),
	}

	return &sp
}

func (sp *registeredServicesProvider) RegisterHandler(h common.RegisterHandler) {
	sp.handlers = append(sp.handlers, h)
}

func (sp *registeredServicesProvider) Start() {
	for _, h := range sp.handlers {
		for _, ri := range sp.registrants {
			h.OnServiceRegistered() <- ri
		}
	}
}

func getRegistrants() []common.RegistrantInfo {
	return []common.RegistrantInfo{
		{
			ServiceName:            common.EchoServiceName,
			ServiceBalancerAddress: "localhost:10010",
		},
		{
			ServiceName:            common.PingPongServiceName,
			ServiceBalancerAddress: "localhost:9090",
		},
	}
}
