package actions_test

import (
	"testing"

	"github.com/tenderly/tenderly-cli/model/actions"
)

func TestTransactionSimple(t *testing.T) {
	_ = MustReadTriggerAndValidate("trigger_transaction_simple")
}

func TestTransactionSimpleWithBody(t *testing.T) {
	trigger := MustReadTriggerAndValidate("trigger_transaction_simple_body")
	if trigger.TransactionSimple == nil {
		t.Fatal("transactionsimple body not parsed")
	}
	if trigger.ToRequest() == nil {
		t.Fatal("expected trigger request")
	}
}

func TestTransactionSimpleRequestType(t *testing.T) {
	trigger := actions.Trigger{Type: actions.TransactionSimpleType}
	if trigger.ToRequestType().Value() != "TRANSACTIONSIMPLE" {
		t.Fatal("unexpected trigger request type")
	}
}
