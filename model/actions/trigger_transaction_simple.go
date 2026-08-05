package actions

import (
	"github.com/tenderly/tenderly-cli/rest/payloads/generated/actions"
)

// TransactionSimpleTrigger has no configuration - the trigger fires for
// transactions without any filtering.
type TransactionSimpleTrigger struct {
}

func (t *TransactionSimpleTrigger) Validate(ctx ValidatorContext) (response ValidateResponse) {
	return response
}

func (t *TransactionSimpleTrigger) ToRequest() actions.Trigger {
	return actions.NewTriggerFromTransactionsimple(actions.TransactionSimpleTrigger{})
}
