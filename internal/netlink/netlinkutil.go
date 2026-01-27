package netlinkutil
package netlinkutil

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"goddns/internal/log"
	"goddns/internal/config"

	xnet "golang.org/x/net/proxy"
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
// Implementation varies by platform: Linux uses netlink, others use command parsing

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

	return bestCandidate.IP.String(), nil
}

// createHTTPClient creates an HTTP client with optional proxy support
func createHTTPClient(cfg config.Config) (*http.Client, error) {
	transport := &http.Transport{}

	if cfg.Proxy != "" {
		proxyURL, err := url.Parse(cfg.Proxy)
		if err != nil {
			return nil, err
		}

		switch proxyURL.Scheme {
		case "http", "https":
			transport.Proxy = http.ProxyURL(proxyURL)
		case "socks5", "socks5h":
			var auth *xnet.Auth
			if proxyURL.User != nil {
				pw, _ := proxyURL.User.Password()
				username := proxyURL.User.Username()
				auth = &xnet.Auth{User: username, Password: pw}
			}
			dialer, err := xnet.SOCKS5("tcp", proxyURL.Host, auth, xnet.Direct)
			if err != nil {
				return nil, err
			}
			transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			}
		default:
			return nil, errors.New("unsupported proxy scheme: " + proxyURL.Scheme)
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
	}, nil
}

// GetIPv6Fallback queries remote API for an IPv6 address
func GetIPv6Fallback(cfg config.Config, quiet bool) ([]IPv6Info, error) {
	var urls []string
	
	// 优先使用新的URLs数组字段，如果没有则使用旧的URL字段
	if len(cfg.GetIP.URLs) > 0 {
		urls = cfg.GetIP.URLs
	} else if cfg.GetIP.URL != "" {
		urls = []string{cfg.GetIP.URL}
	}
	
	if len(urls) == 0 {
		return nil, errors.New("no IP API URL configured in 'get_ip.urls' or 'get_ip.url'")
	}

	const retries = 2

	for i, url := range urls {
		if !quiet {
			log.Info(quiet, "Trying fallback API %d/%d: %s", i+1, len(urls), url)
		}

		// 创建支持代理的HTTP客户端
		client, err := createHTTPClient(cfg)
		if err != nil {
			return nil, errors.New("failed to create HTTP client with proxy: " + err.Error())
		}

		for attempt := 0; attempt <= retries; attempt++ {
			resp, err := client.Get(url)
			if err != nil {
				if !quiet && attempt == retries {
					log.Error(quiet, "Fallback API %s (attempt %d/%d) failed: %v", url, attempt+1, retries, err)
				}
				if attempt < retries {
					time.Sleep(time.Second * 2)
				}
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				if !quiet {
					log.Error(quiet, "Fallback API %s returned status: %d", url, resp.StatusCode)
				}
				continue // 继续尝试下一个URL，而不是跳出循环
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				if !quiet {
					log.Error(quiet, "Failed to read response from %s: %v", url, err)
				}
				continue // 继续尝试下一个URL
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
				continue // 继续尝试下一个URL
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
		log.Error(false, "All fallback APIs failed. Tried %d URLs: %v", len(urls), urls)
	}
	return nil, errors.New("all fallback APIs invalid")
}