package fetcher

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type Checkpoint struct {
	db *sql.DB
}

func NewCheckpoint(path string) (*Checkpoint, error) {
	// ensure directory exists
	if dir := filepath.Dir(path); dir != "." {
		os.MkdirAll(dir, 0o755)
	}
	db, err := sql.Open("sqlite3", path+"?_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}
	cp := &Checkpoint{db: db}
	if err := cp.init(); err != nil {
		return nil, err
	}
	return cp, nil
}

func (c *Checkpoint) init() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS processed (wallet TEXT PRIMARY KEY, balance_wei TEXT, updated_at INTEGER);`,
		`CREATE INDEX IF NOT EXISTS idx_processed_wallet ON processed(wallet);`,
	}
	for _, s := range stmts {
		if _, err := c.db.Exec(s); err != nil {
			return fmt.Errorf("init stmt: %w", err)
		}
	}
	return nil
}

func (c *Checkpoint) Close() error {
	return c.db.Close()
}

func (c *Checkpoint) Get(wallet string) (string, bool, error) {
	var bal string
	err := c.db.QueryRow("SELECT balance_wei FROM processed WHERE wallet = ?", wallet).Scan(&bal)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return bal, true, nil
}

func (c *Checkpoint) Put(wallet string, balanceWei string, ts int64) error {
	_, err := c.db.Exec("INSERT OR REPLACE INTO processed (wallet, balance_wei, updated_at) VALUES (?, ?, ?)", wallet, balanceWei, ts)
	return err
}
