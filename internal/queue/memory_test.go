package queue

import (
	"testing"
)

func TestMemoryMonitor(t *testing.T) {
	monitor := NewMemoryMonitor()

	t.Run("GetMemoryUsage", func(t *testing.T) {
		usage, err := monitor.GetMemoryUsage()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if usage == 0 {
			t.Error("Expected non-zero memory usage")
		}
	})

	t.Run("GetMemoryThreshold", func(t *testing.T) {
		threshold := monitor.GetMemoryThreshold()
		expected := uint64(28 * 1024 * 1024 * 1024) // 28GB
		if threshold != expected {
			t.Errorf("Expected threshold %d, got %d", expected, threshold)
		}
	})

	t.Run("SetMemoryThreshold", func(t *testing.T) {
		newThreshold := uint64(16 * 1024 * 1024 * 1024) // 16GB
		monitor.SetMemoryThreshold(newThreshold)
		
		if monitor.GetMemoryThreshold() != newThreshold {
			t.Errorf("Expected threshold %d, got %d", newThreshold, monitor.GetMemoryThreshold())
		}
	})

	t.Run("IsMemoryPressure", func(t *testing.T) {
		// Set a very high threshold to ensure no pressure
		monitor.SetMemoryThreshold(1024 * 1024 * 1024 * 1024) // 1TB
		if monitor.IsMemoryPressure() {
			t.Error("Expected no memory pressure with high threshold")
		}

		// Set a very low threshold to ensure pressure
		monitor.SetMemoryThreshold(1) // 1 byte
		if !monitor.IsMemoryPressure() {
			t.Error("Expected memory pressure with low threshold")
		}
	})

	t.Run("GetMemoryStats", func(t *testing.T) {
		stats, err := monitor.GetMemoryStats()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if stats == nil {
			t.Fatal("Expected non-nil stats")
		}
		if stats.HeapInuse == 0 {
			t.Error("Expected non-zero heap usage")
		}
	})
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    uint64
		expected string
	}{
		{0, "0 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{28 * 1024 * 1024 * 1024, "28.0 GB"},
	}

	for _, test := range tests {
		result := FormatBytes(test.bytes)
		if result != test.expected {
			t.Errorf("FormatBytes(%d) = %s, expected %s", test.bytes, result, test.expected)
		}
	}
}