package fetcher

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	//"time"
)

// Run is the main entry for the fetcher package. It wires components and runs the pipeline.
func Run(cfg Config) error {
	// Basic validation
	if cfg.BatchSize <= 0 || cfg.BatchSize > 20 {
		return fmt.Errorf("batch-size must be between 1 and 20")
	}
	if cfg.RateRPS <= 0 {
		return fmt.Errorf("rate must be > 0")
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 1
	}

	// Open checkpoint
	cp, err := NewCheckpoint(cfg.CheckpointDB)
	if err != nil {
		return fmt.Errorf("checkpoint init: %w", err)
	}
	defer cp.Close()

	// Setup Etherscan client
	client := NewEtherscanClient(cfg.APIKeys, "1") // Default to chainID "1" (Ethereum mainnet) for now ; can be made configurable if needed
	rl := NewRateLimiter(cfg.RateRPS)
	defer rl.Stop()

	// Channels
	jobs := make(chan job, cfg.Concurrency*2)
	results := make(chan result, cfg.Concurrency*cfg.BatchSize*2)
	done := make(chan struct{})

	// Start writer goroutine
	go writerLoop(cfg, results, done, cp)

	// Start workers
	var wg sync.WaitGroup
	ctx := context.Background()
	for i := 0; i < cfg.Concurrency; i++ {
		wg.Add(1)
		go workerLoop(ctx, i, client, rl, jobs, results, &wg)
	}

	// Read input and feed jobs
	var inFile *os.File
	var scanner *bufio.Scanner
	if cfg.InputPath == "-" {
		inFile = os.Stdin
	} else {
		f, err := os.Open(cfg.InputPath)
		if err != nil {
			return fmt.Errorf("open input: %w", err)
		}
		inFile = f
		defer inFile.Close()
	}
	scanner = bufio.NewScanner(inFile)
	// increase buffer for long lines if needed
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	// Optionally skip addresses already in checkpoint when resume=true by reading checkpoint keys into a set
	skipped := 0
	existing := map[string]bool{}
	if cfg.Resume {
		// naive approach: scan checkpoint table to mark existing processed wallets
		rows, err := cp.db.Query("SELECT wallet FROM processed")
		if err == nil {
			for rows.Next() {
				var w string
				if err := rows.Scan(&w); err == nil {
					existing[w] = true
				}
			}
			rows.Close()
		}
	}

	batch := make([]string, 0, cfg.BatchSize)
	total := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		w := strings.ToLower(line)
		if cfg.Resume {
			if existing[w] {
				skipped++
				continue
			}
		}
		batch = append(batch, w)
		if len(batch) >= cfg.BatchSize {
			jobs <- job{addrs: append([]string(nil), batch...)}
			batch = batch[:0]
		}
		total++
	}
	// send remaining batch
	if len(batch) > 0 {
		jobs <- job{addrs: append([]string(nil), batch...)}
	}
	close(jobs)
	// wait workers
	wg.Wait()
	// close results and wait writer to finish
	close(results)
	<-done
	fmt.Printf("Processed ~%d addresses (skipped %d due to resume)\n", total, skipped)
	return nil
}
