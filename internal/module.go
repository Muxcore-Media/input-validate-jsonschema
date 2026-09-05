package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/Muxcore-Media/core/pkg/contracts"
	modulesdk "github.com/Muxcore-Media/core/sdk/go/module"
	"github.com/santhosh-tekuri/jsonschema/v5"

	"github.com/Muxcore-Media/input-validate-jsonschema/internal/grpctls"
	inputvalidatev1 "github.com/Muxcore-Media/input-validate-jsonschema/muxcore/inputvalidate/v1"
)

const defaultGRPCAddr = "127.0.0.1:9665"

var namedSchemaRe = regexp.MustCompile(`^[a-z][a-z0-9_.:-]{0,127}$`)

type Module struct {
	inputvalidatev1.UnimplementedInputValidateServiceServer

	mu       sync.RWMutex
	id       string
	grpcAddr string
	dataDir  string
	grpcSrv  *grpc.Server
	lis      net.Listener
}

type Config struct {
	ID       string
	GRPCAddr string
	DataDir  string
}

func NewModule(cfg Config) *Module {
	if cfg.ID == "" {
		cfg.ID = "input-validate-jsonschema"
	}
	if cfg.GRPCAddr == "" {
		cfg.GRPCAddr = defaultGRPCAddr
	}
	if cfg.DataDir == "" {
		cfg.DataDir = "./schemas"
	}
	if v := os.Getenv("VALIDATE_GRPC_ADDR"); v != "" {
		cfg.GRPCAddr = v
	}
	cfg.GRPCAddr = resolveGRPCAddr(cfg.GRPCAddr)
	if v := os.Getenv("VALIDATE_DATA_DIR"); v != "" {
		cfg.DataDir = v
	}
	return &Module{id: cfg.ID, grpcAddr: cfg.GRPCAddr, dataDir: cfg.DataDir}
}

func (m *Module) Info() contracts.ModuleInfo {
	return contracts.ModuleInfo{
		ID:           m.id,
		Name:         "Input Validate JSON Schema",
		Version:      Version,
		Roles:        []string{"infrastructure"},
		Description:  "JSON Schema and regex based input validation provider",
		Author:       "MuxCore",
		Capabilities: []string{contracts.CapabilityInputValidate, "settings"},
		Contracts: []contracts.ContractDeclaration{
			{
				Repo:      "github.com/Muxcore-Media/core/pkg/contracts",
				Interface: "InputValidator",
				Version:   "v0.5.0",
			},
		},
		MinCoreVersion: "0.5.0",
		HTTPAddr:       m.grpcAddr,
	}
}

func (m *Module) Init(ctx context.Context) error {
	if err := os.MkdirAll(m.dataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir %s: %w", m.dataDir, err)
	}

	lis, err := net.Listen("tcp", m.grpcAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", m.grpcAddr, err)
	}
	m.lis = lis
	slog.Info("input-validate-jsonschema initialized", "addr", m.grpcAddr, "data_dir", m.dataDir)
	return nil
}

func (m *Module) Start(ctx context.Context) error {
	maxRecv := defaultMaxPayloadBytes + 1024
	tlsCfg, err := grpctls.ServerConfig()
	if err != nil {
		return fmt.Errorf("gRPC TLS: %w", err)
	}
	var grpcOpts []grpc.ServerOption
	grpcOpts = append(grpcOpts, grpc.MaxRecvMsgSize(maxRecv))
	if tlsCfg != nil {
		grpcOpts = append(grpcOpts, grpc.Creds(credentials.NewTLS(tlsCfg)))
		slog.Info("input-validate-jsonschema gRPC TLS enabled", "addr", m.grpcAddr)
	} else {
		slog.Warn("input-validate-jsonschema gRPC listening without TLS (dev only)",
			"addr", m.grpcAddr,
			"hint", "unset MUXCORE_INSECURE_DISABLE_TLS for production",
		)
	}
	srv := grpc.NewServer(grpcOpts...)
	m.grpcSrv = srv
	inputvalidatev1.RegisterInputValidateServiceServer(srv, m)
	modulesdk.RegisterSettings(srv, m.id, m)
	lis := m.lis
	go func() {
		slog.Info("input-validate-jsonschema gRPC service started", "addr", m.grpcAddr)
		if err := srv.Serve(lis); err != nil {
			slog.Error("input-validate-jsonschema gRPC serve error", "error", err)
		}
	}()
	return nil
}

func resolveGRPCAddr(addr string) string {
	if !grpctls.InsecureAllowed() {
		return addr
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	if host == "" || host == "0.0.0.0" {
		return "127.0.0.1:" + port
	}
	return addr
}

func (m *Module) Stop(ctx context.Context) error {
	if m.grpcSrv != nil {
		m.grpcSrv.GracefulStop()
		m.grpcSrv = nil
	}
	if m.lis != nil {
		_ = m.lis.Close()
		m.lis = nil
	}
	slog.Info("input-validate-jsonschema stopped")
	return nil
}

func (m *Module) Health(ctx context.Context) error {
	if m.lis == nil {
		return fmt.Errorf("listener not initialized")
	}
	m.mu.RLock()
	dir := m.dataDir
	m.mu.RUnlock()
	fi, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("data dir %q: %w", dir, err)
	}
	if !fi.IsDir() {
		return fmt.Errorf("data dir %q is not a directory", dir)
	}
	return nil
}

// GRPCListenAddr returns the bound TCP address after Init.
func (m *Module) GRPCListenAddr() string {
	if m.lis != nil {
		return m.lis.Addr().String()
	}
	return ""
}

func (m *Module) Validate(ctx context.Context, req *inputvalidatev1.ValidateRequest) (*inputvalidatev1.ValidateResponse, error) {
	schema := req.GetSchema()
	data := req.GetData()

	if len(schema) > defaultMaxPayloadBytes {
		return &inputvalidatev1.ValidateResponse{
			Valid:  false,
			Errors: []string{fmt.Sprintf("schema exceeds max size (%d > %d bytes)", len(schema), defaultMaxPayloadBytes)},
		}, nil
	}
	if len(data) > defaultMaxPayloadBytes {
		return &inputvalidatev1.ValidateResponse{
			Valid:  false,
			Errors: []string{fmt.Sprintf("data exceeds max size (%d > %d bytes)", len(data), defaultMaxPayloadBytes)},
		}, nil
	}

	if strings.HasPrefix(schema, "json:") {
		return m.validateJSON(ctx, schema, data)
	}
	if strings.HasPrefix(schema, "regex:") {
		return m.validateRegex(ctx, schema, data)
	}
	return &inputvalidatev1.ValidateResponse{
		Valid:  false,
		Errors: []string{fmt.Sprintf("unsupported schema type: %q", schema)},
	}, nil
}

func (m *Module) validateJSON(ctx context.Context, schema string, data []byte) (*inputvalidatev1.ValidateResponse, error) {
	_ = ctx

	schemaName := strings.TrimPrefix(schema, "json:")
	schemaName = strings.TrimSpace(schemaName)
	if schemaName == "" {
		return &inputvalidatev1.ValidateResponse{
			Valid:  false,
			Errors: []string{"json schema name is empty"},
		}, nil
	}

	m.mu.RLock()
	dataDir := m.dataDir
	m.mu.RUnlock()

	var compiledSchema *jsonschema.Schema
	if strings.HasPrefix(schemaName, "{") {
		compiler := m.newSchemaCompiler(dataDir)
		if err := compiler.AddResource("schema.json", strings.NewReader(schemaName)); err != nil {
			return &inputvalidatev1.ValidateResponse{
				Valid:  false,
				Errors: []string{fmt.Sprintf("compile json schema: %v", err)},
			}, nil
		}
		s, err := compiler.Compile("schema.json")
		if err != nil {
			return &inputvalidatev1.ValidateResponse{
				Valid:  false,
				Errors: []string{fmt.Sprintf("compile json schema: %v", err)},
			}, nil
		}
		compiledSchema = s
	} else {
		schemaPath, err := m.resolveNamedSchemaPath(schemaName)
		if err != nil {
			return &inputvalidatev1.ValidateResponse{
				Valid:  false,
				Errors: []string{err.Error()},
			}, nil
		}
		schemaBytes, err := os.ReadFile(schemaPath)
		if err != nil {
			if os.IsNotExist(err) {
				return &inputvalidatev1.ValidateResponse{
					Valid:  false,
					Errors: []string{fmt.Sprintf("schema file not found: %s", schemaName)},
				}, nil
			}
			return &inputvalidatev1.ValidateResponse{
				Valid:  false,
				Errors: []string{fmt.Sprintf("read schema file %s: %v", schemaPath, err)},
			}, nil
		}
		compiler := m.newSchemaCompiler(dataDir)
		url := schemaName + ".json"
		if err := compiler.AddResource(url, bytes.NewReader(schemaBytes)); err != nil {
			return &inputvalidatev1.ValidateResponse{
				Valid:  false,
				Errors: []string{fmt.Sprintf("compile json schema %q: %v", schemaName, err)},
			}, nil
		}
		s, err := compiler.Compile(url)
		if err != nil {
			return &inputvalidatev1.ValidateResponse{
				Valid:  false,
				Errors: []string{fmt.Sprintf("compile json schema %q: %v", schemaName, err)},
			}, nil
		}
		compiledSchema = s
	}

	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return &inputvalidatev1.ValidateResponse{
			Valid:  false,
			Errors: []string{fmt.Sprintf("input is not valid JSON: %v", err)},
		}, nil
	}

	if err := compiledSchema.Validate(v); err != nil {
		var ve *jsonschema.ValidationError
		if ok := asValidationError(err, &ve); ok {
			errs := flattenValidationError(ve)
			if len(errs) == 0 {
				errs = []string{err.Error()}
			}
			return &inputvalidatev1.ValidateResponse{
				Valid:  false,
				Errors: errs,
			}, nil
		}
		return &inputvalidatev1.ValidateResponse{
			Valid:  false,
			Errors: []string{err.Error()},
		}, nil
	}

	return &inputvalidatev1.ValidateResponse{
		Valid:     true,
		Sanitized: data,
	}, nil
}

func (m *Module) validateRegex(ctx context.Context, schema string, data []byte) (*inputvalidatev1.ValidateResponse, error) {
	pattern := strings.TrimPrefix(schema, "regex:")
	re, err := regexp.Compile(pattern)
	if err != nil {
		return &inputvalidatev1.ValidateResponse{
			Valid:  false,
			Errors: []string{fmt.Sprintf("compile regex: %v", err)},
		}, nil
	}

	str := string(bytes.TrimSpace(data))

	matchCtx := ctx
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		matchCtx, cancel = context.WithTimeout(ctx, defaultRegexTimeout)
		defer cancel()
	}

	type result struct {
		matched bool
	}
	ch := make(chan result, 1)
	go func() {
		ch <- result{matched: re.MatchString(str)}
	}()

	select {
	case <-matchCtx.Done():
		return &inputvalidatev1.ValidateResponse{
			Valid:  false,
			Errors: []string{"regex validation timed out"},
		}, nil
	case r := <-ch:
		if r.matched {
			return &inputvalidatev1.ValidateResponse{
				Valid:     true,
				Sanitized: []byte(str),
			}, nil
		}
	}

	return &inputvalidatev1.ValidateResponse{
		Valid:  false,
		Errors: []string{fmt.Sprintf("input %q does not match pattern %q", str, pattern)},
	}, nil
}

func (m *Module) SupportedSchemas(ctx context.Context, req *inputvalidatev1.SupportedSchemasRequest) (*inputvalidatev1.SupportedSchemasResponse, error) {
	_ = ctx
	_ = req

	schemas := []string{"json:{…}", "regex:<pattern>"}

	m.mu.RLock()
	dataDir := m.dataDir
	m.mu.RUnlock()

	entries, err := os.ReadDir(dataDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			name := strings.TrimSuffix(entry.Name(), ".json")
			if namedSchemaRe.MatchString(name) {
				schemas = append(schemas, "json:"+name)
			}
		}
	}

	return &inputvalidatev1.SupportedSchemasResponse{Schemas: schemas}, nil
}

func (m *Module) newSchemaCompiler(dataDir string) *jsonschema.Compiler {
	base, err := m.resolveDataDirBase(dataDir)
	if err != nil {
		base = filepath.Clean(dataDir)
	}

	c := jsonschema.NewCompiler()
	c.LoadURL = func(rawURL string) (io.ReadCloser, error) {
		u, err := url.Parse(rawURL)
		if err != nil {
			return nil, fmt.Errorf("parse schema reference %q: %w", rawURL, err)
		}
		switch strings.ToLower(u.Scheme) {
		case "http", "https":
			return nil, fmt.Errorf("remote schema references are not allowed: %s", rawURL)
		case "file":
			path := u.Path
			if u.Host != "" && u.Host != "localhost" {
				return nil, fmt.Errorf("schema reference outside data dir: %s", rawURL)
			}
			clean := filepath.Clean(path)
			rel, err := filepath.Rel(base, clean)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
				return nil, fmt.Errorf("schema reference outside data dir: %s", rawURL)
			}
			f, err := os.Open(clean)
			if err != nil {
				return nil, fmt.Errorf("open schema reference %s: %w", rawURL, err)
			}
			return f, nil
		default:
			return nil, fmt.Errorf("unsupported schema reference: %s", rawURL)
		}
	}
	return c
}

func (m *Module) resolveDataDirBase(dataDir string) (string, error) {
	base, err := filepath.Abs(dataDir)
	if err != nil {
		return "", fmt.Errorf("resolve data dir: %w", err)
	}
	base = filepath.Clean(base)
	if resolvedBase, err := filepath.EvalSymlinks(base); err == nil {
		base = resolvedBase
	}
	return base, nil
}

func (m *Module) resolveNamedSchemaPath(name string) (string, error) {
	if strings.ContainsAny(name, `/\`) || name == "." || name == ".." || !namedSchemaRe.MatchString(name) {
		return "", fmt.Errorf("invalid schema name %q: must match %s and must not contain path separators", name, namedSchemaRe.String())
	}

	m.mu.RLock()
	dataDir := m.dataDir
	m.mu.RUnlock()
	base, err := m.resolveDataDirBase(dataDir)
	if err != nil {
		return "", err
	}

	candidate := filepath.Clean(filepath.Join(base, name+".json"))
	rel, err := filepath.Rel(base, candidate)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("schema path escapes data dir")
	}

	if fi, err := os.Lstat(candidate); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			resolved, err := filepath.EvalSymlinks(candidate)
			if err != nil {
				return "", fmt.Errorf("resolve schema symlink: %w", err)
			}
			resolved = filepath.Clean(resolved)
			rel, err := filepath.Rel(base, resolved)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
				return "", fmt.Errorf("schema path escapes data dir")
			}
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("stat schema path: %w", err)
	}

	return candidate, nil
}

func flattenValidationError(ve *jsonschema.ValidationError) []string {
	if ve == nil {
		return nil
	}
	var out []string
	var walk func(*jsonschema.ValidationError)
	walk = func(e *jsonschema.ValidationError) {
		if len(e.Causes) == 0 {
			loc := e.InstanceLocation
			if loc == "" {
				loc = e.AbsoluteKeywordLocation
			}
			if loc != "" {
				out = append(out, fmt.Sprintf("%s: %s", loc, e.Message))
			} else if e.Message != "" {
				out = append(out, e.Message)
			}
			return
		}
		for _, c := range e.Causes {
			walk(c)
		}
	}
	walk(ve)
	if len(out) == 0 && ve.Message != "" {
		out = append(out, ve.Message)
	}
	return out
}

func asValidationError(err error, target **jsonschema.ValidationError) bool {
	for err != nil {
		if ve, ok := err.(*jsonschema.ValidationError); ok {
			*target = ve
			return true
		}
		err = errorsUnwrap(err)
	}
	return false
}

func errorsUnwrap(err error) error {
	type unwrapper interface {
		Unwrap() error
	}
	u, ok := err.(unwrapper)
	if !ok {
		return nil
	}
	return u.Unwrap()
}

var _ contracts.Module = (*Module)(nil)
