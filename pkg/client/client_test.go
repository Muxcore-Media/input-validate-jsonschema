package client_test

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/Muxcore-Media/core/pkg/contracts"
	"github.com/Muxcore-Media/input-validate-jsonschema/internal"
	inputvalidatev1 "github.com/Muxcore-Media/input-validate-jsonschema/muxcore/inputvalidate/v1"
	"github.com/Muxcore-Media/input-validate-jsonschema/pkg/client"
)

const bufSize = 1 << 20

func startTestModule(t *testing.T, cfg internal.Config) *bufconn.Listener {
	t.Helper()
	mod := internal.NewModule(cfg)
	lis := bufconn.Listen(bufSize)
	maxRecv := (4 << 20) + 1024
	gs := grpc.NewServer(grpc.MaxRecvMsgSize(maxRecv))
	inputvalidatev1.RegisterInputValidateServiceServer(gs, mod)
	go func() { _ = gs.Serve(lis) }()
	t.Cleanup(func() {
		gs.Stop()
		_ = lis.Close()
	})
	return lis
}

func dialClient(t *testing.T, lis *bufconn.Listener) *client.Client {
	t.Helper()
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return client.New(conn)
}

func TestClientValidateJSON(t *testing.T) {
	lis := startTestModule(t, internal.Config{})
	cl := dialClient(t, lis)
	ctx := context.Background()

	result, err := cl.Validate(ctx, []byte(`{"name":"alice"}`), `json:{"type":"object","required":["name"],"properties":{"name":{"type":"string"}}}`)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Valid {
		t.Fatalf("expected valid, errors=%v", result.Errors)
	}
}

func TestClientSupportedSchemas(t *testing.T) {
	dir := t.TempDir()
	lis := startTestModule(t, internal.Config{DataDir: dir})
	cl := dialClient(t, lis)

	schemas := cl.SupportedSchemas()
	if len(schemas) < 2 {
		t.Fatalf("expected type prefixes, got %v", schemas)
	}
}

var _ contracts.InputValidator = (*client.Client)(nil)
