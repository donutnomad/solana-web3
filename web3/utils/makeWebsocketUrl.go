package utils

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

var URL_RE = regexp.MustCompile(`^[^:]+:\/\/([^:[]+|\[[^\]]+\])(:\d+)?(.*)`)

// MakeWebsocketURL generates a WebSocket URL based on the provided endpoint
func MakeWebsocketURL(endpoint string) (string, error) {
	matches := URL_RE.FindStringSubmatch(endpoint)
	if matches == nil {
		return "", errors.New("Failed to validate endpoint URL `" + endpoint + "`")
	}

	_, hostish, portWithColon, rest := matches[0], matches[1], matches[2], matches[3]

	protocol := "ws:"
	if strings.HasPrefix(endpoint, "https:") {
		protocol = "wss:"
	}

	var startPort *int
	if portWithColon != "" {
		port, err := strconv.Atoi(portWithColon[1:])
		if err != nil {
			return "", err
		}
		startPort = &port
	}

	var websocketPort string
	if startPort != nil {
		// Only shift the port by +1 as a convention for ws(s) only if given endpoint
		// is explicitly specifying the endpoint port (HTTP-based RPC), assuming
		// we're directly trying to connect to solana-validator's ws listening port.
		// When the endpoint omits the port, we're connecting to the protocol
		// default ports: http(80) or https(443) and it's assumed we're behind a reverse
		// proxy which manages WebSocket upgrade and backend port redirection.
		websocketPort = ":" + strconv.Itoa(*startPort+1)
	}

	return protocol + "//" + hostish + websocketPort + rest, nil
}
