# Utility Tools

These are standalone utility scripts for managing UTXOs and testing transactions. They use the `// +build ignore` directive so they won't interfere with the main server build.

## Available Tools

### analyze-utxos.go
Analyzes UTXO distribution for the publishing address.

```bash
go run analyze-utxos.go
```

### consolidate-utxos.go
Consolidates multiple small UTXOs into a single larger UTXO.

```bash
go run consolidate-utxos.go
```

### send-to-funding.go
Sends funds from publishing address back to funding address (uses internal BSV package).

```bash
go run send-to-funding.go
```

### send-to-funding-2.go
Alternative method to send funds (standalone, no internal dependencies).

```bash
go run send-to-funding-2.go
```

## Notes

- These scripts are not compiled with the main server
- They output transaction hex that can be broadcast manually
- Use with caution as they work with real mainnet funds
- Always verify transaction details before broadcasting
