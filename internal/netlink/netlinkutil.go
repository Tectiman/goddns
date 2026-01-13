package netlinkutil

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	stdnetlink "github.com/vishvananda/netlink"
	"netlink_example/internal/log"
	"netlink_example/internal/config"
)

// IPv6Info contains information about an IPv6 address
type IPv6Info struct {
	IP             net.IP
	Scope          string
	AddressState   string
	PreferredLft   time.Duration
	ValidLft       time.Duration
	IsDeprecated   bool
	IsUniqueLocal  bool
	IsCandidate    bool // Whether it is a DDNS candidate
}

// GetAvailableIPv6 returns IPv6 addresses from an interface
func GetAvailableIPv6(interfaceName string) ([]IPv6Info, error) {
	link, err := stdnetlink.LinkByName(interfaceName)
	if err != nil {
		return nil, fmt.Errorf("failed to find interface %s: %w", interfaceName, err)
	}

	addrList, err := stdnetlink.AddrList(link, stdnetlink.FAMILY_V6)
	if err != nil {
		return nil, fmt.Errorf("failed to get address list for %s: %w", interfaceName, err)
	}

	var infos []IPv6Info
	for _, addr := range addrList {
		if addr.IP.To4() != nil {
			continue
		}
		if addr.IP.IsLinkLocalUnicast() {
			continue
		}

		info := IPv6Info{
			IP:           addr.IP,
			PreferredLft: time.Duration(addr.PreferedLft) * time.Second,
			ValidLft:     time.Duration(addr.ValidLft) * time.Second,
		}
		populateInfo(&info)
		infos = append(infos, info)
	}

	if len(infos) == 0 {
		return nil, fmt.Errorf("no IPv6 address found on interface %s", interfaceName)
	}

	return infos, nil
}

func populateInfo(info *IPv6Info) {
	ipBytes := info.IP
	info.IsUniqueLocal = ipBytes[0] == 0xfc || ipBytes[0] == 0xfd

	if info.IP.IsLinkLocalUnicast() {
		info.Scope = "Link Local"
	} else if info.IsUniqueLocal {
		info.Scope = "Unique Local (ULA)"
	} else {
		info.Scope = "Global Unicast"
	}

	info.IsDeprecated = info.PreferredLft.Seconds() <= 0 && info.ValidLft.Seconds() > 0

	if info.ValidLft.Seconds() == 0 {
		info.AddressState = "Expired"
	} else if info.IsDeprecated {
		info.AddressState = "Deprecated"
	} else if info.PreferredLft.Seconds() < info.ValidLft.Seconds() {
		info.AddressState = "Preferred/Dynamic"
	} else {
		info.AddressState = "Preferred/Static"
	}

	info.IsCandidate = info.Scope == "Global Unicast" && !info.IsDeprecated && !info.IsUniqueLocal
}

// IsPrivateOrLocalIP returns true for non-global addresses
func IsPrivateOrLocalIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLinkLocalUnicast() {
		return true
	}
	if ip[0] == 0xfc || ip[0] == 0xfd {
		return true
	}
	if ip.IsLoopback() {
		return true
	}
	return false
}

// SelectBestIPv6 selects the best IPv6 based on PreferredLft
func SelectBestIPv6(config config.Config, infos []IPv6Info) (string, error) {
	candidates := make([]IPv6Info, 0)
	for _, info := range infos {
		if info.IsCandidate {
			candidates = append(candidates, info)
		}
	}

	if len(candidates) == 0 {
		return "", errors.New("no suitable DDNS Candidate (Global Unicast, not deprecated) found")
	}

	var bestCandidate IPv6Info
	maxPreferredLft := time.Duration(0)
	for _, info := range candidates {
		if info.PreferredLft > maxPreferredLft {
			maxPreferredLft = info.PreferredLft
			bestCandidate = info
		}
	}

	log.Info(false, "Selected IP via default 'Max Preferred Lifetime' strategy: %s", bestCandidate.IP.String())
	return bestCandidate.IP.String(), nil
}

// GetIPv6Fallback queries remote API for an IPv6 address
func GetIPv6Fallback(cfg config.Config, quiet bool) ([]IPv6Info, error) {
	var urls []string
	if cfg.GetIP.URL != "" {
		urls = append(urls, cfg.GetIP.URL)
	}
	if len(urls) == 0 {
		return nil, errors.New("no IP API URL configured in 'get_ip.url'")
	}

	const timeout = 5 * time.Second
	const retries = 1

	for i, url := range urls {
		if !quiet {
			log.Info(quiet, "Trying fallback API %d/%d: %s", i+1, len(urls), url)
		}

		for attempt := 0; attempt <= retries; attempt++ {
			client := &http.Client{Timeout: timeout}
			resp, err := client.Get(url)
			if err != nil {
				if !quiet && attempt == retries {
					log.Error(quiet, "Fallback API %s (attempt %d/%d) failed: %v", url, attempt+1, retries, err)
				}
				if attempt < retries {
					time.Sleep(time.Second)
				}
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				if !quiet {
					log.Error(quiet, "Fallback API %s returned status: %d", url, resp.StatusCode)
				}
				break
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				if !quiet {
					log.Error(quiet, "Failed to read response from %s: %v", url, err)
				}
				break
			}

			lines := strings.Split(string(body), "\n")
			var candidateIPStr string
			for _, line := range lines {
				ipStr := strings.TrimSpace(line)
				if ipStr == "" {
					continue
				}
				if !strings.Contains(ipStr, ":") || strings.Contains(ipStr, "<") || strings.Contains(ipStr, "{") {
					continue
				}
				ip := net.ParseIP(ipStr)
				if ip != nil && ip.To4() == nil && !ip.IsLinkLocalUnicast() && !ip.IsLoopback() && !IsPrivateOrLocalIP(ip) {
					candidateIPStr = ipStr
					break
				}
			}

			if candidateIPStr == "" {
				if !quiet {
					log.Error(quiet, "No valid IPv6 found in response from %s: '%s' (must be pure Global Unicast IPv6 on first valid line)", url, string(body))
				}
				break
			}

			ip := net.ParseIP(candidateIPStr)
			info := IPv6Info{
				IP:           ip,
				PreferredLft: time.Hour * 24 * 365 * 10,
				ValidLft:     time.Hour * 24 * 365 * 10,
			}
			populateInfo(&info)

			if !quiet {
				log.Info(quiet, "Fallback API %s succeeded: %s", url, candidateIPStr)
			}
			return []IPv6Info{info}, nil
		}
	}

	if !quiet {
		log.Error(false, "All fallback APIs failed. Response must contain a pure text Global Unicast IPv6 address on the first valid line.")
	}
	return nil, errors.New("all fallback APIs invalid")
}
