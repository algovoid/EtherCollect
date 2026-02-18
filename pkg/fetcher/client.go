package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const EtherscanBase = "https://api.etherscan.io/v2/api"

type EtherscanClient struct {
	Keys       []string
	keyIndex   int
	httpClient *http.Client
	// rate limiter provided by caller
}

type balanceResult struct {
	Account string `json:"account"`
	Balance string `json:"balance"`
}

type apiResponse struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Result  json.RawMessage `json:"result"`
}

func NewEtherscanClient(keys []string) *EtherscanClient {
	return &EtherscanClient{
		Keys:       keys,
		keyIndex:   0,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// rotateKey returns the next API key (round-robin).
func (c *EtherscanClient) rotateKey() string {
	if len(c.Keys) == 0 {
		return ""
	}
	k := c.Keys[c.keyIndex%len(c.Keys)]
	c.keyIndex = (c.keyIndex + 1) % len(c.Keys)
	return k
}

// BuildBalancemultiURL constructs the URL for balancemulti call.
func BuildBalancemultiURL(addrs []string, apiKey string) string {
	q := url.Values{}
	q.Set("module", "account")
	q.Set("action", "balancemulti")
	q.Set("address", strings.Join(addrs, ","))
	q.Set("tag", "latest")
	q.Set("apikey", apiKey)
	return EtherscanBase + "?" + q.Encode()
}

// FetchBalances calls balancemulti and decodes results.
func (c *EtherscanClient) FetchBalances(ctx context.Context, addrs []string) (map[string]string, error) {
	apiKey := c.rotateKey()
	u := BuildBalancemultiURL(addrs, apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var ar apiResponse
	if err := json.Unmarshal(body, &ar); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}
	// If status == "0" and message contains rate limit / throttle, return error to trigger retry
	if ar.Status == "0" {
		return nil, fmt.Errorf("etherscan status 0: %s", ar.Message)
	}
	var res []balanceResult
	if err := json.Unmarshal(ar.Result, &res); err != nil {
		// Sometimes result may be a single object
		var single balanceResult
		if err2 := json.Unmarshal(ar.Result, &single); err2 == nil {
			res = append(res, single)
		} else {
			return nil, fmt.Errorf("failed to decode result: %w", err)
		}
	}
	out := make(map[string]string, len(res))
	for _, r := range res {
		out[strings.ToLower(r.Account)] = r.Balance
	}
	return out, nil
}
