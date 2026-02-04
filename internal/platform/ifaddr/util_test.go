package ifaddr

import (
    "testing"
    "net"
    "time"
    cfgpkg "goddns/internal/config"
)

func TestSelectBestIPv6(t *testing.T) {
    // build candidate addresses
    a1 := IPv6Info{IP: net.ParseIP("2001:db8::1"), PreferredLft: time.Hour, ValidLft: time.Hour * 2}
    a2 := IPv6Info{IP: net.ParseIP("2001:db8::2"), PreferredLft: time.Hour * 24, ValidLft: time.Hour * 24}
    a3 := IPv6Info{IP: net.ParseIP("fc00::1"), PreferredLft: time.Hour * 24 * 7, ValidLft: time.Hour * 24 * 7}

    populateInfo(&a1)
    populateInfo(&a2)
    populateInfo(&a3)

    infos := []IPv6Info{a1, a2, a3}

    best, err := SelectBestIPv6(cfgpkg.Config{}, infos)
    if err != nil {
        t.Fatalf("SelectBestIPv6 returned error: %v", err)
    }
    if best != "2001:db8::2" {
        t.Fatalf("expected 2001:db8::2 got %s", best)
    }
}
