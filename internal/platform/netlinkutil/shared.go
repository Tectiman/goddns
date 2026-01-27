package netlinkutil

import (
	"net"
	"time"
)

// IPv6Info contains information about an IPv6 address
// 统一结构体，供各平台实现复用
// 只在 shared.go 定义，其他文件引用
//
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

// populateInfo 填充 IPv6Info 的附加属性
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
