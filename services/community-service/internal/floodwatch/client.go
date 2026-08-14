// Package floodwatch surfaces Google's flood forecasts for CivicOS
// communities.
//
// # What this is, and what it is not
//
// Google Research runs the models. They ingest rainfall, terrain and gauge
// data and produce a forecast for river basins that are often completely
// uninstrumented — that is the hard part, and it happens entirely upstream.
// What arrives here is the OUTPUT: a severity, a trend, a time range and a
// coordinate.
//
// So there is no model in this package, no inference and nothing learned.
// It is an HTTP client, a distance matcher, a change detector and a
// notifier. CivicOS must never present these forecasts as its own — if one
// is wrong, it has to be visibly Google's forecast that was wrong, with a
// link to Flood Hub. Every surface that renders this data attributes it.
//
// # Why it lives in community-service
//
// It persists, it runs on a timer, and it emits notifications people act
// on. civicai-service has no database, no scheduler, and a contract that
// every response is a suggestion which auto-acts on nothing. This is the
// opposite of that on all three counts. Communities and notifications both
// live here, so the join and the fan-out stay in-process.
//
// # Pilot API
//
// The Flood Forecasting API is in pilot and Google state breaking changes
// should be expected. Everything is therefore behind an interval switch:
// set FLOOD_POLL_INTERVAL_MINUTES=0 and the feature disappears without a
// deploy.
package floodwatch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	// defaultBaseURL is Google's Flood Forecasting service endpoint.
	defaultBaseURL = "https://floodforecasting.googleapis.com"

	// searchPageSize is what we ask for per page. The API allows up to
	// 20,000; Nigeria has far fewer gauges than that, so a single page is
	// the expected case and pagination is the safety net rather than the
	// norm.
	searchPageSize = 20000

	// maxPages bounds the pagination loop. Without it a malformed
	// nextPageToken that never empties would spin forever against a
	// rate-limited API.
	maxPages = 20
)

// Severity mirrors the API's enum. Ordered by how much attention it
// deserves so callers can compare with severityRank rather than
// reimplementing the ordering each time.
type Severity string

const (
	SeverityUnknown     Severity = "UNKNOWN"
	SeverityNoFlooding  Severity = "NO_FLOODING"
	SeverityAboveNormal Severity = "ABOVE_NORMAL"
	SeveritySevere      Severity = "SEVERE"
	SeverityExtreme     Severity = "EXTREME"
)

// severityRank orders severities for comparison. UNKNOWN sits with
// NO_FLOODING at zero on purpose: the API returns it when it has nothing
// to say, and treating "we don't know" as an escalation would alarm people
// on missing data.
func severityRank(s Severity) int {
	switch s {
	case SeverityAboveNormal:
		return 1
	case SeveritySevere:
		return 2
	case SeverityExtreme:
		return 3
	default:
		return 0
	}
}

// IsAlerting reports whether a severity is worth telling a citizen about.
func (s Severity) IsAlerting() bool { return severityRank(s) > 0 }

// Trend mirrors the API's forecastTrend.
type Trend string

const (
	TrendRise     Trend = "RISE"
	TrendFall     Trend = "FALL"
	TrendNoChange Trend = "NO_CHANGE"
)

// LatLng is Google's standard geographic point.
type LatLng struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// TimeRange is the window a forecast applies to.
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// FloodStatus is one gauge's forecast.
//
// Only the fields CivicOS actually uses are declared. The API also returns
// inundation map references and notification polygon ids; rendering those
// needs a map surface we do not have, and decoding data we cannot display
// would invite someone to assume it is being shown.
type FloodStatus struct {
	GaugeID         string    `json:"gaugeId"`
	QualityVerified bool      `json:"qualityVerified"`
	GaugeLocation   LatLng    `json:"gaugeLocation"`
	IssuedTime      time.Time `json:"issuedTime"`
	ForecastRange   TimeRange `json:"forecastTimeRange"`
	Trend           Trend     `json:"forecastTrend"`
	Severity        Severity  `json:"severity"`
	Source          string    `json:"source"`
}

// Gauge is the metadata behind a gauge id — chiefly the river name, which
// is what makes an alert legible ("the River Benue", not "gauge
// hybas_1121455890").
type Gauge struct {
	GaugeID         string `json:"gaugeId"`
	Location        LatLng `json:"location"`
	SiteName        string `json:"siteName"`
	River           string `json:"river"`
	CountryCode     string `json:"countryCode"`
	QualityVerified bool   `json:"qualityVerified"`
	HasModel        bool   `json:"hasModel"`
}

// Client talks to Google's Flood Forecasting API.
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		baseURL: defaultBaseURL,
		apiKey:  apiKey,
		// Generous but bounded: a single call can return every gauge in the
		// country, and a hung request must not wedge the poll goroutine.
		http: &http.Client{Timeout: 60 * time.Second},
	}
}

// WithBaseURL points the client at a different host. Tests use it; nothing
// in production should.
func (c *Client) WithBaseURL(u string) *Client {
	c.baseURL = u
	return c
}

// SearchFloodStatusByRegion returns every flood status for a CLDR region
// code — "NG" for Nigeria.
//
// One request covers the whole country, which is why polling is cheap: the
// API allows 200 requests per minute and an hourly sweep of Nigeria costs
// one of them.
func (c *Client) SearchFloodStatusByRegion(ctx context.Context, regionCode string) ([]FloodStatus, error) {
	var all []FloodStatus
	pageToken := ""

	for page := 0; page < maxPages; page++ {
		body := map[string]any{
			"regionCode": regionCode,
			"pageSize":   searchPageSize,
		}
		if pageToken != "" {
			body["pageToken"] = pageToken
		}

		var out struct {
			FloodStatuses []FloodStatus `json:"floodStatuses"`
			NextPageToken string        `json:"nextPageToken"`
		}
		if err := c.post(ctx, "/v1/floodStatus:searchLatestFloodStatusByArea", body, &out); err != nil {
			return nil, err
		}
		all = append(all, out.FloodStatuses...)
		if out.NextPageToken == "" {
			return all, nil
		}
		pageToken = out.NextPageToken
	}
	// Ran out of pages rather than results. Return what we have and say so:
	// a partial sweep is still worth acting on, and failing outright would
	// throw away real warnings.
	return all, fmt.Errorf("floodwatch: stopped after %d pages, results may be incomplete", maxPages)
}

// BatchGetGauges resolves gauge metadata for the given ids.
func (c *Client) BatchGetGauges(ctx context.Context, gaugeIDs []string) ([]Gauge, error) {
	if len(gaugeIDs) == 0 {
		return nil, nil
	}
	var out struct {
		Gauges []Gauge `json:"gauges"`
	}
	if err := c.post(ctx, "/v1/gauges:batchGet", map[string]any{"gaugeIds": gaugeIDs}, &out); err != nil {
		return nil, err
	}
	return out.Gauges, nil
}

func (c *Client) post(ctx context.Context, path string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("floodwatch: encode request: %w", err)
	}

	endpoint := c.baseURL + path + "?key=" + url.QueryEscape(c.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("floodwatch: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("floodwatch: %s: %w", path, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		// Capped read: an error body is for a log line, and an upstream
		// that returns megabytes of HTML should not become our memory
		// problem.
		snippet, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		// The key is in the query string, so the URL must never reach a log.
		return fmt.Errorf("floodwatch: %s: upstream %d: %s", path, res.StatusCode, bytes.TrimSpace(snippet))
	}
	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		return fmt.Errorf("floodwatch: decode %s: %w", path, err)
	}
	return nil
}
