export function getApiBaseUrl(): string {
  if (typeof window !== 'undefined') {
    // In browser, always use relative path so requests route through Nginx reverse proxy
    // on the current host/origin (e.g. 192.168.64.3, localhost, public IP)
    return '';
  }
  let internal = process.env.INTERNAL_API_URL || 'http://127.0.0.1:8080';
  if (internal.endsWith('/')) {
    internal = internal.slice(0, -1);
  }
  if (internal.endsWith('/api')) {
    internal = internal.slice(0, -4);
  }
  return internal;
}

export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
    details?: any;
    request_id?: string;
  };
  meta?: {
    page?: number;
    limit?: number;
    total?: number;
  };
}

export interface User {
  id: string;
  email: string;
  full_name: string;
  is_active: boolean;
  is_superadmin: boolean;
  default_org_id: string;
  role: string;
}

export interface Organization {
  id: string;
  name: string;
  slug: string;
  plan_tier: string;
  max_servers: number;
  max_websites: number;
}

export interface ServerCapabilities {
  web_servers?: string[];
  php_versions?: string[];
  databases?: string[];
  services?: string[];
}

export interface Server {
  id: string;
  organization_id: string;
  name: string;
  hostname: string;
  ip_address: string;
  os_name: string;
  os_version: string;
  architecture: string;
  kernel_version?: string;
  agent_version: string;
  status: 'online' | 'offline' | 'connecting' | 'maintenance' | 'error';
  cpu_cores: number;
  cpu_model?: string;
  ram_total_mb: number;
  disk_total_gb: number;
  capabilities?: ServerCapabilities;
  is_maintenance?: boolean;
  draining?: boolean;
  max_websites?: number;
  max_databases?: number;
  last_heartbeat_at?: string;
  uptime_seconds?: number;
  created_at: string;
}

export interface ServerWorkloadSummary {
  server_id: string;
  status: string;
  websites_count: number;
  databases_count: number;
  mailboxes_count: number;
  active_workloads: number;
  can_decommission: boolean;
}

export interface MigrationLogItem {
  timestamp: string;
  level: string;
  message: string;
}

export interface CrossNodeMigration {
  id: string;
  organization_id: string;
  source_server_id: string;
  destination_server_id: string;
  website_id?: string;
  database_id?: string;
  status: 'pending' | 'validating' | 'snapshotting' | 'transferring' | 'verifying' | 'restoring' | 'cutover_ready' | 'completed' | 'failed' | 'rolled_back';
  progress: number;
  current_step: string;
  source_checksum?: string;
  destination_checksum?: string;
  transferred_bytes: number;
  total_bytes: number;
  error_message?: string;
  logs: MigrationLogItem[];
  created_by?: string;
  started_at?: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
}

export interface PreflightCheckItem {
  name: string;
  status: 'pass' | 'warn' | 'fail';
  message: string;
}

export interface PreflightCheckResponse {
  compatible: boolean;
  checks: PreflightCheckItem[];
  estimated_size_mb: number;
  source_server_name: string;
  dest_server_name: string;
}

export interface EnrollmentTokenResponse {
  token_id: string;
  raw_token: string;
  label: string;
  expires_at: string;
  install_command: string;
}

export interface Website {
  id: string;
  server_id: string;
  organization_id?: string;
  primary_domain: string;
  document_root: string;
  system_user?: string;
  php_version?: string;
  web_server_type?: string;
  app_type: 'php' | 'static' | 'proxy' | 'nodejs' | 'python' | 'go' | 'wordpress' | 'laravel' | string;
  proxy_port?: number;
  status: 'active' | 'suspended' | 'disabled';
  ssl_enabled: boolean;
  ssl_days_left?: number;
  backup_count?: number;
  backup_status?: string;
  category?: string;
  remarks?: string;
  expiration?: string;
  requests_count?: number;
  waf_status?: 'Active' | 'Inactive';
  traffic_history?: number[];
  account_username?: string;
  account_domain?: string;
  account_plan?: string;
  created_at: string;
}

export interface Database {
  id: string;
  server_id?: string;
  db_type: 'mysql' | 'mariadb' | 'postgresql' | 'pgsql' | 'sqlserver' | 'mongodb' | 'redis';
  name: string;
  username?: string;
  password?: string;
  character_set?: string;
  collation?: string;
  size_bytes?: number;
  quota?: string;
  backup_status?: string;
  backup_count?: number;
  location?: string;
  note?: string;
  host_allow?: string;
  in_recycle_bin?: boolean;
  created_at: string;
}

export interface DatabaseUser {
  id: string;
  server_id: string;
  db_type: 'mysql' | 'mariadb' | 'postgresql';
  username: string;
  host_allow: string;
  created_at: string;
}

export interface SSLCertificate {
  id: string;
  website_id?: string;
  domain_list: string[];
  issuer: string;
  cert_path: string;
  key_path: string;
  issued_at: string;
  expires_at: string;
  auto_renew: boolean;
  status: 'valid' | 'expired' | 'renewing' | 'failed' | 'expiring_soon';
  created_at: string;
  is_wildcard?: boolean;
  days_remaining?: number;
  dns_provider?: string;
}

export interface SSLChallengeInfo {
  challenge_id: string;
  domain: string;
  txt_host: string;
  txt_value: string;
  token: string;
  provider: string;
  status: string;
  created_at: string;
  expires_at: string;
}


export interface AuditLog {
  id: string;
  action: string;
  resource_type: string;
  resource_id?: string;
  ip_address?: string;
  status: 'success' | 'failure';
  error_message?: string;
  metadata?: Record<string, any>;
  created_at: string;
}

export interface SystemVersionInfo {
  api_version: string;
  agent_version: string;
  db_schema_version: number;
  channel: 'stable' | 'beta' | 'nightly';
  os_arch: string;
  go_version: string;
  uptime: string;
  update_available: boolean;
  latest_version: string;
}

export interface ReleaseMetadata {
  version: string;
  channel: 'stable' | 'beta' | 'nightly';
  component: string;
  release_notes: string;
  min_supported_version: string;
  package_url: string;
  package_size_bytes: number;
  sha256_checksum?: string;
  arch_compatibility: string[];
  os_compatibility: string[];
  released_at: string;
}

export interface CompatibilityReport {
  compatible: boolean;
  can_upgrade: boolean;
  is_downgrade: boolean;
  reasons: string[];
  os_supported: boolean;
  arch_supported: boolean;
  min_version_met: boolean;
}

export interface UpdateStep {
  id: string;
  job_id: string;
  step_name: string;
  status: 'pending' | 'in_progress' | 'completed' | 'failed' | 'skipped';
  started_at?: string;
  completed_at?: string;
  error_message?: string;
  details?: Record<string, any>;
}

export interface UpdateJob {
  id: string;
  target_version: string;
  previous_version: string;
  channel: 'stable' | 'beta' | 'nightly';
  component: string;
  status:
    | 'pending'
    | 'prechecking'
    | 'backing_up'
    | 'downloading'
    | 'verifying'
    | 'preparing'
    | 'migrating'
    | 'installing'
    | 'activating'
    | 'health_checking'
    | 'completed'
    | 'failed'
    | 'rolled_back';
  progress_percent: number;
  current_step_description?: string;
  error_message?: string;
  initiated_by?: string;
  started_at: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
  steps?: UpdateStep[];
}

export interface TerminalInfo {
  hostname: string;
  os: string;
  arch: string;
  user: string;
  default_cwd: string;
  shell: string;
  quick_cmds: string[];
}

export interface CreateTerminalSessionRequest {
  request_id?: string;
  cols?: number;
  rows?: number;
  cwd?: string;
}

export interface CreateTerminalSessionResponse {
  request_id: string;
  session_id: string;
  ws_url: string;
  user: string;
  cwd: string;
  cols: number;
  rows: number;
}

export interface TerminalExecutionResult {
  command: string;
  cwd: string;
  stdout: string;
  stderr: string;
  exit_code: number;
  duration_ms: number;
  timestamp: string;
}

export type AppCategory = 'process_manager' | 'web_server' | 'database' | 'runtime' | 'security' | 'tools';

export interface AppPackage {
  id: string;
  name: string;
  display_name: string;
  version: string;
  category: AppCategory;
  description: string;
  developer: string;
  price: string;
  icon: string;
  service_name?: string;
  binary_path?: string;
  config_path?: string;
  default_port?: number;
  is_installed: boolean;
  status: 'running' | 'stopped' | 'not_installed' | 'installing' | 'uninstalling';
  installed_version?: string;
}

export interface AppInstallJob {
  id: string;
  app_id: string;
  action: 'install' | 'uninstall';
  status: 'pending' | 'running' | 'completed' | 'failed';
  progress: number;
  logs: string[];
  error?: string;
  started_at: string;
  ended_at?: string;
}

export interface DashboardTelemetry {
  load: {
    text: string;
    avg: string;
    percent: number;
    load_1m: number;
    load_5m: number;
    load_15m: number;
  };
  cpu: {
    cores: number;
    model: string;
    percent: number;
  };
  ram: {
    used_mb: number;
    total_mb: number;
    used: string;
    total: string;
    percent: number;
  };
  disk: {
    path: string;
    used_gb: number;
    total_gb: number;
    used: string;
    total: string;
    percent: number;
  };
  network: {
    interface: string;
    upstream_mb: string;
    downstream_mb: string;
    total_sent_gb: string;
    total_received_gb: string;
    upstream_kbps: number;
    downstream_kbps: number;
  };
  disk_io: {
    read_mb: string;
    write_mb: string;
    total_read_gb: string;
    total_write_gb: string;
    read_kbps: number;
    write_kbps: number;
  };
  uptime_seconds: number;
  hostname: string;
  os_name: string;
  os_version: string;
  kernel_version: string;
}

export interface DashboardCounts {
  websites_running: number;
  websites_stopped: number;
  websites_total: number;
  databases_total: number;
  ftp_accounts_total: number;
  servers_total: number;
  security_risks: number;
  last_security_scan: string;
}

export interface DashboardOverview {
  telemetry: DashboardTelemetry;
  counts: DashboardCounts;
}

export interface SystemFixResponse {
  success: boolean;
  logs: string[];
  health: string;
}

export function getStoredToken(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem('hostvra_access_token');
}

export function setStoredToken(token: string) {
  if (typeof window !== 'undefined') {
    localStorage.setItem('hostvra_access_token', token);
    document.cookie = `hostvra_token=${encodeURIComponent(token)}; path=/; max-age=604800; SameSite=Lax`;
  }
}

export function clearStoredAuth() {
  if (typeof window !== 'undefined') {
    localStorage.removeItem('hostvra_access_token');
    localStorage.removeItem('hostvra_user');
    document.cookie = 'hostvra_token=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT; max-age=0; SameSite=Lax';
  }
}

export function getStoredWebmailToken(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem('hostvra_webmail_token');
}

export function setStoredWebmailToken(token: string) {
  if (typeof window !== 'undefined') {
    localStorage.setItem('hostvra_webmail_token', token);
    document.cookie = `hostvra_webmail_token=${encodeURIComponent(token)}; path=/; max-age=86400; SameSite=Lax`;
  }
}

export function clearStoredWebmailAuth() {
  if (typeof window !== 'undefined') {
    localStorage.removeItem('hostvra_webmail_token');
    localStorage.removeItem('hostvra_webmail_accounts');
    localStorage.removeItem('hostvra_webmail_active_account');
    document.cookie = 'hostvra_webmail_token=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT; max-age=0; SameSite=Lax';
  }
}

async function executeFetch<T>(
  baseUrl: string,
  endpoint: string,
  options: RequestInit,
  headers: Record<string, string>
): Promise<ApiResponse<T>> {
  const isLongRunning =
    endpoint.includes('terminal') ||
    endpoint.includes('backup') ||
    endpoint.includes('update') ||
    endpoint.includes('docker') ||
    endpoint.includes('websites') ||
    endpoint.includes('import') ||
    endpoint.includes('export') ||
    endpoint.includes('databases');
  const timeoutMs = isLongRunning ? 900000 : 60000;

  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), timeoutMs);

  const formattedEndpoint = endpoint.startsWith('/') ? endpoint : `/${endpoint}`;
  let cleanBase = baseUrl ? (baseUrl.endsWith('/') ? baseUrl.slice(0, -1) : baseUrl) : '';
  if (cleanBase.endsWith('/api') && formattedEndpoint.startsWith('/api')) {
    cleanBase = cleanBase.slice(0, -4);
  }
  const requestUrl = cleanBase ? `${cleanBase}${formattedEndpoint}` : formattedEndpoint;

  try {
    const res = await fetch(requestUrl, {
      credentials: 'include',
      ...options,
      headers,
      signal: options.signal || controller.signal,
    });

    clearTimeout(timeoutId);

    // Auto-intercept 401 Unauthorized on authenticated routes to clear stale session and redirect
    if (res.status === 401 && !endpoint.includes('/auth/login') && !endpoint.includes('/auth/register') && !endpoint.includes('/webmail/auth') && !endpoint.includes('/webmail/sso/validate') && !endpoint.includes('/webmail/session')) {
      if (endpoint.includes('/webmail') || (typeof window !== 'undefined' && window.location.pathname.startsWith('/webmail'))) {
        clearStoredWebmailAuth();
        if (typeof window !== 'undefined' && window.location.pathname !== '/webmail/login' && !window.location.pathname.startsWith('/webmail/sso')) {
          window.location.href = '/webmail/login';
        }
      } else {
        clearStoredAuth();
        if (typeof window !== 'undefined' && !window.location.pathname.startsWith('/login') && !window.location.pathname.startsWith('/register')) {
          const currentPath = window.location.pathname + window.location.search;
          const redirectParam = currentPath && currentPath !== '/' ? `?redirect=${encodeURIComponent(currentPath)}` : '';
          window.location.href = `/login${redirectParam}`;
        }
      }
    }

    const contentType = res.headers.get('content-type') || '';
    let data: any = null;
    if (contentType.includes('application/json')) {
      try {
        data = await res.json();
      } catch {
        data = null;
      }
    }

    if (!data) {
      const text = await res.text().catch(() => '');
      return {
        success: false,
        error: {
          code: `HTTP_${res.status}`,
          message: res.status === 404
            ? `API endpoint not found (404) at ${endpoint}. Please restart the Hostvra Go API backend.`
            : (text.slice(0, 150) || `Request failed with HTTP status ${res.status}`),
        },
      };
    }

    if (!res.ok && !data.error) {
      return {
        success: false,
        error: {
          code: 'HTTP_ERROR',
          message: data.message || `Request failed with status ${res.status}`,
        },
      };
    }
    return data;
  } catch (err: any) {
    clearTimeout(timeoutId);
    if (err.name === 'AbortError') {
      return {
        success: false,
        error: {
          code: 'TIMEOUT',
          message: 'Connection timed out. The Hostvra API server did not respond in time.',
        },
      };
    }
    return {
      success: false,
      error: {
        code: 'NETWORK_ERROR',
        message: err.message || 'Unable to connect to Hostvra API server',
      },
    };
  }
}

export async function apiFetch<T>(endpoint: string, options: RequestInit = {}): Promise<ApiResponse<T>> {
  const isStandaloneWebmailRoute =
    typeof window !== 'undefined' && window.location.pathname.startsWith('/webmail');
  const webmailToken = getStoredWebmailToken();
  const cpToken = getStoredToken();

  // If on standalone /webmail, webmailToken is preferred if available.
  // Everywhere else (Control Panel, e.g. /email, /websites, /dashboard),
  // cpToken is ALWAYS preferred because the user is an authenticated CP administrator/user.
  let token: string | null = null;
  if (isStandaloneWebmailRoute) {
    token = webmailToken || cpToken;
  } else {
    token = cpToken || webmailToken;
  }

  const headers: Record<string, string> = {
    ...((options.headers as Record<string, string>) || {}),
  };

  if (typeof FormData !== 'undefined' && options.body instanceof FormData) {
    delete headers['Content-Type'];
  } else if (!headers['Content-Type']) {
    headers['Content-Type'] = 'application/json';
  }

  if (token && !headers['Authorization']) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  // Always propagate the Control Panel token via X-CP-Token whenever available,
  // providing backend handlers proof of admin/owner identity
  if (cpToken && !headers['X-CP-Token']) {
    headers['X-CP-Token'] = cpToken;
  }

  // Attach X-Webmail-Token only when on standalone webmail or when webmailToken is explicitly active
  if (isStandaloneWebmailRoute && webmailToken && !headers['X-Webmail-Token']) {
    headers['X-Webmail-Token'] = webmailToken;
  }

  return executeFetch<T>(getApiBaseUrl(), endpoint, options, headers);
}

export interface FirewallStatus {
  backend: string;
  is_active: boolean;
  default_incoming: string;
  default_outgoing: string;
  rules_count: number;
  ssh_port_protected: boolean;
  fail2ban_installed: boolean;
  fail2ban_active: boolean;
  jails_count: number;
  total_banned_ips: number;
}

export interface FirewallRule {
  id: string;
  number: number;
  to: string;
  action: string;
  from: string;
  protocol: string;
  comment: string;
}

export interface Fail2banJail {
  name: string;
  service: string;
  is_active: boolean;
  currently_failed: number;
  total_failed: number;
  currently_banned: number;
  total_banned: number;
  banned_ips: string[];
}

export interface BannedIPItem {
  ip: string;
  jail: string;
  banned_at: string;
}

export interface ThreatItem {
  file_path: string;
  threat_name: string;
  line_number?: number;
  severity: 'critical' | 'high' | 'medium';
  quarantined?: boolean;
}

export interface SecurityScan {
  id: string;
  organization_id: string;
  server_id?: string;
  website_id?: string;
  scan_type: 'clamav' | 'heuristic' | 'combined';
  target_path: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  total_files: number;
  infected_files: number;
  findings: ThreatItem[];
  error_message?: string;
  started_at?: string;
  completed_at?: string;
  created_at: string;
}

export interface QuarantineItem {
  id: string;
  scan_id?: string;
  organization_id: string;
  server_id?: string;
  website_id?: string;
  original_path: string;
  quarantine_filename: string;
  file_size: number;
  file_mode: string;
  file_owner: string;
  file_group: string;
  sha256_hash: string;
  threat_name: string;
  status: 'quarantined' | 'restored' | 'deleted';
  restored_at?: string;
  deleted_at?: string;
  quarantined_at: string;
}

export interface SecurityEngineStatus {
  heuristic_active: boolean;
  heuristic_rules: number;
  clamav_installed: boolean;
  clamav_daemon: boolean;
  clamav_binary?: string;
  clamav_db_updated_at?: string;
  quarantine_dir: string;
}

export interface TwoFactorStatus {
  two_factor_enabled: boolean;
  recovery_codes_remaining: number;
}

export interface CronDaemonStatus {
  is_active: boolean;
  daemon: string;
  jobs_count: number;
}

export interface CronJob {
  id: string;
  schedule: string;
  command: string;
  system_user: string;
  description: string;
  is_enabled: boolean;
  last_run_at?: string;
  last_status?: string;
  last_output?: string;
}

export interface CronExecutionResult {
  job_id?: string;
  command: string;
  system_user: string;
  exit_code: number;
  stdout: string;
  stderr?: string;
  duration_ms: number;
  timestamp: string;
  success: boolean;
}

export interface DockerStatus {
  is_installed: boolean;
  is_daemon_running: boolean;
  server_version?: string;
  containers_total?: number;
  containers_running?: number;
  containers_paused?: number;
  containers_stopped?: number;
  images_total?: number;
  storage_driver?: string;
}

export interface DockerContainer {
  id: string;
  names: string;
  image: string;
  command?: string;
  status: string;
  state: 'running' | 'exited' | 'paused' | 'restarting';
  ports: string;
  created_at: string;
}

export interface DockerImage {
  id: string;
  repository: string;
  tag: string;
  size: string;
  created_at: string;
}

export interface DockerStats {
  id: string;
  name: string;
  cpu_perc: string;
  mem_usage: string;
  mem_perc: string;
  net_io: string;
  block_io: string;
  pids: string;
}

export interface FTPDaemonStatus {
  is_installed: boolean;
  is_active: boolean;
  daemon_name: string;
  port: number;
  users_count: number;
  server_ip?: string;
}

export interface FTPUser {
  username: string;
  home_dir: string;
  uid: number;
  gid: number;
  quota_mb?: number;
  upload_bandwidth_kbps?: number;
  download_bandwidth_kbps?: number;
  max_sessions?: number;
  is_enabled: boolean;
  created_at?: string;
}

export interface BackupRecord {
  id: string;
  server_id: string;
  type: 'website' | 'database' | 'full_config' | 'full_account';
  target_name: string;
  storage: 'local' | 's3' | 'wasabi' | 'gcs' | 'r2' | 'b2' | 'spaces' | 'minio' | 'sftp';
  storage_id?: string;
  size_bytes: number;
  sha256?: string;
  item_count?: number;
  status: 'completed' | 'failed' | 'in_progress';
  file_name: string;
  remote_key?: string;
  verified_at?: string;
  integrity_status?: 'unverified' | 'verified' | 'corrupted';
  created_at: string;
  completed_at?: string;
  error_message?: string;
}

export interface BackupDestination {
  id: string;
  name: string;
  type: 's3' | 'wasabi' | 'gcs' | 'r2' | 'b2' | 'spaces' | 'minio' | 'sftp' | 'local';
  endpoint: string;
  region: string;
  bucket: string;
  access_key: string;
  secret_key?: string;
  prefix?: string;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export interface BackupSchedule {
  id: string;
  name: string;
  scope: 'website' | 'database' | 'full_config' | 'full_account';
  target_name: string;
  destination_id: string;
  frequency: 'daily' | 'weekly' | 'monthly' | 'cron';
  cron_expr?: string;
  retention: number;
  enabled: boolean;
  last_run_at?: string;
  next_run_at?: string;
  created_at: string;
}

export interface IntegrityReport {
  backup_id: string;
  file_name: string;
  target_name: string;
  type: string;
  storage: string;
  expected_sha256: string;
  calculated_sha256: string;
  sha256_match: boolean;
  archive_valid: boolean;
  item_count: number;
  uncompressed_size: number;
  status: 'verified' | 'corrupted';
  message: string;
  verified_at: string;
}

export interface PreflightCheck {
  name: string;
  passed: boolean;
  details: string;
}

export interface PreflightRestoreReport {
  backup_id: string;
  target_name: string;
  type: string;
  preflight_ok: boolean;
  disk_space_required: number;
  disk_space_available: number;
  service_status: string;
  existing_target_found: boolean;
  target_conflict_risk: 'none' | 'overwrite_warning';
  integrity_status: string;
  checks: PreflightCheck[];
  message: string;
}


export interface ResourceLimits {
  memory_max_mb: number;
  memory_high_mb?: number;
  cpu_quota: number;
  cpu_weight?: number;
  tasks_max: number;
  io_read_iops?: number;
  io_write_iops?: number;
  io_read_mbps?: number;
  io_write_mbps?: number;
  oom_policy?: 'kill' | 'throttle' | string;
  open_basedir: boolean;
}

export interface UserIsolationInfo {
  username: string;
  group_name?: string;
  uid: number;
  gid: number;
  home_dir: string;
  document_root?: string;
  php_version?: string;
  php_pool_socket: string;
  php_pool_config?: string;
  cgroup_path?: string;
  slice_name?: string;
  limits: ResourceLimits;
  current_memory_mb?: number;
  memory_used_mb?: number;
  memory_peak_mb?: number;
  cpu_usage_percent?: number;
  cpu_usage_perc?: number;
  current_pids?: number;
  tasks_current?: number;
  throttled_usec?: number;
  oom_kills?: number;
  oom_events?: number;
  high_events?: number;
  is_systemd_slice?: boolean;
  status?: string;
  created_at?: string;
}

export interface ResourceQuotaEvent {
  id: string;
  website_id: string;
  username: string;
  event_type: 'oom_kill' | 'memory_high' | 'cpu_throttled' | 'quota_provisioned' | 'quota_updated' | string;
  severity: 'info' | 'warning' | 'critical' | string;
  details: string;
  metric_value: number;
  threshold_value: number;
  created_at: string;
}

export interface WAFStatus {
  is_installed: boolean;
  is_enabled: boolean;
  engine: string;
  mode: 'On' | 'DetectionOnly' | 'Off';
  paranoia_level: number;
  anomaly_threshold: number;
  rules_count: number;
  active_categories_count: number;
  total_attacks_blocked: number;
  total_attacks_detected: number;
  last_blocked_at?: string;
  active_websites_count: number;
  config_path: string;
}

export interface WAFRuleCategory {
  id: string;
  name: string;
  description: string;
  crs_range: string;
  rules_count: number;
  is_enabled: boolean;
}

export interface WebsiteWAFConfig {
  domain: string;
  enabled: boolean;
  mode: 'Inherit' | 'On' | 'DetectionOnly' | 'Off';
  paranoia_level: number;
  cms_preset: 'none' | 'wordpress' | 'drupal' | 'nextjs';
  excluded_rule_ids: number[];
  last_attack_at?: string;
  updated_at: string;
}

export interface WAFAttackEvent {
  id: string;
  timestamp: string;
  client_ip: string;
  domain: string;
  method: string;
  uri: string;
  rule_id: number;
  rule_category: string;
  severity: 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW';
  action: 'BLOCKED' | 'DETECTED' | 'PASSED';
  message: string;
  anomaly_score: number;
  matched_data: string;
}

export interface OneClickAppTemplate {
  id: string;
  name: string;
  version: string;
  category: 'cms' | 'framework' | 'tool';
  description: string;
  icon: string;
  min_php_version?: string;
  requires_db: boolean;
  recommended_ram: string;
  admin_path: string;
}

export interface InstalledAppInfo {
  app_id: string;
  name: string;
  version: string;
  document_root: string;
  installed_at: string;
  db_name?: string;
  db_user?: string;
  admin_url: string;
  config_file: string;
  status: 'healthy' | 'warning' | 'unconfigured';
}

export interface WebsiteAppStatus {
  has_app: boolean;
  app?: InstalledAppInfo;
}

export interface WordPressDetails {
  is_installed: boolean;
  version: string;
  site_title: string;
  site_url: string;
  admin_url: string;
  admin_email: string;
  db_name: string;
  db_user: string;
  db_prefix: string;
  db_host: string;
  maintenance_mode: boolean;
  debug_mode: boolean;
  auto_updates: boolean;
  plugins_count: number;
  active_plugins: number;
  themes_count: number;
  active_theme: string;
  php_version?: string;
  wp_cli_installed: boolean;
  config_path: string;
  extra_constants?: Record<string, string>;
}

export interface WordPressPlugin {
  name: string;
  title: string;
  status: 'active' | 'inactive' | 'must-use';
  version: string;
  update_available: boolean;
  update_version?: string;
  description?: string;
  author?: string;
}

export interface WordPressTheme {
  name: string;
  title: string;
  status: 'active' | 'inactive';
  version: string;
  update_available: boolean;
  author?: string;
}

export interface PopularPlugin {
  slug: string;
  name: string;
  description: string;
  category: string;
  icon: string;
}

// ============================================================================
// Developer & Application Management (Phase 6)
// ============================================================================

export interface ManagedAppRecord {
  id: string;
  website_id: string;
  organization_id: string;
  name: string;
  app_type: 'nodejs' | 'python';
  runtime_version: string;
  app_root: string;
  entrypoint: string;
  start_command: string;
  port: number;
  system_user: string;
  env_vars: Record<string, string>;
  status: 'running' | 'stopped' | 'failed';
  restart_count: number;
  created_at: string;
  updated_at: string;
}

export interface AppProcessStatus {
  id: string;
  status: 'running' | 'stopped' | 'failed' | 'not_configured';
  pid?: number;
  port?: number;
  uptime_sec?: number;
  memory_mb?: number;
  cpu_percent?: number;
  restart_count?: number;
  last_log_tail?: string;
}

export interface RuntimeInfo {
  language: 'nodejs' | 'python';
  version: string;
  binary: string;
  pkg_mgr: string;
  features: string[];
}

export interface GitRepositoryRecord {
  id: string;
  website_id: string;
  organization_id: string;
  provider: 'github' | 'gitlab' | 'custom';
  repo_url: string;
  branch: string;
  webhook_secret: string;
  deploy_key_pub: string;
  deploy_key_priv?: string;
  auto_deploy: boolean;
  build_command: string;
  deploy_path: string;
  release_keep_count: number;
  created_at: string;
  updated_at: string;
}

export interface GitDeploymentRecord {
  id: string;
  repository_id: string;
  website_id: string;
  commit_hash: string;
  commit_message: string;
  author: string;
  status: 'queued' | 'cloning' | 'building' | 'success' | 'failed' | 'rolled_back';
  build_logs: string;
  release_path: string;
  duration_seconds: number;
  created_at: string;
}

export interface WordPressStagingRecord {
  id: string;
  website_id: string;
  organization_id: string;
  staging_domain: string;
  staging_doc_root: string;
  staging_db_name: string;
  staging_db_user: string;
  auth_enabled: boolean;
  auth_user: string;
  auth_password?: string;
  status: 'active' | 'syncing' | 'deleted';
  last_sync_at?: string;
  created_at: string;
  updated_at: string;
}

// ============================================================================
// PHASE SEVEN: CACHING ENGINE, REMOTE DB ACCESS, DATABASE MAINTENANCE
// ============================================================================

export interface WebsiteCacheConfig {
  id?: string;
  website_id: string;
  organization_id?: string;
  cache_type: 'redis' | 'memcached' | 'none';
  connection_type: 'socket' | 'tcp';
  socket_path?: string;
  host: string;
  port: number;
  password?: string;
  max_memory_mb: number;
  eviction_policy: string;
  wp_object_cache_enabled: boolean;
  status: 'active' | 'disabled' | 'failed';
  created_at?: string;
  updated_at?: string;
}

export interface CacheServiceStatus {
  service_type: 'redis' | 'memcached';
  installed: boolean;
  running: boolean;
  version: string;
  binary_path: string;
  port: number;
  socket_path: string;
  error?: string;
}

export interface CacheMetrics {
  cache_type: string;
  running: boolean;
  uptime_seconds?: number;
  connected_clients?: number;
  used_memory_bytes?: number;
  used_memory_human?: string;
  max_memory_bytes?: number;
  total_keys?: number;
  hits?: number;
  misses?: number;
  hit_rate_percent?: number;
}

export interface CacheBenchmarkResult {
  cache_type: string;
  operations: number;
  total_duration_ms: number;
  avg_latency_ms: number;
  min_latency_ms: number;
  max_latency_ms: number;
  ops_per_sec: number;
}

export interface DatabaseRemoteAccess {
  id: string;
  database_id: string;
  organization_id: string;
  db_user: string;
  authorized_ip: string;
  granted_privileges: string;
  firewall_rule_applied: boolean;
  status: 'active' | 'revoked';
  notes?: string;
  expires_at?: string;
  created_at: string;
  updated_at: string;
}

export interface TableHealthInfo {
  table_name: string;
  engine: string;
  rows: number;
  data_size_bytes: number;
  index_size_bytes: number;
  data_free_bytes: number;
  fragmentation_percent: number;
}

export interface DatabaseHealthReport {
  database_name: string;
  total_size_bytes: number;
  total_data_bytes: number;
  total_index_bytes: number;
  total_data_free_bytes: number;
  overall_fragmentation_percent: number;
  table_count: number;
  fragmented_tables_count: number;
  tables: TableHealthInfo[];
}

export interface TableCheckResult {
  table_name: string;
  status: 'OK' | 'warning' | 'error';
  message: string;
}

export interface DatabaseMaintenanceResult {
  operation: 'check' | 'optimize' | 'analyze';
  database_name: string;
  tables_processed: number;
  bytes_reclaimed: number;
  duration_ms: number;
  status: 'success' | 'warning' | 'failed';
  details: string;
  table_results?: TableCheckResult[];
}

export interface DatabaseMaintenanceLog {
  id: string;
  database_id: string;
  organization_id: string;
  operation: 'check' | 'optimize' | 'analyze' | 'repair';
  status: 'success' | 'warning' | 'failed';
  tables_processed: number;
  bytes_reclaimed: number;
  duration_ms: number;
  details: string;
  initiated_by: string;
  created_at: string;
}

// ============================================================================
// PHASE EIGHT: EMAIL DELIVERABILITY, SPAM & MIGRATION TYPES
// ============================================================================

export interface EmailDomain {
  id: string;
  domain: string;
  mail_hostname: string;
  status: string;
  storage_limit_bytes: number;
  storage_used_bytes: number;
  mailbox_count: number;
  alias_count: number;
  dkim_selector: string;
  is_dns_verified?: boolean;
  dns_verified_at?: string;
  created_at: string;
}

export interface EmailMailbox {
  id: string;
  domain_id: string;
  email: string;
  local_part: string;
  name: string;
  quota_bytes: number;
  used_bytes: number;
  rate_limit_per_hour?: number;
  rate_limit_per_day?: number;
  is_active: boolean;
  is_suspended: boolean;
}

export interface EmailSMTPRelay {
  id: string;
  organization_id: string;
  domain_id?: string;
  name: string;
  provider: 'ses' | 'sendgrid' | 'mailgun' | 'brevo' | 'postmark' | 'custom' | string;
  host: string;
  port: number;
  username: string;
  password_encrypted?: string;
  auth_type: string;
  encryption: 'starttls' | 'ssl' | 'none' | string;
  sender_domain: string;
  is_verified: boolean;
  verified_at?: string;
  status: 'active' | 'disabled' | 'error' | string;
  rate_limit_per_second: number;
  rate_limit_per_day: number;
  last_test_at?: string;
  last_test_status: 'pending' | 'success' | 'failed' | string;
  last_test_error?: string;
  created_at: string;
  updated_at: string;
}

export interface RelayConnectionTestResult {
  success: boolean;
  banner: string;
  tls_version?: string;
  tls_cipher?: string;
  cert_issuer?: string;
  cert_expiry?: string;
  auth_succeeded: boolean;
  latency_ms: number;
  error?: string;
  trace?: string[];
}

export interface DeliverabilityRecordStatus {
  record_type: 'SPF' | 'DKIM' | 'DMARC' | 'MX' | string;
  domain: string;
  found: boolean;
  expected_value: string;
  actual_values: string[];
  status: 'valid' | 'missing' | 'mismatch' | 'warning' | string;
  instructions: string;
}

export interface DeliverabilityAuditResult {
  domain: string;
  provider: string;
  is_fully_verified: boolean;
  records: DeliverabilityRecordStatus[];
  recommendations: string[];
  checked_at: string;
}

export interface EmailSpamSettings {
  id?: string;
  organization_id?: string;
  domain_id?: string;
  reject_score: number;
  add_header_score: number;
  greylist_score: number;
  greylist_enabled: boolean;
  bayes_enabled: boolean;
  fuzzy_enabled: boolean;
  dmarc_reporting_enabled: boolean;
  custom_rules?: string;
  created_at?: string;
  updated_at?: string;
}

export interface RspamdTelemetry {
  scanned: number;
  clean: number;
  greylisted: number;
  spam: number;
  rejected: number;
  learned_ham: number;
  learned_spam: number;
  uptime_seconds: number;
  version: string;
  is_alive: boolean;
}

export interface SpamQuarantineItem {
  id: string;
  organization_id: string;
  domain_id?: string;
  message_id: string;
  sender: string;
  recipient: string;
  subject: string;
  score: number;
  symbols: string[];
  action_taken: string;
  mail_path: string;
  status: 'quarantined' | 'released' | 'learned_ham' | 'deleted' | string;
  created_at: string;
}

export interface SourceConnectionOptions {
  host: string;
  port: number;
  username: string;
  password?: string;
  use_ssl: boolean;
  skip_ssl_verify?: boolean;
  auth_type?: string;
  exclude_folders?: string[];
  batch_size?: number;
  max_concurrency?: number;
}

export interface IMAPMigration {
  id: string;
  organization_id: string;
  destination_mailbox_id: string;
  destination_email: string;
  source_host: string;
  source_port: number;
  source_username: string;
  source_password_encrypted?: string;
  source_use_ssl: boolean;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled' | string;
  total_folders: number;
  current_folder: string;
  total_messages: number;
  synced_messages: number;
  skipped_duplicates: number;
  failed_messages: number;
  bytes_transferred: number;
  error_message?: string;
  started_at?: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
}

export interface IMAPTestResult {
  success: boolean;
  greeting?: string;
  folders_found?: string[];
  error?: string;
}

export interface HostingPlan {
  id: string;
  name: string;
  slug: string;
  description: string;
  tier: 'starter' | 'business' | 'enterprise' | 'reseller';
  price_monthly: number;
  price_yearly: number;
  currency: string;
  setup_fee?: number;
  trial_allowed?: boolean;
  trial_days?: number;
  is_featured?: boolean;
  cpu_limit?: number;
  ram_limit_mb?: number;
  disk_space_mb: number;
  bandwidth_mb: number;
  max_websites: number;
  max_databases: number;
  max_mailboxes: number;
  max_ftp: number;
  max_cron?: number;
  max_subdomains?: number;
  dedicated_ip: boolean;
  free_ssl: boolean;
  features: string[];
  is_active: boolean;
  sort_order: number;
  created_at?: string;
  updated_at?: string;
}

export interface Subscription {
  id: string;
  user_id: string;
  organization_id: string;
  plan_id: string;
  plan_name: string;
  server_id?: string;
  status: 'active' | 'pending' | 'suspended' | 'cancelled' | 'expired' | 'trial';
  billing_cycle: 'monthly' | 'yearly';
  amount: number;
  currency: string;
  disk_used_mb: number;
  bandwidth_used_mb: number;
  websites_count: number;
  next_billing_date: string;
  trial_started_at?: string;
  trial_ends_at?: string;
  auto_renew: boolean;
  created_at: string;
  updated_at: string;
}

export interface TrialSettings {
  enabled: boolean;
  default_days: number;
  require_payment_method: boolean;
  one_trial_per_customer: boolean;
  auto_suspend_on_expiry: boolean;
  updated_at?: string;
}

export interface Invoice {
  id: string;
  invoice_number: string;
  user_id: string;
  subscription_id?: string;
  plan_id: string;
  description: string;
  subtotal: number;
  tax: number;
  discount: number;
  total: number;
  currency: string;
  status: 'paid' | 'unpaid' | 'overdue' | 'cancelled';
  payment_method?: string;
  transaction_id?: string;
  due_date: string;
  paid_at?: string;
  created_at: string;
}

export interface PaymentGatewayConfig {
  gateway: string;
  display_name: string;
  enabled: boolean;
  test_mode: boolean;
  api_key?: string;
  secret_key?: string;
  merchant_id?: string;
  updated_at?: string;
}

export interface HostingAccount {
  id: string;
  organization_id: string;
  user_id: string;
  subscription_id?: string;
  server_id?: string;
  server_name?: string;
  domain: string;
  username: string;
  document_root: string;
  plan_id: string;
  plan_name: string;
  status: 'active' | 'suspended' | 'pending' | 'terminated';
  suspend_reason?: string;
  disk_limit_mb: number;
  disk_used_mb: number;
  bandwidth_limit_mb: number;
  bandwidth_used_mb: number;
  websites_limit: number;
  databases_limit: number;
  mailboxes_limit: number;
  memory_limit_mb?: number;
  cpu_limit_percent?: number;
  inodes_limit?: number;
  inodes_used?: number;
  tasks_max?: number;
  io_read_mbps?: number;
  io_write_mbps?: number;
  ip_address?: string;
  php_version?: string;
  ssl_active: boolean;
  contact_email?: string;
  contact_phone?: string;
  notes?: string;
  subdomains_limit?: number;
  ftp_users_limit?: number;
  cron_jobs_limit?: number;
  max_email_send_rate?: number;
  quota_enforced?: boolean;
  quota_status?: string;
  last_sync_at?: string;
  target_server_id?: string;
  migration_status?: string;
  migration_log?: string;
  expires_at?: string;
  suspended_at?: string;
  created_at: string;
  updated_at: string;
}

export interface WHMStats {
  total_accounts: number;
  active_accounts: number;
  suspended_accounts: number;
  pending_accounts: number;
  terminated_accounts: number;
  total_storage_alloc_mb: number;
  total_storage_used_mb: number;
  total_bandwidth_alloc_mb: number;
  total_bandwidth_used_mb: number;
  total_domains: number;
  total_databases: number;
  total_mailboxes: number;
  total_inodes_used: number;
  healthy_nodes: number;
  total_nodes: number;
  expiring_ssl_certs: number;
  over_quota_count: number;
  last_sync_time?: string;
}

export async function getWHMStats(): Promise<ApiResponse<WHMStats>> {
  return apiFetch<WHMStats>('/api/v1/accounts/stats');
}

export async function resetAccountPassword(accountId: string, newPassword?: string) {
  return apiFetch<{ message: string; auto_generated?: boolean; temporary_password?: string }>(
    `/api/v1/accounts/${accountId}/reset-password`,
    {
      method: 'POST',
      body: JSON.stringify({ new_password: newPassword }),
    }
  );
}

export async function updateAccountDetails(
  accountId: string,
  details: { contact_email?: string; contact_phone?: string; notes?: string; expires_at?: string }
) {
  return apiFetch<HostingAccount>(`/api/v1/accounts/${accountId}/details`, {
    method: 'PUT',
    body: JSON.stringify(details),
  });
}

export async function syncAccountQuota(accountId: string) {
  return apiFetch<HostingAccount>(`/api/v1/accounts/${accountId}/sync-quota`, {
    method: 'POST',
  });
}

export async function terminateAccount(accountId: string, confirmUsername: string) {
  return apiFetch<HostingAccount>(`/api/v1/accounts/${accountId}/terminate`, {
    method: 'POST',
    body: JSON.stringify({ confirm_username: confirmUsername }),
  });
}

export async function restoreAccount(accountId: string) {
  return apiFetch<HostingAccount>(`/api/v1/accounts/${accountId}/restore`, {
    method: 'POST',
  });
}

export async function migrateAccount(accountId: string, targetServerId: string) {
  return apiFetch<HostingAccount>(`/api/v1/accounts/${accountId}/migrate`, {
    method: 'POST',
    body: JSON.stringify({ target_server_id: targetServerId }),
  });
}

export async function updateAccountLimits(
  accountId: string,
  limits: {
    memory_limit_mb?: number;
    cpu_limit_percent?: number;
    disk_limit_mb?: number;
    bandwidth_limit_mb?: number;
    inodes_limit?: number;
    tasks_max?: number;
    websites_limit?: number;
    databases_limit?: number;
    mailboxes_limit?: number;
    subdomains_limit?: number;
    ftp_users_limit?: number;
    cron_jobs_limit?: number;
    max_email_send_rate?: number;
    io_read_mbps?: number;
    io_write_mbps?: number;
    php_version?: string;
    open_basedir?: boolean;
  }
): Promise<ApiResponse<HostingAccount>> {
  return apiFetch<HostingAccount>(`/api/v1/accounts/${accountId}/limits`, {
    method: 'PUT',
    body: JSON.stringify(limits),
  });
}

export interface TLDPricing {
  id: string;
  tld: string;
  register_price: number;
  renew_price: number;
  transfer_price: number;
  currency: string;
  min_years: number;
  max_years: number;
  enabled: boolean;
  is_popular: boolean;
  category: string;
  updated_at: string;
}

export interface WhoisRecord {
  domain: string;
  registrar: string;
  whois_server?: string;
  created_date?: string;
  expiry_date?: string;
  updated_date?: string;
  status?: string[];
  nameservers?: string[];
  dnssec?: string;
  registrant?: string;
  admin_email?: string;
  raw_whois?: string;
  queried_at: string;
}

export interface DomainRegistrarConfig {
  id: string;
  organization_id?: string;
  registrar: 'namecheap' | 'resellerclub' | 'cloudflare' | 'enom' | string;
  display_name: string;
  api_user?: string;
  api_key?: string;
  sandbox: boolean;
  enabled: boolean;
  is_default: boolean;
  webhook_secret?: string;
  updated_at: string;
}

export interface DomainSearchResultItem {
  domain: string;
  tld: string;
  available: boolean;
  register_price: number;
  renew_price: number;
  transfer_price: number;
  currency: string;
  is_popular: boolean;
}

export interface DomainOrderPayload {
  domain: string;
  action: 'register' | 'transfer' | 'renew';
  years: number;
  whois_privacy: boolean;
  auto_renew: boolean;
  client_name?: string;
  client_email?: string;
  client_phone?: string;
  client_address?: string;
  payment_method?: string;
}

export type TicketDepartment = 'technical' | 'billing' | 'sales' | 'abuse';
export type TicketPriority = 'low' | 'medium' | 'high' | 'urgent';
export type TicketStatus = 'open' | 'in_progress' | 'answered' | 'customer_reply' | 'closed';

export interface Ticket {
  id: string;
  ticket_number: string;
  organization_id: string;
  user_id: string;
  user_email: string;
  user_name: string;
  department: TicketDepartment;
  priority: TicketPriority;
  status: TicketStatus;
  subject: string;
  related_service?: string;
  replies_count: number;
  last_reply_at: string;
  created_at: string;
  updated_at: string;
  closed_at?: string;
}

export interface TicketReply {
  id: string;
  ticket_id: string;
  user_id: string;
  user_email: string;
  user_name: string;
  is_staff: boolean;
  is_private_note?: boolean;
  message: string;
  attachments?: string[];
  created_at: string;
}

export interface KnowledgeArticle {
  id: string;
  title: string;
  slug: string;
  category: string;
  content: string;
  summary: string;
  views: number;
  helpful_votes: number;
  unhelpful_votes: number;
  is_published: boolean;
  created_at: string;
  updated_at: string;
}

export interface CannedResponse {
  id: string;
  title: string;
  shortcut: string;
  department: TicketDepartment;
  content: string;
  created_at: string;
}

export interface SupportStats {
  total_tickets: number;
  open_tickets: number;
  answered_tickets: number;
  closed_tickets: number;
  avg_response_mins: number;
  resolution_rate: number;
  total_articles: number;
  article_helpful_pct: number;
}

export interface AIAssistantResponse {
  answer: string;
  confidence: string;
  recommended_action: string;
  related_articles: KnowledgeArticle[];
  suggested_ticket: boolean;
}

export interface SystemSettings {
  panel_domain: string;
  panel_port: string;
  security_entrance: string;
  ssl_enabled: boolean;
  ssl_days_remaining: number;
  dev_mode: boolean;
  api_enabled: boolean;
  api_key: string;
  panel_user: string;
  panel_pass?: string;
  bound_account: string;
  menu_bar_hidden: string;
  close_panel: boolean;
  ipv6_enabled: boolean;
  offline_mode: boolean;
  cdn_proxy: boolean;
  home_bulletin: boolean;
  site_monitor: boolean;
  auto_fetch_favicon: boolean;
  auto_backup_panel: boolean;
  panel_theme: string;
  panel_language: string;
  panel_alias: string;
  session_timeout: string;
  default_site_folder: string;
  default_backup_folder: string;
  server_ip: string;
  mail_server_ip_mode?: 'auto' | 'manual';
  mail_server_public_ip?: string;
  detected_public_ip?: string;
  server_time: string;
  timezone_region: string;
  timezone_city: string;
  security_alarm: boolean;
  basic_auth: boolean;
  google_auth: boolean;
  strong_password: boolean;
  authorized_ip: string;
  not_logged_in_response: string;
  password_expire: string;
  updated_at?: string;
}

// ----------------------------------------------------------------------------
// Domain Reseller & Management Types
// ----------------------------------------------------------------------------

export interface RegisteredDomain {
  id: string;
  user_id: string;
  organization_id?: string;
  domain_name: string;
  tld: string;
  status: 'active' | 'pending' | 'expired' | 'suspended' | 'cancelled' | 'transferring';
  reseller_order_id?: string;
  reseller_customer_id?: string;
  auto_renew: boolean;
  privacy_protected: boolean;
  is_locked: boolean;
  registration_date?: string;
  expiry_date?: string;
  nameservers: string[];
  epp_code?: string;
  created_at: string;
  updated_at: string;
}

export interface DomainOrder {
  id: string;
  user_id: string;
  domain_name: string;
  action: 'register' | 'renew' | 'transfer';
  years: number;
  amount: number;
  currency: string;
  status: 'pending_payment' | 'processing' | 'completed' | 'failed' | 'cancelled';
  payment_status: 'unpaid' | 'paid' | 'refunded';
  payment_method?: string;
  reseller_order_id?: string;
  error_message?: string;
  retry_count: number;
  created_at: string;
  updated_at: string;
}

export interface DomainContact {
  id?: string;
  first_name: string;
  last_name: string;
  email: string;
  phone: string;
  company_name?: string;
  address1: string;
  address2?: string;
  city: string;
  state: string;
  postal_code: string;
  country: string;
}

export interface DomainDnsRecordItem {
  id: string;
  type: string;
  name: string;
  content: string;
  ttl: number;
  priority?: number;
}

export interface DomainAdminMetrics {
  total_domains: number;
  active_domains: number;
  expiring_30_days: number;
  total_revenue: number;
  total_cost: number;
  gross_profit: number;
}

export interface DomainAdminPrice {
  id: string;
  tld: string;
  register_price?: number;
  registration_price?: number;
  renew_price?: number;
  renewal_price?: number;
  transfer_price?: number;
  cost_price?: number;
  registration_cost?: number;
  currency: string;
  min_years?: number;
  max_years?: number;
  enabled: boolean;
  is_popular?: boolean;
  category?: string;
  updated_at?: string;
}

export interface ResellerClubTestResult {
  connected?: boolean;
  status?: string;
  mode?: string;
  reseller_id?: string;
  base_url?: string;
  latency_ms?: number;
  message?: string;
}

// ----------------------------------------------------------------------------
// Domain Reseller API Helper Functions
// ----------------------------------------------------------------------------

export async function fetchRegisteredDomains(): Promise<ApiResponse<RegisteredDomain[]>> {
  return apiFetch<RegisteredDomain[]>('/api/v1/domains');
}

export async function fetchRegisteredDomain(id: string): Promise<ApiResponse<RegisteredDomain>> {
  return apiFetch<RegisteredDomain>(`/api/v1/domains/${id}`);
}

export async function searchDomainAvailability(query: string): Promise<ApiResponse<DomainSearchResultItem[]>> {
  return apiFetch<DomainSearchResultItem[]>(`/api/v1/domains/search?query=${encodeURIComponent(query)}`);
}

export async function fetchTLDPricings(): Promise<ApiResponse<TLDPricing[]>> {
  return apiFetch<TLDPricing[]>('/api/v1/domains/pricing/tlds');
}

export async function orderDomainRegistration(payload: {
  domain: string;
  years: number;
  nameservers?: string[];
  payment_method?: string;
  auto_renew?: boolean;
  registrant?: DomainContact;
}): Promise<ApiResponse<{ order: DomainOrder; invoice: Invoice; domain: string; years: number; amount: number; message: string }>> {
  return apiFetch('/api/v1/domains/order', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function initiateDomainTransfer(payload: {
  domain: string;
  auth_code: string;
  years: number;
  nameservers?: string[];
  payment_method?: string;
}): Promise<ApiResponse<{ order: DomainOrder; invoice: Invoice; message: string }>> {
  return apiFetch('/api/v1/domains/transfer', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function renewDomainSubscription(domainId: string, payload: {
  years: number;
  payment_method?: string;
}): Promise<ApiResponse<{ order: DomainOrder; invoice: Invoice; message: string }>> {
  return apiFetch(`/api/v1/domains/${domainId}/renew`, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function fetchDomainNameservers(domainId: string): Promise<ApiResponse<string[]>> {
  return apiFetch<string[]>(`/api/v1/domains/${domainId}/nameservers`);
}

export async function updateDomainNameservers(domainId: string, nameservers: string[]): Promise<ApiResponse<{ nameservers: string[] }>> {
  return apiFetch(`/api/v1/domains/${domainId}/nameservers`, {
    method: 'PUT',
    body: JSON.stringify({ nameservers }),
  });
}

export async function fetchDomainDnsRecords(domainId: string): Promise<ApiResponse<DomainDnsRecordItem[]>> {
  return apiFetch<DomainDnsRecordItem[]>(`/api/v1/domains/${domainId}/dns`);
}

export async function addDomainDnsRecord(domainId: string, record: {
  type: string;
  name: string;
  content: string;
  ttl?: number;
  priority?: number;
}): Promise<ApiResponse<DomainDnsRecordItem>> {
  return apiFetch(`/api/v1/domains/${domainId}/dns`, {
    method: 'POST',
    body: JSON.stringify(record),
  });
}

export async function deleteDomainDnsRecord(domainId: string, recordId: string): Promise<ApiResponse<{ message: string }>> {
  return apiFetch(`/api/v1/domains/${domainId}/dns/${recordId}`, {
    method: 'DELETE',
  });
}

export async function fetchDomainRegistrarLock(domainId: string): Promise<ApiResponse<{ locked: boolean }>> {
  return apiFetch<{ locked: boolean }>(`/api/v1/domains/${domainId}/lock`);
}

export async function setDomainRegistrarLock(domainId: string, locked: boolean): Promise<ApiResponse<{ locked: boolean }>> {
  return apiFetch(`/api/v1/domains/${domainId}/lock`, {
    method: 'POST',
    body: JSON.stringify({ locked }),
  });
}

export async function fetchDomainEPPCode(domainId: string): Promise<ApiResponse<{ epp_code: string; domain: string }>> {
  return apiFetch<{ epp_code: string; domain: string }>(`/api/v1/domains/${domainId}/epp-code`);
}

export async function fetchDomainContacts(domainId: string): Promise<ApiResponse<{ registrant: DomainContact; admin?: DomainContact; tech?: DomainContact; billing?: DomainContact }>> {
  return apiFetch(`/api/v1/domains/${domainId}/contacts`);
}

// ----------------------------------------------------------------------------
// Admin Domain Reseller API Helper Functions
// ----------------------------------------------------------------------------

export async function fetchDomainAdminMetrics(): Promise<ApiResponse<DomainAdminMetrics>> {
  return apiFetch<DomainAdminMetrics>('/api/v1/admin/domains/metrics');
}

export async function fetchDomainAdminOrders(params?: {
  page?: number;
  limit?: number;
  status?: string;
}): Promise<ApiResponse<DomainOrder[]>> {
  const query = new URLSearchParams();
  if (params?.page) query.set('page', params.page.toString());
  if (params?.limit) query.set('limit', params.limit.toString());
  if (params?.status) query.set('status', params.status);
  const qStr = query.toString() ? `?${query.toString()}` : '';
  return apiFetch<DomainOrder[]>(`/api/v1/admin/domains/orders${qStr}`);
}

export async function retryDomainAdminOrder(orderId: string): Promise<ApiResponse<{ message: string }>> {
  return apiFetch(`/api/v1/admin/domains/orders/${orderId}/retry`, {
    method: 'POST',
  });
}

export async function fetchDomainAdminPrices(): Promise<ApiResponse<DomainAdminPrice[]>> {
  return apiFetch<DomainAdminPrice[]>('/api/v1/admin/domains/prices');
}

export async function updateDomainAdminPrice(id: string, payload: Partial<DomainAdminPrice>): Promise<ApiResponse<DomainAdminPrice>> {
  return apiFetch(`/api/v1/admin/domains/prices/${id}`, {
    method: 'PUT',
    body: JSON.stringify(payload),
  });
}

export async function testResellerClubConnection(): Promise<ApiResponse<ResellerClubTestResult>> {
  return apiFetch<ResellerClubTestResult>('/api/v1/admin/domains/test-connection', {
    method: 'POST',
  });
}

export async function triggerDomainReconciliation(): Promise<ApiResponse<{ message: string }>> {
  return apiFetch('/api/v1/admin/domains/reconcile', {
    method: 'POST',
  });
}

// ==========================================
// USER PLANS, QUOTAS & ADMIN USER MANAGEMENT
// ==========================================

export interface UserPlanOverride {
  user_id: string;
  plan_id?: string;
  max_websites?: number;
  max_databases?: number;
  max_mailboxes?: number;
  max_ftp?: number;
  max_cron?: number;
  max_subdomains?: number;
  disk_space_mb?: number;
  bandwidth_mb?: number;
  permission_terminal?: boolean;
  permission_backups?: boolean;
  permission_dns?: boolean;
  permission_ssl?: boolean;
  permission_file_manager?: boolean;
  permission_cron?: boolean;
  permission_apps?: boolean;
  permission_php_selector?: boolean;
  notes?: string;
}

export interface UserResourceUsage {
  websites_count: number;
  databases_count: number;
  mailboxes_count: number;
  ftp_count: number;
  cron_count: number;
  subdomains_count: number;
  disk_used_mb: number;
  bandwidth_used_mb: number;
}

export interface UserStorageQuotaDetails {
  quota_bytes: number;
  used_bytes: number;
  free_bytes: number;
  usage_percent: number;
  is_unlimited: boolean;
  source: string;
}

export interface EffectiveUserPlan {
  user_id: string;
  user_email: string;
  user_name: string;
  role: string;
  is_active: boolean;
  is_superadmin: boolean;
  plan_id: string;
  plan_name: string;
  plan_slug: string;
  plan_tier: string;
  subscription_status: string;
  max_websites: number;
  max_databases: number;
  max_mailboxes: number;
  max_ftp: number;
  max_cron: number;
  max_subdomains: number;
  disk_space_mb: number;
  bandwidth_mb: number;
  permissions: Record<string, boolean>;
  usage: UserResourceUsage;
  storage?: UserStorageQuotaDetails;
  overrides?: UserPlanOverride;
}

export function formatBytes(bytes: number): string {
  if (bytes <= 0 || isNaN(bytes)) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(k)), sizes.length - 1);
  const num = bytes / Math.pow(k, i);
  return `${num >= 10 || i === 0 ? num.toFixed(0) : num.toFixed(1)} ${sizes[i]}`;
}

export interface AdminUserListItem {
  id: string;
  email: string;
  full_name: string;
  role: string;
  is_active: boolean;
  is_superadmin: boolean;
  last_login_at?: string;
  created_at: string;
  effective_plan?: EffectiveUserPlan;
  has_custom_overrides: boolean;
}

export function getStoredUserRole(): { role: string; isSuperAdmin: boolean } {
  if (typeof window === 'undefined') return { role: 'customer', isSuperAdmin: false };
  try {
    const raw = localStorage.getItem('hostvra_user');
    if (raw) {
      const u = JSON.parse(raw);
      return { role: u.role || 'customer', isSuperAdmin: Boolean(u.is_superadmin) };
    }
    const token = localStorage.getItem('hostvra_access_token');
    if (token) {
      const parts = token.split('.');
      if (parts.length === 3) {
        const payload = JSON.parse(atob(parts[1]));
        return { role: payload.role || 'customer', isSuperAdmin: Boolean(payload.is_superadmin) };
      }
    }
  } catch {}
  return { role: 'customer', isSuperAdmin: false };
}

export async function fetchUserEffectivePlan(refresh?: boolean): Promise<ApiResponse<EffectiveUserPlan>> {
  const url = refresh ? '/api/v1/user/plan?refresh=true' : '/api/v1/user/plan';
  return apiFetch<EffectiveUserPlan>(url);
}

export async function fetchAdminUsers(): Promise<ApiResponse<AdminUserListItem[]>> {
  return apiFetch<AdminUserListItem[]>('/api/v1/admin/users');
}

export async function fetchAdminUserDetail(userId: string): Promise<ApiResponse<{ user: User; effective_plan: EffectiveUserPlan }>> {
  return apiFetch(`/api/v1/admin/users/${userId}`);
}

export async function updateAdminUserPlan(userId: string, planId: string, billingCycle: 'monthly' | 'yearly'): Promise<ApiResponse<{ message: string; effective_plan: EffectiveUserPlan }>> {
  return apiFetch(`/api/v1/admin/users/${userId}/plan`, {
    method: 'PUT',
    body: JSON.stringify({ plan_id: planId, billing_cycle: billingCycle }),
  });
}

export async function updateAdminUserOverrides(userId: string, overrides: Partial<UserPlanOverride>): Promise<ApiResponse<{ message: string; effective_plan: EffectiveUserPlan }>> {
  return apiFetch(`/api/v1/admin/users/${userId}/overrides`, {
    method: 'PUT',
    body: JSON.stringify(overrides),
  });
}

export async function deleteAdminUserOverrides(userId: string): Promise<ApiResponse<{ message: string; effective_plan: EffectiveUserPlan }>> {
  return apiFetch(`/api/v1/admin/users/${userId}/overrides`, {
    method: 'DELETE',
  });
}

export async function updateAdminUserStatus(userId: string, isActive: boolean): Promise<ApiResponse<{ message: string }>> {
  return apiFetch(`/api/v1/admin/users/${userId}/status`, {
    method: 'PUT',
    body: JSON.stringify({ is_active: isActive }),
  });
}

export async function updateAdminUserRole(userId: string, role: string): Promise<ApiResponse<{ message: string; role: string }>> {
  return apiFetch(`/api/v1/admin/users/${userId}/role`, {
    method: 'PUT',
    body: JSON.stringify({ role }),
  });
}

export async function deleteAdminUser(userId: string): Promise<ApiResponse<{ message: string }>> {
  return apiFetch(`/api/v1/admin/users/${userId}`, {
    method: 'DELETE',
  });
}

export async function updateAdminUserEmail(userId: string, email: string): Promise<ApiResponse<{ success: boolean; email: string; message: string }>> {
  return apiFetch(`/api/v1/admin/users/${userId}/email`, {
    method: 'PUT',
    body: JSON.stringify({ email }),
  });
}

export async function updateAdminUserPassword(userId: string, password: string): Promise<ApiResponse<{ success: boolean; message: string }>> {
  return apiFetch(`/api/v1/admin/users/${userId}/password`, {
    method: 'PUT',
    body: JSON.stringify({ password }),
  });
}

export async function impersonateAdminUser(userId: string): Promise<ApiResponse<{
  tokens: { access_token: string; refresh_token: string };
  user: User;
  role: string;
  expires_at: string;
  message: string;
}>> {
  return apiFetch(`/api/v1/admin/users/${userId}/impersonate`, {
    method: 'POST',
  });
}

// WordPress Toolkit API Functions
export async function getWordPressDetails(websiteId: string): Promise<ApiResponse<WordPressDetails>> {
  return apiFetch<WordPressDetails>(`/api/v1/websites/${websiteId}/wordpress`);
}

export async function installWordPress(websiteId: string, payload: {
  site_title?: string;
  admin_user?: string;
  admin_email?: string;
  admin_password?: string;
  db_name?: string;
  install_plugins?: string[];
}): Promise<ApiResponse<WordPressDetails>> {
  return apiFetch<WordPressDetails>(`/api/v1/websites/${websiteId}/wordpress/install`, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function toggleWordPressMaintenance(websiteId: string, enabled: boolean): Promise<ApiResponse<{ success: boolean; maintenance_mode: boolean }>> {
  return apiFetch<{ success: boolean; maintenance_mode: boolean }>(`/api/v1/websites/${websiteId}/wordpress/maintenance`, {
    method: 'POST',
    body: JSON.stringify({ enabled }),
  });
}

export async function toggleWordPressDebug(websiteId: string, enabled: boolean): Promise<ApiResponse<{ success: boolean; debug_mode: boolean }>> {
  return apiFetch<{ success: boolean; debug_mode: boolean }>(`/api/v1/websites/${websiteId}/wordpress/debug`, {
    method: 'POST',
    body: JSON.stringify({ enabled }),
  });
}

export async function flushWordPressCache(websiteId: string): Promise<ApiResponse<{ success: boolean; message: string }>> {
  return apiFetch<{ success: boolean; message: string }>(`/api/v1/websites/${websiteId}/wordpress/cache/flush`, {
    method: 'POST',
  });
}

export async function optimizeWordPressDB(websiteId: string): Promise<ApiResponse<{ success: boolean; message: string }>> {
  return apiFetch<{ success: boolean; message: string }>(`/api/v1/websites/${websiteId}/wordpress/db/optimize`, {
    method: 'POST',
  });
}

export async function resetWordPressPassword(websiteId: string, username: string, newPassword: string): Promise<ApiResponse<{ success: boolean; message: string }>> {
  return apiFetch<{ success: boolean; message: string }>(`/api/v1/websites/${websiteId}/wordpress/reset-password`, {
    method: 'POST',
    body: JSON.stringify({ username, new_password: newPassword }),
  });
}

export async function listWordPressPlugins(websiteId: string): Promise<ApiResponse<WordPressPlugin[]>> {
  return apiFetch<WordPressPlugin[]>(`/api/v1/websites/${websiteId}/wordpress/plugins`);
}

export async function toggleWordPressPlugin(websiteId: string, plugin: string, activate: boolean): Promise<ApiResponse<{ success: boolean; plugin: string; action: string }>> {
  return apiFetch<{ success: boolean; plugin: string; action: string }>(`/api/v1/websites/${websiteId}/wordpress/plugins/toggle`, {
    method: 'POST',
    body: JSON.stringify({ plugin, activate }),
  });
}

export async function installWordPressPlugin(websiteId: string, plugin: string, activate: boolean = true): Promise<ApiResponse<{ success: boolean; plugin: string }>> {
  return apiFetch<{ success: boolean; plugin: string }>(`/api/v1/websites/${websiteId}/wordpress/plugins/install`, {
    method: 'POST',
    body: JSON.stringify({ plugin, activate }),
  });
}

export async function deleteWordPressPlugin(websiteId: string, plugin: string): Promise<ApiResponse<{ success: boolean; plugin: string }>> {
  return apiFetch<{ success: boolean; plugin: string }>(`/api/v1/websites/${websiteId}/wordpress/plugins/${plugin}`, {
    method: 'DELETE',
  });
}

export async function listWordPressThemes(websiteId: string): Promise<ApiResponse<WordPressTheme[]>> {
  return apiFetch<WordPressTheme[]>(`/api/v1/websites/${websiteId}/wordpress/themes`);
}

export async function activateWordPressTheme(websiteId: string, theme: string): Promise<ApiResponse<{ success: boolean; theme: string }>> {
  return apiFetch<{ success: boolean; theme: string }>(`/api/v1/websites/${websiteId}/wordpress/themes/activate`, {
    method: 'POST',
    body: JSON.stringify({ theme }),
  });
}

export async function updateWordPressCore(websiteId: string): Promise<ApiResponse<{ success: boolean; message: string }>> {
  return apiFetch<{ success: boolean; message: string }>(`/api/v1/websites/${websiteId}/wordpress/update-core`, {
    method: 'POST',
  });
}

export async function getPopularWordPressPlugins(): Promise<ApiResponse<PopularPlugin[]>> {
  return apiFetch<PopularPlugin[]>(`/api/v1/wordpress/catalog`);
}

// --- Multi-Server Fleet Management & Cross-Node Migrations ---
export async function toggleServerMaintenance(serverId: string, enabled: boolean): Promise<ApiResponse<Server>> {
  return apiFetch<Server>(`/api/v1/servers/${serverId}/maintenance`, {
    method: 'POST',
    body: JSON.stringify({ enabled }),
  });
}

export async function drainServerNode(serverId: string, enabled: boolean): Promise<ApiResponse<{ server_id: string; draining: boolean; workload_summary: ServerWorkloadSummary; message: string }>> {
  return apiFetch(`/api/v1/servers/${serverId}/drain`, {
    method: 'POST',
    body: JSON.stringify({ enabled }),
  });
}

export async function getServerWorkloads(serverId: string): Promise<ApiResponse<ServerWorkloadSummary>> {
  return apiFetch<ServerWorkloadSummary>(`/api/v1/servers/${serverId}/workloads`);
}

export async function deleteServer(serverId: string, force: boolean = false): Promise<ApiResponse<{ success: boolean; message: string }>> {
  return apiFetch(`/api/v1/servers/${serverId}${force ? '?force=true' : ''}`, {
    method: 'DELETE',
  });
}

export async function rotateServerAgentKey(serverId: string): Promise<ApiResponse<{ server_id: string; agent_key: string; message: string }>> {
  return apiFetch(`/api/v1/servers/${serverId}/rotate-key`, {
    method: 'POST',
  });
}

export async function preflightMigration(payload: {
  source_server_id: string;
  destination_server_id: string;
  website_id?: string;
  database_id?: string;
}): Promise<ApiResponse<PreflightCheckResponse>> {
  return apiFetch<PreflightCheckResponse>('/api/v1/servers/migrations/preflight', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function createCrossNodeMigration(payload: {
  source_server_id: string;
  destination_server_id: string;
  website_id?: string;
  database_id?: string;
}): Promise<ApiResponse<CrossNodeMigration>> {
  return apiFetch<CrossNodeMigration>('/api/v1/servers/migrations', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function listCrossNodeMigrations(): Promise<ApiResponse<CrossNodeMigration[]>> {
  return apiFetch<CrossNodeMigration[]>('/api/v1/servers/migrations');
}

export async function getCrossNodeMigration(migrationId: string): Promise<ApiResponse<CrossNodeMigration>> {
  return apiFetch<CrossNodeMigration>(`/api/v1/servers/migrations/${migrationId}`);
}

export async function cutoverMigration(migrationId: string): Promise<ApiResponse<CrossNodeMigration>> {
  return apiFetch<CrossNodeMigration>(`/api/v1/servers/migrations/${migrationId}/cutover`, {
    method: 'POST',
  });
}

export async function rollbackMigration(migrationId: string): Promise<ApiResponse<CrossNodeMigration>> {
  return apiFetch<CrossNodeMigration>(`/api/v1/servers/migrations/${migrationId}/rollback`, {
    method: 'POST',
  });
}

// ============================================================================
// PHASE THREE: WHMCS/BLESTA WEBHOOKS & RESELLER MULTI-TENANCY ENGINE
// ============================================================================

export interface ProvisioningIntegration {
  id: string;
  organization_id: string;
  provider: 'whmcs' | 'blesta' | 'custom';
  name: string;
  secret_key_hash?: string;
  raw_secret_key?: string;
  default_server_id?: string;
  default_plan_id?: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface ProvisioningWebhookLog {
  id: string;
  organization_id: string;
  integration_id?: string;
  provider: string;
  action: string;
  username?: string;
  domain?: string;
  payload: any;
  status: 'success' | 'failed' | 'replayed';
  http_status: number;
  response_body?: string;
  error_message?: string;
  ip_address?: string;
  created_at: string;
}

export interface ResellerProfile {
  id: string;
  organization_id: string;
  user_id: string;
  max_accounts: number;
  allocated_disk_mb: number;
  allocated_bandwidth_mb: number;
  allow_overselling: boolean;
  brand_name: string;
  brand_logo_url?: string;
  custom_nameservers: string[];
  custom_email_sender: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface ResellerUsageSummary {
  max_accounts: number;
  used_accounts: number;
  allocated_disk_mb: number;
  used_disk_mb: number;
  allocated_bandwidth_mb: number;
  used_bandwidth_mb: number;
  allow_overselling: boolean;
}

export interface BrandingInfo {
  brand_name: string;
  brand_logo_url: string;
  custom_nameservers: string[];
  custom_email_sender: string;
}

export async function listProvisioningIntegrations(): Promise<ApiResponse<ProvisioningIntegration[]>> {
  return apiFetch<ProvisioningIntegration[]>('/api/v1/provisioning/integrations');
}

export async function createProvisioningIntegration(payload: {
  provider: string;
  name: string;
  default_server_id?: string;
  default_plan_id?: string;
  is_active?: boolean;
}): Promise<ApiResponse<ProvisioningIntegration>> {
  return apiFetch<ProvisioningIntegration>('/api/v1/provisioning/integrations', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function deleteProvisioningIntegration(id: string): Promise<ApiResponse<{ status: string }>> {
  return apiFetch<{ status: string }>(`/api/v1/provisioning/integrations/${id}`, {
    method: 'DELETE',
  });
}

export async function listProvisioningWebhookLogs(): Promise<ApiResponse<ProvisioningWebhookLog[]>> {
  return apiFetch<ProvisioningWebhookLog[]>('/api/v1/provisioning/logs');
}

export async function replayWebhookLog(id: string): Promise<ApiResponse<{ status: string; message: string; result: any }>> {
  return apiFetch<{ status: string; message: string; result: any }>(`/api/v1/provisioning/logs/${id}/replay`, {
    method: 'POST',
  });
}

export async function listResellerProfiles(): Promise<ApiResponse<ResellerProfile[]>> {
  return apiFetch<ResellerProfile[]>('/api/v1/reseller/profiles');
}

export async function getResellerProfile(id?: string): Promise<ApiResponse<ResellerProfile>> {
  return apiFetch<ResellerProfile>(`/api/v1/reseller/profile${id ? `/${id}` : ''}`);
}

export async function updateResellerProfile(id: string, payload: Partial<ResellerProfile>): Promise<ApiResponse<ResellerProfile>> {
  return apiFetch<ResellerProfile>(`/api/v1/reseller/profile/${id}`, {
    method: 'PUT',
    body: JSON.stringify(payload),
  });
}

export async function getResellerUsage(userId?: string): Promise<ApiResponse<ResellerUsageSummary>> {
  return apiFetch<ResellerUsageSummary>(`/api/v1/reseller/usage${userId ? `?user_id=${userId}` : ''}`);
}

export async function getBranding(resellerId?: string): Promise<ApiResponse<BrandingInfo>> {
  return apiFetch<BrandingInfo>(`/api/v1/branding${resellerId ? `?reseller_id=${resellerId}` : ''}`);
}

export async function updateResellerBranding(payload: Partial<BrandingInfo>): Promise<ApiResponse<BrandingInfo>> {
  return apiFetch<BrandingInfo>('/api/v1/reseller/branding', {
    method: 'PUT',
    body: JSON.stringify(payload),
  });
}

// ============================================================================
// Developer & Application Management API Functions (Phase 6)
// ============================================================================

export async function getManagedApp(websiteId: string): Promise<ApiResponse<ManagedAppRecord | null>> {
  return apiFetch<ManagedAppRecord | null>(`/api/v1/websites/${websiteId}/app-runtime`);
}

export async function saveManagedApp(websiteId: string, payload: {
  name: string;
  app_type: 'nodejs' | 'python';
  runtime_version: string;
  app_root?: string;
  entrypoint?: string;
  start_command?: string;
  port?: number;
  env_vars?: Record<string, string>;
}): Promise<ApiResponse<ManagedAppRecord>> {
  return apiFetch<ManagedAppRecord>(`/api/v1/websites/${websiteId}/app-runtime`, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function controlManagedApp(websiteId: string, action: 'start' | 'stop' | 'restart'): Promise<ApiResponse<{
  success: boolean;
  action: string;
  status: string;
  metrics?: AppProcessStatus;
}>> {
  return apiFetch<{ success: boolean; action: string; status: string; metrics?: AppProcessStatus }>(
    `/api/v1/websites/${websiteId}/app-runtime/control`,
    {
      method: 'POST',
      body: JSON.stringify({ action }),
    }
  );
}

export async function getManagedAppStatus(websiteId: string): Promise<ApiResponse<AppProcessStatus>> {
  return apiFetch<AppProcessStatus>(`/api/v1/websites/${websiteId}/app-runtime/status`);
}

export async function deleteManagedApp(websiteId: string): Promise<ApiResponse<{ deleted: boolean }>> {
  return apiFetch<{ deleted: boolean }>(`/api/v1/websites/${websiteId}/app-runtime`, {
    method: 'DELETE',
  });
}

export async function detectRuntimes(): Promise<ApiResponse<RuntimeInfo[]>> {
  return apiFetch<RuntimeInfo[]>('/api/v1/runtimes');
}

export async function getGitRepo(websiteId: string): Promise<ApiResponse<GitRepositoryRecord | null>> {
  return apiFetch<GitRepositoryRecord | null>(`/api/v1/websites/${websiteId}/git`);
}

export async function saveGitRepo(websiteId: string, payload: {
  provider?: string;
  repo_url: string;
  branch: string;
  auto_deploy: boolean;
  build_command?: string;
  deploy_path?: string;
  release_keep_count?: number;
}): Promise<ApiResponse<GitRepositoryRecord>> {
  return apiFetch<GitRepositoryRecord>(`/api/v1/websites/${websiteId}/git`, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function deleteGitRepo(websiteId: string): Promise<ApiResponse<{ deleted: boolean }>> {
  return apiFetch<{ deleted: boolean }>(`/api/v1/websites/${websiteId}/git`, {
    method: 'DELETE',
  });
}

export async function triggerGitDeploy(websiteId: string, branch?: string): Promise<ApiResponse<GitDeploymentRecord>> {
  return apiFetch<GitDeploymentRecord>(`/api/v1/websites/${websiteId}/git/deploy`, {
    method: 'POST',
    body: JSON.stringify({ branch }),
  });
}

export async function listGitDeployments(websiteId: string): Promise<ApiResponse<GitDeploymentRecord[]>> {
  return apiFetch<GitDeploymentRecord[]>(`/api/v1/websites/${websiteId}/git/deployments`);
}

export async function rollbackGitDeployment(websiteId: string, deploymentId: string): Promise<ApiResponse<GitDeploymentRecord>> {
  return apiFetch<GitDeploymentRecord>(`/api/v1/websites/${websiteId}/git/rollback/${deploymentId}`, {
    method: 'POST',
  });
}

export async function getWordPressStaging(websiteId: string): Promise<ApiResponse<WordPressStagingRecord | null>> {
  return apiFetch<WordPressStagingRecord | null>(`/api/v1/websites/${websiteId}/staging`);
}

export async function createWordPressStaging(websiteId: string, payload: {
  staging_subdomain?: string;
  auth_user?: string;
  auth_password?: string;
}): Promise<ApiResponse<{
  success: boolean;
  staging_domain: string;
  staging_root: string;
  auth_user: string;
  auth_password?: string;
  message: string;
}>> {
  return apiFetch<{
    success: boolean;
    staging_domain: string;
    staging_root: string;
    auth_user: string;
    auth_password?: string;
    message: string;
  }>(`/api/v1/websites/${websiteId}/staging`, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function pushWordPressStagingToLive(websiteId: string): Promise<ApiResponse<{
  success: boolean;
  backup_path: string;
  completed_at: string;
  message: string;
}>> {
  return apiFetch<{
    success: boolean;
    backup_path: string;
    completed_at: string;
    message: string;
  }>(`/api/v1/websites/${websiteId}/staging/push-to-live`, {
    method: 'POST',
  });
}

export async function deleteWordPressStaging(websiteId: string): Promise<ApiResponse<{ deleted: boolean }>> {
  return apiFetch<{ deleted: boolean }>(`/api/v1/websites/${websiteId}/staging`, {
    method: 'DELETE',
  });
}

// ============================================================================
// PHASE SEVEN: CACHING ENGINE API FUNCTIONS
// ============================================================================

export async function getCacheServices(): Promise<ApiResponse<CacheServiceStatus[]>> {
  return apiFetch<CacheServiceStatus[]>('/api/v1/caching/services');
}

export async function getWebsiteCacheConfig(websiteId: string): Promise<ApiResponse<WebsiteCacheConfig>> {
  return apiFetch<WebsiteCacheConfig>(`/api/v1/websites/${websiteId}/caching`);
}

export async function saveWebsiteCacheConfig(websiteId: string, payload: Partial<WebsiteCacheConfig>): Promise<ApiResponse<WebsiteCacheConfig>> {
  return apiFetch<WebsiteCacheConfig>(`/api/v1/websites/${websiteId}/caching`, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function disableWebsiteCache(websiteId: string): Promise<ApiResponse<{ message: string }>> {
  return apiFetch<{ message: string }>(`/api/v1/websites/${websiteId}/caching`, {
    method: 'DELETE',
  });
}

export async function getWebsiteCacheMetrics(websiteId: string): Promise<ApiResponse<CacheMetrics>> {
  return apiFetch<CacheMetrics>(`/api/v1/websites/${websiteId}/caching/metrics`);
}

export async function flushWebsiteCache(websiteId: string): Promise<ApiResponse<{ message: string }>> {
  return apiFetch<{ message: string }>(`/api/v1/websites/${websiteId}/caching/flush`, {
    method: 'POST',
  });
}

export async function benchmarkWebsiteCache(websiteId: string, iterations = 30): Promise<ApiResponse<CacheBenchmarkResult>> {
  return apiFetch<CacheBenchmarkResult>(`/api/v1/websites/${websiteId}/caching/benchmark`, {
    method: 'POST',
    body: JSON.stringify({ iterations }),
  });
}

export async function toggleWordPressObjectCache(websiteId: string, enabled: boolean): Promise<ApiResponse<WebsiteCacheConfig>> {
  return apiFetch<WebsiteCacheConfig>(`/api/v1/websites/${websiteId}/caching/wp-object-cache`, {
    method: 'POST',
    body: JSON.stringify({ enabled }),
  });
}

// ============================================================================
// PHASE SEVEN: REMOTE DATABASE ACCESS API FUNCTIONS
// ============================================================================

export async function listDatabaseRemoteAccess(databaseId: string): Promise<ApiResponse<DatabaseRemoteAccess[]>> {
  return apiFetch<DatabaseRemoteAccess[]>(`/api/v1/databases/${databaseId}/remote-access`);
}

export async function grantDatabaseRemoteAccess(databaseId: string, payload: {
  db_user?: string;
  authorized_ip: string;
  privileges?: string;
  allow_unrestricted_wildcard?: boolean;
  notes?: string;
  expires_at?: string;
}): Promise<ApiResponse<DatabaseRemoteAccess>> {
  return apiFetch<DatabaseRemoteAccess>(`/api/v1/databases/${databaseId}/remote-access`, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function revokeDatabaseRemoteAccess(databaseId: string, grantId: string): Promise<ApiResponse<{ message: string }>> {
  return apiFetch<{ message: string }>(`/api/v1/databases/${databaseId}/remote-access/${grantId}`, {
    method: 'DELETE',
  });
}

export async function testDatabaseRemoteAccess(databaseId: string, payload: {
  host: string;
  port?: number;
  db_user: string;
  password?: string;
}): Promise<ApiResponse<{ success: boolean; latency_ms: number; message: string; error?: string }>> {
  return apiFetch<{ success: boolean; latency_ms: number; message: string; error?: string }>(`/api/v1/databases/${databaseId}/remote-access/test`, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

// ============================================================================
// PHASE SEVEN: DATABASE MAINTENANCE API FUNCTIONS
// ============================================================================

export async function getDatabaseHealth(databaseId: string): Promise<ApiResponse<DatabaseHealthReport>> {
  return apiFetch<DatabaseHealthReport>(`/api/v1/databases/${databaseId}/maintenance/health`);
}

export async function checkDatabaseIntegrity(databaseId: string, tableNames?: string[]): Promise<ApiResponse<DatabaseMaintenanceResult>> {
  return apiFetch<DatabaseMaintenanceResult>(`/api/v1/databases/${databaseId}/maintenance/check`, {
    method: 'POST',
    body: JSON.stringify({ table_names: tableNames || [] }),
  });
}

export async function optimizeDatabase(databaseId: string, tableNames?: string[]): Promise<ApiResponse<DatabaseMaintenanceResult>> {
  return apiFetch<DatabaseMaintenanceResult>(`/api/v1/databases/${databaseId}/maintenance/optimize`, {
    method: 'POST',
    body: JSON.stringify({ table_names: tableNames || [] }),
  });
}

export async function analyzeDatabase(databaseId: string, tableNames?: string[]): Promise<ApiResponse<DatabaseMaintenanceResult>> {
  return apiFetch<DatabaseMaintenanceResult>(`/api/v1/databases/${databaseId}/maintenance/analyze`, {
    method: 'POST',
    body: JSON.stringify({ table_names: tableNames || [] }),
  });
}

export async function listDatabaseMaintenanceLogs(databaseId: string, limit = 30): Promise<ApiResponse<DatabaseMaintenanceLog[]>> {
  return apiFetch<DatabaseMaintenanceLog[]>(`/api/v1/databases/${databaseId}/maintenance/logs?limit=${limit}`);
}

// ============================================================================
// PHASE EIGHT: EMAIL DELIVERABILITY & RELAY API FUNCTIONS
// ============================================================================

export async function listEmailRelays(domainId?: string): Promise<ApiResponse<EmailSMTPRelay[]>> {
  const q = domainId ? `?domain_id=${encodeURIComponent(domainId)}` : '';
  return apiFetch<EmailSMTPRelay[]>(`/api/v1/email/relays${q}`);
}

export async function getEmailRelay(id: string): Promise<ApiResponse<EmailSMTPRelay>> {
  return apiFetch<EmailSMTPRelay>(`/api/v1/email/relays/${id}`);
}

export async function createEmailRelay(payload: {
  domain_id?: string;
  name: string;
  provider: string;
  host: string;
  port: number;
  username: string;
  password?: string;
  auth_type?: string;
  encryption?: string;
  sender_domain: string;
  rate_limit_per_second?: number;
  rate_limit_per_day?: number;
  test_on_save?: boolean;
}): Promise<ApiResponse<EmailSMTPRelay>> {
  return apiFetch<EmailSMTPRelay>('/api/v1/email/relays', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function updateEmailRelay(id: string, payload: Partial<{
  domain_id?: string;
  name: string;
  provider: string;
  host: string;
  port: number;
  username: string;
  password?: string;
  auth_type?: string;
  encryption?: string;
  sender_domain: string;
  rate_limit_per_second?: number;
  rate_limit_per_day?: number;
}>): Promise<ApiResponse<EmailSMTPRelay>> {
  return apiFetch<EmailSMTPRelay>(`/api/v1/email/relays/${id}`, {
    method: 'PUT',
    body: JSON.stringify(payload),
  });
}

export async function deleteEmailRelay(id: string): Promise<ApiResponse<{ success: boolean; message: string }>> {
  return apiFetch<{ success: boolean; message: string }>(`/api/v1/email/relays/${id}`, {
    method: 'DELETE',
  });
}

export async function testEmailRelay(id: string): Promise<ApiResponse<RelayConnectionTestResult>> {
  return apiFetch<RelayConnectionTestResult>(`/api/v1/email/relays/${id}/test`, {
    method: 'POST',
  });
}

export async function verifyDomainDeliverability(id: string): Promise<ApiResponse<DeliverabilityAuditResult>> {
  return apiFetch<DeliverabilityAuditResult>(`/api/v1/email/relays/${id}/verify-dns`, {
    method: 'POST',
  });
}

// ============================================================================
// PHASE EIGHT: ANTI-SPAM FILTER & QUARANTINE API FUNCTIONS
// ============================================================================

export async function getEmailSpamSettings(domainId?: string): Promise<ApiResponse<EmailSpamSettings>> {
  const q = domainId ? `?domain_id=${encodeURIComponent(domainId)}` : '';
  return apiFetch<EmailSpamSettings>(`/api/v1/email/spam/settings${q}`);
}

export async function updateEmailSpamSettings(payload: Partial<EmailSpamSettings>): Promise<ApiResponse<EmailSpamSettings>> {
  return apiFetch<EmailSpamSettings>('/api/v1/email/spam/settings', {
    method: 'PUT',
    body: JSON.stringify(payload),
  });
}

export async function getEmailSpamTelemetry(): Promise<ApiResponse<RspamdTelemetry>> {
  return apiFetch<RspamdTelemetry>('/api/v1/email/spam/telemetry');
}

export async function listSpamQuarantine(domainId?: string): Promise<ApiResponse<SpamQuarantineItem[]>> {
  const q = domainId ? `?domain_id=${encodeURIComponent(domainId)}` : '';
  return apiFetch<SpamQuarantineItem[]>(`/api/v1/email/spam/quarantine${q}`);
}

export async function releaseQuarantineItem(id: string): Promise<ApiResponse<{ success: boolean; error?: string }>> {
  return apiFetch<{ success: boolean; error?: string }>(`/api/v1/email/spam/quarantine/${id}/release`, {
    method: 'POST',
  });
}

export async function learnHamQuarantineItem(id: string): Promise<ApiResponse<{ success: boolean; message: string }>> {
  return apiFetch<{ success: boolean; message: string }>(`/api/v1/email/spam/quarantine/${id}/learn-ham`, {
    method: 'POST',
  });
}

export async function deleteQuarantineItem(id: string): Promise<ApiResponse<{ success: boolean; message: string }>> {
  return apiFetch<{ success: boolean; message: string }>(`/api/v1/email/spam/quarantine/${id}`, {
    method: 'DELETE',
  });
}

// ============================================================================
// PHASE EIGHT: LIVE IMAP MAILBOX MIGRATION API FUNCTIONS
// ============================================================================

export async function listIMAPMigrations(): Promise<ApiResponse<IMAPMigration[]>> {
  return apiFetch<IMAPMigration[]>('/api/v1/email/migrations');
}

export async function testSourceIMAP(opts: SourceConnectionOptions): Promise<ApiResponse<IMAPTestResult>> {
  return apiFetch<IMAPTestResult>('/api/v1/email/migrations/test-source', {
    method: 'POST',
    body: JSON.stringify(opts),
  });
}

export async function startIMAPMigration(payload: {
  destination_mailbox_id: string;
  source: SourceConnectionOptions;
}): Promise<ApiResponse<IMAPMigration>> {
  return apiFetch<IMAPMigration>('/api/v1/email/migrations', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function getIMAPMigration(id: string): Promise<ApiResponse<IMAPMigration>> {
  return apiFetch<IMAPMigration>(`/api/v1/email/migrations/${id}`);
}

export async function cancelIMAPMigration(id: string): Promise<ApiResponse<{ success: boolean; message: string }>> {
  return apiFetch<{ success: boolean; message: string }>(`/api/v1/email/migrations/${id}/cancel`, {
    method: 'POST',
  });
}

export async function deleteIMAPMigration(id: string): Promise<ApiResponse<{ success: boolean; message: string }>> {
  return apiFetch<{ success: boolean; message: string }>(`/api/v1/email/migrations/${id}`, {
    method: 'DELETE',
  });
}

// ============================================================================
// PHASE NINE: INCREMENTAL BACKUPS, REPOSITORIES & DISASTER RECOVERY
// ============================================================================

export interface BackupRepository {
  id: string;
  organization_id: string;
  destination_id?: string;
  name: string;
  provider: 'restic' | 'borg' | 'native';
  repo_uri: string;
  encryption_algorithm: string;
  key_id?: string;
  is_initialized: boolean;
  last_checked_at?: string;
  integrity_status: 'healthy' | 'checking' | 'error' | 'unknown';
  total_snapshots: number;
  total_size_bytes: number;
  deduplicated_size_bytes: number;
  compression_ratio: number;
  deletion_protection: boolean;
  created_at: string;
  updated_at: string;
}

export interface BackupIncrementalSnapshot {
  id: string;
  repository_id: string;
  organization_id: string;
  snapshot_hash: string;
  parent_snapshot_id?: string;
  scope: 'website' | 'database' | 'mail' | 'full_account';
  target_name: string;
  paths_included: string;
  paths_excluded: string;
  files_new: number;
  files_changed: number;
  files_unmodified: number;
  total_files: number;
  raw_size_bytes: number;
  stored_size_bytes: number;
  database_snapshot_consistent: boolean;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'pruned';
  integrity_status: string;
  error_message?: string;
  started_at: string;
  completed_at?: string;
  created_at: string;
}

export interface BackupRetentionPolicy {
  id: string;
  repository_id: string;
  organization_id: string;
  name: string;
  keep_last: number;
  keep_hourly: number;
  keep_daily: number;
  keep_weekly: number;
  keep_monthly: number;
  keep_yearly: number;
  prune_schedule: string;
  is_active: boolean;
  last_pruned_at?: string;
  pruned_snapshots_count: number;
  bytes_reclaimed: number;
}

export interface SnapshotFileNode {
  name: string;
  path: string;
  type: 'file' | 'dir';
  size_bytes: number;
  mode: string;
  modified_at: string;
}

export interface DisasterRecoveryStep {
  step_number: number;
  title: string;
  description: string;
  commands: string[];
  safety_note?: string;
}

export interface DisasterRecoveryRunbook {
  title: string;
  generated_at: string;
  summary: string;
  steps: DisasterRecoveryStep[];
  markdown: string;
}

export interface DestinationTestResult {
  success: boolean;
  auth_passed: boolean;
  upload_passed: boolean;
  listing_passed: boolean;
  restore_passed: boolean;
  deletion_passed: boolean;
  latency_ms: number;
  error_message?: string;
  tested_at: string;
}

export interface IncrementalRestoreResult {
  success: boolean;
  files_restored: number;
  bytes_restored: number;
  database_restored?: boolean;
  mailboxes_restored?: boolean;
  target_directory: string;
  error_message?: string;
  completed_at: string;
}

export interface PruneResult {
  snapshots_kept: number;
  snapshots_removed: number;
  bytes_reclaimed: number;
  success: boolean;
  error_message?: string;
  completed_at: string;
}

export interface RepositoryIntegrityResult {
  healthy: boolean;
  total_snapshots: number;
  total_packs: number;
  total_size_bytes: number;
  errors_found?: string[];
  duration_seconds: number;
  checked_at: string;
}

// Client API Methods for Incremental Backups
export async function listBackupRepositories(): Promise<ApiResponse<BackupRepository[]>> {
  return apiFetch<BackupRepository[]>('/api/v1/backups/repositories');
}

export async function createBackupRepository(data: {
  name: string;
  provider: 'restic' | 'borg' | 'native';
  destination_type: string;
  repo_uri?: string;
  storage_config?: Record<string, string>;
  encryption_algorithm?: string;
  password?: string;
  deletion_protection?: boolean;
}): Promise<ApiResponse<BackupRepository>> {
  return apiFetch<BackupRepository>('/api/v1/backups/repositories', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function getBackupRepository(id: string): Promise<ApiResponse<BackupRepository>> {
  return apiFetch<BackupRepository>(`/api/v1/backups/repositories/${id}`);
}

export async function updateBackupRepository(id: string, data: {
  name?: string;
  deletion_protection?: boolean;
}): Promise<ApiResponse<BackupRepository>> {
  return apiFetch<BackupRepository>(`/api/v1/backups/repositories/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  });
}

export async function deleteBackupRepository(id: string): Promise<ApiResponse<{ status: string; repo_id: string }>> {
  return apiFetch<{ status: string; repo_id: string }>(`/api/v1/backups/repositories/${id}`, {
    method: 'DELETE',
  });
}

export async function checkRepositoryIntegrity(id: string): Promise<ApiResponse<RepositoryIntegrityResult>> {
  return apiFetch<RepositoryIntegrityResult>(`/api/v1/backups/repositories/${id}/check`, {
    method: 'POST',
  });
}

export async function pruneRepository(id: string, dryRun: boolean = false): Promise<ApiResponse<PruneResult>> {
  return apiFetch<PruneResult>(`/api/v1/backups/repositories/${id}/prune`, {
    method: 'POST',
    body: JSON.stringify({ dry_run: dryRun }),
  });
}

export async function getRepositoryRetention(id: string): Promise<ApiResponse<BackupRetentionPolicy>> {
  return apiFetch<BackupRetentionPolicy>(`/api/v1/backups/repositories/${id}/retention`);
}

export async function saveRepositoryRetention(id: string, policy: Partial<BackupRetentionPolicy>): Promise<ApiResponse<BackupRetentionPolicy>> {
  return apiFetch<BackupRetentionPolicy>(`/api/v1/backups/repositories/${id}/retention`, {
    method: 'POST',
    body: JSON.stringify(policy),
  });
}

export async function listIncrementalSnapshots(repoId?: string): Promise<ApiResponse<BackupIncrementalSnapshot[]>> {
  const url = repoId ? `/api/v1/backups/snapshots?repository_id=${encodeURIComponent(repoId)}` : '/api/v1/backups/snapshots';
  return apiFetch<BackupIncrementalSnapshot[]>(url);
}

export async function createIncrementalSnapshot(data: {
  repository_id: string;
  scope: 'website' | 'database' | 'mail' | 'full_account';
  target_name?: string;
  include_paths?: string[];
  exclude_paths?: string[];
  database_type?: string;
  database_name?: string;
  database_user?: string;
  database_password?: string;
}): Promise<ApiResponse<BackupIncrementalSnapshot>> {
  return apiFetch<BackupIncrementalSnapshot>('/api/v1/backups/snapshots', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function getSnapshotFileTree(snapshotId: string, path: string = ''): Promise<ApiResponse<SnapshotFileNode[]>> {
  const q = path ? `?path=${encodeURIComponent(path)}` : '';
  return apiFetch<SnapshotFileNode[]>(`/api/v1/backups/snapshots/${snapshotId}/tree${q}`);
}

export async function restoreSnapshotFiles(snapshotId: string, data: {
  selected_paths: string[];
  target_directory?: string;
  restore_mode?: 'in_place' | 'alternate';
}): Promise<ApiResponse<IncrementalRestoreResult>> {
  return apiFetch<IncrementalRestoreResult>(`/api/v1/backups/snapshots/${snapshotId}/restore-file`, {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function restoreFullAccountSnapshot(snapshotId: string, data: {
  target_directory?: string;
  restore_database?: boolean;
  database_name?: string;
  database_user?: string;
  database_password?: string;
  restore_mailboxes?: boolean;
  dry_run?: boolean;
}): Promise<ApiResponse<IncrementalRestoreResult>> {
  return apiFetch<IncrementalRestoreResult>(`/api/v1/backups/snapshots/${snapshotId}/restore-full`, {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function getDisasterRecoveryRunbook(repositoryId: string, targetHostname?: string): Promise<ApiResponse<DisasterRecoveryRunbook>> {
  return apiFetch<DisasterRecoveryRunbook>('/api/v1/backups/disaster-recovery/runbook', {
    method: 'POST',
    body: JSON.stringify({
      repository_id: repositoryId,
      target_hostname: targetHostname || 'recovery-host.hostvra.internal',
    }),
  });
}

export async function testBackupDestinationConfig(cfg: {
  type: string;
  endpoint?: string;
  region?: string;
  bucket?: string;
  access_key?: string;
  secret_key?: string;
  prefix?: string;
  local_path?: string;
}): Promise<ApiResponse<DestinationTestResult>> {
  return apiFetch<DestinationTestResult>('/api/v1/backups/destinations/verify', {
    method: 'POST',
    body: JSON.stringify(cfg),
  });
}
