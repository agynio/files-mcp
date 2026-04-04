//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var png1x1 = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44,
	0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x06, 0x00, 0x00, 0x00, 0x1f,
	0x15, 0xc4, 0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x60,
	0x00, 0x00, 0x00, 0x02, 0x00, 0x01, 0xe5, 0x27, 0xd4, 0xa2, 0x00, 0x00, 0x00, 0x00, 0x49,
	0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
}

func TestReadFileTextRoundTrip(t *testing.T) {
	session := connectMCP(t)
	content := "hello files-mcp"
	fileID := uploadTestFile(t, "hello.txt", "text/plain", []byte(content))

	result := callReadFile(t, session, fileID)
	if result.IsError {
		t.Fatalf("expected success result")
	}
	text := requireTextContent(t, result)
	if text.Text != content {
		t.Fatalf("text content = %q, want %q", text.Text, content)
	}
}

func TestReadFileJSONRoundTrip(t *testing.T) {
	session := connectMCP(t)
	content := `{"ok":true}`
	fileID := uploadTestFile(t, "payload.json", "application/json", []byte(content))

	result := callReadFile(t, session, fileID)
	if result.IsError {
		t.Fatalf("expected success result")
	}
	text := requireTextContent(t, result)
	if text.Text != content {
		t.Fatalf("json content = %q, want %q", text.Text, content)
	}
}

func TestReadFileImageRoundTrip(t *testing.T) {
	session := connectMCP(t)
	fileID := uploadTestFile(t, "pixel.png", "image/png", png1x1)

	result := callReadFile(t, session, fileID)
	if result.IsError {
		t.Fatalf("expected success result")
	}
	image := requireImageContent(t, result)
	if image.MIMEType != "image/png" {
		t.Fatalf("mime type = %q, want %q", image.MIMEType, "image/png")
	}
	if !bytes.Equal(image.Data, png1x1) {
		t.Fatalf("image data mismatch")
	}
}

func TestReadFileBinaryRoundTrip(t *testing.T) {
	session := connectMCP(t)
	content := []byte("%PDF-1.4\n%EOF\n")
	fileID := uploadTestFile(t, "document.pdf", "application/pdf", content)

	result := callReadFile(t, session, fileID)
	if result.IsError {
		t.Fatalf("expected success result")
	}
	resource := requireResourceContent(t, result)
	if resource.URI != fmt.Sprintf("agyn://file/%s", fileID) {
		t.Fatalf("resource uri = %q, want %q", resource.URI, fmt.Sprintf("agyn://file/%s", fileID))
	}
	if resource.MIMEType != "application/pdf" {
		t.Fatalf("resource mime = %q, want %q", resource.MIMEType, "application/pdf")
	}
	if !bytes.Equal(resource.Blob, content) {
		t.Fatalf("resource blob mismatch")
	}
}

func TestReadFileNotFound(t *testing.T) {
	session := connectMCP(t)
	missingID := uuid.NewString()

	result := callReadFile(t, session, missingID)
	if !result.IsError {
		t.Fatalf("expected error result")
	}
	text := requireTextContent(t, result)
	expected := fmt.Sprintf("file not found: %s", missingID)
	if !strings.Contains(text.Text, expected) {
		t.Fatalf("error text = %q, want %q", text.Text, expected)
	}
}

func TestReadFileEmptyID(t *testing.T) {
	session := connectMCP(t)

	result := callReadFile(t, session, "")
	if !result.IsError {
		t.Fatalf("expected error result")
	}
	text := requireTextContent(t, result)
	if text.Text != "file_id is required" {
		t.Fatalf("error text = %q, want %q", text.Text, "file_id is required")
	}
}

func TestReadFileListTools(t *testing.T) {
	session := connectMCP(t)

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	t.Cleanup(cancel)
	result, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(result.Tools) != 1 {
		t.Fatalf("tools length = %d, want 1", len(result.Tools))
	}
	if result.Tools[0].Name != "read_file" {
		t.Fatalf("tool name = %q, want %q", result.Tools[0].Name, "read_file")
	}
}

func requireTextContent(t *testing.T, result *mcp.CallToolResult) *mcp.TextContent {
	t.Helper()
	if len(result.Content) != 1 {
		t.Fatalf("content length = %d, want 1", len(result.Content))
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content type = %T, want *mcp.TextContent", result.Content[0])
	}
	return text
}

func requireImageContent(t *testing.T, result *mcp.CallToolResult) *mcp.ImageContent {
	t.Helper()
	if len(result.Content) != 1 {
		t.Fatalf("content length = %d, want 1", len(result.Content))
	}
	image, ok := result.Content[0].(*mcp.ImageContent)
	if !ok {
		t.Fatalf("content type = %T, want *mcp.ImageContent", result.Content[0])
	}
	return image
}

func requireResourceContent(t *testing.T, result *mcp.CallToolResult) *mcp.ResourceContents {
	t.Helper()
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
