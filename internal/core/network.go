package core

import (
	"fmt"
	"net"
	"strings"
	"time"

	gopsutil_net "github.com/shirou/gopsutil/v3/net"
)

type NetworkStatus struct {
	Online   bool
	Counters gopsutil_net.IOCountersStat
	Time     time.Time
	Err      error
}

func CheckNetworkStatus() NetworkStatus {
	online := localOnline()

	// Get per-interface counters to filter out loopback/virtual traffic
	perIface, err := gopsutil_net.IOCounters(true)
	if err != nil {
		return NetworkStatus{Online: online, Err: err, Time: time.Now()}
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

	return NetworkStatus{Online: online, Counters: gopsutil_net.IOCountersStat{BytesSent: totalSent, BytesRecv: totalRecv}, Time: time.Now()}
}

// isVirtualInterface returns true for loopback or known virtual interface prefixes.
// This avoids counting local chatter (Docker, VMs, multicast) as WAN activity.
func isVirtualInterface(name string) bool {
	virtualPrefixes := []string{"lo", "Loopback Pseudo-Interface", "docker", "veth", "br-", "virbr", "vbox", "vmnet",
		"tailscale", "tun", "utun", "tap", "wg"}
	for _, prefix := range virtualPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// localOnline determines likely online status without external probes, cross-platform.
// It returns true if a real interface is up with a carrier (running) and holds
// a global unicast address; that excludes link-local, so DHCP-less APIPA too.
func localOnline() bool {
	ifaces, err := net.Interfaces()
	if err != nil {
		return false
	}
	for _, iface := range ifaces {
		running := iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagRunning != 0
		if !running || iface.Flags&net.FlagLoopback != 0 || isVirtualInterface(iface.Name) {
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
			if ip.IsGlobalUnicast() {
				return true
			}
		}
	}
	return false
}

func FormatTraffic(bps float64) string {
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
