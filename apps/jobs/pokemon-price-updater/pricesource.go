package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// HTTPPriceSource reads quotes from a JSON price feed:
//
//	GET {baseURL}?ids=501773,501774
//	→ {"prices": {"501773": {"Normal": 1.42, "Holofoil": 5.00}}}
//
// Ids the feed can't quote are absent from the response. Any TCGplayer-
// compatible proxy can sit behind this contract.
type HTTPPriceSource struct {
	BaseURL string
	Client  *http.Client
}

func NewHTTPPriceSource(baseURL string) *HTTPPriceSource {
	return &HTTPPriceSource{
		BaseURL: baseURL,
		Client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *HTTPPriceSource) Prices(ctx context.Context, tcgplayerIDs []int) (map[int]map[string]float64, error) {
	idStrs := make([]string, len(tcgplayerIDs))
	for i, id := range tcgplayerIDs {
		idStrs[i] = strconv.Itoa(id)
	}
	u, err := url.Parse(s.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("price feed url: %w", err)
	}
	q := u.Query()
	q.Set("ids", strings.Join(idStrs, ","))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	res, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("price feed returned %d", res.StatusCode)
	}
	var body struct {
		Prices map[string]map[string]float64 `json:"prices"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("price feed body: %w", err)
	}
	out := make(map[int]map[string]float64, len(body.Prices))
	for idStr, prices := range body.Prices {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			continue // tolerate junk keys rather than failing the run
		}
		out[id] = prices
	}
	return out, nil
}
