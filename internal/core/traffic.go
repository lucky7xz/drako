package core

import "time"

// TrafficMeter keeps a sliding window of network counter samples and reports
// the average byte rate across the retained window. It holds no presentation
// state — callers turn its numbers into labels.
type TrafficMeter struct {
	// WindowSeconds is how far back samples are kept.
	WindowSeconds float64

	times []time.Time
	sent  []uint64
	recv  []uint64
}

// Sample records cumulative sent/recv counters observed at now, then drops
// any samples older than the window.
func (tm *TrafficMeter) Sample(sent, recv uint64, now time.Time) {
	tm.times = append(tm.times, now)
	tm.sent = append(tm.sent, sent)
	tm.recv = append(tm.recv, recv)

	cutoff := now.Add(-time.Duration(tm.WindowSeconds * float64(time.Second)))
	first := 0
	for i, t := range tm.times {
		if !t.Before(cutoff) {
			break
		}
		first = i + 1
	}
	if first > 0 && len(tm.times) > first {
		tm.times = tm.times[first:]
		tm.sent = tm.sent[first:]
		tm.recv = tm.recv[first:]
	}
}

// Samples returns how many readings are currently retained.
func (tm *TrafficMeter) Samples() int {
	return len(tm.times)
}

// Rates returns the average bytes/sec for sent and recv across the window.
// ok is false until at least two samples span a positive duration — the
// caller distinguishes "no data yet" (Samples <= 1) from "degenerate span"
// (ok false despite enough samples).
func (tm *TrafficMeter) Rates() (sentBps, recvBps float64, ok bool) {
	if len(tm.times) <= 1 {
		return 0, 0, false
	}
	duration := tm.times[len(tm.times)-1].Sub(tm.times[0]).Seconds()
	if duration <= 0 {
		return 0, 0, false
	}
	sentDelta := tm.sent[len(tm.sent)-1] - tm.sent[0]
	recvDelta := tm.recv[len(tm.recv)-1] - tm.recv[0]
	return float64(sentDelta) / duration, float64(recvDelta) / duration, true
}
