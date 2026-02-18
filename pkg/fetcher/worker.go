package fetcher

import (
	"context"
	//"fmt"
	"log"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"
)

type job struct {
	addrs []string
}

type result struct {
	wallet string
	wei    string
}

// simple rate limiter using time.Ticker channels shared across workers
type RateLimiter struct {
	ticker *time.Ticker
	mu     sync.Mutex
}

func NewRateLimiter(rps float64) *RateLimiter {
	interval := time.Duration(math.Max(1.0/rps*1000.0, 1.0)) * time.Millisecond
	return &RateLimiter{ticker: time.NewTicker(interval)}
}

func (r *RateLimiter) Wait() {
	<-r.ticker.C
}

func (r *RateLimiter) Stop() {
	r.ticker.Stop()
}

// Exponential backoff with jitter
func backoffSleep(attempt int) {
	base := math.Min(60.0, math.Pow(2, float64(attempt))*1.0)
	jitter := rand.Float64() * 0.5 * float64(attempt+1)
	d := time.Duration((base + jitter) * float64(time.Second))
	time.Sleep(d)
}

func workerLoop(ctx context.Context, id int, c *EtherscanClient, rl *RateLimiter, jobs <-chan job, results chan<- result, wg *sync.WaitGroup) {
	defer wg.Done()
	attemptLimit := 6
	for j := range jobs {
		// Build unique, lowercase addresses; skip empties
		batch := make([]string, 0, len(j.addrs))
		seen := map[string]bool{}
		for _, a := range j.addrs {
			a = strings.TrimSpace(strings.ToLower(a))
			if a == "" {
				continue
			}
			if seen[a] {
				continue
			}
			seen[a] = true
			batch = append(batch, a)
		}
		if len(batch) == 0 {
			continue
		}
		// Retry loop
		var res map[string]string
		var err error
		for attempt := 0; attempt < attemptLimit; attempt++ {
			// rate limit wait
			rl.Wait()
			// fetch
			res, err = c.FetchBalances(ctx, batch)
			if err == nil {
				break
			}
			// log and backoff then retry
			log.Printf("[worker %d] fetch error (attempt %d): %v", id, attempt+1, err)
			backoffSleep(attempt)
		}
		if err != nil {
			// on permanent failure, set zeros
			log.Printf("[worker %d] permanent failure for batch: %v. marking balances zero", id, err)
			for _, a := range batch {
				results <- result{wallet: a, wei: "0"}
			}
			continue
		}
		// push results in deterministic order
		for _, a := range batch {
			w := strings.ToLower(a)
			wei := res[w]
			if wei == "" {
				wei = "0"
			}
			results <- result{wallet: w, wei: wei}
		}
	}
}
