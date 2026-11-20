# HOSTVRA — P11 BACKUP AND ROLLBACK REPORT

**Execution Date & Time:** 2026-11-20T17:08:00+06:00  
**Deployment Script:** `update.sh` (Hardened in Phase P11)

---

## 1. Automated Pre-Deployment Backup Procedure (`update.sh`)

Before pulling new code or replacing any binary/frontend build on the VPS, `update.sh` creates a timestamped snapshot directory at:
```text
/var/lib/hostvra/updates_backup/<UNIX_TIMESTAMP>/
```
Containing:
1. **`pre_deploy_commit.txt`**: Exact Git commit hash running prior to deployment (`git rev-parse HEAD`).
2. **`service_states.txt`**: Pre-deployment `systemctl is-active` state of `hostvra-api`, `hostvra-web`, and `hostvra-agent`.
3. **`hostvra-api.bak` & `hostvra-agent.bak`**: Preserved copies of `/usr/local/bin/hostvra-api` and `/usr/local/bin/hostvra-agent`.
4. **`web_next_build.tar.gz`**: Full archive of the previous working `apps/web/.next` production frontend build.
5. **`api.env.bak`, `store.json.bak`, `crypto_treasuries.json.bak`**: Preserved configuration and persistent state snapshots.
6. **`hostvra_db.sql` & `mariadb_all.sql`**: Full PostgreSQL (`pg_dump hostvra`) and MariaDB/MySQL (`mysqldump --single-transaction --all-databases`) database dumps.

---

## 2. Automatic Rollback on Deployment Failure

`update.sh` registers `trap rollback ERR`. If binary compilation, Next.js build (`npm run build`), or post-restart `/health` verification fails:
- `hostvra-api.bak` and `hostvra-agent.bak` are restored to `/usr/local/bin/`.
- `web_next_build.tar.gz` is extracted back to `apps/web/.next`.
- `api.env.bak` and persistent store backups are verified/restored without overwriting customer data.
- `systemctl restart hostvra-api hostvra-web hostvra-agent` restores the services immediately.

---

## 3. Manual Operator Rollback Procedure (If Needed After Cutover)

To manually roll back to the most recent backup snapshot on `root@hostvra:~#`:

```bash
LATEST_BAK=$(ls -td /var/lib/hostvra/updates_backup/* | head -n 1)
echo "Rolling back from: $LATEST_BAK"
cp -fp "$LATEST_BAK/hostvra-api.bak" /usr/local/bin/hostvra-api
[[ -f "$LATEST_BAK/hostvra-agent.bak" ]] && cp -fp "$LATEST_BAK/hostvra-agent.bak" /usr/local/bin/hostvra-agent
if [[ -f "$LATEST_BAK/web_next_build.tar.gz" ]]; then
  cd /root/Hostvra && rm -rf apps/web/.next && tar -xzf "$LATEST_BAK/web_next_build.tar.gz" -C apps/web
fi
systemctl restart hostvra-api hostvra-web hostvra-agent
curl -fsS http://127.0.0.1:8080/health && echo "Rollback Healthy"
```
*(Note: Database migrations `0001`–`0037` are additive `IF NOT EXISTS` schema extensions and remain backward-compatible with prior binaries without destructive schema rollback.)*
