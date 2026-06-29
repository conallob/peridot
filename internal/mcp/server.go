package mcp

import (
	"github.com/mark3labs/mcp-go/server"
)

// Serve starts an MCP stdio server that bridges tool calls to the peridot daemon
// over the IPC socket at socketPath. It blocks until stdin is closed.
func Serve(socketPath string) error {
	srv := server.NewMCPServer(
		"peridot",
		"0.1.0",
		server.WithToolCapabilities(true),
	)
	registerTools(srv, newBridge(socketPath))
	return server.ServeStdio(srv)
}
