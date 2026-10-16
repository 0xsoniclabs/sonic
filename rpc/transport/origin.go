// Copyright 2026 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

package transport

import (
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/log"
)

// originChecker returns the WebSocket origin check of go-ethereum: a request
// without Origin header passes (the check protects against browser based
// attacks and browsers always set it), "*" admits any origin, no configuration
// admits localhost and the host name of the machine.
func originChecker(allowedOrigins []string) func(*http.Request) bool {
	origins := map[string]bool{}
	allowAll := false
	for _, origin := range allowedOrigins {
		if origin == "*" {
			allowAll = true
		}
		if origin != "" {
			origins[origin] = true
		}
	}
	if len(origins) == 0 {
		origins["http://localhost"] = true
		if hostname, err := os.Hostname(); err == nil {
			origins["http://"+hostname] = true
		}
	}
	return func(req *http.Request) bool {
		if _, ok := req.Header["Origin"]; !ok {
			return true
		}
		origin := strings.ToLower(req.Header.Get("Origin"))
		if allowAll || originIsAllowed(origins, origin) {
			return true
		}
		log.Warn("Rejected WebSocket connection", "origin", origin)
		return false
	}
}

func originIsAllowed(allowed map[string]bool, browserOrigin string) bool {
	for origin := range allowed {
		if ruleAllowsOrigin(origin, browserOrigin) {
			return true
		}
	}
	return false
}

func ruleAllowsOrigin(allowedOrigin string, browserOrigin string) bool {
	allowedScheme, allowedHostname, allowedPort, err := parseOriginURL(allowedOrigin)
	if err != nil {
		log.Warn("Error parsing allowed origin specification", "spec", allowedOrigin, "error", err)
		return false
	}
	browserScheme, browserHostname, browserPort, err := parseOriginURL(browserOrigin)
	if err != nil {
		log.Warn("Error parsing browser 'Origin' field", "Origin", browserOrigin, "error", err)
		return false
	}
	if allowedScheme != "" && allowedScheme != browserScheme {
		return false
	}
	if allowedHostname != "" && allowedHostname != browserHostname {
		return false
	}
	if allowedPort != "" && allowedPort != browserPort {
		return false
	}
	return true
}

func parseOriginURL(origin string) (string, string, string, error) {
	parsedURL, err := url.Parse(strings.ToLower(origin))
	if err != nil {
		return "", "", "", err
	}
	var scheme, hostname, port string
	if strings.Contains(origin, "://") {
		scheme = parsedURL.Scheme
		hostname = parsedURL.Hostname()
		port = parsedURL.Port()
	} else {
		scheme = ""
		hostname = parsedURL.Scheme
		port = parsedURL.Opaque
		if hostname == "" {
			hostname = origin
		}
	}
	return scheme, hostname, port, nil
}

// isWebsocket checks the header of an HTTP request for a WebSocket upgrade.
func isWebsocket(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket") &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}
