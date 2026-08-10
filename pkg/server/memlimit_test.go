package server

import (
	"os"
	"path/filepath"
	"testing"
)

// TestReadCgroupMemoryMax pins the parsing of both cgroup layouts and the two
// ways a limit is spelled "unlimited" — v2 uses the literal "max", v1 uses a
// sentinel near max int64. Reading either as a real limit would set a
// nonsensical GOMEMLIMIT.
func TestReadCgroupMemoryMax(t *testing.T) {
	tests := []struct {
		name     string
		contents string
		want     int64
		wantOK   bool
	}{
		{"v2 concrete limit", "1610612736\n", 1610612736, true},
		{"v2 unlimited", "max\n", 0, false},
		{"v1 unlimited sentinel", "9223372036854771712\n", 0, false},
		{"empty", "", 0, false},
		{"garbage", "not-a-number\n", 0, false},
		{"zero", "0\n", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "memory.max")
			if err := os.WriteFile(path, []byte(tt.contents), 0o644); err != nil {
				t.Fatal(err)
			}
			restore := cgroupMemoryMaxPaths
			cgroupMemoryMaxPaths = []string{path}
			defer func() { cgroupMemoryMaxPaths = restore }()

			got, ok := readCgroupMemoryMax()
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("readCgroupMemoryMax() = (%d, %v), want (%d, %v)", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestApplyMemoryLimitRespectsExplicitEnv: an operator who set GOMEMLIMIT has
// already had it applied by the runtime, and must not be overridden.
func TestApplyMemoryLimitRespectsExplicitEnv(t *testing.T) {
	t.Setenv("GOMEMLIMIT", "512MiB")
	if got := ApplyMemoryLimit(); got != 0 {
		t.Errorf("ApplyMemoryLimit() = %d, want 0 (no change when GOMEMLIMIT is set)", got)
	}
}

// TestApplyMemoryLimitNoCgroup: outside a container there is no limit to derive
// from, and the runtime default must be left alone.
func TestApplyMemoryLimitNoCgroup(t *testing.T) {
	restore := cgroupMemoryMaxPaths
	cgroupMemoryMaxPaths = []string{filepath.Join(t.TempDir(), "absent")}
	defer func() { cgroupMemoryMaxPaths = restore }()

	if got := ApplyMemoryLimit(); got != 0 {
		t.Errorf("ApplyMemoryLimit() = %d, want 0 when no cgroup limit is readable", got)
	}
}
