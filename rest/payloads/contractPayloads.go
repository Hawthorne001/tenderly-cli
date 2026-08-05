package payloads

import (
	"regexp"

	"github.com/tenderly/tenderly-cli/providers"
)

type UploadContractsRequest struct {
	Contracts []providers.Contract `json:"contracts"`
	Config    *Config              `json:"config,omitempty"`
	Tag       string               `json:"tag,omitempty"`
}

type UploadContractsResponse struct {
	Contracts []providers.ApiContract `json:"contracts"`
	Error     *ApiError               `json:"error"`
}

type GetContractsResponse struct {
	Contracts []providers.ApiContract `json:"contracts"`
	Error     *ApiError               `json:"error"`
}

type RemoveContractsRequest struct {
	// ContractIDs are in the "eth:{networkID}:{address}" format.
	ContractIDs []string `json:"contract_ids"`
}

type RemoveContractsResponse struct {
	Error *ApiError `json:"error"`
}

type RenameContractRequest struct {
	DisplayName string `json:"display_name"`
}

type RenameContractResponse struct {
	Error *ApiError `json:"error"`
}

type Config struct {
	CompilerVersion    string            `json:"compiler_version,omitempty"`
	OptimizationsUsed  *bool             `json:"optimizations_used,omitempty"`
	OptimizationsCount *int              `json:"optimizations_count,omitempty"`
	EvmVersion         *string           `json:"evm_version,omitempty"`
	ViaIR              *bool             `json:"via_ir,omitempty"`
	Remappings         []string          `json:"remappings,omitempty"`
	Libraries          map[string]string `json:"libraries,omitempty"`
	Metadata           *ConfigMetadata   `json:"metadata,omitempty"`
	Details            *ConfigDetails    `json:"details,omitempty"`
}

type ConfigMetadata struct {
	UseLiteralContent *bool   `json:"useLiteralContent,omitempty"`
	BytecodeHash      *string `json:"bytecodeHash,omitempty"`
	AppendCBOR        *bool   `json:"appendCBOR,omitempty"`
}

type ConfigDetails struct {
	Peephole          *bool       `json:"peephole,omitempty"`
	JumpdestRemover   *bool       `json:"jumpdestRemover,omitempty"`
	OrderLiterals     *bool       `json:"orderLiterals,omitempty"`
	Deduplicate       *bool       `json:"deduplicate,omitempty"`
	Cse               *bool       `json:"cse,omitempty"`
	ConstantOptimizer *bool       `json:"constantOptimizer,omitempty"`
	Yul               *bool       `json:"yul,omitempty"`
	Inliner           *bool       `json:"inliner,omitempty"`
	YulDetails        *YulDetails `json:"yulDetails,omitempty"`
}

type YulDetails struct {
	StackAllocation *bool   `json:"stackAllocation,omitempty"`
	OptimizerSteps  *string `json:"optimizerSteps,omitempty"`
}

// DefaultOptimizerRuns is solc's default optimizer runs value. The API forces
// a missing optimizations_count to 200 anyway, so we always send it explicitly.
const DefaultOptimizerRuns = 200

// exactVersionRegexp matches exact solc versions ("0.8.20", "v0.8.20+commit.a1b79de6").
// Version ranges ("^0.8.0", ">=0.7.0 <0.9.0", "pragma") must not be sent as
// compiler_version — the API would use them instead of the exact version from
// the build artifact.
var exactVersionRegexp = regexp.MustCompile(`^v?\d+\.\d+\.\d+(\+.*)?$`)

func ParseNewTruffleConfig(compilers map[string]providers.Compiler) *Config {
	return parseSolcConfig(compilers)
}

func ParseOldTruffleConfig(solc map[string]providers.Optimizer) *Config {
	optimizer, exists := solc["optimizer"]
	if !exists {
		return nil
	}

	payload := &Config{}
	applyOptimizer(payload, &optimizer)
	return payload
}

func ParseSolcConfigWithOptimizer(compilers map[string]providers.Compiler) *Config {
	return parseSolcConfig(compilers)
}

func ParseSolcConfigWithSettings(compilers map[string]providers.Compiler) *Config {
	return parseSolcConfig(compilers)
}

// parseSolcConfig flattens a provider compiler entry into the API config
// payload, preferring the nested standard-JSON settings over the legacy
// top-level fields.
func parseSolcConfig(compilers map[string]providers.Compiler) *Config {
	compiler, exists := compilers["solc"]
	if !exists {
		return nil
	}

	payload := &Config{
		EvmVersion: compiler.EvmVersion,
		Remappings: compiler.Remappings,
	}

	if exactVersionRegexp.MatchString(compiler.Version) {
		payload.CompilerVersion = compiler.Version
	}

	optimizer := compiler.Optimizer
	if compiler.Settings != nil {
		settings := compiler.Settings
		if settings.EvmVersion != nil {
			payload.EvmVersion = settings.EvmVersion
		}
		if len(settings.Remappings) > 0 {
			payload.Remappings = settings.Remappings
		}
		payload.ViaIR = settings.ViaIR
		payload.Libraries = settings.Libraries
		if settings.Metadata != nil {
			payload.Metadata = &ConfigMetadata{
				UseLiteralContent: settings.Metadata.UseLiteralContent,
				BytecodeHash:      settings.Metadata.BytecodeHash,
				AppendCBOR:        settings.Metadata.AppendCBOR,
			}
		}
		if settings.Optimizer != nil {
			optimizer = settings.Optimizer
		}
	}

	applyOptimizer(payload, optimizer)
	return payload
}

// applyOptimizer sets explicit optimizer values so the API never applies its
// own defaults: a missing optimizations_used means "disabled" server-side and
// a missing optimizations_count is forced to 200.
func applyOptimizer(payload *Config, optimizer *providers.Optimizer) {
	enabled := false
	runs := DefaultOptimizerRuns

	if optimizer != nil {
		if optimizer.Enabled != nil {
			enabled = *optimizer.Enabled
		}
		if optimizer.Runs != nil {
			runs = *optimizer.Runs
		}
	}

	payload.OptimizationsUsed = &enabled
	payload.OptimizationsCount = &runs

	if optimizer == nil || optimizer.Details == nil {
		return
	}

	details := optimizer.Details
	payload.Details = &ConfigDetails{
		Peephole:          details.Peephole,
		JumpdestRemover:   details.JumpdestRemover,
		OrderLiterals:     details.OrderLiterals,
		Deduplicate:       details.Deduplicate,
		Cse:               details.Cse,
		ConstantOptimizer: details.ConstantOptimizer,
		Yul:               details.Yul,
		Inliner:           details.Inliner,
	}
	if details.YulDetails != nil {
		payload.Details.YulDetails = &YulDetails{
			StackAllocation: details.YulDetails.StackAllocation,
			OptimizerSteps:  details.YulDetails.OptimizerSteps,
		}
	}
}
