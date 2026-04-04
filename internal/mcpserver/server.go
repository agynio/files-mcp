package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	filesv1 "github.com/agynio/files-mcp/.gen/go/agynio/api/files/v1"
	"github.com/agynio/files-mcp/internal/contenttype"
	"github.com/agynio/files-mcp/internal/filesclient"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	readFileToolName        = "read_file"
	readFileToolDescription = "Read a file from the platform. Use this tool to access file content when you see agyn://file/ references in the conversation."
)

var readFileInputSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"file_id": map[string]any{
			"type":        "string",
			"description": "The file ID from an agyn://file/ URI (the part after agyn://file/)",
		},
	},
	"required": []string{"file_id"},
}

type FilesClient interface {
	GetFileMetadata(ctx context.Context, fileID string) (*filesv1.FileInfo, error)
	GetFileContent(ctx context.Context, fileID string) (filesclient.FileContentStream, error)
}

type Options struct {
	MaxFileSize int64
}

type Server struct {
	filesClient FilesClient
	maxFileSize int64
	server      *mcp.Server
	handler     http.Handler
}

type readFileInput struct {
	FileID string `json:"file_id"`
}

func New(filesClient FilesClient, opts Options) *Server {
	maxSize := opts.MaxFileSize
	if maxSize <= 0 {
		panic("mcpserver.New: MaxFileSize must be positive")
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "files-mcp",
		Version: "0.1.0",
	}, &mcp.ServerOptions{Capabilities: &mcp.ServerCapabilities{Tools: &mcp.ToolCapabilities{}}})

	tool := &mcp.Tool{
		Name:        readFileToolName,
		Description: readFileToolDescription,
		InputSchema: readFileInputSchema,
	}

	mcpServer := &Server{
		filesClient: filesClient,
		maxFileSize: maxSize,
		server:      server,
	}

	server.AddTool(tool, mcpServer.handleReadFile)
	serverHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, nil)
	mcpServer.handler = serverHandler

	return mcpServer
}

func (s *Server) Handler() http.Handler {
	return s.handler
}

func (s *Server) handleReadFile(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := parseReadFileInput(req)
	if err != nil {
		return toolErrorResult(err.Error()), nil
	}

	metadata, err := s.filesClient.GetFileMetadata(ctx, input.FileID)
	if err != nil {
		return toolErrorResult(mapServiceError(err, input.FileID)), nil
	}

	sizeBytes := metadata.GetSizeBytes()
	if sizeBytes > s.maxFileSize {
		return toolErrorResult(fmt.Sprintf("file too large: %d bytes exceeds limit of %d bytes", sizeBytes, s.maxFileSize)), nil
	}

	stream, err := s.filesClient.GetFileContent(ctx, input.FileID)
	if err != nil {
		return toolErrorResult(mapServiceError(err, input.FileID)), nil
	}

	data, err := readAllContent(stream)
	if err != nil {
		return toolErrorResult(mapServiceError(err, input.FileID)), nil
	}

	content := buildContent(metadata.GetContentType(), input.FileID, data)
	return &mcp.CallToolResult{Content: []mcp.Content{content}}, nil
}

func parseReadFileInput(req *mcp.CallToolRequest) (readFileInput, error) {
	if len(req.Params.Arguments) == 0 {
		return readFileInput{}, errors.New("file_id is required")
	}
	var input readFileInput
	if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
		return readFileInput{}, errors.New("invalid tool arguments")
	}
	input.FileID = strings.TrimSpace(input.FileID)
	if input.FileID == "" {
		return readFileInput{}, errors.New("file_id is required")
	}
	return input, nil
}

func buildContent(contentType string, fileID string, data []byte) mcp.Content {
	contentKind := contenttype.Classify(contentType)
	switch contentKind {
	case contenttype.KindImage:
		return &mcp.ImageContent{Data: data, MIMEType: contentType}
	case contenttype.KindText:
		return &mcp.TextContent{Text: string(data)}
	case contenttype.KindResource:
		return &mcp.EmbeddedResource{
			Resource: &mcp.ResourceContents{
				URI:      fmt.Sprintf("agyn://file/%s", fileID),
				MIMEType: contentType,
				Blob:     data,
			},
		}
	default:
		panic(fmt.Sprintf("buildContent: unhandled content kind %d", contentKind))
	}
}

func readAllContent(stream filesclient.FileContentStream) ([]byte, error) {
	var buffer bytes.Buffer
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if _, err := buffer.Write(resp.GetChunkData()); err != nil {
			return nil, err
		}
	}
	return buffer.Bytes(), nil
}

func toolErrorResult(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
	}
}

func mapServiceError(err error, fileID string) string {
	if isNotFound(err) {
		return fmt.Sprintf("file not found: %s", fileID)
	}
	if isUnavailable(err) {
		return "platform service unavailable"
	}
	return fmt.Sprintf("failed to download file: %s", errorDetails(err))
}

func isNotFound(err error) bool {
	statusErr, ok := status.FromError(err)
	return ok && statusErr.Code() == codes.NotFound
}

func isUnavailable(err error) bool {
	statusErr, ok := status.FromError(err)
	if ok && (statusErr.Code() == codes.Unavailable || statusErr.Code() == codes.DeadlineExceeded) {
		return true
	}
	return errors.Is(err, context.DeadlineExceeded)
}

func errorDetails(err error) string {
	statusErr, ok := status.FromError(err)
	if ok && statusErr.Message() != "" {
		return statusErr.Message()
	}
	return err.Error()
}
