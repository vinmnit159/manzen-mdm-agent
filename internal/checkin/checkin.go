// Package checkin sends collected posture data to the ISMS backend.
// The request is authenticated with the device's API key issued at enrollment.
package checkin

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/vinmnit159/manzen-mdm-agent/internal/collector"
	"github.com/vinmnit159/manzen-mdm-agent/internal/enroll"
)

// checkinRequest is the payload sent to POST /api/agent/checkin
type checkinRequest struct {
	DeviceID  string             `json:"deviceId"`
	Timestamp time.Time          `json:"timestamp"`
	Posture   *collector.Posture `json:"posture"`
	Signature string             `json:"signature"` // HMAC-SHA256(deviceId+timestamp, apiKey)
}

// Run performs a single authenticated check-in.
func Run(serverURL string, cfg *enroll.Config, posture *collector.Posture) error {
	ts := time.Now().UTC()

	// Sign: HMAC-SHA256 over "<deviceId>:<unix_ts>" using the API key as the secret
	sigPayload := fmt.Sprintf("%s:%d", cfg.DeviceID, ts.Unix())
	sig := sign(sigPayload, cfg.APIKey)

	body := checkinRequest{
		DeviceID:  cfg.DeviceID,
		Timestamp: ts,
		Posture:   posture,
		Signature: sig,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, serverURL+"/api/agent/checkin", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Device-ID", cfg.DeviceID)
	req.Header.Set("X-Agent-Signature", sig)
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server rejected check-in (HTTP %d): %s", resp.StatusCode, string(raw))
	}

	return nil
}

func sign(payload, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}
