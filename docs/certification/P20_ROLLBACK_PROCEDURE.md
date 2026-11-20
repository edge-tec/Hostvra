# HOSTVRA — P20.10: ROLLBACK PROCEDURE

**Date / UTC Timestamp:** `2026-11-20T15:00:48Z`  
**Environment:** Repository `update.sh` & `install.sh`  
**Status:** **PASS (SCRIPT & ISOLATED RECOVERY VERIFIED)**

## Verified Rollback Mechanism (`update.sh`)
1. Creates `/var/backups/hostvra/pre-update-YYYYMMDD-HHMMSS/` **before** any Git pull or binary replacement.
2. Preserves `/etc/hostvra/hostvra.env` (`0600`), `/var/lib/hostvra/store.json`, `/var/lib/hostvra/vault/hd_master.enc` (`0600`), `web-next.tar.gz`, binaries, and DB dumps.
3. Retains the latest 5 valid backups automatically.
4. `trap 'rollback' ERR` restores previous Git commit, binaries, Next.js build, and `store.json` if any deployment step or health check fails.
