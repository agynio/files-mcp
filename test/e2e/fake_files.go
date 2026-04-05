//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"

	filesv1 "github.com/agynio/files-mcp/.gen/go/agynio/api/files/v1"
	"github.com/agynio/files-mcp/internal/filesclient"
	"github.com/agynio/files-mcp/internal/mcpserver"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeFiles struct {
	mu    sync.Mutex
	files map[string]fakeFile
}

type fakeFile struct {
	info    *filesv1.FileInfo
	content []byte
}

func newFakeFiles() *fakeFiles {
	return &fakeFiles{files: make(map[string]fakeFile)}
}

func (f *fakeFiles) Upload(filename string, contentType string, content []byte) (string, error) {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return "", errors.New("filename is required")
	}
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		return "", errors.New("content type is required")
	}

	id := uuid.NewString()
	record := fakeFile{
		info: &filesv1.FileInfo{
			Id:          id,
			Filename:    filename,
			ContentType: contentType,
			SizeBytes:   int64(len(content)),
		},
		content: append([]byte(nil), content...),
	}

	f.mu.Lock()
	f.files[id] = record
	f.mu.Unlock()

	return id, nil
}

func (f *fakeFiles) GetFileMetadata(ctx context.Context, fileID string) (*filesv1.FileInfo, error) {
	record, ok := f.get(fileID)
	if !ok {
		return nil, status.Error(codes.NotFound, "file not found")
	}
	metadata := *record.info
	return &metadata, nil
}

func (f *fakeFiles) GetFileContent(ctx context.Context, fileID string) (filesclient.FileContentStream, error) {
	record, ok := f.get(fileID)
	if !ok {
		return nil, status.Error(codes.NotFound, "file not found")
	}
	return &fakeFileStream{data: append([]byte(nil), record.content...)}, nil
}

func (f *fakeFiles) get(fileID string) (fakeFile, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	record, ok := f.files[fileID]
	return record, ok
}

type fakeFileStream struct {
	data []byte
	sent bool
}

func (f *fakeFileStream) Recv() (*filesv1.GetFileContentResponse, error) {
	if f.sent {
		return nil, io.EOF
	}
	f.sent = true
	return &filesv1.GetFileContentResponse{ChunkData: f.data}, nil
}

var _ mcpserver.FilesClient = (*fakeFiles)(nil)
