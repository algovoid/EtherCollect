# EtherCollect
CLI tool to get wallet funding in JSON/CSV from supplied wallet list
# EtherCollect (Go)

Lightweight CLI tool to fetch Ethereum ETH balances for large lists of wallet addresses using the Etherscan API (balancemulti), built in Go. Designed for large inputs, respects free-tier rate limits, supports multiple API key rotation, and persists progress in a SQLite checkpoint for resumability.
