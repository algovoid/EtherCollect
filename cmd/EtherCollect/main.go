package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/algovoid/EtherCollect/pkg/fetcher"
)

func main() {
	var (
		inputPath     = flag.String("input", "", "Input file path (one address per line) or '-' for stdin (required)")
		outputPath    = flag.String("output", "", "Output file path (CSV or NDJSON) (required)")
		format        = flag.String("format", "csv", "Output format: csv or json (ndjson)")
		apiKeysFlag   = flag.String("api-keys", "", "Comma-separated Etherscan API keys (or set ETHERSCAN_API_KEYS env var)")
		batchSize     = flag.Int("batch-size", 20, "Addresses per balancemulti request (max 20)")
		rate          = flag.Float64("rate", 2.0, "Global requests per second (default 2.0)")
		concurrency   = flag.Int("concurrency", 1, "Number of worker goroutines")
		resume        = flag.Bool("resume", false, "Use sqlite checkpoint to resume and skip already processed addresses")
		checkpoint    = flag.String("checkpoint", "checkpoint.db", "SQLite checkpoint file path")
		progressTotal = flag.Int("progress-total", 0, "Optional: total addresses for progress display")
	)

	flag.Parse()

	if *inputPath == "" || *outputPath == "" {
		flag.Usage()
		os.Exit(2)
	}

	// Load API keys from flag or env var
	apiKeys := []string{}
	if *apiKeysFlag != "" {
		apiKeys = strings.Split(*apiKeysFlag, ",")
	} else if v := os.Getenv("ETHERSCAN_API_KEYS"); v != "" {
		apiKeys = strings.Split(v, ",")
	}
	for i := range apiKeys {
		apiKeys[i] = strings.TrimSpace(apiKeys[i])
	}
	if len(apiKeys) == 0 {
		log.Fatal("No Etherscan API key provided. Use --api-keys or set ETHERSCAN_API_KEYS")
	}

	cfg := fetcher.Config{
		InputPath:     *inputPath,
		OutputPath:    *outputPath,
		Format:        strings.ToLower(*format),
		APIKeys:       apiKeys,
		BatchSize:     *batchSize,
		RateRPS:       *rate,
		Concurrency:   *concurrency,
		Resume:        *resume,
		CheckpointDB:  *checkpoint,
		ProgressTotal: *progressTotal,
	}

	start := time.Now()
	if err := fetcher.Run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Done in %s\n", time.Since(start).Round(time.Second))
}
