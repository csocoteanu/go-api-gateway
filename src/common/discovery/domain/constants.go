package domain

import (
	discovery "common/discovery/domain/protos"
	"fmt"
	"github.com/pkg/errors"
	"strings"
)

const (
	ACK                 = "ACK"
	PingPongServiceName = "pingpong"
	EchoServiceName     = "echo"
)

var (
	ErrHeartBeatFailed   = errors.New("heartbeat request failed")
	ErrRegisterFailed    = errors.New("failed registering registrant")
	ErrRegistrantExists  = errors.New("registrant exists")
	ErrRegistrantMissing = errors.New("reqistrant does not exist")
	ErrInvalidRequest    = errors.New("invalid request")

	RegistryAddress = ToGRPCAddress("localhost", 8500)
)

// AggregateErrors aggregates errors in a single error
func AggregateErrors(errs ...*discovery.Error) error {
	var sb strings.Builder
	for _, err := range errs {
		sb.WriteString(fmt.Sprintf("[%d] %s;", err.Code, err.Message))
	}

	return fmt.Errorf("encountered errors: %s", sb.String())
}

// ToGRPCAddress creates a new gRPC address
func ToGRPCAddress(hostname string, port uint32) string {
	return fmt.Sprintf("%s:%d", hostname, port)
}
