package internal

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	inputvalidatev1 "github.com/Muxcore-Media/input-validate-jsonschema/muxcore/inputvalidate/v1"
)

func TestModuleInfo(t *testing.T) {
	Version = "0.1.3"
	m := NewModule(Config{})
	info := m.Info()
	if info.ID == "" {
		t.Error("module ID must not be empty")
	}
	if info.Version != "0.1.3" {
		t.Errorf("version = %q, want 0.1.3", info.Version)
	}
	if info.MinCoreVersion == "" {
		t.Error("MinCoreVersion must not be empty")
	}
	if len(info.Capabilities) == 0 || info.Capabilities[0] != "input.validate" {
		t.Errorf("expected input.validate capability, got %v", info.Capabilities)
	}
	if info.HTTPAddr != "127.0.0.1:9665" {
		t.Errorf("HTTPAddr = %q, want 127.0.0.1:9665", info.HTTPAddr)
	}
}

func TestSupportedSchemasListsNamedFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "user-create.json"), []byte(`{"type":"object"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.txt"), []byte("skip"), 0o600); err != nil {
		t.Fatal(err)
	}

	m := NewModule(Config{DataDir: dir})
	resp, err := m.SupportedSchemas(context.Background(), &inputvalidatev1.SupportedSchemasRequest{})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range resp.Schemas {
		if s == "json:user-create" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected json:user-create in %v", resp.Schemas)
	}
}

func TestValidateJSONSchemaInline(t *testing.T) {
	m := NewModule(Config{})
	ctx := context.Background()

	schema := `{"type": "object", "properties": {"name": {"type": "string"}, "age": {"type": "number"}}, "required": ["name"]}`
	data := []byte(`{"name": "alice", "age": 30}`)

	resp, err := m.Validate(ctx, &inputvalidatev1.ValidateRequest{
		Schema: "json:" + schema,
		Data:   data,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Valid {
		t.Fatalf("expected valid, got errors: %v", resp.Errors)
	}
}

func TestValidateJSONSchemaInvalid(t *testing.T) {
	m := NewModule(Config{})
	ctx := context.Background()

	schema := `{"type": "object", "properties": {"email": {"type": "string", "format": "email"}}, "required": ["email"]}`
	data := []byte(`{"email": "not-an-email"}`)

	resp, err := m.Validate(ctx, &inputvalidatev1.ValidateRequest{
		Schema: "json:" + schema,
		Data:   data,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Valid {
		t.Fatal("expected invalid")
	}
	if len(resp.Errors) == 0 {
		t.Fatal("expected validation errors")
	}
}

func TestValidateJSONSchemaMissingField(t *testing.T) {
	m := NewModule(Config{})
	ctx := context.Background()

	schema := `{"type": "object", "required": ["required_field"]}`
	data := []byte(`{}`)

	resp, err := m.Validate(ctx, &inputvalidatev1.ValidateRequest{
		Schema: "json:" + schema,
		Data:   data,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Valid {
		t.Fatal("expected invalid for missing required field")
	}
	if len(resp.Errors) == 0 {
		t.Fatal("expected validation errors")
	}
}

func TestValidateJSONSchemaCompileFailure(t *testing.T) {
	m := NewModule(Config{})
	resp, err := m.Validate(context.Background(), &inputvalidatev1.ValidateRequest{
		Schema: "json:{not valid json",
		Data:   []byte(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Valid {
		t.Fatal("expected invalid")
	}
	if len(resp.Errors) == 0 {
		t.Fatal("expected compile error message")
	}
}

func TestValidateNamedSchemaMissingFile(t *testing.T) {
	m := NewModule(Config{DataDir: t.TempDir()})
	resp, err := m.Validate(context.Background(), &inputvalidatev1.ValidateRequest{
		Schema: "json:missing-schema",
		Data:   []byte(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Valid {
		t.Fatal("expected invalid")
	}
	if len(resp.Errors) == 0 {
		t.Fatal("expected missing file error")
	}
}

func TestValidateNamedSchemaInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "broken.json"), []byte(`{`), 0o600); err != nil {
		t.Fatal(err)
	}
	m := NewModule(Config{DataDir: dir})
	resp, err := m.Validate(context.Background(), &inputvalidatev1.ValidateRequest{
		Schema: "json:broken",
		Data:   []byte(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Valid {
		t.Fatal("expected invalid")
	}
	if len(resp.Errors) == 0 {
		t.Fatal("expected compile error")
	}
}

func TestValidateRegexMatch(t *testing.T) {
	m := NewModule(Config{})
	ctx := context.Background()

	resp, err := m.Validate(ctx, &inputvalidatev1.ValidateRequest{
		Schema: "regex:^[a-z]+$",
		Data:   []byte("hello"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Valid {
		t.Fatalf("expected valid, got errors: %v", resp.Errors)
	}
}

func TestValidateRegexNoMatch(t *testing.T) {
	m := NewModule(Config{})
	ctx := context.Background()

	resp, err := m.Validate(ctx, &inputvalidatev1.ValidateRequest{
		Schema: "regex:^[a-z]+$",
		Data:   []byte("Hello123"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Valid {
		t.Fatal("expected invalid")
	}
}

func TestValidateRegexCompileFailure(t *testing.T) {
	m := NewModule(Config{})
	resp, err := m.Validate(context.Background(), &inputvalidatev1.ValidateRequest{
		Schema: "regex:[",
		Data:   []byte("test"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Valid {
		t.Fatal("expected invalid")
	}
	if len(resp.Errors) == 0 {
		t.Fatal("expected compile error")
	}
}

func TestValidateRegexTimeout(t *testing.T) {
	m := NewModule(Config{})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(2 * time.Millisecond)

	resp, err := m.Validate(ctx, &inputvalidatev1.ValidateRequest{
		Schema: "regex:^[a-z]+$",
		Data:   []byte("hello"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Valid {
		t.Fatal("expected timeout failure")
	}
	if len(resp.Errors) == 0 || !strings.Contains(resp.Errors[0], "timed out") {
		t.Fatalf("expected timeout error, got %v", resp.Errors)
	}
}

func TestValidateOversizeData(t *testing.T) {
	m := NewModule(Config{})
	data := make([]byte, defaultMaxPayloadBytes+1)
	resp, err := m.Validate(context.Background(), &inputvalidatev1.ValidateRequest{
		Schema: `json:{"type":"object"}`,
		Data:   data,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Valid {
		t.Fatal("expected invalid")
	}
	if len(resp.Errors) == 0 {
		t.Fatal("expected size error")
	}
}

func TestValidateOversizeSchema(t *testing.T) {
	m := NewModule(Config{})
	schema := "json:" + strings.Repeat("a", defaultMaxPayloadBytes)
	resp, err := m.Validate(context.Background(), &inputvalidatev1.ValidateRequest{
		Schema: schema,
		Data:   []byte(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Valid {
		t.Fatal("expected invalid")
	}
}

func TestValidateSchemaRefRejectsRemoteHTTP(t *testing.T) {
	m := NewModule(Config{DataDir: t.TempDir()})
	schema := `{"$ref":"http://127.0.0.1/schema.json"}`
	resp, err := m.Validate(context.Background(), &inputvalidatev1.ValidateRequest{
		Schema: "json:" + schema,
		Data:   []byte(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Valid {
		t.Fatal("expected invalid")
	}
	if len(resp.Errors) == 0 {
		t.Fatal("expected ref rejection error")
	}
}

func TestValidateSchemaRefRejectsOutsideFile(t *testing.T) {
	m := NewModule(Config{DataDir: t.TempDir()})
	schema := `{"$ref":"file:///etc/passwd"}`
	resp, err := m.Validate(context.Background(), &inputvalidatev1.ValidateRequest{
		Schema: "json:" + schema,
		Data:   []byte(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Valid {
		t.Fatal("expected invalid")
	}
	if len(resp.Errors) == 0 {
		t.Fatal("expected ref rejection error")
	}
}

func TestValidateUnsupportedSchema(t *testing.T) {
	m := NewModule(Config{})
	ctx := context.Background()

	resp, err := m.Validate(ctx, &inputvalidatev1.ValidateRequest{
		Schema: "sql:3",
		Data:   []byte("test"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Valid {
		t.Fatal("expected invalid for unsupported schema type")
	}
}

func TestValidateNamedSchemaRejectsTraversal(t *testing.T) {
	m := NewModule(Config{DataDir: t.TempDir()})
	ctx := context.Background()
	data := []byte(`{}`)

	for _, name := range []string{
		"../etc/passwd",
		"..\\etc\\passwd",
		"foo/bar",
		`foo\bar`,
		".",
		"..",
		"Foo",
		"",
	} {
		schema := "json:" + name
		resp, err := m.Validate(ctx, &inputvalidatev1.ValidateRequest{
			Schema: schema,
			Data:   data,
		})
		if err != nil {
			t.Fatalf("%q: unexpected error: %v", name, err)
		}
		if resp.Valid {
			t.Fatalf("%q: expected invalid", name)
		}
		if len(resp.Errors) == 0 {
			t.Fatalf("%q: expected error message", name)
		}
	}
}

func TestValidateNamedSchemaFile(t *testing.T) {
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "user-create.json")
	if err := os.WriteFile(schemaPath, []byte(`{"type":"object","required":["name"],"properties":{"name":{"type":"string"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	m := NewModule(Config{DataDir: dir})
	ctx := context.Background()

	resp, err := m.Validate(ctx, &inputvalidatev1.ValidateRequest{
		Schema: "json:user-create",
		Data:   []byte(`{"name":"alice"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Valid {
		t.Fatalf("expected valid, got errors: %v", resp.Errors)
	}
}

func TestValidateNamedSchemaRejectsEscapingSymlink(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.json")
	if err := os.WriteFile(outside, []byte(`{"type":"object"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "escape.json")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}

	m := NewModule(Config{DataDir: dir})
	resp, err := m.Validate(context.Background(), &inputvalidatev1.ValidateRequest{
		Schema: "json:escape",
		Data:   []byte(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Valid {
		t.Fatal("expected invalid for escaping symlink")
	}
	if len(resp.Errors) == 0 {
		t.Fatal("expected error message")
	}
}

func TestValidateNotJSON(t *testing.T) {
	m := NewModule(Config{})
	ctx := context.Background()

	schema := `{"type": "object"}`
	resp, err := m.Validate(ctx, &inputvalidatev1.ValidateRequest{
		Schema: "json:" + schema,
		Data:   []byte("not json at all"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Valid {
		t.Fatal("expected invalid for non-JSON input")
	}
}

func TestLifecycle(t *testing.T) {
	m := NewModule(Config{GRPCAddr: ":0", DataDir: t.TempDir()})
	ctx := context.Background()
	if err := m.Init(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m.Health(ctx); err != nil {
		t.Fatal("expected health to pass")
	}
	if err := m.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m.Health(ctx); err == nil {
		t.Fatal("expected health to fail after stop")
	}
}

func TestInitCreatesDataDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "schemas")
	m := NewModule(Config{GRPCAddr: ":0", DataDir: dir})
	if err := m.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !fi.IsDir() {
		t.Fatal("expected directory")
	}
	_ = m.Stop(context.Background())
}

func TestStopClosesListenerWithoutStart(t *testing.T) {
	m := NewModule(Config{GRPCAddr: ":0", DataDir: t.TempDir()})
	if err := m.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := m.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := m.Health(context.Background()); err == nil {
		t.Fatal("expected health to fail when listener closed")
	}
}

func TestSettingsDataDirLive(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()
	if err := os.WriteFile(filepath.Join(dirB, "person.json"), []byte(`{"type":"object","required":["name"],"properties":{"name":{"type":"string"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	m := NewModule(Config{DataDir: dirA, GRPCAddr: "127.0.0.1:0"})
	defs := m.Settings()
	if len(defs) != 1 || defs[0].Key != "data_dir" || defs[0].Value != dirA {
		t.Fatalf("Settings=%+v", defs)
	}
	if err := m.UpdateSetting("data_dir", dirB); err != nil {
		t.Fatal(err)
	}
	path, err := m.resolveNamedSchemaPath("person")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(path) != dirB {
		t.Fatalf("resolved path %q not under %q", path, dirB)
	}
}

func TestUpdateSettingRejectsEmptyDataDir(t *testing.T) {
	m := NewModule(Config{DataDir: t.TempDir()})
	if err := m.UpdateSetting("data_dir", ""); err == nil {
		t.Fatal("expected error for empty data_dir")
	}
}

func TestUpdateSettingRejectsNonDirectory(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := NewModule(Config{DataDir: dir})
	if err := m.UpdateSetting("data_dir", file); err == nil {
		t.Fatal("expected error for non-directory")
	}
}

func TestUpdateSettingRejectsMissingDirectory(t *testing.T) {
	m := NewModule(Config{DataDir: t.TempDir()})
	if err := m.UpdateSetting("data_dir", filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("expected error for missing directory")
	}
}
