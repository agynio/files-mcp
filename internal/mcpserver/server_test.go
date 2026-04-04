package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"

	filesv1 "github.com/agynio/files-mcp/.gen/go/agynio/api/files/v1"
	"github.com/agynio/files-mcp/internal/filesclient"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeFilesClient struct {
	metadata    *filesv1.FileInfo
	metadataErr error
	contentErr  error
	chunks      [][]byte
	streamErr   error
}

func (f *fakeFilesClient) GetFileMetadata(context.Context, string) (*filesv1.FileInfo, error) {
	if f.metadataErr != nil {
		return nil, f.metadataErr
	}
	return f.metadata, nil
}

func (f *fakeFilesClient) GetFileContent(context.Context, string) (filesclient.FileContentStream, error) {
	if f.contentErr != nil {
		return nil, f.contentErr
	}
	return &fakeStream{chunks: f.chunks, err: f.streamErr}, nil
}

type fakeStream struct {
	chunks [][]byte
	err    error
	index  int
}

func (f *fakeStream) Recv() (*filesv1.GetFileContentResponse, error) {
	if f.err != nil {
		err := f.err
		f.err = nil
		return nil, err
	}
	if f.index >= len(f.chunks) {
		return nil, io.EOF
	}
	resp := &filesv1.GetFileContentResponse{ChunkData: f.chunks[f.index]}
	f.index++
	return resp, nil
}

func TestReadFileTextContent(t *testing.T) {
	client := &fakeFilesClient{
		metadata: &filesv1.FileInfo{
			Id:          "file-1",
			ContentType: "text/plain",
			SizeBytes:   11,
		},
		chunks: [][]byte{[]byte("hello "), []byte("world")},
	}
	server := New(client, Options{MaxFileSize: 20})

	result := callReadFile(t, server, "file-1")
	text := expectTextContent(t, result)
	if text != "hello world" {
		t.Fatalf("text content = %q, want %q", text, "hello world")
	}
}

func TestReadFileImageContent(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03}
	client := &fakeFilesClient{
		metadata: &filesv1.FileInfo{
			Id:          "file-2",
			ContentType: "image/png",
			SizeBytes:   int64(len(data)),
		},
		chunks: [][]byte{data},
	}
	server := New(client, Options{MaxFileSize: 20})

	result := callReadFile(t, server, "file-2")
	image := expectImageContent(t, result)
	if !bytes.Equal(image.Data, data) {
		t.Fatalf("image data = %v, want %v", image.Data, data)
	}
	if image.MIMEType != "image/png" {
		t.Fatalf("mime type = %q, want %q", image.MIMEType, "image/png")
	}
}

func TestReadFileBinaryContent(t *testing.T) {
	data := []byte("%PDF-1.4")
	client := &fakeFilesClient{
		metadata: &filesv1.FileInfo{
			Id:          "file-3",
			ContentType: "application/pdf",
			SizeBytes:   int64(len(data)),
		},
		chunks: [][]byte{data},
	}
	server := New(client, Options{MaxFileSize: 20})

	result := callReadFile(t, server, "file-3")
	resource := expectResourceContent(t, result)
	if resource.URI != "agyn://file/file-3" {
		t.Fatalf("resource uri = %q, want %q", resource.URI, "agyn://file/file-3")
	}
	if resource.MIMEType != "application/pdf" {
		t.Fatalf("resource mime type = %q, want %q", resource.MIMEType, "application/pdf")
	}
	if !bytes.Equal(resource.Blob, data) {
		t.Fatalf("resource blob = %v, want %v", resource.Blob, data)
	}
}

func TestReadFileJSONContent(t *testing.T) {
	data := []byte(`{"ok":true}`)
	client := &fakeFilesClient{
		metadata: &filesv1.FileInfo{
			Id:          "file-4",
			ContentType: "application/json",
			SizeBytes:   int64(len(data)),
		},
		chunks: [][]byte{data},
	}
	server := New(client, Options{MaxFileSize: 20})

	result := callReadFile(t, server, "file-4")
	text := expectTextContent(t, result)
	if text != string(data) {
		t.Fatalf("text content = %q, want %q", text, string(data))
	}
}

func TestReadFileNotFound(t *testing.T) {
	client := &fakeFilesClient{metadataErr: status.Error(codes.NotFound, "missing")}
	server := New(client, Options{MaxFileSize: 20})

	result := callReadFile(t, server, "missing-file")
	text := expectErrorContent(t, result)
	if text != "file not found: missing-file" {
		t.Fatalf("error = %q, want %q", text, "file not found: missing-file")
	}
}

func TestReadFileTooLarge(t *testing.T) {
	client := &fakeFilesClient{
		metadata: &filesv1.FileInfo{
			Id:          "file-5",
			ContentType: "text/plain",
			SizeBytes:   25,
		},
	}
	server := New(client, Options{MaxFileSize: 20})

	result := callReadFile(t, server, "file-5")
	text := expectErrorContent(t, result)
	if text != "file too large: 25 bytes exceeds limit of 20 bytes" {
		t.Fatalf("error = %q, want %q", text, "file too large: 25 bytes exceeds limit of 20 bytes")
	}
}

func TestReadFileEmptyID(t *testing.T) {
	client := &fakeFilesClient{}
	server := New(client, Options{MaxFileSize: 20})

	result := callReadFile(t, server, "")
	text := expectErrorContent(t, result)
	if text != "file_id is required" {
		t.Fatalf("error = %q, want %q", text, "file_id is required")
	}
}

func TestReadFileServiceUnavailable(t *testing.T) {
	client := &fakeFilesClient{metadataErr: status.Error(codes.Unavailable, "gateway down")}
	server := New(client, Options{MaxFileSize: 20})

	result := callReadFile(t, server, "file-6")
	text := expectErrorContent(t, result)
	if text != "platform service unavailable" {
		t.Fatalf("error = %q, want %q", text, "platform service unavailable")
	}
}

func callReadFile(t *testing.T, server *Server, fileID string) *mcp.CallToolResult {
	t.Helper()
	args, err := json.Marshal(map[string]any{"file_id": fileID})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Arguments: args},
	}
	result, err := server.handleReadFile(context.Background(), req)
	if err != nil {
		t.Fatalf("handleReadFile error: %v", err)
	}
	return result
}

func expectTextContent(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if result.IsError {
		t.Fatalf("expected success result")
	}
	if len(result.Content) != 1 {
		t.Fatalf("content length = %d, want 1", len(result.Content))
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content type = %T, want *mcp.TextContent", result.Content[0])
	}
	return text.Text
}

func expectImageContent(t *testing.T, result *mcp.CallToolResult) *mcp.ImageContent {
	t.Helper()
	if result.IsError {
		t.Fatalf("expected success result")
	}
	if len(result.Content) != 1 {
		t.Fatalf("content length = %d, want 1", len(result.Content))
	}
	image, ok := result.Content[0].(*mcp.ImageContent)
	if !ok {
		t.Fatalf("content type = %T, want *mcp.ImageContent", result.Content[0])
	}
	return image
}

func expectResourceContent(t *testing.T, result *mcp.CallToolResult) *mcp.ResourceContents {
	t.Helper()
	if result.IsError {
		t.Fatalf("expected success result")
	}
	if len(result.Content) != 1 {
		t.Fatalf("content length = %d, want 1", len(result.Content))
	}
	resource, ok := result.Content[0].(*mcp.EmbeddedResource)
	if !ok {
		t.Fatalf("content type = %T, want *mcp.EmbeddedResource", result.Content[0])
	}
	if resource.Resource == nil {
		t.Fatalf("resource contents missing")
	}
	return resource.Resource
}

func expectErrorContent(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if !result.IsError {
		t.Fatalf("expected error result")
	}
	if len(result.Content) != 1 {
		t.Fatalf("content length = %d, want 1", len(result.Content))
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content type = %T, want *mcp.TextContent", result.Content[0])
	}
	return text.Text
}
