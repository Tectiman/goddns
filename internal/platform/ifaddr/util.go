package ifaddr

import (
    "context"
    "errors"
    "fmt"
    "io"
    "net"
    "net/http"
    "net/url"
    "strings"
    "time"

    "goddns/internal/config"
    "goddns/internal/log"

    xnet "golang.org/x/net/proxy"
)

// SelectBestIPv6 selects the best IPv6 based on PreferredLft
func SelectBestIPv6(cfg config.Config, infos []IPv6Info) (string, error) {
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
    if len(cfg.GetIP.URLs) > 0 {
        urls = cfg.GetIP.URLs
    } else if cfg.GetIP.URL != "" {
        urls = []string{cfg.GetIP.URL}
    }

    if len(urls) == 0 {
        return nil, errors.New("no IP API URL configured in 'get_ip.urls' or 'get_ip.url'")
    }

    const retries = 2

    // create result channel for concurrent requests
    resultChan := make(chan struct {
        info []IPv6Info
        err  error
        url  string
    })
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    for _, u := range urls {
        go func(u string) {
            client, err := createHTTPClient(cfg)
            if err != nil {
                select {
                case resultChan <- struct {
                    info []IPv6Info
                    err  error
                    url  string
                }{nil, errors.New("failed to create HTTP client: " + err.Error()), u}:
                case <-ctx.Done():
                    return
                }
                return
            }

            for attempt := 0; attempt <= retries; attempt++ {
                select {
                case <-ctx.Done():
                    return
                default:
                }

                if !quiet {
                    log.Info("Trying fallback API %s (attempt %d/%d)", u, attempt+1, retries+1)
                }

                resp, err := client.Get(u)
                if err != nil {
                    if attempt == retries {
                        select {
                        case resultChan <- struct {
                            info []IPv6Info
                            err  error
                            url  string
                        }{nil, fmt.Errorf("API request failed: %v", err), u}:
                        case <-ctx.Done():
                            return
                        }
                    }
                    if attempt < retries {
                        time.Sleep(time.Second * 2)
                    }
                    continue
                }

                body, err := io.ReadAll(resp.Body)
                resp.Body.Close()
                if err != nil {
                    if attempt == retries {
                        select {
                        case resultChan <- struct {
                            info []IPv6Info
                            err  error
                            url  string
                        }{nil, fmt.Errorf("failed to read response: %v", err), u}:
                        case <-ctx.Done():
                            return
                        }
                    }
                    continue
                }

                if resp.StatusCode != http.StatusOK {
                    if attempt == retries {
                        select {
                        case resultChan <- struct {
                            info []IPv6Info
                            err  error
                            url  string
                        }{nil, fmt.Errorf("API returned status: %d", resp.StatusCode), u}:
                        case <-ctx.Done():
                            return
                        }
                    }
                    continue
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
                    if attempt == retries {
                        select {
                        case resultChan <- struct {
                            info []IPv6Info
                            err  error
                            url  string
                        }{nil, errors.New("no valid IPv6 found in response"), u}:
                        case <-ctx.Done():
                            return
                        }
                    }
                    continue
                }

                ip := net.ParseIP(candidateIPStr)
                info := IPv6Info{
                    IP:           ip,
                    PreferredLft: time.Hour * 24 * 365 * 10,
                    ValidLft:     time.Hour * 24 * 365 * 10,
                }
                populateInfo(&info)

                if !quiet {
                    log.Info("Fallback API %s succeeded: %s", u, candidateIPStr)
                }
                select {
                case resultChan <- struct {
                    info []IPv6Info
                    err  error
                    url  string
                }{[]IPv6Info{info}, nil, u}:
                case <-ctx.Done():
                    return
                }
                return
            }
        }(u)
    }

    var lastErr error
    for range urls {
        select {
        case res := <-resultChan:
            if res.err == nil {
                return res.info, nil
            }
            lastErr = res.err
            if !quiet {
                log.Error("API %s failed: %v", res.url, res.err)
            }
        case <-ctx.Done():
            return nil, errors.New("all API requests timed out")
        }
    }

    if !quiet {
        log.Error("All fallback APIs failed. Tried %d URLs: %v", len(urls), urls)
    }
    return nil, lastErr
}
