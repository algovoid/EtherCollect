# EtherCollect (Go)

Lightweight CLI tool to fetch Ethereum ETH balances for large lists of wallet addresses using the Etherscan API (balancemulti), built in Go. Designed for large inputs, respects free-tier rate limits, supports multiple API key rotation, and persists progress in a SQLite checkpoint for resumability.

Works on **Linux, macOS, and Windows**. (tested on linux and windows only)

## Features

- Batch `balancemulti` requests (default 20 addresses per call)
- Respectable rate limiting (default 2 requests/sec)
- Exponential backoff with jitter for transient errors and rate limits
- Rotates across multiple Etherscan API keys to increase throughput
- SQLite checkpoint to persist processed addresses and balances
- Output to CSV or NDJSON (newline-delimited JSON)
- Streaming (low memory footprint), suitable for millions of addresses

## Limitations

- Free Etherscan tier may still throttle/bucket; rotating keys helps but does not guarantee unbounded throughput.
- `balancemulti` historically supports up to 20 addresses per request, keep batch size ≤ 20.
- For ERC-20 token balances, a different approach is required (not supported here).
- SQLite checkpoint adds disk I/O.

## Build & Requirements

- Go 1.20+
- CGO enabled for `github.com/mattn/go-sqlite3` (default on most Linux/macOS setups).

```bash
git clone github.com/algovoid/EtherCollect.git
cd EtherCollect
go build -o EtherCollect ./cmd/EtherCollect
```

### Build instructions

Clone the repository and build the binary:

```bash
git clone https://github.com/algovoid/EtherCollect.git
cd EtherCollect
```

### Linux / macOS:

```bash
go build -o EtherCollect ./cmd/EtherCollect
```

### Windows (with MinGW-w64):

```powershell
set CGO_ENABLED=1
go build -o EtherCollect.exe ./cmd/EtherCollect
```

After build, you'll have an executable (EtherCollect or EtherCollect.exe).

## Usage

### Basic command structure

```powershell
./EtherCollect \
  --input addresses.txt \
  --output balances.csv \
  --format csv \
  --api-keys "key1,key2" \
  --batch-size 20 \
  --rate 2.0 \
  --resume
```

## On Windows, use backslashes and .exe:

```powershell
EtherCollect.exe ^
  --input addresses.txt ^
  --output balances.csv ^
  --format csv ^
  --api-keys "key1,key2" ^
  --batch-size 20 ^
  --rate 2.0 ^
  --resume
```

## Flags

| Flag             | Description                                                            | Default       |
| ---------------- | ---------------------------------------------------------------------- | ------------- |
| --input          | Input file (one address per line) or - for stdin                       | required      |
| --output         | Output file path (CSV or NDJSON)                                       | required      |
| --format         | Output format: csv or json (ndjson)                                    | csv           |
| --api-keys       | Comma-separated Etherscan API keys (or set ETHERSCAN_API_KEYS env var) | required      |
| --batch-size     | Addresses per balancemulti request (≤20)                               | 20            |
| --rate           | Requests per second (across all workers/keys)                          | 2.0           |
| --concurrency    | Number of concurrent workers                                           | 1             |
| --resume         | Use SQLite checkpoint to resume and skip already-processed addresses   | false         |
| --checkpoint     | Path to SQLite checkpoint file                                         | checkpoint.db |
| --progress-total | Expected total addresses for progress output                           | (optional)    |

## Scripts for Easier Use

For convenience, we provide wrapper scripts that set sensible defaults and ask for missing parameters interactively.

### Bash script (Linux/macOS)

Save as `fetch_balances.sh` and make executable (`chmod +x fetch_balances.sh`):

```bash
#!/bin/bash

# default values
BATCH_SIZE=20
RATE=2.0
FORMAT="csv"
RESUME="--resume"

# prompt for required inputs if not provided via arguments
if [ -z "$1" ]; then
    read -p "Input file (addresses.txt): " INPUT
    INPUT=${INPUT:-addresses.txt}
else
    INPUT=$1
fi

if [ -z "$2" ]; then
    read -p "Output file (balances.csv): " OUTPUT
    OUTPUT=${OUTPUT:-balances.csv}
else
    OUTPUT=$2
fi

if [ -z "$3" ]; then
    read -p "Etherscan API keys (comma-separated): " API_KEYS
else
    API_KEYS=$3
fi

# run the tool
./EtherCollect \
    --input "$INPUT" \
    --output "$OUTPUT" \
    --format "$FORMAT" \
    --api-keys "$API_KEYS" \
    --batch-size "$BATCH_SIZE" \
    --rate "$RATE" \
    $RESUME

```

### Usage:

```bash
./fetch_balances.sh [input_file] [output_file] [api_keys]
```

#### If arguments are omitted, the script prompts interactively.

### Batch script (Windows)

Save as `fetch_balances.bat:`

```powershell
@echo off

set BATCH_SIZE=20
set RATE=2.0
set FORMAT=csv
set RESUME=--resume

if "%1"=="" (
    set /p INPUT="Input file (addresses.txt): "
    if "!INPUT!"=="" set INPUT=addresses.txt
) else (
    set INPUT=%1
)

if "%2"=="" (
    set /p OUTPUT="Output file (balances.csv): "
    if "!OUTPUT!"=="" set OUTPUT=balances.csv
) else (
    set OUTPUT=%2
)

if "%3"=="" (
    set /p API_KEYS="Etherscan API keys (comma-separated): "
) else (
    set API_KEYS=%3
)

EtherCollect.exe ^
    --input "%INPUT%" ^
    --output "%OUTPUT%" ^
    --format %FORMAT% ^
    --api-keys "%API_KEYS%" ^
    --batch-size %BATCH_SIZE% ^
    --rate %RATE% ^
    %RESUME%
```

### Usage:

Open Command Prompt, navigate to the folder, and run:

`fetch_balances.bat [input_file] [output_file] [api_keys]`

#### If arguments are omitted, you'll be prompted.

#

# Advanced: Direct CLI Usage

Experienced users can call the binary directly with all flags as shown in the basic command structure. This gives full control over batching, rate limiting, concurrency, and checkpointing.

Example with environment variable for API keys:

```bash
export ETHERSCAN_API_KEYS="key1,key2"
./EtherCollect --input huge_list.txt --output results.json --format json --rate 5 --concurrency 2
```

### On Windows (PowerShell):

```powershell
$env:ETHERSCAN_API_KEYS="key1,key2"
.\EtherCollect.exe --input huge_list.txt --output results.json --format json --rate 5 --concurrency 2
```

## How it works (short)

1. Reads input addresses streaming from file/stdin.
2. Deduplicates on-the-fly (optional).
3. Batches addresses and sends `balancemulti` requests, rotating API keys each request and respecting global rate limit.
4. Writes results immediately to output file and to SQLite checkpoint for resumability.
5. Retries transient errors with exponential backoff and jitter.

## Notes & Next steps

- If you want token balances, or historical balances at a particular block, or to run faster for millions of addresses, consider a paid Etherscan plan or running an archive node / indexer.
- You can run multiple instances each with different API keys and a shared checkpoint DB for parallelism (careful with DB locking).
