"""
Hostvra AI Engineering — Hostvra API Tools
Read-only wrappers for the Hostvra REST API.
Requires HOSTVRA_API_URL and HOSTVRA_API_JWT environment variables.
"""

import json
import os
from typing import Any, Dict, List, Optional
from urllib import request, error


API_URL = os.environ.get("HOSTVRA_API_URL", "http://localhost:8080")
API_JWT = os.environ.get("HOSTVRA_API_JWT", "")


class HostvraAPIError(Exception):
    def __init__(self, status: int, body: str):
        self.status = status
        self.body = body
        super().__init__(f"HTTP {status}: {body[:200]}")


def _get(path: str) -> Dict[str, Any]:
    """Make an authenticated GET request to the Hostvra API."""
    if not API_JWT:
        raise HostvraAPIError(401, "HOSTVRA_API_JWT is not set — cannot make authenticated requests")

    url = f"{API_URL}{path}"
    req = request.Request(
        url,
        headers={
            "Authorization": f"Bearer {API_JWT}",
            "Accept": "application/json",
        },
    )

    try:
        with request.urlopen(req, timeout=10) as resp:
            return json.loads(resp.read().decode())
    except error.HTTPError as e:
        body = e.read().decode()
        raise HostvraAPIError(e.code, body)
    except error.URLError as e:
        raise HostvraAPIError(0, f"Connection error: {e.reason}")


def check_health() -> Dict:
    """Check if the API is up and responding."""
    url = f"{API_URL}/health"
    req = request.Request(url, headers={"Accept": "application/json"})
    try:
        with request.urlopen(req, timeout=5) as resp:
            return {"reachable": True, "body": json.loads(resp.read().decode())}
    except Exception as e:
        return {"reachable": False, "error": str(e)}


def get_servers(org_id: Optional[str] = None) -> List[Dict]:
    """List servers in the organization."""
    data = _get("/api/v1/servers")
    return data.get("data", data)


def get_server_metrics(server_id: str, limit: int = 10) -> Dict:
    """Get recent metrics for a server."""
    return _get(f"/api/v1/servers/{server_id}/metrics?limit={limit}")


def get_websites(server_id: Optional[str] = None) -> List[Dict]:
    """List websites."""
    path = f"/api/v1/websites" if not server_id else f"/api/v1/servers/{server_id}/websites"
    data = _get(path)
    return data.get("data", data)


def get_audit_logs(limit: int = 50) -> List[Dict]:
    """Get recent audit logs."""
    data = _get(f"/api/v1/audit-logs?limit={limit}")
    return data.get("data", data)


def get_firewall_rules() -> Dict:
    """Get firewall rules (UFW + Fail2ban)."""
    return _get("/api/v1/firewall/rules")


def get_ssl_certificates() -> List[Dict]:
    """List SSL certificates."""
    data = _get("/api/v1/ssl")
    return data.get("data", data)


def get_databases(server_id: str) -> List[Dict]:
    """List managed databases on a server."""
    data = _get(f"/api/v1/databases?server_id={server_id}")
    return data.get("data", data)


def get_dashboard_overview() -> Dict:
    """Get dashboard overview telemetry."""
    return _get("/api/v1/dashboard/overview")


def get_update_jobs() -> List[Dict]:
    """Get system update job history."""
    data = _get("/api/v1/system/updates/jobs")
    return data.get("data", data)
