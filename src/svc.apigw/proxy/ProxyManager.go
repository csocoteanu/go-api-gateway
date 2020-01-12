package proxy

import (
	"common/discovery/registry"
	"common/discovery/utils"
	protos "common/svcprotos/gen"
	"context"
	"fmt"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"log"
)

type registerCallback func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) (err error)

type ProxyManager struct {
	serverMux         *runtime.ServeMux
	handler           registry.ServiceRegisteredHandler
	grpcOpts          []grpc.DialOption
	registerCallbacks map[string]registerCallback
}

func NewProxyManager(handler registry.ServiceRegisteredHandler) *ProxyManager {
	callbacks := map[string]registerCallback{
		utils.PingPongServiceName: protos.RegisterPingPongServiceHandlerFromEndpoint,
		utils.EchoServiceName:     protos.RegisterEchoServiceHandlerFromEndpoint,
	}
	opts := []grpc.DialOption{grpc.WithInsecure()}

	m := ProxyManager{
		serverMux:         runtime.NewServeMux(),
		handler:           handler,
		grpcOpts:          opts,
		registerCallbacks: callbacks,
	}

	m.listenForRegistrants()

	return &m
}

func (m *ProxyManager) GetServerMux() *runtime.ServeMux {
	return m.serverMux
}

func (m *ProxyManager) listenForRegistrants() {
	go func() {
		for registrant := range m.handler.OnServiceRegistered() {
			fmt.Printf("Registering GRPC service=%s address=%s\n", registrant.ServiceName, registrant.ServiceBalancerAddress)

			callback, ok := m.registerCallbacks[registrant.ServiceName]
			if !ok {
				log.Printf("No service register handler for %s", registrant.ServiceName)
			}

			err := callback(context.Background(), m.serverMux, registrant.ServiceBalancerAddress, m.grpcOpts)
			if err != nil {
				fmt.Printf("Error encountered: %s", err.Error())
			}
		}
	}()
}
