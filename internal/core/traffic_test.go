package core

import (
	"testing"
	"time"
)

func TestTrafficMeter(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("no data until two samples", func(t *testing.T) {
		var tm TrafficMeter
		tm.WindowSeconds = 10
		if _, _, ok := tm.Rates(); ok {
			t.Error("empty meter reported rates")
		}
		tm.Sample(100, 200, base)
		if tm.Samples() != 1 {
			t.Errorf("Samples = %d, want 1", tm.Samples())
		}
		if _, _, ok := tm.Rates(); ok {
			t.Error("single sample reported rates")
		}
	})

	t.Run("rate is delta over duration", func(t *testing.T) {
		var tm TrafficMeter
		tm.WindowSeconds = 10
		tm.Sample(0, 0, base)
		tm.Sample(1000, 2000, base.Add(time.Second))
		sent, recv, ok := tm.Rates()
		if !ok {
			t.Fatal("ok = false, want true")
		}
		if sent != 1000 || recv != 2000 {
			t.Errorf("rates = (%v, %v), want (1000, 2000)", sent, recv)
		}
	})

	t.Run("samples older than the window are dropped", func(t *testing.T) {
		var tm TrafficMeter
		tm.WindowSeconds = 2
		tm.Sample(0, 0, base) // dropped once now advances past window
		tm.Sample(500, 500, base.Add(time.Second))
		tm.Sample(1500, 1500, base.Add(3*time.Second))
		if tm.Samples() != 2 {
			t.Fatalf("Samples = %d, want 2 (oldest trimmed)", tm.Samples())
		}
		// Window now spans t+1..t+3: delta 1000 over 2s = 500 B/s each.
		sent, recv, ok := tm.Rates()
		if !ok || sent != 500 || recv != 500 {
			t.Errorf("rates = (%v, %v, %v), want (500, 500, true)", sent, recv, ok)
		}
	})

	t.Run("degenerate same-timestamp span is not ok", func(t *testing.T) {
		var tm TrafficMeter
		tm.WindowSeconds = 10
		tm.Sample(0, 0, base)
		tm.Sample(1000, 1000, base)
		if tm.Samples() != 2 {
			t.Fatalf("Samples = %d, want 2", tm.Samples())
		}
		if _, _, ok := tm.Rates(); ok {
			t.Error("zero-duration span reported rates")
		}
	})
}
