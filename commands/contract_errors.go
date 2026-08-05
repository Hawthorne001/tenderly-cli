package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tenderly/tenderly-cli/rest/payloads"
)

const (
	BytecodeMismatchSlug       = "bytecode_mismatch"
	InvalidCompilerVersionSlug = "invalid_compiler_semantic_version"
	QuotaLimitReachedSlug      = "quota_limit_reached"
	ForbiddenSlug              = "forbidden"
)

// ContractErrorMessage enriches known verification API error slugs with
// actionable guidance. Unknown slugs fall back to the raw API message.
func ContractErrorMessage(apiErr *payloads.ApiError) string {
	switch apiErr.Slug {
	case BytecodeMismatchSlug:
		message := "The compiled bytecode doesn't match the bytecode deployed on-chain."
		if details := bytecodeMismatchDetails(apiErr.Data); details != "" {
			message = fmt.Sprintf("%s %s.", message, details)
		}
		return fmt.Sprintf("%s\n"+
			"This usually means the compiler settings differ from the ones used at deployment. "+
			"Check the compiler version, optimizer settings (enabled/runs), viaIR and metadata settings in your project configuration.",
			message,
		)
	case InvalidCompilerVersionSlug:
		return "The compiler version couldn't be determined or is invalid. " +
			"Make sure your build artifacts contain the compiler version (try recompiling your project) " +
			"or set an exact compiler version (e.g. \"0.8.20\") in your project configuration."
	case QuotaLimitReachedSlug:
		return fmt.Sprintf("%s\n"+
			"Your current plan doesn't allow this operation. Upgrade your plan or contact support at support@tenderly.co.",
			apiErr.Message,
		)
	case ForbiddenSlug:
		return fmt.Sprintf("%s\n"+
			"One of the networks in this request isn't included in your plan. "+
			"Check which networks your plan supports in the Tenderly dashboard, or retry with --networks limited to supported ones.",
			apiErr.Message,
		)
	default:
		return apiErr.Message
	}
}

// bytecodeMismatchDetails extracts the mismatched contracts from the error
// data, which holds one or more objects with NetworkID and Address fields.
func bytecodeMismatchDetails(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var mismatches []map[string]interface{}
	if err := json.Unmarshal(raw, &mismatches); err != nil {
		var single map[string]interface{}
		if err := json.Unmarshal(raw, &single); err != nil {
			return ""
		}
		mismatches = []map[string]interface{}{single}
	}

	var parts []string
	for _, mismatch := range mismatches {
		address, network := mismatch["Address"], mismatch["NetworkID"]
		if address == nil && network == nil {
			continue
		}
		parts = append(parts, fmt.Sprintf("%v on network %v", address, network))
	}
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf("Mismatched: %s", strings.Join(parts, ", "))
}
