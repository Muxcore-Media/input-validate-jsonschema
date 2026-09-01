package client

import (
	"context"

	"google.golang.org/grpc"

	"github.com/Muxcore-Media/core/pkg/contracts"
	inputvalidatev1 "github.com/Muxcore-Media/input-validate-jsonschema/muxcore/inputvalidate/v1"
)

// Client implements contracts.InputValidator over InputValidateService gRPC.
type Client struct {
	rpc inputvalidatev1.InputValidateServiceClient
}

// New returns an InputValidator backed by conn.
func New(conn grpc.ClientConnInterface) *Client {
	return &Client{rpc: inputvalidatev1.NewInputValidateServiceClient(conn)}
}

var _ contracts.InputValidator = (*Client)(nil)

// Validate checks data against schema and returns a contracts.ValidationResult.
func (c *Client) Validate(ctx context.Context, data []byte, schema string) (contracts.ValidationResult, error) {
	resp, err := c.rpc.Validate(ctx, &inputvalidatev1.ValidateRequest{
		Schema: schema,
		Data:   data,
	})
	if err != nil {
		return contracts.ValidationResult{}, err
	}
	return contracts.ValidationResult{
		Valid:     resp.GetValid(),
		Sanitized: resp.GetSanitized(),
		Errors:    append([]string(nil), resp.GetErrors()...),
	}, nil
}

// SupportedSchemas returns schema identifiers this validator can handle.
func (c *Client) SupportedSchemas() []string {
	resp, err := c.rpc.SupportedSchemas(context.Background(), &inputvalidatev1.SupportedSchemasRequest{})
	if err != nil {
		return nil
	}
	return append([]string(nil), resp.GetSchemas()...)
}
