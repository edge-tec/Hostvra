export function getApiBaseUrl(): string {
  if (typeof window !== 'undefined' && window.location.hostname && window.location.hostname !== 'localhost' && window.location.hostname !== '127.0.0.1') {
    return `${window.location.protocol}//${window.location.hostname}:8080`;
  }
  return process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
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

    const data: ApiResponse<T> = await res.json();
    if (!res.ok && !data.error) {
      return {
        success: false,
        error: {
          code: 'HTTP_ERROR',
          message: `Request failed with status ${res.status}`,
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

