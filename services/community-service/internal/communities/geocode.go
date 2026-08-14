package communities

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Geocoder turns an administrative name into a point.
//
// This exists so an admin creating a community does not have to leave the
// form, find the place on a map, copy coordinates and paste them back. It
// produces a STARTING POINT, never an answer: an LGA is a polygon and the
// geocoder returns something like its centre, which may be nowhere near
// the town people live in or the river that floods.
//
// So every surface that uses this presents the result as a suggestion to
// be checked, and the admin still confirms on a map before saving. That is
// the same line the rest of this feature holds — coordinates are set by a
// human who looked, never derived and trusted.
type Geocoder struct {
	apiKey string
	http   *http.Client
}

func NewGeocoder(apiKey string) *Geocoder {
	return &Geocoder{
		apiKey: apiKey,
		http:   &http.Client{Timeout: 10 * time.Second},
	}
}

// Enabled reports whether geocoding is configured. Without a key the
// endpoint says so plainly and the admin UI hides the button rather than
// offering something that will always fail.
func (g *Geocoder) Enabled() bool { return g != nil && g.apiKey != "" }

// Suggestion is a proposed point for a place.
type Suggestion struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	// FormattedAddress is what Google matched. Shown to the admin so they
	// can see whether it resolved the place they meant — "Ikeja, Lagos,
	// Nigeria" is reassuring, "Ikeja, Nigeria" less so.
	FormattedAddress string `json:"formattedAddress"`
	// LocationType is Google's own confidence signal. APPROXIMATE is the
	// normal result for an LGA and is worth surfacing, because it is
	// precisely the case where the point needs checking.
	LocationType string `json:"locationType"`
	// Partial is set when Google reports it did not match the whole query.
	Partial bool `json:"partialMatch"`
}

// Lookup geocodes "LGA, State, Nigeria".
//
// Country is pinned rather than left to the geocoder: "Kano" and "Ibadan"
// exist elsewhere in the world, and a community placed on the wrong
// continent would be excluded from every flood match without anything
// looking wrong.
func (g *Geocoder) Lookup(ctx context.Context, state, lga string) (*Suggestion, error) {
	if !g.Enabled() {
		return nil, &AppError{
			Code:    "GEOCODING_UNAVAILABLE",
			Message: "Automatic location lookup is not configured. Enter coordinates manually.",
			Status:  http.StatusServiceUnavailable,
		}
	}

	parts := make([]string, 0, 3)
	if s := strings.TrimSpace(lga); s != "" {
		parts = append(parts, s)
	}
	if s := strings.TrimSpace(state); s != "" {
		parts = append(parts, s)
	}
	if len(parts) == 0 {
		return nil, &AppError{Code: "VALIDATION_ERROR", Message: "Provide a state and LGA", Status: http.StatusBadRequest}
	}
	parts = append(parts, "Nigeria")

	endpoint := "https://maps.googleapis.com/maps/api/geocode/json?" + url.Values{
		"address": {strings.Join(parts, ", ")},
		// Bias to Nigeria as well as naming it, so a partial match cannot
		// wander abroad.
		"region":     {"ng"},
		"components": {"country:NG"},
		"key":        {g.apiKey},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("geocode: build request: %w", err)
	}
	res, err := g.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("geocode: %w", err)
	}
	defer res.Body.Close()

	var payload struct {
		Status  string `json:"status"`
		Results []struct {
			FormattedAddress string `json:"formatted_address"`
			PartialMatch     bool   `json:"partial_match"`
			Geometry         struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
				LocationType string `json:"location_type"`
			} `json:"geometry"`
		} `json:"results"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("geocode: decode: %w", err)
	}

	// ZERO_RESULTS is an ordinary outcome, not a failure — plenty of LGA
	// names do not geocode cleanly. The admin types the point instead.
	if payload.Status == "ZERO_RESULTS" || len(payload.Results) == 0 {
		return nil, &AppError{
			Code:    "GEOCODING_NO_MATCH",
			Message: "Could not find that place automatically. Enter coordinates manually.",
			Status:  http.StatusNotFound,
		}
	}
	if payload.Status != "OK" {
		// Quota, bad key, denied request. Never surfaced verbatim: the
		// upstream message can name the project and the key restriction.
		return nil, fmt.Errorf("geocode: upstream status %s", payload.Status)
	}

	top := payload.Results[0]
	return &Suggestion{
		Latitude:         top.Geometry.Location.Lat,
		Longitude:        top.Geometry.Location.Lng,
		FormattedAddress: top.FormattedAddress,
		LocationType:     top.Geometry.LocationType,
		Partial:          top.PartialMatch,
	}, nil
}
