package main

import (
	"hostex-matrix-bridge/pkg/connector"
	
	"maunium.net/go/mautrix/bridgev2/matrix/mxmain"
)

var (
	// These will be filled by the build system
	Tag       = "unknown"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	m := mxmain.BridgeMain{
		Name:        "mautrix-hostex",
		Description: "A Matrix bridge for Hostex property management system",
		URL:         "https://github.com/keithah/matrix-hostex-bridge",
		Version:     "0.1.2",
		Connector:   &connector.HostexConnector{}, // Switch back to full connector
	}
	m.InitVersion(Tag, Commit, BuildTime)
	m.Run()
}