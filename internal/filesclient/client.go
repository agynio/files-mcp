package filesclient

import (
	"context"

	filesv1 "github.com/agynio/files-mcp/.gen/go/agynio/api/files/v1"
	gatewayv1 "github.com/agynio/files-mcp/.gen/go/agynio/api/gateway/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	client gatewayv1.FilesGatewayClient
}

type FileContentStream interface {
	Recv() (*filesv1.GetFileContentResponse, error)
}

type BearerToken struct {
	Token string
}

func (b BearerToken) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Bearer " + b.Token}, nil
}

func (BearerToken) RequireTransportSecurity() bool {
	return false
}

func Dial(ctx context.Context, address string, token string) (*grpc.ClientConn, error) {
	options := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if token != "" {
		options = append(options, grpc.WithPerRPCCredentials(BearerToken{Token: token}))
	}
	return grpc.DialContext(ctx, address, options...)
}

func New(conn *grpc.ClientConn) *Client {
	return &Client{client: gatewayv1.NewFilesGatewayClient(conn)}
}

func (c *Client) GetFileMetadata(ctx context.Context, fileID string) (*filesv1.FileInfo, error) {
	resp, err := c.client.GetFileMetadata(ctx, &filesv1.GetFileMetadataRequest{FileId: fileID})
	if err != nil {
		return nil, err
	}
	return resp.GetFile(), nil
}

func (c *Client) GetFileContent(ctx context.Context, fileID string) (FileContentStream, error) {
	return c.client.GetFileContent(ctx, &filesv1.GetFileContentRequest{FileId: fileID})
}
