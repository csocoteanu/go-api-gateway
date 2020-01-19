package domain

type RegistrantInfo struct {
	ServiceName            string
	ServiceBalancerAddress string
	ServiceLocalAddress    string
	ControlAddress         string
}

type RegistrantInfoRepository interface {
	StoreRegistrantInfo(*RegistrantInfo) error
	RemoveRegistrantInfo(*RegistrantInfo) error
	GetAllRegistrantInfos() ([]*RegistrantInfo, error)
}

type RegisterHandler interface {
	OnServiceRegistered() chan RegistrantInfo
	OnServiceUnregistered() chan RegistrantInfo
}

type ServiceRegistry interface {
	RegisterHandler(RegisterHandler)
}

type TCPLoadBalancer interface {
	AddProxy(proxy string)
	RemoveProxy(proxy string)
	GetProxies() []string
}

func NewRegistrantInfo(
	controlAddress string,
	serviceName, balancerAddress, localAddress string) RegistrantInfo {

	ri := RegistrantInfo{
		ControlAddress:         controlAddress,
		ServiceName:            serviceName,
		ServiceBalancerAddress: balancerAddress,
		ServiceLocalAddress:    localAddress,
	}

	return ri
}
