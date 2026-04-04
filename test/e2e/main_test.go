//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	filesv1 "github.com/agynio/files-mcp/.gen/go/agynio/api/files/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const requestTimeout = 30 * time.Second

var (
	filesAddress = envOrDefault("FILES_ADDRESS", "files:50051")
	mcpBaseURL   = envOrDefault("MCP_BASE_URL", "http://files-mcp:8100")
)

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
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
	conn, err := grpc.NewClient(filesAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial files service: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})
	client := filesv1.NewFilesServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	t.Cleanup(cancel)
	stream, err := client.UploadFile(ctx)
	if err != nil {
		t.Fatalf("create upload stream: %v", err)
	}

	metadata := &filesv1.UploadFileMetadata{
		Filename:    filename,
		ContentType: contentType,
		SizeBytes:   int64(len(content)),
	}
	if err := stream.Send(&filesv1.UploadFileRequest{Payload: &filesv1.UploadFileRequest_Metadata{Metadata: metadata}}); err != nil {
		t.Fatalf("send metadata: %v", err)
	}
	if err := stream.Send(&filesv1.UploadFileRequest{Payload: &filesv1.UploadFileRequest_Chunk{Chunk: &filesv1.UploadFileChunk{Data: content}}}); err != nil {
		t.Fatalf("send chunk: %v", err)
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		t.Fatalf("close upload stream: %v", err)
	}
	if resp == nil || resp.GetFile() == nil || resp.GetFile().GetId() == "" {
		t.Fatalf("upload response missing file info")
	}
	return resp.GetFile().GetId()
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

func TestE2EPlaceholder(t *testing.T) {
	t.Skip("e2e tests require the full platform stack")
}
