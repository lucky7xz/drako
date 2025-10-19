package main

import (
	"fmt"
	"net"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	gopsutil_net "github.com/shirou/gopsutil/v3/net"
)

func checkNetworkStatus() tea.Cmd {
	return func() tea.Msg {
		online := localOnline()

		// Get per-interface counters to filter out loopback/virtual traffic
		perIface, err := gopsutil_net.IOCounters(true)
		if err != nil {
			return networkStatusMsg{online: online, err: err, t: time.Now()}
		}

		// Sum only real NICs (exclude lo, docker, vbox, etc.)
		var totalSent, totalRecv uint64
		for _, c := range perIface {
			if isVirtualInterface(c.Name) {
				continue
			}
			totalSent += c.BytesSent
			totalRecv += c.BytesRecv
		}

		return networkStatusMsg{online: online, counters: gopsutil_net.IOCountersStat{BytesSent: totalSent, BytesRecv: totalRecv}, t: time.Now()}
	}
}

// isVirtualInterface returns true for loopback or known virtual interface prefixes.
// This avoids counting local chatter (Docker, VMs, multicast) as WAN activity.
func isVirtualInterface(name string) bool {
	virtualPrefixes := []string{"lo", "docker", "veth", "br-", "vbox", "vmnet", "tailscale", "tun", "tap"}
	for _, prefix := range virtualPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// localOnline determines likely online status without external probes, cross-platform.
// It returns true if there is at least one non-loopback, up interface with a
// non-link-local global unicast IP address.
func localOnline() bool {
	ifaces, err := net.Interfaces()
	if err != nil {
		return false
	}
	for _, iface := range ifaces {
		if (iface.Flags&net.FlagUp) == 0 || (iface.Flags&net.FlagLoopback) != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil {
				continue
			}
			if ip.IsGlobalUnicast() && !ip.IsLinkLocalUnicast() && !isAPIPA(ip) {
				return true
			}
		}
	}
	return false
}

// isAPIPA reports whether the IP is an IPv4 Automatic Private IP Address (169.254.0.0/16).
// We exclude APIPA because it indicates no DHCP/gateway and thus no real upstream connectivity.
func isAPIPA(ip net.IP) bool {
	ip4 := ip.To4()
	return ip4 != nil && ip4[0] == 169 && ip4[1] == 254
}

func networkTick() tea.Cmd {
	return tea.Tick(2500*time.Millisecond, func(t time.Time) tea.Msg {
		return checkNetworkStatus()()
	})
}

func formatTraffic(bps float64) string {
	const (
		kb = 1024
		mb = 1024 * kb
	)
	switch {
	case bps > mb:
		return fmt.Sprintf("%.2f MB/s", bps/mb)
	case bps > kb:
		return fmt.Sprintf("%.2f KB/s", bps/kb)
	default:
		return fmt.Sprintf("%.0f B/s", bps)
	}
}
