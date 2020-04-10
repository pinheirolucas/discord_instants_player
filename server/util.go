package server

import (
	"strconv"
	"strings"
)

func getHostAndPortFromAddress(address string) (string, int) {
	addrport := strings.Split(address, ":")
	if len(addrport) != 2 {
		return "", 0
	}

	port, err := strconv.Atoi(addrport[1])
	if err != nil {
		return "", 0
	}

	return addrport[0], port
}
