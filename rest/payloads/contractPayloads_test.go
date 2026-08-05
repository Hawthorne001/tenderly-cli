package payloads

import (
	"testing"

	"github.com/tenderly/tenderly-cli/providers"
)

func boolPtr(b bool) *bool       { return &b }
func intPtr(i int) *int          { return &i }
func stringPtr(s string) *string { return &s }

func TestParseSolcConfigWithSettings(t *testing.T) {
	compilers := map[string]providers.Compiler{
		"solc": {
			Version: "0.8.20",
			Settings: &providers.CompilerSettings{
				EvmVersion: stringPtr("paris"),
				ViaIR:      boolPtr(true),
				Remappings: []string{"@openzeppelin/=node_modules/@openzeppelin/"},
				Libraries:  map[string]string{"MyLib": "0x1234"},
				Metadata: &providers.CompilerSettingsMetadata{
					UseLiteralContent: boolPtr(true),
					BytecodeHash:      stringPtr("none"),
					AppendCBOR:        boolPtr(false),
				},
				Optimizer: &providers.Optimizer{
					Enabled: boolPtr(true),
					Runs:    intPtr(999),
				},
			},
		},
	}

	config := ParseSolcConfigWithSettings(compilers)

	if config == nil {
		t.Fatal("expected config, got nil")
	}
	if config.CompilerVersion != "0.8.20" {
		t.Errorf("expected compiler version 0.8.20, got %q", config.CompilerVersion)
	}
	if config.ViaIR == nil || !*config.ViaIR {
		t.Error("expected viaIR true")
	}
	if config.EvmVersion == nil || *config.EvmVersion != "paris" {
		t.Error("expected evmVersion paris")
	}
	if len(config.Remappings) != 1 {
		t.Errorf("expected 1 remapping, got %d", len(config.Remappings))
	}
	if config.Libraries["MyLib"] != "0x1234" {
		t.Error("expected MyLib library address")
	}
	if config.Metadata == nil || config.Metadata.BytecodeHash == nil || *config.Metadata.BytecodeHash != "none" {
		t.Error("expected metadata bytecodeHash none")
	}
	if config.Metadata.AppendCBOR == nil || *config.Metadata.AppendCBOR {
		t.Error("expected metadata appendCBOR false")
	}
	if config.OptimizationsUsed == nil || !*config.OptimizationsUsed {
		t.Error("expected optimizations used true")
	}
	if config.OptimizationsCount == nil || *config.OptimizationsCount != 999 {
		t.Error("expected optimizations count 999")
	}
}

func TestParseSolcConfigExplicitOptimizerDefaults(t *testing.T) {
	compilers := map[string]providers.Compiler{
		"solc": {Version: "0.8.20"},
	}

	config := ParseNewTruffleConfig(compilers)

	if config == nil {
		t.Fatal("expected config, got nil")
	}
	if config.OptimizationsUsed == nil || *config.OptimizationsUsed {
		t.Error("expected explicit optimizations_used false when optimizer is absent")
	}
	if config.OptimizationsCount == nil || *config.OptimizationsCount != DefaultOptimizerRuns {
		t.Errorf("expected explicit optimizations_count %d when optimizer is absent", DefaultOptimizerRuns)
	}
}

func TestParseSolcConfigVersionRangeNotSent(t *testing.T) {
	for _, version := range []string{"^0.8.0", ">=0.7.0 <0.9.0", "pragma", "native", ""} {
		compilers := map[string]providers.Compiler{
			"solc": {Version: version},
		}
		config := ParseNewTruffleConfig(compilers)
		if config.CompilerVersion != "" {
			t.Errorf("version %q should not be sent as compiler_version, got %q", version, config.CompilerVersion)
		}
	}

	for _, version := range []string{"0.8.20", "v0.8.20", "0.8.20+commit.a1b79de6"} {
		compilers := map[string]providers.Compiler{
			"solc": {Version: version},
		}
		config := ParseNewTruffleConfig(compilers)
		if config.CompilerVersion != version {
			t.Errorf("exact version %q should be sent as compiler_version, got %q", version, config.CompilerVersion)
		}
	}
}

func TestParseSolcConfigTopLevelOptimizer(t *testing.T) {
	// Brownie/Buidler-style config: optimizer and evmVersion at the compiler
	// top level, no nested settings. Regression test for the nil-deref that
	// used to read compiler.Settings inside this branch.
	compilers := map[string]providers.Compiler{
		"solc": {
			Version:    "0.7.6",
			EvmVersion: stringPtr("istanbul"),
			Optimizer: &providers.Optimizer{
				Enabled: boolPtr(true),
				Runs:    intPtr(200),
				Details: &providers.OptimizerDetails{
					Yul: boolPtr(true),
					YulDetails: &providers.YulDetails{
						StackAllocation: boolPtr(true),
					},
				},
			},
		},
	}

	config := ParseSolcConfigWithOptimizer(compilers)

	if config == nil {
		t.Fatal("expected config, got nil")
	}
	if config.EvmVersion == nil || *config.EvmVersion != "istanbul" {
		t.Error("expected evmVersion istanbul")
	}
	if config.Details == nil || config.Details.Yul == nil || !*config.Details.Yul {
		t.Error("expected optimizer details yul true")
	}
	if config.Details.YulDetails == nil || config.Details.YulDetails.StackAllocation == nil {
		t.Error("expected yulDetails stackAllocation")
	}
}

func TestParseOldTruffleConfig(t *testing.T) {
	config := ParseOldTruffleConfig(map[string]providers.Optimizer{
		"optimizer": {Enabled: boolPtr(true), Runs: intPtr(500)},
	})

	if config == nil {
		t.Fatal("expected config, got nil")
	}
	if config.OptimizationsUsed == nil || !*config.OptimizationsUsed {
		t.Error("expected optimizations used true")
	}
	if config.OptimizationsCount == nil || *config.OptimizationsCount != 500 {
		t.Error("expected optimizations count 500")
	}

	if ParseOldTruffleConfig(map[string]providers.Optimizer{}) != nil {
		t.Error("expected nil config when optimizer key is absent")
	}
}
