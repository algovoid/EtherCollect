package fetcher

type Config struct {
	InputPath    string
	OutputPath   string
	Format       string // "csv" or "json"
	APIKeys      []string
	BatchSize    int
	RateRPS      float64
	Concurrency  int
	Resume       bool
	CheckpointDB string
	ProgressTotal int
}
