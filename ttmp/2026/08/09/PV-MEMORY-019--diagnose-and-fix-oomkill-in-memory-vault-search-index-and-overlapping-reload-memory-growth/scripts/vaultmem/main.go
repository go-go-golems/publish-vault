// Command vaultmem measures the resident memory cost of loading a real
// Obsidian vault through the same code paths the production server uses:
//
//	vault.New(root)        -> pkg/vault.LoadAll (parse every .md, render HTML,
//	                          build wiki-link index, backlinks, asset index)
//	search.New(v)          -> pkg/search, bleve.NewMemOnly, one doc per note
//
// It reports runtime.MemStats after each phase, computes per-note averages and
// a multiplier against the on-disk markdown byte total, and (with -second)
// builds a SECOND vault+index while the first is still referenced, which is
// exactly what RuntimeState.Reload does in production.
//
// Usage (from the go workspace root so go.work resolves the module):
//
//	go run ./publish-vault/ttmp/2026/08/09/PV-MEMORY-019--*/scripts/vaultmem \
//	    -vault /home/manuel/code/wesen/go-go-golems/go-go-parc \
//	    -second -memprofile /tmp/vault.heap
//
// This program does not modify application source; it only calls exported APIs.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/go-go-golems/publish-vault/pkg/search"
	"github.com/go-go-golems/publish-vault/pkg/vault"
)

type phase struct {
	Name      string
	Elapsed   time.Duration
	HeapAlloc uint64
	HeapInuse uint64
	HeapSys   uint64
	Sys       uint64
	NextGC    uint64
	NumGC     uint32
	MaxRSS    uint64
}

var phases []phase

func mib(b uint64) float64 { return float64(b) / (1024 * 1024) }

func record(name string, elapsed time.Duration) phase {
	// Force a GC so HeapAlloc reflects LIVE data, not garbage awaiting sweep.
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	p := phase{
		Name:      name,
		Elapsed:   elapsed,
		HeapAlloc: m.HeapAlloc,
		HeapInuse: m.HeapInuse,
		HeapSys:   m.HeapSys,
		Sys:       m.Sys,
		NextGC:    m.NextGC,
		NumGC:     m.NumGC,
		MaxRSS:    maxRSSBytes(),
	}
	phases = append(phases, p)
	fmt.Printf("%-28s elapsed=%-10s heapAlloc=%8.1f MiB  heapInuse=%8.1f MiB  heapSys=%8.1f MiB  sys=%8.1f MiB  nextGC=%8.1f MiB  numGC=%d  maxRSS=%8.1f MiB\n",
		name, elapsed.Round(time.Millisecond), mib(p.HeapAlloc), mib(p.HeapInuse),
		mib(p.HeapSys), mib(p.Sys), mib(p.NextGC), p.NumGC, mib(p.MaxRSS))
	return p
}

// maxRSSBytes reads VmHWM (peak resident set size) from /proc/self/status.
// This is the number a container memory limit is enforced against, so it is
// the number that matters for an OOMKill, not HeapAlloc.
func maxRSSBytes() uint64 {
	b, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "VmHWM:") {
			var kb uint64
			_, _ = fmt.Sscanf(strings.TrimPrefix(line, "VmHWM:"), "%d kB", &kb)
			return kb * 1024
		}
	}
	return 0
}

func currentRSSBytes() uint64 {
	b, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			var kb uint64
			_, _ = fmt.Sscanf(strings.TrimPrefix(line, "VmRSS:"), "%d kB", &kb)
			return kb * 1024
		}
	}
	return 0
}

// diskStats walks the vault the same way vault.LoadAll does (skipping dot
// directories) and totals markdown bytes so the report can express memory as a
// multiplier of on-disk source size.
func diskStats(root string) (mdFiles int, mdBytes int64, assetFiles int, assetBytes int64) {
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") && p != root {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			mdFiles++
			mdBytes += info.Size()
		} else {
			assetFiles++
			assetBytes += info.Size()
		}
		return nil
	})
	return
}

func main() {
	var (
		vaultPath  = flag.String("vault", "", "path to the Obsidian vault root (required)")
		second     = flag.Bool("second", false, "build a SECOND vault+index while holding the first (simulates RuntimeState.Reload overlap)")
		third      = flag.Bool("third", false, "also build a THIRD snapshot (simulates two overlapping reloads)")
		noSearch   = flag.Bool("no-search", false, "skip building the bleve search index")
		persistDir = flag.String("persist", "", "if set, build a PERSISTENT bleve index under this dir instead of an in-memory one (mirrors --search-index-path)")
		memProfile = flag.String("memprofile", "", "write a heap profile to this path at peak")
		memLimit   = flag.Int64("memlimit", 0, "if >0, call debug.SetMemoryLimit with this many bytes (simulates GOMEMLIMIT)")
	)
	flag.Parse()

	if *vaultPath == "" {
		fmt.Fprintln(os.Stderr, "ERROR: -vault is required")
		flag.Usage()
		os.Exit(2)
	}
	if _, err := os.Stat(*vaultPath); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: cannot stat vault %q: %v\n", *vaultPath, err)
		os.Exit(1)
	}
	if *memLimit > 0 {
		debug.SetMemoryLimit(*memLimit)
		fmt.Printf("debug.SetMemoryLimit(%d) => %.1f MiB\n", *memLimit, mib(uint64(*memLimit)))
	}

	fmt.Printf("GOMAXPROCS=%d GOGC=%q GOMEMLIMIT=%q\n", runtime.GOMAXPROCS(0), os.Getenv("GOGC"), os.Getenv("GOMEMLIMIT"))
	fmt.Printf("vault=%s\n\n", *vaultPath)

	dStart := time.Now()
	mdFiles, mdBytes, assetFiles, assetBytes := diskStats(*vaultPath)
	fmt.Printf("on-disk: mdFiles=%d mdBytes=%d (%.1f MiB)  assetFiles=%d assetBytes=%d (%.1f MiB)  walk=%s\n\n",
		mdFiles, mdBytes, float64(mdBytes)/(1024*1024),
		assetFiles, assetBytes, float64(assetBytes)/(1024*1024),
		time.Since(dStart).Round(time.Millisecond))

	base := record("00-baseline", 0)

	// ---- Snapshot 1 -------------------------------------------------------
	t0 := time.Now()
	v1, err := vault.New(*vaultPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: vault.New: %v\n", err)
		os.Exit(1)
	}
	afterVault1 := record("01-vault1-loaded", time.Since(t0))

	var s1 *search.Index
	afterSearch1 := afterVault1
	if !*noSearch {
		t1 := time.Now()
		s1, err = buildIndex(v1, *persistDir, "1")
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: build search index: %v\n", err)
			os.Exit(1)
		}
		afterSearch1 = record("02-search1-built", time.Since(t1))
	}

	// Per-note breakdown of the retained Note fields.
	notes := v1.AllNotes()
	var htmlBytes, sourceHTMLApprox, excerptBytes, titleBytes, pathBytes, slugBytes int64
	var wikiLinkCount, backlinkCount, tagCount int64
	for _, n := range notes {
		htmlBytes += int64(len(n.HTML))
		excerptBytes += int64(len(n.Excerpt))
		titleBytes += int64(len(n.Title))
		pathBytes += int64(len(n.Path))
		slugBytes += int64(len(n.Slug))
		wikiLinkCount += int64(len(n.WikiLinks))
		backlinkCount += int64(len(n.Backlinks))
		tagCount += int64(len(n.Tags))
	}
	// sourceHTML is unexported; it is the parser output HTML before vault-level
	// rewriting, so it is approximately the same magnitude as HTML.
	sourceHTMLApprox = htmlBytes

	fmt.Println()
	fmt.Println("== per-note retained field totals (snapshot 1) ==")
	fmt.Printf("notes                 = %d\n", len(notes))
	fmt.Printf("HTML bytes            = %d (%.1f MiB)\n", htmlBytes, float64(htmlBytes)/(1024*1024))
	fmt.Printf("sourceHTML bytes (~)  = %d (%.1f MiB)  [unexported, approximated as == HTML]\n", sourceHTMLApprox, float64(sourceHTMLApprox)/(1024*1024))
	fmt.Printf("Excerpt bytes         = %d (%.1f KiB)\n", excerptBytes, float64(excerptBytes)/1024)
	fmt.Printf("Title bytes           = %d\n", titleBytes)
	fmt.Printf("Path bytes            = %d\n", pathBytes)
	fmt.Printf("Slug bytes            = %d\n", slugBytes)
	fmt.Printf("WikiLink refs         = %d\n", wikiLinkCount)
	fmt.Printf("Backlink entries      = %d\n", backlinkCount)
	fmt.Printf("Tag entries           = %d\n", tagCount)
	if mdBytes > 0 {
		fmt.Printf("HTML / markdown ratio = %.2fx\n", float64(htmlBytes)/float64(mdBytes))
	}
	fmt.Println()

	// ---- Snapshot 2 (overlapping reload) ----------------------------------
	var v2 *vault.Vault
	var s2 *search.Index
	if *second {
		t2 := time.Now()
		v2, err = vault.New(*vaultPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: vault.New (2): %v\n", err)
			os.Exit(1)
		}
		record("03-vault2-loaded", time.Since(t2))
		if !*noSearch {
			t3 := time.Now()
			s2, err = buildIndex(v2, *persistDir, "2")
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: build search index (2): %v\n", err)
				os.Exit(1)
			}
			record("04-search2-built", time.Since(t3))
		}
	}

	// ---- Snapshot 3 (two overlapping reloads) -----------------------------
	var v3 *vault.Vault
	var s3 *search.Index
	if *third {
		t4 := time.Now()
		v3, err = vault.New(*vaultPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: vault.New (3): %v\n", err)
			os.Exit(1)
		}
		record("05-vault3-loaded", time.Since(t4))
		if !*noSearch {
			t5 := time.Now()
			s3, err = buildIndex(v3, *persistDir, "3")
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: build search index (3): %v\n", err)
				os.Exit(1)
			}
			record("06-search3-built", time.Since(t5))
		}
	}

	peak := phases[len(phases)-1]

	if *memProfile != "" {
		f, err := os.Create(*memProfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: create memprofile: %v\n", err)
		} else {
			runtime.GC()
			if err := pprof.WriteHeapProfile(f); err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: write heap profile: %v\n", err)
			} else {
				fmt.Printf("\nheap profile written to %s\n", *memProfile)
			}
			_ = f.Close()
		}
	}

	// ---- Summary ----------------------------------------------------------
	fmt.Println()
	fmt.Println("== summary ==")
	vaultOnly := int64(afterVault1.HeapAlloc) - int64(base.HeapAlloc)
	searchOnly := int64(afterSearch1.HeapAlloc) - int64(afterVault1.HeapAlloc)
	oneSnapshot := int64(afterSearch1.HeapAlloc) - int64(base.HeapAlloc)
	fmt.Printf("vault only (live heap delta)   = %.1f MiB\n", mib(uint64(max64(vaultOnly, 0))))
	fmt.Printf("search only (live heap delta)  = %.1f MiB\n", mib(uint64(max64(searchOnly, 0))))
	fmt.Printf("one snapshot (vault+search)    = %.1f MiB\n", mib(uint64(max64(oneSnapshot, 0))))
	if len(notes) > 0 {
		fmt.Printf("bytes per note (one snapshot)  = %.0f\n", float64(oneSnapshot)/float64(len(notes)))
	}
	if mdBytes > 0 {
		fmt.Printf("multiplier vs on-disk markdown = %.2fx\n", float64(oneSnapshot)/float64(mdBytes))
	}
	fmt.Printf("peak phase                     = %s\n", peak.Name)
	fmt.Printf("peak heapAlloc (live)          = %.1f MiB\n", mib(peak.HeapAlloc))
	fmt.Printf("peak heapSys                   = %.1f MiB\n", mib(peak.HeapSys))
	fmt.Printf("peak runtime Sys               = %.1f MiB\n", mib(peak.Sys))
	fmt.Printf("peak RSS (VmHWM)               = %.1f MiB\n", mib(peak.MaxRSS))
	fmt.Printf("current RSS (VmRSS)            = %.1f MiB\n", mib(currentRSSBytes()))
	fmt.Println()
	fmt.Println("NOTE: VmHWM is the number a Kubernetes memory limit is enforced against.")
	fmt.Println("      A 1536 MiB limit is breached when peak RSS exceeds it, regardless of live heap.")

	// Keep everything alive to the very end so the GC cannot reclaim snapshots
	// early and understate the overlap cost.
	runtime.KeepAlive(v1)
	runtime.KeepAlive(s1)
	runtime.KeepAlive(v2)
	runtime.KeepAlive(s2)
	runtime.KeepAlive(v3)
	runtime.KeepAlive(s3)
}

// buildIndex builds either an in-memory bleve index (persistDir == "", the
// production default because --search-index-path defaults to empty) or a
// persistent on-disk index (mirrors server.buildSearchIndex when
// --search-index-path is set: build to a directory, close, reopen).
func buildIndex(v *vault.Vault, persistDir, tag string) (*search.Index, error) {
	if persistDir == "" {
		return search.New(v)
	}
	dir := filepath.Join(persistDir, "snap"+tag, "index")
	si, err := search.NewPersistent(v, dir)
	if err != nil {
		return nil, err
	}
	// Production closes the build handle and reopens the finished index, which
	// is what determines steady-state resident memory.
	if err := si.Close(); err != nil {
		return nil, err
	}
	return search.OpenPersistent(dir)
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
