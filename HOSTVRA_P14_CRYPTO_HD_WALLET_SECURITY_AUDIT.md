# HOSTVRA — P14.5 IN-HOUSE CRYPTO GATEWAY & HD WALLET SECURITY AUDIT

**Audit Date & Time:** 2026-11-20T18:16:30+06:00

---

## 1. Blockchain Network & Asset Inventory (`apps/api/internal/store/crypto_store.go`)

- **Why live `/api/v1/public/status` reported `7 active blockchain networks monitored`**:
  - The live production `/var/lib/hostvra/store.json` was initialized when 7 networks (`tron`, `bsc`, `ethereum`, `polygon`, `bitcoin`, `litecoin`, `dogecoin`) were seeded; `seedCryptoData()` in `crypto_store.go` now defines 8 networks (`tron`, `bsc`, `ethereum`, `polygon`, `bitcoin`, `solana`, `litecoin`, `dogecoin`) and 11 currencies (`USDT_TRC20`, `USDT_BEP20`, `USDT_ERC20`, `BNB`, `ETH`, `POL`, `TRX`, `BTC`, `SOL`, `LTC`, `DOGE`).
- **Deterministic Derivation Standards Verified (`TestHDWallet_DeterministicMultiChainDerivationAndWatchOnlyXPub`)**:
  - `ethereum`, `bsc`, `polygon`: BIP-44 `m/44'/60'/0'/0/index` (secp256k1 Keccak-256 EIP-55)
  - `tron`: BIP-44 `m/44'/195'/0'/0/index` (Base58Check `T...`)
  - `bitcoin`: BIP-84 `m/84'/0'/0'/0/index` (Native SegWit Bech32 `bc1q...`)
  - `litecoin`: BIP-84 `m/84'/2'/0'/0/index` (Native SegWit Bech32 `ltc1q...`)
  - `dogecoin`: BIP-44 `m/44'/3'/0'/0/index` (Base58Check `D...`)
  - `solana`: SLIP-0010 Ed25519 `m/44'/501'/index'/0'`
- **Index Isolation**: Index `0` is reserved for the merchant's primary Trust Wallet address; customer deposit addresses start at `NextDerivationIndex >= 1` (`ReserveNextDerivationIndex`).
- **Vault & Signer Security (`TestIsolatedSigner_PolicyEnforcementAndNoSecretLeakage`)**:
  - Master mnemonic is encrypted at rest using AES-256-GCM; never returned by any API endpoint and never logged.
  - TRON TRC20 sweeps construct `TriggerSmartContract` (`a9059cbb` ABI-encoded `transfer(address,uint256)`), verified by `TestTronTRC20_ABIEncoding`.
- **P14 Persistence Fix (`P14-01`)**:
  - Added `m.saveToDiskLocked()` to `UpdateHDWalletAddressStatus`, `CreateTreasurySweepJob`, `UpdateTreasurySweepJob`, `RecordTreasurySweepLog`, and `RecordWalletDerivationAudit` in `apps/api/internal/store/hd_wallet_store.go`, verified across cold disk reload in `TestP14_ConcurrentPayInvoiceWithWalletAndPlanScopedCouponsAndSweepPersistence`.
