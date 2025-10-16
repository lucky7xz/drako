package main

import (
	"fmt"
	"net"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	gopsutil_net "github.com/shirou/gopsutil/v3/net"
)

func checkNetworkStatus() tea.Cmd {
	return func() tea.Msg {
		_, err := net.DialTimeout("tcp", "8.8.8.8:53", 2*time.Second)
		online := err == nil

		counters, err := gopsutil_net.IOCounters(false)
		if err != nil || len(counters) == 0 {
			return networkStatusMsg{online: online, err: err, t: time.Now()}
		}

		return networkStatusMsg{online: online, counters: counters[0], t: time.Now()}
	}
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
