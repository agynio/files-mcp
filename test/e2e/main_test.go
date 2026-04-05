//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/agynio/files-mcp/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	requestTimeout = 30 * time.Second
	maxFileSize    = int64(20 * 1024 * 1024)
)

var (
	mcpBaseURL       string
	mcpServer        *httptest.Server
	fakeFilesBackend *fakeFiles
)

func TestMain(m *testing.M) {
	fakeFilesBackend = newFakeFiles()
	server := mcpserver.New(fakeFilesBackend, mcpserver.Options{MaxFileSize: maxFileSize})
	mcpServer = httptest.NewServer(server.Handler())
	mcpBaseURL = mcpServer.URL

	exitCode := m.Run()
	mcpServer.Close()
	os.Exit(exitCode)
}

func connectMCP(t *testing.T) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "files-mcp-e2e", Version: "0.1.0"}, nil)
	transport := &mcp.StreamableClientTransport{Endpoint: mcpBaseURL}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	t.Cleanup(cancel)

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("connect mcp: %v", err)
	}
	t.Cleanup(func() {
		_ = session.Close()
	})
	return session
}

func uploadTestFile(t *testing.T, filename string, contentType string, content []byte) string {
	t.Helper()
	fileID, err := fakeFilesBackend.Upload(filename, contentType, content)
	if err != nil {
		t.Fatalf("upload file: %v", err)
	}
	return fileID
}

func callReadFile(t *testing.T, session *mcp.ClientSession, fileID string) *mcp.CallToolResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	t.Cleanup(cancel)
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "read_file",
		Arguments: map[string]any{"file_id": fileID},
	})
	if err != nil {
		t.Fatalf("call read_file: %v", err)
	}
	return result
}
