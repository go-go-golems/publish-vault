// Command 02-vault-load measures the structural parser's real vault load.
// It reads the vault but does not write to it or print note contents.
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/go-go-golems/publish-vault/pkg/vault"
)

func main() {
	root := flag.String("vault", "", "vault directory")
	flag.Parse()
	if *root == "" {
		fmt.Fprintln(os.Stderr, "-vault is required")
		os.Exit(2)
	}
	started := time.Now()
	v, err := vault.New(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	runtime.GC()
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	fmt.Printf("notes=%d elapsed=%s live_heap_mib=%.1f\n", v.Count(), time.Since(started).Round(time.Millisecond), float64(stats.HeapAlloc)/(1024*1024))
	runtime.KeepAlive(v)
}
