export function getApiBaseUrl(): string {
  if (typeof window !== 'undefined') {
    if (process.env.NEXT_PUBLIC_API_URL) {
      return process.env.NEXT_PUBLIC_API_URL;
    }
    return '';
  }
  return process.env.INTERNAL_API_URL || process.env.NEXT_PUBLIC_API_URL || 'http://127.0.0.1:8080';
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
  last_heartbeat_at?: string;
  uptime_seconds?: number;
  created_at: string;
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
  app_type: 'php' | 'static' | 'proxy' | 'nodejs' | 'python' | 'go';
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
  }
}

export function clearStoredAuth() {
  if (typeof window !== 'undefined') {
    localStorage.removeItem('hostvra_access_token');
    localStorage.removeItem('hostvra_user');
  }
}

export async function apiFetch<T>(endpoint: string, options: RequestInit = {}): Promise<ApiResponse<T>> {
  const token = getStoredToken();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string> || {}),
  };

  if (token && !headers['Authorization']) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  try {
    const res = await fetch(`${getApiBaseUrl()}${endpoint}`, {
      ...options,
      headers,
    });

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
    return {
      success: false,
      error: {
        code: 'NETWORK_ERROR',
        message: err.message || 'Unable to connect to Hostvra API server',
      },
    };
  }
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
  type: 'website' | 'database' | 'full_config';
  target_name: string;
  storage: 'local' | 's3' | 'r2' | 'b2' | 'sftp';
  storage_id?: string;
  size_bytes: number;
  sha256?: string;
  item_count?: number;
  status: 'completed' | 'failed' | 'in_progress';
  file_name: string;
  remote_key?: string;
  created_at: string;
  completed_at?: string;
  error_message?: string;
}

export interface BackupDestination {
  id: string;
  name: string;
  type: 's3' | 'r2' | 'b2' | 'sftp' | 'local';
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
  scope: 'website' | 'database' | 'full_config';
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

export interface ResourceLimits {
  memory_max_mb: number;
  cpu_quota: number;
  tasks_max: number;
  io_read_mbps?: number;
  io_write_mbps?: number;
  open_basedir: boolean;
}

export interface UserIsolationInfo {
  username: string;
  group_name: string;
  uid: number;
  gid: number;
  home_dir: string;
  document_root: string;
  php_version: string;
  php_pool_socket: string;
  php_pool_config: string;
  slice_name: string;
  limits: ResourceLimits;
  memory_used_mb: number;
  cpu_usage_perc: number;
  tasks_current: number;
  is_systemd_slice: boolean;
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

export interface HostingPlan {
  id: string;
  name: string;
  slug: string;
  description: string;
  tier: 'starter' | 'business' | 'enterprise' | 'reseller';
  price_monthly: number;
  price_yearly: number;
  currency: string;
  disk_space_mb: number;
  bandwidth_mb: number;
  max_websites: number;
  max_databases: number;
  max_mailboxes: number;
  max_ftp: number;
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
  status: 'active' | 'pending' | 'suspended' | 'cancelled' | 'expired';
  billing_cycle: 'monthly' | 'yearly';
  amount: number;
  currency: string;
  disk_used_mb: number;
  bandwidth_used_mb: number;
  websites_count: number;
  next_billing_date: string;
  auto_renew: boolean;
  created_at: string;
  updated_at: string;
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
  ip_address?: string;
  php_version?: string;
  ssl_active: boolean;
  suspended_at?: string;
  created_at: string;
  updated_at: string;
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
