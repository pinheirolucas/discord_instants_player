package server

import (
	"fmt"
	"os"

	"github.com/grandcat/zeroconf"
)

func newAutodiscoveryServer(service string, port int) (*zeroconf.Server, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	instanceName := fmt.Sprintf(
		"%s-%d",
		hostname,
		port,
	)

	server, err := zeroconf.Register(
		instanceName,
		service,
		"local.",
		port,
		nil,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return server, nil
}
