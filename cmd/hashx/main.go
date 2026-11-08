package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/haunterhx/hashx/internal/ioc"
	"github.com/haunterhx/hashx/internal/scan"
)

var buildType = "dev"
var buildVersion = "1.0"

const (
	colorReset   = "\033[0m"
	colorRedBold = "\033[1;31m"
	colorYellow  = "\033[33m"
)

func main() {
	PrintBanner(os.Stderr)
	if buildType == "dev" {
		fmt.Printf("\n\tRunning %v Build %v\n", buildType, buildVersion)
	}

	// flags
	root := flag.String("path", ".", "directory to scan")
	iocFile := flag.String("iocs", "", "file of SHA256 IOCs, one per line")
	entThresh := flag.Float64("entropy", 7.2, "flag files at or above this entropy")
	workers := flag.Int("workers", runtime.NumCPU(), "concurrent workers")
	maxMB := flag.Int64("maxmb", 512, "skip files larger than this MB size")
	asJSON := flag.Bool("json", false, "emit JSON lines instead of a table")
	showVer := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVer {
		fmt.Printf("hashx %s (%s/%s, %s)\n",
			buildVersion, runtime.GOOS, runtime.GOARCH, runtime.Version())
		return
	}

	iocs, err := ioc.LoadIOCs(*iocFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading IOCs: %v\n", err)
		os.Exit(1)
	}

	start := time.Now()
	paths := make(chan string, 256)
	results := make(chan scan.Result, 256)

	// Worker Pool
	var wg sync.WaitGroup
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range paths {
				r := scan.Analyze(p, *maxMB*1024*1024)
				if r.Err == "" {
					_, r.IOCHit = iocs[r.SHA256]
					r.HighEnt = r.Entropy >= *entThresh
				}
				results <- r
			}
		}()
	}

	// Producer: walk the dir tree, regular files only
	go func() {
		filepath.WalkDir(*root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.Type().IsRegular() {
				paths <- p
			}
			return nil
		})
		close(paths)
	}()

	go func() { wg.Wait(); close(results) }()

	var all []scan.Result
	var hits, high, errs int

	enc := json.NewEncoder(os.Stdout)

	for r := range results {
		if *asJSON {
			enc.Encode(r)
		}
		all = append(all, r)

		switch {
		case r.Err != "":
			errs++
		case r.IOCHit:
			hits++
		}

		if r.HighEnt {
			high++
		}
	}

	if !*asJSON {
		sort.Slice(all, func(i, j int) bool { return all[i].Entropy > all[j].Entropy })

		fmt.Printf("%-6s %-8s %-10s %-64s %s\n", "FLAG", "ENTROPY", "SIZE", "SHA256", "PATH")

		for _, r := range all {
			if !r.IOCHit && !r.HighEnt {
				continue
			}
			if r.IOCHit {
				fmt.Println(colorize(colorRedBold, fmt.Sprintf("%-6s %-8.3f %-10d %64s %s",
					"IOC!", r.Entropy, r.Size, r.SHA256, r.Path)))
			} else {
				fmt.Printf("%s %-8.3f %-10d %-64s %s\n",
					colorize(colorYellow, fmt.Sprintf("%-6s", "HIGH")), r.Entropy, r.Size, r.SHA256, r.Path)
			}
		}
	}

	fmt.Fprintf(os.Stderr, "\nscanned=%d ioc_hits=%d high_entropy=%d errors=%d elapsed=%s\n\n",
		len(all), hits, high, errs, time.Since(start).Round(time.Millisecond))

	if hits > 0 {
		os.Exit(2)
	}
}
