package ga4

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"runtime"
	"time"

	"github.com/ryanfowler/uuid"
)

// Send measurement data to the GA4 server
// The data is sent in the form of a measurement protocol payload
// https://developers.google.com/analytics/devguides/collection/protocol/ga4

// NewGA4Client creates a new GA4Client object with the measurementID and apiSecret.
func NewGA4Client(measurementID, apiSecret, userID string, debug bool) (*GA4Client, error) {
	// If userID is not provided, generate a new one
	if userID == "" {
		u, err := uuid.NewV7(time.Now())
		if err != nil {
			return nil, err
		}

		userID = u.String()
	}

	return &GA4Client{
		measurementID: measurementID,
		apiSecret:     apiSecret,
		userID:        userID,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		debug: debug,
	}, nil
}

// SendEvent sends one event to Google Analytics
func (g *GA4Client) SendEvent(event Event) error {
	query := url.Values{}
	query.Add("api_secret", g.apiSecret)
	query.Add("measurement_id", g.measurementID)

	var uri string
	if g.debug {
		uri = fmt.Sprintf("https://www.google-analytics.com/debug/mp/collect?%s", query.Encode())
	} else {
		uri = fmt.Sprintf("https://www.google-analytics.com/mp/collect?%s", query.Encode())
	}

	// append event params
	if event.Params == nil {
		event.Params = map[string]interface{}{}
	}
	event.Params["os"] = runtime.GOOS
	event.Params["arch"] = runtime.GOARCH
	event.Params["version"] = VERSION

	payload := Payload{
		ClientID:        fmt.Sprintf("%d.%d", rand.Int31(), time.Now().Unix()),
		UserID:          g.userID,
		TimestampMicros: time.Now().UnixMicro(),
		Events:          []Event{event},
	}

	if g.debug {
		fmt.Printf("[DEBUG] send GA4 event %s %+v", uri, payload)
	}

	bs, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%s", "marshal GA4 request payload failed")
	}

	body := bytes.NewReader(bs)
	res, err := g.httpClient.Post(uri, "application/json", body)
	if err != nil {
		return fmt.Errorf("%s", "request GA4 failed")
	}

	if res.StatusCode >= 300 {
		return fmt.Errorf("validation response got unexpected status %d", res.StatusCode)
	}

	if !g.debug {
		return nil
	}

	bs, err = io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("%s", "read GA4 response body failed")
	}

	validationResponse := ValidationResponse{}
	err = json.Unmarshal(bs, &validationResponse)
	if err != nil {
		return fmt.Errorf("%s", "unmarshal GA4 response body failed")
	}

	if g.debug {
		fmt.Printf("[DEBUG] get GA4 validation response %d %+v", res.StatusCode, validationResponse)
	}

	return nil
}
