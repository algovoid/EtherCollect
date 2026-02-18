package fetcher

import (
	//"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"
)

func writeHeaderCSV(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	return w.Write([]string{"wallet", "balance_wei", "balance_eth", "updated_at_unix"})
}

func appendCSV(path string, rec []string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if err := w.Write(rec); err != nil {
		return err
	}
	w.Flush()
	return w.Error()
}

func appendNDJSON(path string, obj interface{}) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	return enc.Encode(obj)
}

func weiToEthString(weiStr string) string {
	// simple integer division display using float64 for readable output; for precision-sensitive use big.Int
	var wei float64 = 0
	if v, err := strconv.ParseFloat(weiStr, 64); err == nil {
		wei = v
	}
	eth := wei / 1e18
	return fmt.Sprintf("%.18f", eth)
}

func writerLoop(cfg Config, results <-chan result, done chan<- struct{}, cp *Checkpoint) {
	defer close(done)
	// Prepare output header if not resume
	if !cfg.Resume {
		if cfg.Format == "csv" {
			if err := writeHeaderCSV(cfg.OutputPath); err != nil {
				panic(err)
			}
		}
	}
	for r := range results {
		ts := time.Now().Unix()
		if err := cp.Put(r.wallet, r.wei, ts); err != nil {
			// log but continue
			fmt.Fprintf(os.Stderr, "checkpoint write error: %v\n", err)
		}
		if cfg.Format == "csv" {
			rec := []string{r.wallet, r.wei, weiToEthString(r.wei), strconv.FormatInt(ts, 10)}
			if err := appendCSV(cfg.OutputPath, rec); err != nil {
				fmt.Fprintf(os.Stderr, "write error: %v\n", err)
			}
		} else {
			obj := map[string]interface{}{
				"wallet":          r.wallet,
				"balance_wei":     r.wei,
				"balance_eth":     weiToEthString(r.wei),
				"updated_at_unix": ts,
			}
			if err := appendNDJSON(cfg.OutputPath, obj); err != nil {
				fmt.Fprintf(os.Stderr, "write error: %v\n", err)
			}
		}
	}
}
