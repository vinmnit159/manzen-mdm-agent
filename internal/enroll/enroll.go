// Package enroll handles one-time device enrollment against the ISMS backend.
// On success it saves a config file to /Library/Application Support/ManzenAgent/config.json
// (or ~/.manzen-agent/config.json on non-root runs) containing the device's
// API key and device ID for all subsequent check-ins.
package enroll

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Config is persisted after successful enrollment.
type Config struct {
	DeviceID   string    `json:"deviceId"`
	APIKey     string    `json:"apiKey"`
	ServerURL  string    `json:"serverURL"`
	EnrolledAt time.Time `json:"enrolledAt"`
}

// enrollRequest is sent to POST /api/agent/enroll
type enrollRequest struct {
	EnrollmentToken string `json:"enrollmentToken"`
	Hostname        string `json:"hostname"`
	OSType          string `json:"osType"`
	OSVersion       string `json:"osVersion"`
	SerialNumber    string `json:"serialNumber"`
}

// enrollResponse is returned by the backend on success.
type enrollResponse struct {
	DeviceID string `json:"deviceId"`
	APIKey   string `json:"apiKey"`
}

// Run performs enrollment and saves the resulting config.
func Run(serverURL, token string) error {
	hostname, _ := os.Hostname()
	osVersion := run("sw_vers", "-productVersion")
	serial := serialNumber()

	body := enrollRequest{
		EnrollmentToken: token,
		Hostname:        hostname,
		OSType:          runtime.GOOS,
		OSVersion:       osVersion,
		SerialNumber:    serial,
	}

	data, _ := json.Marshal(body)
	resp, err := http.Post(serverURL+"/api/agent/enroll", "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("enrollment rejected (HTTP %d): %s", resp.StatusCode, string(raw))
	}

	var result enrollResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return fmt.Errorf("invalid response: %w", err)
	}

	cfg := Config{
		DeviceID:   result.DeviceID,
		APIKey:     result.APIKey,
		ServerURL:  serverURL,
		EnrolledAt: time.Now().UTC(),
	}

	if err := SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Enrolled successfully!\n  Device ID : %s\n  Config    : %s\n", cfg.DeviceID, configPath())
	return nil
}

// ─── Config persistence ───────────────────────────────────────────────────────

func configPath() string {
	// Prefer system-wide path when running as root
	if os.Getuid() == 0 {
		return "/Library/Application Support/ManzenAgent/config.json"
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".manzen-agent", "config.json")
}

func SaveConfig(cfg Config) error {
	p := configPath()
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0600)
}

func LoadConfig() (*Config, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return nil, fmt.Errorf("config not found at %s: %w", configPath(), err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func run(name string, args ...string) string {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func serialNumber() string {
	out := run("system_profiler", "SPHardwareDataType")
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "Serial Number") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}
