//go:build linux

package ifaddr

import (
    "encoding/json"
    "fmt"
    "time"
    "net"

    stdnetlink "github.com/vishvananda/netlink"
)

type netlinkAddrInfo struct {
    Addr   string `json:"addr"`
    Pltime uint32 `json:"pltime"`
    Vltime uint32 `json:"vltime"`
}

// GetAvailableIPv6 returns IPv6 addresses from an interface using netlink
func GetAvailableIPv6(interfaceName string) ([]IPv6Info, error) {
    link, err := stdnetlink.LinkByName(interfaceName)
    if err != nil {
        return nil, fmt.Errorf("failed to find interface %s: %w", interfaceName, err)
    }

    addrList, err := stdnetlink.AddrList(link, stdnetlink.FAMILY_V6)
    if err != nil {
        return nil, fmt.Errorf("failed to get address list for %s: %w", interfaceName, err)
    }

    var addrInfos []netlinkAddrInfo
    for _, addr := range addrList {
        if addr.IP.To4() != nil {
            continue
        }
        if addr.IP.IsLinkLocalUnicast() {
            continue
        }

        addrInfo := netlinkAddrInfo{
            Addr:   addr.IP.String(),
            Pltime: uint32(addr.PreferedLft),
            Vltime: uint32(addr.ValidLft),
        }
        addrInfos = append(addrInfos, addrInfo)
    }

    if len(addrInfos) == 0 {
        return nil, fmt.Errorf("no IPv6 address found on interface %s", interfaceName)
    }

    // Convert to JSON and back to ensure consistency with other platforms
    jsonBytes, err := json.Marshal(addrInfos)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal addresses to JSON: %w", err)
    }

    var parsedAddrs []netlinkAddrInfo
    if err := json.Unmarshal(jsonBytes, &parsedAddrs); err != nil {
        return nil, fmt.Errorf("failed to unmarshal JSON addresses: %w", err)
    }

    var infos []IPv6Info
    for _, addrInfo := range parsedAddrs {
        ip := net.ParseIP(addrInfo.Addr)
        info := IPv6Info{
            IP:           ip,
            PreferredLft: time.Duration(addrInfo.Pltime) * time.Second,
            ValidLft:     time.Duration(addrInfo.Vltime) * time.Second,
        }
        populateInfo(&info)
        infos = append(infos, info)
    }

    return infos, nil
}
