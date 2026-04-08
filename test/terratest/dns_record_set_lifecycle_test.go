package terratest

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
	"github.com/alikor/terraform-provider-godaddy/test/terratest/helpers"
)

func TestDNSRecordSetLifecycleWithMockAPI(t *testing.T) {
	helpers.RequireTerraformCLI(t)

	mock := newMockDNSAPI(t)
	server := httptest.NewServer(mock)
	defer server.Close()

	_, cliConfig := helpers.BuildLocalProvider(t)
	fixtureDir := filepath.Join(helpers.RepoRoot(t), "test", "terratest", "fixtures", "dns_record_set_mock")
	env := append(os.Environ(),
		"TF_CLI_CONFIG_FILE="+cliConfig,
		"TF_VAR_godaddy_api_key=mock-key",
		"TF_VAR_godaddy_api_secret=mock-secret",
		"TF_VAR_godaddy_base_url="+server.URL,
		"TF_VAR_domain=example.com",
		"TF_VAR_record_name=terratest-mock",
		"TF_VAR_record_value=codex-mock",
	)

	runTerraformCommand(t, fixtureDir, env, 0, "init", "-input=false", "-no-color")
	runTerraformCommand(t, fixtureDir, env, 0, "apply", "-input=false", "-auto-approve", "-no-color")
	runTerraformCommand(t, fixtureDir, env, 0, "plan", "-input=false", "-lock=false", "-detailed-exitcode", "-no-color")
	runTerraformCommand(t, fixtureDir, env, 0, "state", "rm", "godaddy_dns_record_set.test")
	runTerraformCommand(t, fixtureDir, env, 0, "import", "-input=false", "godaddy_dns_record_set.test", "example.com,TXT,terratest-mock")
	runTerraformCommand(t, fixtureDir, env, 0, "plan", "-input=false", "-lock=false", "-detailed-exitcode", "-no-color")
	runTerraformCommand(t, fixtureDir, env, 0, "destroy", "-input=false", "-auto-approve", "-no-color")

	if got := mock.putCount(); got != 1 {
		t.Fatalf("PUT count = %d, want 1", got)
	}
	if got := mock.deleteCount(); got != 1 {
		t.Fatalf("DELETE count = %d, want 1", got)
	}
	if got := mock.getCount(); got < 4 {
		t.Fatalf("GET count = %d, want at least 4", got)
	}
	if mock.exists() {
		t.Fatal("expected mock RRset to be deleted after destroy")
	}
}

func runTerraformCommand(t *testing.T, dir string, env []string, wantExit int, args ...string) string {
	t.Helper()

	cmd := exec.Command("terraform", args...)
	cmd.Dir = dir
	cmd.Env = env
	output, err := cmd.CombinedOutput()

	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !strings.Contains(err.Error(), "exit status") || !errors.As(err, &exitErr) {
			t.Fatalf("terraform %v failed: %v\n%s", args, err, string(output))
		}
		exitCode = exitErr.ExitCode()
	}

	if exitCode != wantExit {
		t.Fatalf("terraform %v exit code = %d, want %d\n%s", args, exitCode, wantExit, string(output))
	}

	return string(output)
}

type mockDNSAPI struct {
	t *testing.T

	mu          sync.Mutex
	existing    bool
	records     []client.DNSRecord
	getRequests int
	putRequests int
	delRequests int
}

func newMockDNSAPI(t *testing.T) *mockDNSAPI {
	t.Helper()
	return &mockDNSAPI{t: t}
}

func (m *mockDNSAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") == "" {
		m.t.Fatalf("missing Authorization header")
	}

	expectedPath := "/v1/domains/example.com/records/TXT/terratest-mock"
	if r.URL.Path != expectedPath {
		m.t.Fatalf("unexpected request path %s", r.URL.Path)
	}

	switch r.Method {
	case http.MethodGet:
		m.handleGet(w)
	case http.MethodPut:
		m.handlePut(w, r)
	case http.MethodDelete:
		m.handleDelete(w)
	default:
		m.t.Fatalf("unexpected method %s", r.Method)
	}
}

func (m *mockDNSAPI) handleGet(w http.ResponseWriter) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.getRequests++
	if !m.existing {
		writeMockAPIError(w, http.StatusNotFound, "NOT_FOUND", "record set not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(m.records)
}

func (m *mockDNSAPI) handlePut(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.putRequests++

	var payload []client.DNSRecord
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		m.t.Fatalf("unable to decode PUT payload: %v", err)
	}
	if len(payload) != 1 || payload[0].Data != "codex-mock" || payload[0].TTL != 600 {
		m.t.Fatalf("unexpected PUT payload: %#v", payload)
	}

	m.records = payload
	m.existing = true
	w.WriteHeader(http.StatusOK)
}

func (m *mockDNSAPI) handleDelete(w http.ResponseWriter) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.delRequests++
	m.records = nil
	m.existing = false
	w.WriteHeader(http.StatusNoContent)
}

func (m *mockDNSAPI) getCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.getRequests
}

func (m *mockDNSAPI) putCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.putRequests
}

func (m *mockDNSAPI) deleteCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.delRequests
}

func (m *mockDNSAPI) exists() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.existing
}

func writeMockAPIError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code":    code,
		"message": message,
	})
}
