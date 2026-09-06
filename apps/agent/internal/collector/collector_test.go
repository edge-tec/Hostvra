package collector

import (
	"testing"
)

func TestCollectorSampling(t *testing.T) {
	c := NewCollector()
	metrics, err := c.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if metrics.Hostname == "" {
		t.Error("Expected Hostname to be populated")
	}

	if metrics.CPUCores <= 0 {
		t.Errorf("Expected CPU cores > 0, got %d", metrics.CPUCores)
	}

	if metrics.RAMTotalMB <= 0 {
		t.Errorf("Expected RAMTotalMB > 0, got %d", metrics.RAMTotalMB)
	}

	if metrics.DiskTotalGB <= 0 {
		t.Errorf("Expected DiskTotalGB > 0, got %d", metrics.DiskTotalGB)
	}
}
