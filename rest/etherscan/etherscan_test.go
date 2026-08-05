package etherscan

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tenderly/tenderly-cli/providers"
	"github.com/tenderly/tenderly-cli/rest/payloads"
)

func boolPtr(b bool) *bool { return &b }
func intPtr(i int) *int    { return &i }

func TestBuildStandardJSONInput(t *testing.T) {
	contracts := []providers.Contract{
		{Name: "Foo", Source: "contract Foo {}", SourcePath: "contracts/Foo.sol"},
		{Name: "Lib", Source: "library Lib {}", SourcePath: "contracts/Lib.sol"},
		{Name: "Empty"}, // no source — skipped
	}
	config := &payloads.Config{
		OptimizationsUsed:  boolPtr(true),
		OptimizationsCount: intPtr(200),
		ViaIR:              boolPtr(true),
		Remappings:         []string{"@oz/=node_modules/@oz/"},
	}

	sourceJSON, err := BuildStandardJSONInput(contracts, config)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	var input StandardJSONInput
	if err := json.Unmarshal([]byte(sourceJSON), &input); err != nil {
		t.Fatalf("output is not valid JSON: %s", err)
	}

	if input.Language != "Solidity" {
		t.Errorf("expected language Solidity, got %q", input.Language)
	}
	if len(input.Sources) != 2 {
		t.Errorf("expected 2 sources, got %d", len(input.Sources))
	}
	if input.Sources["contracts/Foo.sol"].Content != "contract Foo {}" {
		t.Error("Foo source content mismatch")
	}
	if input.Settings.Optimizer == nil || !*input.Settings.Optimizer.Enabled || *input.Settings.Optimizer.Runs != 200 {
		t.Error("optimizer settings not carried over")
	}
	if input.Settings.ViaIR == nil || !*input.Settings.ViaIR {
		t.Error("viaIR not carried over")
	}
	if len(input.Settings.Remappings) != 1 {
		t.Error("remappings not carried over")
	}
}

func TestBuildStandardJSONInputNoSources(t *testing.T) {
	if _, err := BuildStandardJSONInput([]providers.Contract{{Name: "X"}}, nil); err == nil {
		t.Fatal("expected error for empty sources")
	}
}

func TestVerifySourceCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed parsing form: %s", err)
		}
		if r.FormValue("module") != "contract" || r.FormValue("action") != "verifysourcecode" {
			t.Errorf("module/action not set: %q/%q", r.FormValue("module"), r.FormValue("action"))
		}
		if r.FormValue("codeformat") != CodeFormatStandardJSON {
			t.Errorf("unexpected codeformat %q", r.FormValue("codeformat"))
		}
		if r.FormValue("contractname") != "contracts/Foo.sol:Foo" {
			t.Errorf("unexpected contractname %q", r.FormValue("contractname"))
		}
		if r.Header.Get("X-Access-Key") != "test-key" {
			t.Errorf("access key header not set")
		}

		_ = json.NewEncoder(w).Encode(Response{Status: "1", Message: "OK", Result: "0xabc"})
	}))
	defer server.Close()

	verifierClient := NewClient(server.URL+"/verify", "test-key", "", "tenderly-cli/test")
	response, err := verifierClient.VerifySourceCode(&VerifyRequest{
		ContractAddress: "0xabc",
		ContractName:    "contracts/Foo.sol:Foo",
		CompilerVersion: "0.8.20",
		SourceCode:      "{}",
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !response.IsOK() {
		t.Errorf("expected OK response, got %+v", response)
	}
}

func TestVerifySourceCodeAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"slug":"quota_limit_reached","message":"Your plan does not allow contract verification."}}`))
	}))
	defer server.Close()

	verifierClient := NewClient(server.URL+"/verify", "", "token", "tenderly-cli/test")
	_, err := verifierClient.VerifySourceCode(&VerifyRequest{})
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
	if got := err.Error(); got != "verification request failed (403): Your plan does not allow contract verification." {
		t.Errorf("unexpected error message: %s", got)
	}
}
