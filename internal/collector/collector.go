// Package collector gathers macOS security posture data.
// Each check maps to an ISO 27001 Annex A control:
//   - Disk encryption  → A.8.24 (Use of Cryptography)
//   - Screen lock      → A.5.15 (Access Control)
//   - Firewall         → A.8.20 (Networks Security)
//   - OS version       → A.8.8  (Technical Vulnerability Management)
//   - SIP              → A.8.7  (Protection against malware)
package collector

import (
	"bytes"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Posture holds the full security state of the device.
type Posture struct {
	Timestamp         time.Time `json:"timestamp"`
	Hostname          string    `json:"hostname"`
	OSType            string    `json:"osType"`
	OSVersion         string    `json:"osVersion"`
	SerialNumber      string    `json:"serialNumber"`
	DiskEncryption    bool      `json:"diskEncryptionEnabled"`
	ScreenLock        bool      `json:"screenLockEnabled"`
	FirewallEnabled   bool      `json:"firewallEnabled"`
	AntivirusEnabled  bool      `json:"antivirusEnabled"`
	SystemIntegrity   bool      `json:"systemIntegrityProtectionEnabled"`
	AutoUpdateEnabled bool      `json:"autoUpdateEnabled"`
	GatekeeperEnabled bool      `json:"gatekeeperEnabled"`
}

// Collect gathers posture data from the current macOS machine.
func Collect() (*Posture, error) {
	p := &Posture{
		Timestamp: time.Now().UTC(),
		OSType:    runtime.GOOS,
	}

	p.Hostname = run("hostname")
	p.OSVersion = run("sw_vers", "-productVersion")
	p.SerialNumber = serialNumber()
	p.DiskEncryption = checkFilevault()
	p.ScreenLock = checkScreenLock()
	p.FirewallEnabled = checkFirewall()
	p.SystemIntegrity = checkSIP()
	p.AutoUpdateEnabled = checkAutoUpdate()
	p.GatekeeperEnabled = checkGatekeeper()
	// macOS doesn't have a native AV binary — default false unless 3rd party detected
	p.AntivirusEnabled = checkAntivirus()

	return p, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func run(name string, args ...string) string {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func runAndContains(needle string, name string, args ...string) bool {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return false
	}
	return bytes.Contains(bytes.ToLower(out), []byte(strings.ToLower(needle)))
}

func serialNumber() string {
	// system_profiler SPHardwareDataType outputs "Serial Number (system): XXXX"
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

// FileVault — fdesetup status returns "FileVault is On."
func checkFilevault() bool {
	return runAndContains("FileVault is On", "fdesetup", "status")
}

// Screen lock — check that screensaver password is required
func checkScreenLock() bool {
	out := run("defaults", "-currentHost", "read", "com.apple.screensaver", "askForPassword")
	return strings.TrimSpace(out) == "1"
}

// Firewall — /usr/libexec/ApplicationFirewall/socketfilterfw --getglobalstate
func checkFirewall() bool {
	return runAndContains("enabled", "/usr/libexec/ApplicationFirewall/socketfilterfw", "--getglobalstate")
}

// SIP — csrutil status returns "System Integrity Protection status: enabled."
func checkSIP() bool {
	return runAndContains("enabled", "csrutil", "status")
}

// Auto-update — softwareupdate --schedule (prints "Automatic check is on")
func checkAutoUpdate() bool {
	return runAndContains("on", "softwareupdate", "--schedule")
}

// Gatekeeper — spctl --status returns "assessments enabled"
func checkGatekeeper() bool {
	return runAndContains("assessments enabled", "spctl", "--status")
}

// Antivirus — check for common macOS AV processes
func checkAntivirus() bool {
	avProcesses := []string{"com.malwarebytes", "CrowdStrike", "SentinelOne", "com.sophos", "com.symantec"}
	for _, proc := range avProcesses {
		if runAndContains(proc, "ps", "aux") {
			return true
		}
	}
	return false
}
