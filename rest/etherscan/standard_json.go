package etherscan

import (
	"encoding/json"
	"fmt"

	"github.com/tenderly/tenderly-cli/providers"
	"github.com/tenderly/tenderly-cli/rest/payloads"
)

// StandardJSONInput is the solc standard-JSON compilation input. The settings
// are passed through to solc verbatim server-side, so a verification compiles
// with exactly the submitted configuration.
type StandardJSONInput struct {
	Language string                   `json:"language"`
	Sources  map[string]SourceContent `json:"sources"`
	Settings Settings                 `json:"settings"`
}

type SourceContent struct {
	Content string `json:"content"`
}

type Settings struct {
	Remappings []string                 `json:"remappings,omitempty"`
	Optimizer  *Optimizer               `json:"optimizer,omitempty"`
	EvmVersion *string                  `json:"evmVersion,omitempty"`
	ViaIR      *bool                    `json:"viaIR,omitempty"`
	Metadata   *payloads.ConfigMetadata `json:"metadata,omitempty"`
}

type Optimizer struct {
	Enabled *bool                   `json:"enabled,omitempty"`
	Runs    *int                    `json:"runs,omitempty"`
	Details *payloads.ConfigDetails `json:"details,omitempty"`
}

// BuildStandardJSONInput assembles the compilation input from the provider
// build artifacts (all sources, including undeployed library/interface
// contracts) and the parsed compiler configuration.
func BuildStandardJSONInput(contracts []providers.Contract, config *payloads.Config) (string, error) {
	sources := make(map[string]SourceContent)
	for _, contract := range contracts {
		if contract.SourcePath == "" || contract.Source == "" {
			continue
		}
		sources[contract.SourcePath] = SourceContent{Content: contract.Source}
	}
	if len(sources) == 0 {
		return "", fmt.Errorf("no contract sources found in build artifacts")
	}

	input := StandardJSONInput{
		Language: "Solidity",
		Sources:  sources,
	}

	if config != nil {
		input.Settings = Settings{
			Remappings: config.Remappings,
			EvmVersion: config.EvmVersion,
			ViaIR:      config.ViaIR,
			Metadata:   config.Metadata,
		}
		if config.OptimizationsUsed != nil || config.OptimizationsCount != nil || config.Details != nil {
			input.Settings.Optimizer = &Optimizer{
				Enabled: config.OptimizationsUsed,
				Runs:    config.OptimizationsCount,
				Details: config.Details,
			}
		}
	}

	marshaled, err := json.Marshal(input)
	if err != nil {
		return "", err
	}

	return string(marshaled), nil
}
