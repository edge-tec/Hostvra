# HOSTVRA — P11 LIVE VPS DEPLOYMENT REPORT

**Execution Date & Time:** 2026-11-20T17:07:00+06:00  
**Repository:** `edge-tec/Hostvra` (`origin/main`)  
**Certified P10 Baseline Commit:** `59bf61f700524b57797537b6e501508673870bf5`  
**Live VPS Execution Status:** **BLOCKED BY EXTERNAL SSH CREDENTIALS (Operator One-Command Execution Ready)**

---

## 1. Root Cause of Operator's `/opt/hostvra: No such file or directory` & P11 Fix

When the operator executed `cd /opt/hostvra && ...` on the Contabo production VPS (`root@hostvra:~#`), bash returned:
```text
-bash: cd: /opt/hostvra: No such file or directory
```
**Root Cause Identified in Repository Inspection (`install.sh` line 325 vs `docker-install.sh` line 24):**
- Standard bare-metal VPS installations performed by `install.sh` clone the repository to **`/root/Hostvra`** (and store persistent state in `/var/lib/hostvra`), whereas `docker-install.sh` uses `/opt/hostvra`.
- **P11 Permanent Remediation in `update.sh`**:
  1. `update.sh` auto-detects the active repository across `/root/Hostvra`, `/opt/hostvra`, `/var/lib/hostvra/repo`, and `/var/www/hostvra`.
  2. `update.sh` now automatically creates symlinks for both `/opt/hostvra` and `/root/Hostvra` pointing to the detected `$HOSTVRA_DIR` so any future command targeting either `/opt/hostvra` or `/root/Hostvra` works identically.
  3. `update.sh` now performs its **timestamped pre-deployment backup** (`pre_deploy_commit.txt`, `service_states.txt`, `hostvra-api.bak`, `hostvra-agent.bak`, `web_next_build.tar.gz`, `api.env.bak`, `store.json.bak`, `hostvra_db.sql`, `mariadb_all.sql`) **before** `git reset --hard origin/main` and before deleting `apps/web/.next`.

---

## 2. Verified Linux Production Binary Artifact (`bin/hostvra-api-linux-amd64`)

Executed locally on `/Volumes/EDGE/Hostvra-main`:
```text
$ file bin/hostvra-api-linux-amd64
bin/hostvra-api-linux-amd64: ELF 64-bit LSB executable, x86-64, version 1 (SYSV), statically linked, Go BuildID=9KmJn2r0ZqiKOqf1_Mpl/-GmnsvLrp3DY3zvS-dtZ/A5zDQyFi4GzkwYv_-bXC/bS4MsHjfjYDGqEe-SPEh, BuildID[sha1]=8f0b78733968098cae2d9c3f485a2c626add3b8c, with debug_info, not stripped

$ shasum -a 256 bin/hostvra-api-linux-amd64
59ffa98d127382b56d1e55c75b9be7d975feec196fd019a9c00e9d201b875a37  bin/hostvra-api-linux-amd64
```

---

## 3. Exact Operator Deployment Command (Contabo VPS `root@hostvra:~#`)

Because this local workspace has no direct SSH credentials to `root@hostvra`, run the following single command on your Contabo VPS terminal (`root@hostvra:~#`):

```bash
cd /root/Hostvra && git pull origin main && bash update.sh
```
*(Or simply: `curl -fsSL https://raw.githubusercontent.com/edge-tec/Hostvra/main/update.sh | bash`)*
