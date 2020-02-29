package proxy

import (
	"common"
	pb "common/protos/gen"
	"context"
	"fmt"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"log"
)

type registerCallback func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) (err error)

type ProxyManager struct {
	serverMux         *runtime.ServeMux
	added             chan common.RegistrantInfo
	removed           chan common.RegistrantInfo
	grpcOpts          []grpc.DialOption
	registerCallbacks map[string]registerCallback
}

func NewProxyManager() *ProxyManager {
	callbacks := map[string]registerCallback{
		common.PingPongServiceName: pb.RegisterPingPongServiceHandlerFromEndpoint,
		common.EchoServiceName:     pb.RegisterEchoServiceHandlerFromEndpoint,
	}
	opts := []grpc.DialOption{grpc.WithInsecure()}

	m := ProxyManager{
		serverMux:         runtime.NewServeMux(),
		grpcOpts:          opts,
		registerCallbacks: callbacks,
		added:             make(chan common.RegistrantInfo),
		removed:           make(chan common.RegistrantInfo),
	}

	go m.listenForRegistrants()

	return &m
}

func (m *ProxyManager) OnServiceRegistered() chan common.RegistrantInfo {
	return m.added
}

func (m *ProxyManager) OnServiceUnregistered() chan common.RegistrantInfo {
	return m.removed
}

func (m *ProxyManager) GetServerMux() *runtime.ServeMux {
	return m.serverMux
}

func (m *ProxyManager) listenForRegistrants() {
	err := pb.RegisterRegistryServiceHandlerFromEndpoint(context.Background(), m.serverMux, common.RegistryAddress, m.grpcOpts)
	if err != nil {
		log.Printf("Error encountered: %s", err.Error())
	}

	for {
		select {
		case registrant := <-m.added:
			fmt.Printf("Registering GRPC service=%s address=%s\n", registrant.ServiceName, registrant.ServiceBalancerAddress)

			callback, ok := m.registerCallbacks[registrant.ServiceName]
			if !ok {
				log.Printf("No service register handler for %s", registrant.ServiceName)
			}

			err := callback(context.Background(), m.serverMux, registrant.ServiceBalancerAddress, m.grpcOpts)
			if err != nil {
				log.Printf("Error encountered: %s", err.Error())
			}
		case <-m.removed:
		}
	}
}
