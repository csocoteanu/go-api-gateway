package utils

import (
	discovery "common/discovery/protos"
	"fmt"
	"github.com/pkg/errors"
	"strings"
)

var (
	ErrHeartBeatFailed   = errors.New("heartbeat request failed")
	ErrRegisterFailed    = errors.New("failed registering registrant")
	ErrRegistrantExists  = errors.New("registrant exists")
	ErrRegistrantMissing = errors.New("reqistrant does not exist")
	ErrInvalidRequest    = errors.New("invalid request")
)

// AggregateErrors aggregates errors in a single error
func AggregateErrors(errs ...*discovery.Error) error {
	var sb strings.Builder
	for _, err := range errs {
		sb.WriteString(fmt.Sprintf("[%d] %s;", err.Code, err.Message))
	}

	return fmt.Errorf("encountered errors: %s", sb.String())
}
