// Package etherscan implements the client side of Tenderly's
// Etherscan-compatible contract verification API — the same protocol Foundry
// and hardhat-verify speak. Verification is synchronous: a verifysourcecode
// call returns the outcome directly.
package etherscan

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// CodeFormatStandardJSON submits the full solc standard-JSON input, so the
// server recompiles with exactly the submitted compiler settings.
const CodeFormatStandardJSON = "solidity-standard-json-input"

// Client talks to a single verifier URL: either the dashboard API
// (.../etherscan/verify/network/{networkId}[/public]) or a Virtual TestNet
// ({rpc}/verify, where the RPC URL itself authenticates).
type Client struct {
	verifierURL string
	accessKey   string
	token       string
	userAgent   string
	httpClient  *http.Client
}

func NewClient(verifierURL, accessKey, token, userAgent string) *Client {
	return &Client{
		verifierURL: verifierURL,
		accessKey:   accessKey,
		token:       token,
		userAgent:   userAgent,
		httpClient:  &http.Client{Timeout: 5 * time.Minute},
	}
}

type VerifyRequest struct {
	// ContractAddress is the deployed address to verify.
	ContractAddress string
	// ContractName is fully qualified: "contracts/Foo.sol:Foo".
	ContractName string
	// CompilerVersion is an exact solc version, e.g. "0.8.20+commit.a1b79de6".
	CompilerVersion string
	// SourceCode is the marshaled solc standard-JSON input.
	SourceCode string
}

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  string `json:"result"`
}

func (r *Response) IsOK() bool {
	return r.Status == "1"
}

func (c *Client) VerifySourceCode(request *VerifyRequest) (*Response, error) {
	form := url.Values{}
	form.Set("module", "contract")
	form.Set("action", "verifysourcecode")
	form.Set("codeformat", CodeFormatStandardJSON)
	form.Set("contractaddress", request.ContractAddress)
	form.Set("contractname", request.ContractName)
	form.Set("compilerversion", request.CompilerVersion)
	form.Set("sourceCode", request.SourceCode)

	httpRequest, err := http.NewRequest(
		"POST",
		c.verifierURL+"?module=contract&action=verifysourcecode",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, err
	}

	httpRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if c.userAgent != "" {
		httpRequest.Header.Set("User-Agent", c.userAgent)
	}
	if c.accessKey != "" {
		httpRequest.Header.Set("X-Access-Key", c.accessKey)
	} else if c.token != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+c.token)
	}

	httpResponse, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = httpResponse.Body.Close()
	}()

	body, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, err
	}

	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		return nil, fmt.Errorf("verification request failed (%d): %s",
			httpResponse.StatusCode, extractErrorMessage(body, httpResponse.Status))
	}

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed decoding verification response: %s", err)
	}

	return &response, nil
}

// extractErrorMessage pulls the API error message out of an error body,
// falling back to the HTTP status.
func extractErrorMessage(body []byte, status string) string {
	var errorResponse struct {
		Error struct {
			Slug    string `json:"slug"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &errorResponse) == nil && errorResponse.Error.Message != "" {
		return errorResponse.Error.Message
	}
	return status
}
