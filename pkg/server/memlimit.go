package server

import (
	"log"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
)

// cgroupMemoryMaxPaths are the cgroup v2 and v1 files holding the container's
// memory limit, in preference order.
var cgroupMemoryMaxPaths = []string{
	"/sys/fs/cgroup/memory.max",                   // cgroup v2
	"/sys/fs/cgroup/memory/memory.limit_in_bytes", // cgroup v1
}

// memLimitHeadroom is the fraction of the container limit handed to the Go
// heap. The remainder covers everything the Go heap accounting does not see —
// the binary's own pages, goroutine stacks, the page cache backing a
// disk-resident search index, and mmap'd bleve segments. 0.85 leaves ~460 MiB
// of a 3 GiB limit for those.
const memLimitHeadroom = 0.85

// ApplyMemoryLimit derives GOMEMLIMIT from the container's memory limit so the
// Go runtime starts collecting before the kernel OOM-kills the process.
//
// Without this the heap can grow to roughly twice the live heap before GC runs,
// which is exactly how a ~985 MiB live heap produced 1.93 GiB of heap-system
// memory against a 1536 MiB container limit and an exit 137. See PV-MEMORY-019.
//
// A GOMEMLIMIT already present in the environment is left alone: the runtime
// has applied it and an explicit operator choice must win. It returns the limit
// it set, or 0 when it made no change.
//
// This is a soft limit, not a fix for a heap that genuinely does not fit. If
// the live heap approaches the limit the runtime will GC continuously rather
// than OOM — trading a crash for a stall — so it must be paired with actually
// reducing residency (for this service, --search-index-path).
func ApplyMemoryLimit() int64 {
	if _, ok := os.LookupEnv("GOMEMLIMIT"); ok {
		return 0
	}
	limit, ok := readCgroupMemoryMax()
	if !ok {
		return 0
	}
	soft := int64(float64(limit) * memLimitHeadroom)
	if soft <= 0 {
		return 0
	}
	debug.SetMemoryLimit(soft)
	log.Printf("memory: GOMEMLIMIT derived from cgroup: containerLimitBytes=%d softLimitBytes=%d headroom=%.2f",
		limit, soft, memLimitHeadroom)
	return soft
}

// readCgroupMemoryMax returns the container memory limit in bytes. It reports
// false when there is no limit ("max"), when the files are absent (not
// containerised), or when the value is implausibly large — cgroup v1 signals
// "unlimited" with a sentinel near max int64 rather than a keyword.
func readCgroupMemoryMax() (int64, bool) {
	const v1Unlimited = int64(1) << 53 // ~9 PiB; any real limit is far below

	for _, path := range cgroupMemoryMaxPaths {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		text := strings.TrimSpace(string(raw))
		if text == "" || text == "max" {
			return 0, false
		}
		value, err := strconv.ParseInt(text, 10, 64)
		if err != nil || value <= 0 || value >= v1Unlimited {
			continue
		}
		return value, true
	}
	return 0, false
}
