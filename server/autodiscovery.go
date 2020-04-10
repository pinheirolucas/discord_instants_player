package server

import "github.com/grandcat/zeroconf"

func newAutodiscoveryServer(service string, port int) (*zeroconf.Server, error) {
	server, err := zeroconf.Register(
		"DiscordInstantsPlayer",
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
