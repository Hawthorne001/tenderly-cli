package payloads

import (
	"encoding/json"
	"fmt"
)

type ApiError struct {
	Message string          `json:"message"`
	Slug    string          `json:"slug,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (a *ApiError) Error() string {
	return fmt.Sprintf("Got error of type: [%s], with message [%s]", a.Slug, a.Message)
}
