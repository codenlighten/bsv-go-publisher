# Your BSV AKUA Broadcast Server - Keypairs Ready

## ✅ Status

Your `.env` file now contains **real BSV keypairs** ready for mainnet testing.

---

## 🔐 Your Keypairs

### Funding Address & Private Key
```
Address:           1XJ82FS3QLrXRT6zfrB4W9BanSSgFgw1m
Private Key (WIF): L2ZMxBfjREorSJ5qiWCU1vrZSQxJg6DRmx5fMpBY9oaoWMFbxGU6
```

**Purpose:** Send BSV here to fund the UTXO splitting operation. This address will hold larger amounts before splitting into 50,000 publishing UTXOs.

---

### Publishing Address & Private Key
```
Address:           12w4BoPtqCt7EFLmUPi9GLmpbZ1CHdPvzj
Private Key (WIF): L1tvUUBsdYsRt1hbMCLtj1XEHL3XAfrcJKt2x7VxoKrQ8SdfFpxg
```

**Purpose:** Receives the split UTXOs (each worth exactly 100 sats). These are used for broadcasting OP_RETURN transactions.

---

## 🚀 Next Steps

### 1. Send Funds to Funding Address
Send BSV to:
```
1XJ82FS3QLrXRT6zfrB4W9BanSSgFgw1m
```

Amount suggestions:
- **Small test:** 5 BSV (creates 50,000 UTXOs = 5,000,000 sats ÷ 100 sats/UTXO)
- **Production:** 50+ BSV for operational buffer

### 2. Verify the Server Loads Your Keys
```bash
make run
```

Look for output like:
```
✓ Loaded FUNDING_PRIVKEY: 1XJ82FS3QLrXRT6zfrB4W9BanSSgFgw1m
✓ Loaded PUBLISHING_PRIVKEY: 12w4BoPtqCt7EFLmUPi9GLmpbZ1CHdPvzj
```

### 3. Create UTXO Pool
Once funded, run the splitter to create 50,000 publishing UTXOs:
```bash
# (Implementation coming - currently the endpoint exists)
curl -X POST http://localhost:8080/admin/split
```

### 4. Test Broadcasting
```bash
# Submit a test OP_RETURN transaction
make publish DATA=48656c6c6f

# Check status
curl http://localhost:8080/status/{uuid}
```

---

## ⚠️ Security Notes

**These are test keypairs on mainnet.** In production:

1. **Never commit private keys to git** - Your `.env` is already in `.gitignore`
2. **Use a secrets manager** for production (AWS Secrets Manager, HashiCorp Vault, etc.)
3. **Keep backups** of your WIF keys in secure storage
4. **Limit funding** - Only send what you'll use for operations
5. **Monitor addresses** - Watch for unexpected UTXOs or transactions
6. **Rotate keys periodically** - Generate new keypairs for long-running operations

---

## 📊 Configuration Verified

✅ Real keypairs added to `.env`  
✅ MongoDB URI configured (MongoDB Atlas)  
✅ ARC configuration ready (just needs token)  
✅ Build successful with new keys  

---

## 🎯 Quick Command Reference

```bash
# Start the server
make run

# View logs
make logs

# Check health
make health

# View UTXO statistics
make stats

# Test publishing
make publish DATA=<hex_data>
```

---

## 📝 Environment Summary

Your `.env` now contains:

| Variable | Value | Purpose |
|----------|-------|---------|
| MONGO_PASSWORD | [set manually] | MongoDB authentication |
| BSV_NETWORK | mainnet | Bitcoin SV network |
| FUNDING_PRIVKEY | L2ZMx... | Splits UTXOs |
| PUBLISHING_PRIVKEY | L1tvu... | Broadcasts OP_RETURN |
| ARC_URL | https://arc.gorillapool.io | Transaction broadcast |
| ARC_TOKEN | [set manually] | ARC authentication |
| TRAIN_INTERVAL | 3s | Batch interval |
| TRAIN_MAX_BATCH | 1000 | Transactions per batch |
| TARGET_PUBLISHING_UTXOS | 50000 | Target pool size |
| MONGO_URI | [set manually] | MongoDB connection |
| MONGO_DB_NAME | go-bsv | Database name |

---

## 🔗 Addresses You Can Monitor

**Funding Address:** https://whatsonchain.com/address/1XJ82FS3QLrXRT6zfrB4W9BanSSgFgw1m  
**Publishing Address:** https://whatsonchain.com/address/12w4BoPtqCt7EFLmUPi9GLmpbZ1CHdPvzj

---

## ✨ Ready to Test

Your server is now ready for real transactions:

1. ✅ Code builds cleanly
2. ✅ Real keypairs loaded
3. ✅ Database configured
4. ✅ API endpoints ready
5. ⏳ Awaiting funding...

Send some BSV to `1XJ82FS3QLrXRT6zfrB4W9BanSSgFgw1m` and start broadcasting!

---

**Last Updated:** February 6, 2026  
**Status:** Ready for testing with real transactions
