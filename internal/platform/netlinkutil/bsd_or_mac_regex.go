//go:build !linux && !freebsd

package netlinkutil

import (
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// parseLifetime converts lifetime string to seconds
func parseLifetime(s string) (int64, error) {
	"your_module/shared" // Import shared.go for IPv6Info and functions
	if s == "forever" || s == "infinity" {
		return 315360000, nil // 10 years, as a large number for "forever"
	}

	// BSD time is already plain numeric seconds, parse directly
	seconds, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse lifetime number: %w", err)
	}
	return seconds, nil
}

// GetAvailableIPv6 executes "ifconfig" and parses output, for BSD/macOS
func GetAvailableIPv6(ifaceName string) ([]IPv6Info, error) {
	cmd := exec.Command("ifconfig", ifaceName)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute ifconfig %s: %w", ifaceName, err)
	}

	outputStr := string(output)
	var infos []IPv6Info

	re := regexp.MustCompile(`inet6\s+(\S+)\s+prefixlen\s+\d+.*?pltime\s+(\d+)\s+vltime\s+(\d+)`)

	matches := re.FindAllStringSubmatch(outputStr, -1)

	for _, match := range matches {
		ipStr := match[1]
		pltimeStr := match[2]
		vltimeStr := match[3]

		if parts := strings.Split(ipStr, "%"); len(parts) > 1 {
			ipStr = parts[0]
		}

		ip := net.ParseIP(ipStr)
		if ip == nil || ip.To4() != nil {
			continue
		}

		if strings.HasPrefix(ipStr, "fe80:") || strings.HasPrefix(ipStr, "::1") {
			continue
		}

		plSeconds, _ := parseLifetime(pltimeStr)
		vlSeconds, _ := parseLifetime(vltimeStr)

		info := IPv6Info{
			IP:           ip,
			PreferredLft: time.Duration(plSeconds) * time.Second,
			ValidLft:     time.Duration(vlSeconds) * time.Second,
		}

		populateInfo(&info)
		infos = append(infos, info)
	}

	if len(infos) == 0 {
		return nil, fmt.Errorf("no available IPv6 address found. Please check ifconfig output format")
	}

	return infos, nil
}