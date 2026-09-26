'use client';

import React, { useState, useEffect, useMemo } from 'react';
import { DashboardShell } from '@/components/DashboardShell';
import { WebmailClient } from '@/components/WebmailClient';
import { apiFetch } from '@/lib/api';
import {
  Mail,
  Plus,
  ShieldCheck,
  ShieldAlert,
  Server,
  Key,
  ExternalLink,
  Trash2,
  RefreshCw,
  Copy,
  Check,
  Search,
  HardDrive,
  Inbox,
  Send,
  AlertTriangle,
  Lock,
  Globe,
  Sliders,
  Sparkles,
  Layers,
  X,
  Activity,
  Terminal,
  RotateCw,
  CheckCircle2,
  XCircle,
  HelpCircle,
  Cpu,
  Radio,
  FileText,
  UserCheck,
  BookOpen,
  Download,
  UserX,
} from 'lucide-react';
import { EmailUserGuide } from '@/components/EmailUserGuide';

interface EmailDomain {
  id: string;
  domain: string;
  mail_hostname: string;
  status: string;
  storage_limit_bytes: number;
  storage_used_bytes: number;
  mailbox_count: number;
  alias_count: number;
  dkim_selector: string;
  created_at: string;
}

interface EmailMailbox {
  id: string;
  domain_id: string;
  email: string;
  local_part: string;
  name: string;
  quota_bytes: number;
  used_bytes: number;
  is_active: boolean;
  is_suspended: boolean;
}

interface QueueItem {
  id: string;
  size_bytes: number;
  arrival_time: string;
  sender: string;
  recipients: string[];
  reason: string;
  status: string;
}

interface DeliveryLog {
  id: string;
  timestamp: string;
  sender: string;
  recipient: string;
  status: string;
  message_id: string;
  spam_score: number;
  relay: string;
  delay_seconds: number;
  details: string;
}

interface MailServer {
  id: string;
  name: string;
  hostname: string;
  primary_domain: string;
  additional_domains: string[];
  ipv4_address: string;
  ipv6_address?: string;
  timezone: string;
  storage_location: string;
  mailbox_storage_limit_bytes: number;
  max_mailbox_size_bytes: number;
  max_attachment_size_bytes: number;
  smtp_port: number;
  smtp_submission_port: number;
  smtps_port: number;
  imap_port: number;
  imaps_port: number;
  pop3_port: number;
  pop3s_port: number;
  tls_enabled: boolean;
  spam_filter_enabled: boolean;
  antivirus_enabled: boolean;
  dkim_enabled: boolean;
  spf_enabled: boolean;
  dmarc_enabled: boolean;
  webmail_enabled: boolean;
  auto_ssl_enabled: boolean;
  backup_enabled: boolean;
  status: string;
  health_status: string;
  rate_limit_per_mailbox_hr: number;
  rate_limit_per_domain_hr: number;
  rate_limit_per_ip_hr: number;
  auth_failure_threshold: number;
  domain_count?: number;
  mailbox_count?: number;
  created_at: string;
  provisioning_logs?: string;
}

interface PreflightCheckItem {
  name: string;
  category: string;
  status: 'passed' | 'warning' | 'failed';
  message: string;
  details?: string;
}

interface PreflightResult {
  passed: boolean;
  checked_at: string;
  hostname: string;
  checks: PreflightCheckItem[];
}

interface SuppressionItem {
  id: string;
  email: string;
  reason: string;
  details: string;
  created_at: string;
}

interface ServiceStatus {
  name: string;
  status: string;
  uptime_seconds: number;
  memory_bytes: number;
  active_ports: number[];
  version: string;
}

interface DNSRecordItem {
  record_type: string;
  host: string;
  expected: string;
  status: string;
  message: string;
}

interface DNSVerificationResult {
  domain: string;
  mx_valid: boolean;
  spf_valid: boolean;
  dkim_valid: boolean;
  dmarc_valid: boolean;
  mx_records: string[];
  spf_record: string;
  dkim_record: string;
  dmarc_record: string;
  issues: string[];
}

interface HealthDeduction {
  reason: string;
  points: number;
  remediation: string;
}

interface HealthCheck {
  name: string;
  passed: boolean;
  severity: string;
  detail: string;
  remediation: string;
}

interface DeliverabilityHealth {
  score: number;
  rating: string;
  mx_valid: boolean;
  spf_valid: boolean;
  dkim_valid: boolean;
  dmarc_valid: boolean;
  ptr_valid: boolean;
  fcrdns_valid: boolean;
  tls_valid: boolean;
  open_relay_rejected: boolean;
  checks: HealthCheck[];
  deductions: HealthDeduction[];
}

interface SMTPSettings {
  domain: string;
  mail_hostname: string;
  incoming_server: string;
  incoming_imap_port: number;
  incoming_pop3_port: number;
  outgoing_server: string;
  outgoing_smtp_submission_port: number;
  outgoing_smtp_ssl_port: number;
  require_tls: boolean;
  require_auth: boolean;
}

interface MailboxTestResult {
  mailbox_id: string;
  email: string;
  smtp_auth_passed: boolean;
  imap_auth_passed: boolean;
  smtp_latency_ms: number;
  imap_latency_ms: number;
  error?: string;
  details: string;
}

interface TestEmailResult {
  success: boolean;
  message_id: string;
  recipient: string;
  latency_ms: number;
  transcript: string[];
  error?: string;
}

function formatBytes(bytes?: number): string {
  if (!bytes || bytes <= 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

export default function EmailHostingPage() {
  const [activeTab, setActiveTab] = useState<
    'servers' | 'mailboxes' | 'webmail' | 'domains' | 'health' | 'smtp' | 'queue' | 'logs' | 'suppressions' | 'services' | 'tester' | 'guide'
  >('servers');

  const [selectedWebmailEmail, setSelectedWebmailEmail] = useState<string | undefined>(undefined);
  const [domains, setDomains] = useState<EmailDomain[]>([]);
  const [mailboxes, setMailboxes] = useState<EmailMailbox[]>([]);
  const [mailServers, setMailServers] = useState<MailServer[]>([]);
  const [mailServersLoading, setMailServersLoading] = useState(false);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [copiedKey, setCopiedKey] = useState<string | null>(null);

  // Mail Server Modal & Provisioning States
  const [showCreateServerModal, setShowCreateServerModal] = useState(false);
  const [serverLogsModal, setServerLogsModal] = useState<MailServer | null>(null);
  const [preflightLoading, setPreflightLoading] = useState(false);
  const [preflightResult, setPreflightResult] = useState<PreflightResult | null>(null);
  const [creatingServer, setCreatingServer] = useState(false);
  const [serverCreationError, setServerCreationError] = useState<string | null>(null);
  const [serverWizardTab, setServerWizardTab] = useState<'general' | 'storage_ports' | 'security' | 'ratelimits' | 'preflight'>('general');

  // New Server Form States
  const [serverName, setServerName] = useState('Primary Mail Node');
  const [serverHostname, setServerHostname] = useState('mail.example.com');
  const [serverPrimaryDomain, setServerPrimaryDomain] = useState('example.com');
  const [serverAdditionalDomains, setServerAdditionalDomains] = useState('');
  const [serverIPv4, setServerIPv4] = useState('127.0.0.1');
  const [serverIPv6, setServerIPv6] = useState('');
  const [serverTimezone, setServerTimezone] = useState('UTC');
  const [serverStorageLocation, setServerStorageLocation] = useState('/var/mail/vhosts');
  const [serverMailboxStorageLimitGB, setServerMailboxStorageLimitGB] = useState(100);
  const [serverMaxMailboxSizeGB, setServerMaxMailboxSizeGB] = useState(10);
  const [serverMaxAttachmentSizeMB, setServerMaxAttachmentSizeMB] = useState(50);
  const [serverSmtpPort, setServerSmtpPort] = useState(25);
  const [serverSmtpSubmissionPort, setServerSmtpSubmissionPort] = useState(587);
  const [serverSmtpsPort, setServerSmtpsPort] = useState(465);
  const [serverImapPort, setServerImapPort] = useState(143);
  const [serverImapsPort, setServerImapsPort] = useState(993);
  const [serverPop3Port, setServerPop3Port] = useState(110);
  const [serverPop3sPort, setServerPop3sPort] = useState(995);
  const [serverTlsEnabled, setServerTlsEnabled] = useState(true);
  const [serverSpamFilterEnabled, setServerSpamFilterEnabled] = useState(true);
  const [serverAntivirusEnabled, setServerAntivirusEnabled] = useState(false);
  const [serverDkimEnabled, setServerDkimEnabled] = useState(true);
  const [serverSpfEnabled, setServerSpfEnabled] = useState(true);
  const [serverDmarcEnabled, setServerDmarcEnabled] = useState(true);
  const [serverWebmailEnabled, setServerWebmailEnabled] = useState(true);
  const [serverAutoSslEnabled, setServerAutoSslEnabled] = useState(true);
  const [serverBackupEnabled, setServerBackupEnabled] = useState(true);
  const [serverRateLimitMailboxHr, setServerRateLimitMailboxHr] = useState(200);
  const [serverRateLimitDomainHr, setServerRateLimitDomainHr] = useState(1000);
  const [serverRateLimitIpHr, setServerRateLimitIpHr] = useState(500);
  const [serverAuthFailureThreshold, setServerAuthFailureThreshold] = useState(10);

  // Tab specific data
  const [queueItems, setQueueItems] = useState<QueueItem[]>([]);
  const [queueLoading, setQueueLoading] = useState(false);
  const [flushingQueue, setFlushingQueue] = useState(false);

  const [deliveryLogs, setDeliveryLogs] = useState<DeliveryLog[]>([]);
  const [logsLoading, setLogsLoading] = useState(false);
  const [logsFilterStatus, setLogsFilterStatus] = useState<string>('all');
  const [logsSearch, setLogsSearch] = useState('');

  const [suppressions, setSuppressions] = useState<SuppressionItem[]>([]);
  const [suppressionsLoading, setSuppressionsLoading] = useState(false);

  const [services, setServices] = useState<ServiceStatus[]>([]);
  const [servicesLoading, setServicesLoading] = useState(false);
  const [serviceActionLoading, setServiceActionLoading] = useState<string | null>(null);

  const [smtpSettings, setSmtpSettings] = useState<SMTPSettings | null>(null);
  const [selectedSmtpDomain, setSelectedSmtpDomain] = useState<string>('');

  const [healthDomainId, setHealthDomainId] = useState<string>('');
  const [healthData, setHealthData] = useState<DeliverabilityHealth | null>(null);
  const [healthLoading, setHealthLoading] = useState(false);

  // Modals
  const [showAddDomainModal, setShowAddDomainModal] = useState(false);
  const [showAddMailboxModal, setShowAddMailboxModal] = useState(false);
  const [showDNSModal, setShowDNSModal] = useState<EmailDomain | null>(null);
  const [domainDNSRecords, setDomainDNSRecords] = useState<DNSRecordItem[]>([]);
  const [loadingDNS, setLoadingDNS] = useState(false);
  const [regeneratingDKIM, setRegeneratingDKIM] = useState(false);
  const [showVerifyDNSModal, setShowVerifyDNSModal] = useState<DNSVerificationResult | null>(null);
  const [verifyingDNS, setVerifyingDNS] = useState(false);
  const [showPasswordModal, setShowPasswordModal] = useState<EmailMailbox | null>(null);
  const [showClientSetupModal, setShowClientSetupModal] = useState<EmailMailbox | null>(null);
  const [showTestMailboxModal, setShowTestMailboxModal] = useState<EmailMailbox | null>(null);
  const [mailboxTestResult, setMailboxTestResult] = useState<MailboxTestResult | null>(null);
  const [testingMailbox, setTestingMailbox] = useState(false);
  const [newPassword, setNewPassword] = useState('');

  // Form states
  const [newDomainName, setNewDomainName] = useState('');
  const [newDomainMailHost, setNewDomainMailHost] = useState('');
  const [newDomainDKIMSelector, setNewDomainDKIMSelector] = useState('default');
  const [newDomainStorageGB, setNewDomainStorageGB] = useState(50);
  const [addingDomain, setAddingDomain] = useState(false);
  const [newLocalPart, setNewLocalPart] = useState('');
  const [newMailboxDomain, setNewMailboxDomain] = useState('');
  const [newMailboxName, setNewMailboxName] = useState('');
  const [newMailboxPass, setNewMailboxPass] = useState('');
  const [newMailboxQuotaGB, setNewMailboxQuotaGB] = useState(5);
  const [creatingMailbox, setCreatingMailbox] = useState(false);

  // Test email sender form
  const [testEmailFrom, setTestEmailFrom] = useState('');
  const [testEmailTo, setTestEmailTo] = useState('');
  const [testEmailSubject, setTestEmailSubject] = useState('Hostvra Deliverability Test Email');
  const [testEmailBody, setTestEmailBody] = useState('Hello,\n\nThis is a live test email sent via Hostvra Postfix MTA.\nTLS 587 Submission authenticated.');
  const [sendingTestEmail, setSendingTestEmail] = useState(false);
  const [testEmailResult, setTestEmailResult] = useState<TestEmailResult | null>(null);

  const fetchEmailData = async () => {
    try {
      setLoading(true);
      const [domRes, mbRes, srvRes] = await Promise.all([
        apiFetch<EmailDomain[]>('/api/v1/email/domains'),
        apiFetch<EmailMailbox[]>('/api/v1/email/mailboxes'),
        apiFetch<MailServer[]>('/api/v1/email/servers'),
      ]);
      if (domRes.success && domRes.data) {
        setDomains(domRes.data);
        if (domRes.data.length > 0) {
          if (!newMailboxDomain) setNewMailboxDomain(domRes.data[0].id);
          if (!selectedSmtpDomain) setSelectedSmtpDomain(domRes.data[0].domain);
          if (!healthDomainId) setHealthDomainId(domRes.data[0].id);
        }
      }
      if (mbRes.success && mbRes.data) {
        setMailboxes(mbRes.data);
        if (mbRes.data.length > 0 && !testEmailFrom) {
          setTestEmailFrom(mbRes.data[0].email);
        }
      }
      if (srvRes.success && srvRes.data) {
        setMailServers(srvRes.data);
      }
    } catch (err) {
      console.error('Failed to load email data:', err);
    } finally {
      setLoading(false);
    }
  };

  const fetchMailServers = async () => {
    try {
      setMailServersLoading(true);
      const res = await apiFetch<MailServer[]>('/api/v1/email/servers');
      if (res.success && res.data) {
        setMailServers(res.data);
      }
    } catch (err) {
      console.error('Failed to load mail servers:', err);
    } finally {
      setMailServersLoading(false);
    }
  };

  const handleRunPreflight = async () => {
    try {
      setPreflightLoading(true);
      setServerCreationError(null);
      const payload = {
        hostname: serverHostname.trim(),
        ipv4_address: serverIPv4.trim(),
        ipv6_address: serverIPv6.trim() || undefined,
        storage_location: serverStorageLocation.trim(),
        smtp_port: Number(serverSmtpPort),
        smtp_submission_port: Number(serverSmtpSubmissionPort),
        smtps_port: Number(serverSmtpsPort),
        imap_port: Number(serverImapPort),
        imaps_port: Number(serverImapsPort),
        pop3_port: Number(serverPop3Port),
        pop3s_port: Number(serverPop3sPort),
      };
      const res = await apiFetch<PreflightResult>('/api/v1/email/servers/preflight', {
        method: 'POST',
        body: JSON.stringify(payload),
      });
      if (res.success && res.data) {
        setPreflightResult(res.data);
      } else {
        setServerCreationError(res.error?.message || 'Preflight check failed to complete');
      }
    } catch (err: any) {
      setServerCreationError(err.message || 'Preflight request encountered network error');
    } finally {
      setPreflightLoading(false);
    }
  };

  const handleCreateMailServer = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!serverHostname.trim() || !serverPrimaryDomain.trim() || !serverIPv4.trim()) {
      setServerCreationError('Hostname, Primary Domain, and IPv4 Address are required.');
      return;
    }

    const additionalArr = serverAdditionalDomains
      .split(/[\n,]/)
      .map((d) => d.trim().toLowerCase())
      .filter((d) => d.length > 0 && d !== serverPrimaryDomain.toLowerCase().trim());

    const payload = {
      name: serverName.trim() || serverHostname.trim(),
      hostname: serverHostname.trim().toLowerCase(),
      primary_domain: serverPrimaryDomain.trim().toLowerCase(),
      additional_domains: additionalArr,
      ipv4_address: serverIPv4.trim(),
      ipv6_address: serverIPv6.trim() || undefined,
      timezone: serverTimezone.trim() || 'UTC',
      storage_location: serverStorageLocation.trim() || '/var/mail/vhosts',
      mailbox_storage_limit_bytes: (Number(serverMailboxStorageLimitGB) || 100) * 1024 * 1024 * 1024,
      max_mailbox_size_bytes: (Number(serverMaxMailboxSizeGB) || 10) * 1024 * 1024 * 1024,
      max_attachment_size_bytes: (Number(serverMaxAttachmentSizeMB) || 50) * 1024 * 1024,
      smtp_port: Number(serverSmtpPort) || 25,
      smtp_submission_port: Number(serverSmtpSubmissionPort) || 587,
      smtps_port: Number(serverSmtpsPort) || 465,
      imap_port: Number(serverImapPort) || 143,
      imaps_port: Number(serverImapsPort) || 993,
      pop3_port: Number(serverPop3Port) || 110,
      pop3s_port: Number(serverPop3sPort) || 995,
      tls_enabled: serverTlsEnabled,
      spam_filter_enabled: serverSpamFilterEnabled,
      antivirus_enabled: serverAntivirusEnabled,
      dkim_enabled: serverDkimEnabled,
      spf_enabled: serverSpfEnabled,
      dmarc_enabled: serverDmarcEnabled,
      webmail_enabled: serverWebmailEnabled,
      auto_ssl_enabled: serverAutoSslEnabled,
      backup_enabled: serverBackupEnabled,
      rate_limit_per_mailbox_hr: Number(serverRateLimitMailboxHr) || 0,
      rate_limit_per_domain_hr: Number(serverRateLimitDomainHr) || 0,
      rate_limit_per_ip_hr: Number(serverRateLimitIpHr) || 0,
      auth_failure_threshold: Number(serverAuthFailureThreshold) || 10,
    };

    try {
      setCreatingServer(true);
      setServerCreationError(null);
      const res = await apiFetch<MailServer>('/api/v1/email/servers', {
        method: 'POST',
        body: JSON.stringify(payload),
      });
      if (res.success && res.data) {
        setShowCreateServerModal(false);
        setPreflightResult(null);
        await fetchMailServers();
        await fetchEmailData();
        setActiveTab('servers');
      } else {
        setServerCreationError(res.error?.message || 'Failed to provision mail server node.');
      }
    } catch (err: any) {
      setServerCreationError(err.message || 'Error communicating with Hostvra Agent provisioner.');
    } finally {
      setCreatingServer(false);
    }
  };

  const handleDeleteMailServer = async (id: string, name: string) => {
    if (!confirm(`Are you sure you want to remove mail server node "${name}"? Mailboxes and domain bindings associated with this server will be detached.`)) {
      return;
    }
    try {
      const res = await apiFetch(`/api/v1/email/servers/${id}`, { method: 'DELETE' });
      if (res.success) {
        setMailServers((prev) => prev.filter((s) => s.id !== id));
      } else {
        alert(res.error?.message || 'Failed to remove mail server node.');
      }
    } catch (err: any) {
      alert(`Error deleting mail server: ${err.message}`);
    }
  };

  useEffect(() => {
    fetchEmailData();
  }, []);

  // Fetch queue items
  const fetchQueue = async () => {
    try {
      setQueueLoading(true);
      const res = await apiFetch<QueueItem[]>('/api/v1/email/queue');
      if (res.success && res.data) {
        setQueueItems(res.data);
      }
    } catch (err) {
      console.error('Failed to fetch mail queue:', err);
    } finally {
      setQueueLoading(false);
    }
  };

  // Flush queue
  const handleFlushQueue = async () => {
    try {
      setFlushingQueue(true);
      const res = await apiFetch<any>('/api/v1/email/queue/flush', { method: 'POST' });
      if (res.success) {
        alert(res.data?.message || 'Postfix mail queue flushed successfully (postqueue -f).');
        fetchQueue();
      }
    } catch (err: any) {
      alert(`Flush queue failed: ${err.message || 'Error executing postqueue'}`);
    } finally {
      setFlushingQueue(false);
    }
  };

  // Delete single queue item
  const handleDeleteQueueItem = async (id: string) => {
    if (!confirm(`Purge queued message ${id} from Postfix queue?`)) return;
    try {
      const res = await apiFetch(`/api/v1/email/queue/${id}`, { method: 'DELETE' });
      if (res.success) {
        setQueueItems((prev) => prev.filter((item) => item.id !== id));
      }
    } catch (err: any) {
      alert(`Failed to delete queue item: ${err.message || 'Error'}`);
    }
  };

  // Fetch logs
  const fetchLogs = async () => {
    try {
      setLogsLoading(true);
      const res = await apiFetch<DeliveryLog[]>('/api/v1/email/logs');
      if (res.success && res.data) {
        setDeliveryLogs(res.data);
      }
    } catch (err) {
      console.error('Failed to fetch delivery logs:', err);
    } finally {
      setLogsLoading(false);
    }
  };

  // Fetch suppressions
  const fetchSuppressions = async () => {
    try {
      setSuppressionsLoading(true);
      const res = await apiFetch<SuppressionItem[]>('/api/v1/email/suppressions');
      if (res.success && res.data) {
        setSuppressions(res.data);
      }
    } catch (err) {
      console.error('Failed to fetch suppressions:', err);
    } finally {
      setSuppressionsLoading(false);
    }
  };

  // Remove suppression
  const handleRemoveSuppression = async (id: string) => {
    if (!confirm('Remove this address from suppression list?')) return;
    try {
      const res = await apiFetch(`/api/v1/email/suppressions/${id}`, { method: 'DELETE' });
      if (res.success) {
        setSuppressions((prev) => prev.filter((item) => item.id !== id));
      }
    } catch (err: any) {
      alert(`Failed to remove suppression: ${err.message || 'Error'}`);
    }
  };

  // Fetch services
  const fetchServices = async () => {
    try {
      setServicesLoading(true);
      const res = await apiFetch<ServiceStatus[]>('/api/v1/email/services');
      if (res.success && res.data) {
        setServices(res.data);
      }
    } catch (err) {
      console.error('Failed to fetch email services:', err);
    } finally {
      setServicesLoading(false);
    }
  };

  // Service actions
  const handleServiceAction = async (serviceName: string, action: 'restart' | 'reload') => {
    try {
      setServiceActionLoading(`${serviceName}-${action}`);
      const res = await apiFetch<any>(`/api/v1/email/services/${serviceName}/${action}`, { method: 'POST' });
      if (res.success) {
        alert(res.data?.message || `${serviceName} ${action} completed with syntax validation pass.`);
        fetchServices();
      }
    } catch (err: any) {
      alert(`Service ${action} failed: ${err.message || 'Error'}`);
    } finally {
      setServiceActionLoading(null);
    }
  };

  // Fetch SMTP Settings
  const fetchSmtpSettings = async (dom: string) => {
    try {
      const res = await apiFetch<SMTPSettings>(`/api/v1/email/smtp-settings?domain=${encodeURIComponent(dom)}`);
      if (res.success && res.data) {
        setSmtpSettings(res.data);
      }
    } catch (err) {
      console.error('Failed to fetch SMTP settings:', err);
    }
  };

  // Fetch Deliverability Health
  const fetchHealthAudit = async (domainId: string) => {
    if (!domainId) return;
    try {
      setHealthLoading(true);
      const res = await apiFetch<DeliverabilityHealth>(`/api/v1/email/domains/${domainId}/health`);
      if (res.success && res.data) {
        setHealthData(res.data);
      }
    } catch (err) {
      console.error('Failed to run deliverability audit:', err);
    } finally {
      setHealthLoading(false);
    }
  };

  // Verify DNS modal handler
  const handleVerifyDNS = async (domId: string) => {
    try {
      setVerifyingDNS(true);
      let res = await apiFetch<DNSVerificationResult>(`/api/v1/email/domains/${domId}/verify-dns`);
      if (!res.success) {
        res = await apiFetch<DNSVerificationResult>(`/api/v1/email/domains/${domId}/dns/verify`, { method: 'POST' });
      }
      if (res.success && res.data) {
        setShowVerifyDNSModal(res.data);
      }
    } catch (err: any) {
      alert(`DNS Verification failed: ${err.message || 'Error querying DNS'}`);
    } finally {
      setVerifyingDNS(false);
    }
  };

  const openDNSModal = async (dom: EmailDomain) => {
    setShowDNSModal(dom);
    setLoadingDNS(true);
    setDomainDNSRecords([]);
    try {
      const res = await apiFetch<DNSRecordItem[]>(`/api/v1/email/domains/${dom.id}/dns`);
      if (res.success && res.data) {
        setDomainDNSRecords(res.data);
      }
    } catch (err: any) {
      console.error('Failed to load DNS records:', err);
    } finally {
      setLoadingDNS(false);
    }
  };

  const handleRegenerateDKIM = async (domId: string) => {
    if (!confirm('Are you sure you want to regenerate the 2048-bit RSA DKIM key? You will need to update your DNS TXT record.')) {
      return;
    }
    try {
      setRegeneratingDKIM(true);
      const res = await apiFetch(`/api/v1/email/domains/${domId}/dkim/generate`, { method: 'POST' });
      if (res.success) {
        if (showDNSModal && showDNSModal.id === domId) {
          await openDNSModal(showDNSModal);
        }
        alert('A new 2048-bit RSA DKIM key has been generated! Please update your DNS TXT record.');
      }
    } catch (err: any) {
      alert(`Failed to regenerate DKIM key: ${err.message || 'Unknown error'}`);
    } finally {
      setRegeneratingDKIM(false);
    }
  };

  const copyAllZoneRecords = (dom: EmailDomain, records: DNSRecordItem[]) => {
    if (!records || records.length === 0) return;
    const lines = [
      `; Hostvra Email DNS Zone Records for ${dom.domain}`,
      `; Generated on ${new Date().toUTCString()}`,
      `$ORIGIN ${dom.domain}.`,
      `$TTL 3600`,
      '',
    ];
    records.forEach((r) => {
      const host = r.host === '@' ? '@' : r.host;
      if (r.record_type === 'MX') {
        lines.push(`${host.padEnd(20)} 3600 IN MX   ${r.expected}`);
      } else if (r.record_type === 'TXT') {
        lines.push(`${host.padEnd(20)} 3600 IN TXT  "${r.expected}"`);
      } else {
        lines.push(`${host.padEnd(20)} 3600 IN ${r.record_type.padEnd(5)} ${r.expected}`);
      }
    });
    const content = lines.join('\n');
    copyToClipboard(content, 'zone_all');
  };

  // Test mailbox credentials
  const handleTestMailbox = async (mb: EmailMailbox) => {
    setShowTestMailboxModal(mb);
    setMailboxTestResult(null);
    try {
      setTestingMailbox(true);
      const res = await apiFetch<MailboxTestResult>(`/api/v1/email/mailboxes/${mb.id}/test`, { method: 'POST' });
      if (res.success && res.data) {
        setMailboxTestResult(res.data);
      }
    } catch (err: any) {
      setMailboxTestResult({
        mailbox_id: mb.id,
        email: mb.email,
        smtp_auth_passed: false,
        imap_auth_passed: false,
        smtp_latency_ms: 0,
        imap_latency_ms: 0,
        error: err.message || 'Authentication probe failed',
        details: 'Failed to communicate with local mail daemons.',
      });
    } finally {
      setTestingMailbox(false);
    }
  };

  // Send Test Email
  const handleSendTestEmail = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!testEmailFrom || !testEmailTo) return;
    try {
      setSendingTestEmail(true);
      setTestEmailResult(null);
      const res = await apiFetch<TestEmailResult>('/api/v1/email/test-send', {
        method: 'POST',
        body: JSON.stringify({
          from: testEmailFrom,
          to: testEmailTo,
          subject: testEmailSubject,
          body: testEmailBody,
        }),
      });
      if (res.success && res.data) {
        setTestEmailResult(res.data);
      }
    } catch (err: any) {
      setTestEmailResult({
        success: false,
        message_id: '',
        recipient: testEmailTo,
        latency_ms: 0,
        transcript: [`ERROR: ${err.message || 'SMTP Handshake rejected'}`],
        error: err.message || 'Failed to transmit test email',
      });
    } finally {
      setSendingTestEmail(false);
    }
  };

  // React to tab changes
  useEffect(() => {
    if (activeTab === 'servers') fetchMailServers();
    if (activeTab === 'queue') fetchQueue();
    if (activeTab === 'logs') fetchLogs();
    if (activeTab === 'suppressions') fetchSuppressions();
    if (activeTab === 'services') fetchServices();
    if (activeTab === 'smtp' && selectedSmtpDomain) fetchSmtpSettings(selectedSmtpDomain);
    if (activeTab === 'health' && healthDomainId) fetchHealthAudit(healthDomainId);
  }, [activeTab, selectedSmtpDomain, healthDomainId]);

  const copyToClipboard = (text: string, key: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(key);
    setTimeout(() => setCopiedKey(null), 2000);
  };

  const handleAddDomain = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newDomainName.trim()) return;

    const domainClean = newDomainName.toLowerCase().trim();
    const mailHost = newDomainMailHost.trim() ? newDomainMailHost.toLowerCase().trim() : `mail.${domainClean}`;
    const dkimSel = newDomainDKIMSelector.trim() || 'default';
    const storageBytes = (newDomainStorageGB || 50) * 1024 * 1024 * 1024;

    try {
      setAddingDomain(true);
      const res = await apiFetch<EmailDomain>('/api/v1/email/domains', {
        method: 'POST',
        body: JSON.stringify({
          domain: domainClean,
          mail_hostname: mailHost,
          dkim_selector: dkimSel,
          storage_limit_bytes: storageBytes,
        }),
      });

      if (res.success && res.data) {
        const addedDomain = res.data;
        setDomains((prev) => {
          const exists = prev.some((d) => d.id === addedDomain.id);
          return exists ? prev : [...prev, addedDomain];
        });
        setNewMailboxDomain(addedDomain.id);
        setSelectedSmtpDomain(addedDomain.domain);
        setHealthDomainId(addedDomain.id);
        setShowAddDomainModal(false);
        setNewDomainName('');
        setNewDomainMailHost('');
        await openDNSModal(addedDomain);
      } else {
        alert(res.error?.message || 'Failed to add email domain. Please verify server status.');
        await fetchEmailData();
      }
    } catch (err: any) {
      alert(`Failed to add domain: ${err.message || 'Unknown error'}`);
    } finally {
      setAddingDomain(false);
    }
  };

  const handleAddMailbox = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newLocalPart.trim() || !newMailboxPass) return;

    const selectedDom = domains.find((d) => d.id === newMailboxDomain) || domains[0];
    if (!selectedDom) return;
    const fullEmail = `${newLocalPart.toLowerCase().trim()}@${selectedDom.domain}`;

    try {
      setCreatingMailbox(true);
      const res = await apiFetch<EmailMailbox>('/api/v1/email/mailboxes', {
        method: 'POST',
        body: JSON.stringify({
          domain_id: selectedDom.id,
          local_part: newLocalPart.toLowerCase().trim(),
          password: newMailboxPass,
          name: newMailboxName || fullEmail,
          quota_bytes: newMailboxQuotaGB * 1024 * 1024 * 1024,
        }),
      });

      if (res.success && res.data) {
        setMailboxes([...mailboxes, res.data]);
      } else {
        await fetchEmailData();
      }
      setNewLocalPart('');
      setNewMailboxName('');
      setNewMailboxPass('');
      setShowAddMailboxModal(false);
    } catch (err: any) {
      alert(`Failed to create mailbox: ${err.message || 'Unknown error'}`);
    } finally {
      setCreatingMailbox(false);
    }
  };

  const handleDeleteMailbox = async (id: string) => {
    if (confirm('Are you sure you want to delete this mailbox? All stored Maildir messages will be purged.')) {
      try {
        await apiFetch(`/api/v1/email/mailboxes/${id}`, { method: 'DELETE' });
        setMailboxes(mailboxes.filter((m) => m.id !== id));
      } catch (err) {
        console.error('Failed to delete mailbox:', err);
      }
    }
  };

  const handleToggleSuspendMailbox = async (mb: EmailMailbox) => {
    const actionName = mb.is_suspended ? 'Resume' : 'Suspend';
    if (!confirm(`${actionName} mailbox ${mb.email}? When suspended, login and outbound mail submission are disabled.`)) {
      return;
    }
    try {
      const res = await apiFetch<EmailMailbox>(`/api/v1/email/mailboxes/${mb.id}/toggle-suspend?suspend=${!mb.is_suspended}`, {
        method: 'POST',
      });
      if (res.success) {
        setMailboxes((prev) => prev.map((item) => (item.id === mb.id ? { ...item, is_suspended: !mb.is_suspended } : item)));
      } else {
        alert(res.error?.message || `Failed to ${actionName.toLowerCase()} mailbox.`);
      }
    } catch (err: any) {
      alert(`Error updating mailbox status: ${err.message || 'Network error'}`);
    }
  };

  const handleDeleteDomain = async (id: string) => {
    if (confirm('Delete email domain and all associated mailboxes and DKIM keys?')) {
      try {
        await apiFetch(`/api/v1/email/domains/${id}`, { method: 'DELETE' });
        setDomains(domains.filter((d) => d.id !== id));
        setMailboxes(mailboxes.filter((m) => m.domain_id !== id));
      } catch (err) {
        console.error('Failed to delete domain:', err);
      }
    }
  };

  const handleSavePassword = async () => {
    if (!showPasswordModal || !newPassword) return;
    if (newPassword.length < 8) {
      alert('Password must be at least 8 characters long');
      return;
    }
    try {
      await apiFetch(`/api/v1/email/mailboxes/${showPasswordModal.id}/password`, {
        method: 'PUT',
        body: JSON.stringify({ password: newPassword }),
      });
      alert('Mailbox password updated and hashed with Dovecot SHA512-CRYPT.');
      setShowPasswordModal(null);
      setNewPassword('');
    } catch (err: any) {
      alert(err.message || 'Failed to update mailbox password');
    }
  };

  const filteredMailboxes = mailboxes.filter(
    (m) => m.email.toLowerCase().includes(search.toLowerCase()) || m.name.toLowerCase().includes(search.toLowerCase())
  );

  const filteredDomains = domains.filter((d) => d.domain.toLowerCase().includes(search.toLowerCase()));

  const filteredLogs = deliveryLogs.filter((log) => {
    if (logsFilterStatus !== 'all' && log.status.toLowerCase() !== logsFilterStatus) {
      return false;
    }
    if (logsSearch.trim()) {
      const q = logsSearch.toLowerCase();
      return (
        log.sender.toLowerCase().includes(q) ||
        log.recipient.toLowerCase().includes(q) ||
        log.message_id.toLowerCase().includes(q)
      );
    }
    return true;
  });

  return (
    <DashboardShell>
      <div className="space-y-6">
        {/* Header Title & Actions */}
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div>
            <div className="flex items-center gap-2.5">
              <div className="p-2 rounded-xl bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 border border-indigo-500/20">
                <Mail className="w-6 h-6" />
              </div>
              <div>
                <h1 className="text-2xl font-bold text-slate-900 dark:text-white tracking-tight">Email Hosting</h1>
                <p className="text-sm text-slate-500 dark:text-slate-400">
                  Postfix MTA, Dovecot IMAP/POP3, Rspamd Milter, and Roundcube Webmail
                </p>
              </div>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-3">
            <button
              onClick={() => {
                setSelectedWebmailEmail(mailboxes[0]?.email);
                setActiveTab('webmail');
              }}
              className="inline-flex items-center gap-2 px-3.5 py-2 rounded-xl text-sm font-medium bg-indigo-50 dark:bg-indigo-600/10 text-indigo-700 dark:text-indigo-300 hover:bg-indigo-100 dark:hover:bg-indigo-600/20 border border-indigo-200 dark:border-indigo-500/30 transition shadow-xs"
              title="Open Built-in Webmail Client"
            >
              <Inbox className="w-4 h-4 text-indigo-500 dark:text-indigo-400" />
              <span>Webmail (Inbox)</span>
            </button>
            <button
              onClick={() => {
                if (domains.length > 0) {
                  openDNSModal(domains[0]);
                } else {
                  setShowAddDomainModal(true);
                }
              }}
              className="inline-flex items-center gap-2 px-3.5 py-2 rounded-xl text-sm font-medium bg-emerald-50 dark:bg-emerald-600/10 text-emerald-700 dark:text-emerald-300 hover:bg-emerald-100 dark:hover:bg-emerald-600/20 border border-emerald-200 dark:border-emerald-500/30 transition shadow-xs"
              title="View DKIM, SPF, DMARC, MX DNS records"
            >
              <ShieldCheck className="w-4 h-4 text-emerald-500" />
              <span>DNS & Security</span>
            </button>
            <button
              onClick={() => setActiveTab('tester')}
              className="inline-flex items-center gap-2 px-3.5 py-2 rounded-xl text-sm font-medium bg-white dark:bg-surface-800 text-slate-700 dark:text-slate-200 hover:bg-slate-50 dark:hover:bg-surface-700 border border-slate-200 dark:border-surface-700 transition shadow-xs"
            >
              <Terminal className="w-4 h-4 text-emerald-500" />
              <span>Send Test Email</span>
            </button>
            <button
              onClick={() => setShowAddDomainModal(true)}
              className="inline-flex items-center gap-2 px-3.5 py-2 rounded-xl text-sm font-medium bg-white dark:bg-surface-800 text-slate-700 dark:text-slate-200 hover:bg-slate-50 dark:hover:bg-surface-700 border border-slate-200 dark:border-surface-700 transition shadow-xs"
            >
              <Plus className="w-4 h-4 text-indigo-500" />
              <span>Add Domain</span>
            </button>
            <button
              onClick={() => setShowAddMailboxModal(true)}
              className="inline-flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-medium bg-indigo-600 hover:bg-indigo-500 text-white shadow-lg shadow-indigo-600/20 transition active:scale-95"
            >
              <Inbox className="w-4 h-4" />
              <span>Create Mailbox</span>
            </button>
          </div>
        </div>

        {/* Live Summary Metrics */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <div className="p-5 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 shadow-xs flex items-center justify-between">
            <div>
              <p className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                Configured Domains
              </p>
              <p className="text-2xl font-bold text-slate-900 dark:text-white mt-1">{domains.length}</p>
              <p className="text-xs text-indigo-600 dark:text-indigo-400 mt-1 flex items-center gap-1">
                <Globe className="w-3.5 h-3.5" /> Postfix virtual_mailbox_domains
              </p>
            </div>
            <div className="w-12 h-12 rounded-xl bg-indigo-50 dark:bg-indigo-500/10 border border-indigo-200 dark:border-indigo-500/20 flex items-center justify-center text-indigo-600 dark:text-indigo-400">
              <Globe className="w-6 h-6" />
            </div>
          </div>

          <div className="p-5 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 shadow-xs flex items-center justify-between">
            <div>
              <p className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                Active Mailboxes
              </p>
              <p className="text-2xl font-bold text-slate-900 dark:text-white mt-1">{mailboxes.length}</p>
              <p className="text-xs text-slate-500 mt-1">Dovecot Maildir storage</p>
            </div>
            <div className="w-12 h-12 rounded-xl bg-blue-50 dark:bg-blue-500/10 border border-blue-200 dark:border-blue-500/20 flex items-center justify-center text-blue-600 dark:text-blue-400">
              <Inbox className="w-6 h-6" />
            </div>
          </div>

          <div className="p-5 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 shadow-xs flex items-center justify-between">
            <div>
              <p className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                Deliverability Health
              </p>
              <p className="text-2xl font-bold text-slate-900 dark:text-white mt-1">
                {healthData ? `${healthData.score} / 100` : 'Live Audit'}
              </p>
              <p className="text-xs text-emerald-600 dark:text-emerald-400 mt-1">
                {healthData ? healthData.rating : 'RFC 5321 & Zero Open Relay'}
              </p>
            </div>
            <div className="w-12 h-12 rounded-xl bg-emerald-50 dark:bg-emerald-500/10 border border-emerald-200 dark:border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
              <ShieldCheck className="w-6 h-6" />
            </div>
          </div>

          <div className="p-5 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 shadow-xs flex items-center justify-between">
            <div>
              <p className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                Spam & Security Guard
              </p>
              <p className="text-2xl font-bold text-purple-600 dark:text-purple-400 mt-1">Active</p>
              <p className="text-xs text-slate-500 mt-1">Rspamd Milter + TLS 587/465</p>
            </div>
            <div className="w-12 h-12 rounded-xl bg-purple-50 dark:bg-purple-500/10 border border-purple-200 dark:border-purple-500/20 flex items-center justify-center text-purple-600 dark:text-purple-400">
              <Sparkles className="w-6 h-6" />
            </div>
          </div>
        </div>

        {/* Navigation Tabs */}
        <div className="flex items-center gap-1.5 p-1 bg-slate-100 dark:bg-surface-900 rounded-xl overflow-x-auto text-xs font-semibold border border-slate-200 dark:border-surface-800">
          <button
            onClick={() => setActiveTab('servers')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all font-medium whitespace-nowrap ${
              activeTab === 'servers'
                ? 'bg-white dark:bg-surface-800 text-indigo-600 dark:text-indigo-400 shadow-xs'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
            }`}
          >
            <Server className="w-4 h-4" />
            <span>Mail Servers ({mailServers.length})</span>
            <span className="px-1.5 py-0.5 rounded-full text-[10px] font-bold bg-purple-500/20 text-purple-600 dark:text-purple-400">
              Cluster
            </span>
          </button>

          <button
            onClick={() => setActiveTab('mailboxes')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all font-medium whitespace-nowrap ${
              activeTab === 'mailboxes'
                ? 'bg-white dark:bg-surface-800 text-indigo-600 dark:text-indigo-400 shadow-xs'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
            }`}
          >
            <Mail className="w-4 h-4" />
            <span>Mailboxes ({mailboxes.length})</span>
          </button>

          <button
            onClick={() => setActiveTab('webmail')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all font-medium whitespace-nowrap ${
              activeTab === 'webmail'
                ? 'bg-white dark:bg-surface-800 text-indigo-600 dark:text-indigo-400 shadow-xs'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
            }`}
          >
            <Inbox className="w-4 h-4" />
            <span>Webmail Suite</span>
            <span className="px-1.5 py-0.5 rounded-full text-[10px] font-bold bg-indigo-500/20 text-indigo-600 dark:text-indigo-400">
              Live
            </span>
          </button>

          <button
            onClick={() => setActiveTab('domains')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all font-medium whitespace-nowrap ${
              activeTab === 'domains'
                ? 'bg-white dark:bg-surface-800 text-indigo-600 dark:text-indigo-400 shadow-xs'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
            }`}
          >
            <Globe className="w-4 h-4" />
            <span>Email Domains ({domains.length})</span>
          </button>

          <button
            onClick={() => setActiveTab('health')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all font-medium whitespace-nowrap ${
              activeTab === 'health'
                ? 'bg-white dark:bg-surface-800 text-indigo-600 dark:text-indigo-400 shadow-xs'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
            }`}
          >
            <ShieldCheck className="w-4 h-4" />
            <span>Deliverability & Health Audit</span>
          </button>

          <button
            onClick={() => setActiveTab('smtp')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all font-medium whitespace-nowrap ${
              activeTab === 'smtp'
                ? 'bg-white dark:bg-surface-800 text-indigo-600 dark:text-indigo-400 shadow-xs'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
            }`}
          >
            <Layers className="w-4 h-4" />
            <span>SMTP & Client Setup</span>
          </button>

          <button
            onClick={() => setActiveTab('queue')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all font-medium whitespace-nowrap ${
              activeTab === 'queue'
                ? 'bg-white dark:bg-surface-800 text-indigo-600 dark:text-indigo-400 shadow-xs'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
            }`}
          >
            <Send className="w-4 h-4" />
            <span>Mail Queue</span>
          </button>

          <button
            onClick={() => setActiveTab('logs')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all font-medium whitespace-nowrap ${
              activeTab === 'logs'
                ? 'bg-white dark:bg-surface-800 text-indigo-600 dark:text-indigo-400 shadow-xs'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
            }`}
          >
            <Sliders className="w-4 h-4" />
            <span>Delivery Logs</span>
          </button>

          <button
            onClick={() => setActiveTab('suppressions')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all font-medium whitespace-nowrap ${
              activeTab === 'suppressions'
                ? 'bg-white dark:bg-surface-800 text-indigo-600 dark:text-indigo-400 shadow-xs'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
            }`}
          >
            <UserCheck className="w-4 h-4" />
            <span>Suppressions</span>
          </button>

          <button
            onClick={() => setActiveTab('services')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all font-medium whitespace-nowrap ${
              activeTab === 'services'
                ? 'bg-white dark:bg-surface-800 text-indigo-600 dark:text-indigo-400 shadow-xs'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
            }`}
          >
            <Activity className="w-4 h-4" />
            <span>Services & Daemons</span>
          </button>

          <button
            onClick={() => setActiveTab('tester')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all font-medium whitespace-nowrap ${
              activeTab === 'tester'
                ? 'bg-white dark:bg-surface-800 text-indigo-600 dark:text-indigo-400 shadow-xs'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
            }`}
          >
            <Terminal className="w-4 h-4" />
            <span>Send Test Tool</span>
          </button>

          <button
            onClick={() => setActiveTab('guide')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all font-medium whitespace-nowrap ${
              activeTab === 'guide'
                ? 'bg-white dark:bg-surface-800 text-indigo-600 dark:text-indigo-400 shadow-xs'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
            }`}
          >
            <BookOpen className="w-4 h-4" />
            <span>User Guide (35 Topics)</span>
            <span className="px-1.5 py-0.5 rounded-full text-[10px] font-bold bg-indigo-500/20 text-indigo-600 dark:text-indigo-400">
              Docs
            </span>
          </button>
        </div>

        {/* Tab Content: Mail Servers (Cluster Subsystem) */}
        {activeTab === 'servers' && (
          <div className="space-y-6">
            {/* Header / Actions Bar */}
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-5 shadow-xs">
              <div>
                <div className="flex items-center gap-2">
                  <h2 className="text-lg font-bold text-slate-900 dark:text-white flex items-center gap-2">
                    <Server className="w-5 h-5 text-indigo-500" />
                    <span>Enterprise Mail Server Nodes</span>
                  </h2>
                  <span className="px-2 py-0.5 rounded-full text-xs font-bold bg-indigo-50 dark:bg-indigo-950/40 text-indigo-600 dark:text-indigo-400 border border-indigo-200 dark:border-indigo-800">
                    Hostvra Cluster
                  </span>
                </div>
                <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                  Dedicated self-hosted mail instances running Postfix MTA, Dovecot IMAP/POP3, and Rspamd spam milter with zero artificial sending quotas.
                </p>
              </div>

              <div className="flex items-center gap-2.5">
                <button
                  onClick={fetchMailServers}
                  disabled={mailServersLoading}
                  className="p-2.5 rounded-xl border border-slate-200 dark:border-surface-700 bg-white dark:bg-surface-800 text-slate-600 dark:text-slate-300 hover:text-slate-900 dark:hover:text-white transition-all shadow-xs"
                  title="Refresh Mail Server Nodes"
                >
                  <RefreshCw className={`w-4 h-4 ${mailServersLoading ? 'animate-spin text-indigo-500' : ''}`} />
                </button>
                <button
                  onClick={() => {
                    setServerCreationError(null);
                    setPreflightResult(null);
                    setServerWizardTab('general');
                    setShowCreateServerModal(true);
                  }}
                  className="flex items-center gap-2 px-4 py-2.5 rounded-xl text-xs font-semibold bg-indigo-600 hover:bg-indigo-500 text-white shadow-lg shadow-indigo-600/20 transition-all"
                >
                  <Plus className="w-4 h-4" />
                  <span>Create Mail Server</span>
                </button>
              </div>
            </div>

            {/* Uncapped Architecture Notice */}
            <div className="p-4 rounded-2xl bg-gradient-to-r from-purple-500/10 via-indigo-500/10 to-blue-500/10 border border-purple-200/50 dark:border-purple-500/20 flex items-start gap-3">
              <Sparkles className="w-5 h-5 text-purple-600 dark:text-purple-400 flex-shrink-0 mt-0.5" />
              <div className="text-xs">
                <span className="font-bold text-slate-900 dark:text-white">Uncapped Sending Architecture: </span>
                <span className="text-slate-600 dark:text-slate-300">
                  Hostvra enforces no synthetic email sending limits (e.g. 100 or 1,000/day). Sending volume is governed strictly by server compute, disk I/O, mail queue capacity, and recipient server policies. Deliverability is preserved via DKIM, SPF, DMARC, and abuse rate controls.
                </span>
              </div>
            </div>

            {/* Server Nodes Grid */}
            {mailServersLoading && mailServers.length === 0 ? (
              <div className="p-12 text-center bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl space-y-3">
                <RefreshCw className="w-8 h-8 animate-spin mx-auto text-indigo-500" />
                <p className="text-sm font-semibold text-slate-700 dark:text-slate-300">Loading mail server nodes...</p>
              </div>
            ) : mailServers.length === 0 ? (
              <div className="p-12 text-center bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl space-y-4">
                <div className="w-16 h-16 rounded-2xl bg-indigo-50 dark:bg-indigo-950/40 border border-indigo-200 dark:border-indigo-800 flex items-center justify-center mx-auto text-indigo-600 dark:text-indigo-400">
                  <Server className="w-8 h-8" />
                </div>
                <div>
                  <h3 className="text-base font-bold text-slate-900 dark:text-white">No Mail Server Nodes Configured</h3>
                  <p className="text-xs text-slate-500 dark:text-slate-400 max-w-md mx-auto mt-1">
                    Deploy your first standalone in-house email server node with automated Postfix, Dovecot, Rspamd, ClamAV, and Webmail provisioning.
                  </p>
                </div>
                <button
                  onClick={() => {
                    setServerCreationError(null);
                    setPreflightResult(null);
                    setServerWizardTab('general');
                    setShowCreateServerModal(true);
                  }}
                  className="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl text-xs font-semibold bg-indigo-600 hover:bg-indigo-500 text-white shadow-lg shadow-indigo-600/20 transition-all"
                >
                  <Plus className="w-4 h-4" />
                  <span>Provision First Mail Server</span>
                </button>
              </div>
            ) : (
              <div className="grid grid-cols-1 xl:grid-cols-2 gap-5">
                {mailServers.map((srv) => (
                  <div
                    key={srv.id}
                    className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-5 shadow-xs space-y-4 hover:border-slate-300 dark:hover:border-surface-700 transition-all"
                  >
                    {/* Top Row: Name, Hostname, Badges */}
                    <div className="flex items-start justify-between gap-4">
                      <div className="flex items-center gap-3">
                        <div className="w-11 h-11 rounded-xl bg-purple-50 dark:bg-purple-950/40 border border-purple-200 dark:border-purple-800 flex items-center justify-center text-purple-600 dark:text-purple-400 font-bold">
                          <Server className="w-5 h-5" />
                        </div>
                        <div>
                          <div className="flex items-center gap-2">
                            <h3 className="text-sm font-bold text-slate-900 dark:text-white">{srv.name}</h3>
                            <span
                              className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-bold ${
                                srv.status === 'active'
                                  ? 'bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800'
                                  : 'bg-amber-50 dark:bg-amber-950/40 text-amber-600 dark:text-amber-400 border border-amber-200 dark:border-amber-800'
                              }`}
                            >
                              <span className="w-1.5 h-1.5 rounded-full bg-current" />
                              {srv.status.toUpperCase()}
                            </span>
                          </div>
                          <p className="text-xs font-mono text-indigo-600 dark:text-indigo-400 mt-0.5">{srv.hostname}</p>
                        </div>
                      </div>

                      <div className="flex items-center gap-1.5">
                        <span
                          className={`px-2 py-0.5 rounded-md text-[10px] font-bold ${
                            srv.health_status === 'healthy'
                              ? 'bg-emerald-50 text-emerald-600 dark:bg-emerald-950/40 dark:text-emerald-400'
                              : 'bg-rose-50 text-rose-600 dark:bg-rose-950/40 dark:text-rose-400'
                          }`}
                        >
                          {srv.health_status.toUpperCase()}
                        </span>
                      </div>
                    </div>

                    {/* Network & Storage Specifications */}
                    <div className="grid grid-cols-2 gap-3 text-xs bg-slate-50 dark:bg-surface-950 p-3 rounded-xl border border-slate-200/60 dark:border-surface-800/60">
                      <div>
                        <span className="text-slate-500 dark:text-slate-400 block text-[11px]">Primary Domain:</span>
                        <span className="font-semibold text-slate-800 dark:text-slate-200 font-mono">{srv.primary_domain}</span>
                      </div>
                      <div>
                        <span className="text-slate-500 dark:text-slate-400 block text-[11px]">Server IPv4 / IPv6:</span>
                        <span className="font-semibold text-slate-800 dark:text-slate-200 font-mono">
                          {srv.ipv4_address} {srv.ipv6_address ? `/ ${srv.ipv6_address}` : ''}
                        </span>
                      </div>
                      <div>
                        <span className="text-slate-500 dark:text-slate-400 block text-[11px]">Storage Pool Limit:</span>
                        <span className="font-semibold text-slate-800 dark:text-slate-200">
                          {formatBytes(srv.mailbox_storage_limit_bytes)} (Max Box: {formatBytes(srv.max_mailbox_size_bytes)})
                        </span>
                      </div>
                      <div>
                        <span className="text-slate-500 dark:text-slate-400 block text-[11px]">Maildir Location:</span>
                        <span className="font-semibold text-slate-800 dark:text-slate-200 font-mono text-[11px] truncate block">
                          {srv.storage_location}
                        </span>
                      </div>
                    </div>

                    {/* Port Matrix */}
                    <div className="space-y-1.5">
                      <div className="flex items-center justify-between text-[11px] font-semibold text-slate-600 dark:text-slate-300">
                        <span>Active Service Protocols & Ports</span>
                        <span className="text-slate-400 font-normal">Max Attachment: {formatBytes(srv.max_attachment_size_bytes)}</span>
                      </div>
                      <div className="flex flex-wrap gap-1.5 text-[10px] font-mono">
                        <span className="px-2 py-0.5 rounded bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-300 border border-slate-200 dark:border-surface-700">
                          SMTP: {srv.smtp_port}
                        </span>
                        <span className="px-2 py-0.5 rounded bg-indigo-50 dark:bg-indigo-950/40 text-indigo-700 dark:text-indigo-300 border border-indigo-200 dark:border-indigo-800">
                          Submission: {srv.smtp_submission_port} (STARTTLS)
                        </span>
                        <span className="px-2 py-0.5 rounded bg-blue-50 dark:bg-blue-950/40 text-blue-700 dark:text-blue-300 border border-blue-200 dark:border-blue-800">
                          SMTPS: {srv.smtps_port} (TLS)
                        </span>
                        <span className="px-2 py-0.5 rounded bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-300 border border-slate-200 dark:border-surface-700">
                          IMAP: {srv.imap_port}
                        </span>
                        <span className="px-2 py-0.5 rounded bg-emerald-50 dark:bg-emerald-950/40 text-emerald-700 dark:text-emerald-300 border border-emerald-200 dark:border-emerald-800">
                          IMAPS: {srv.imaps_port} (TLS)
                        </span>
                        <span className="px-2 py-0.5 rounded bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-300 border border-slate-200 dark:border-surface-700">
                          POP3: {srv.pop3_port} / {srv.pop3s_port}
                        </span>
                      </div>
                    </div>

                    {/* Security & Feature Badges */}
                    <div className="flex flex-wrap items-center gap-1.5 text-[10px] font-semibold">
                      <span className="px-2 py-0.5 rounded bg-emerald-50 dark:bg-emerald-950/40 text-emerald-700 dark:text-emerald-300">
                        TLS 1.3
                      </span>
                      {srv.spam_filter_enabled && (
                        <span className="px-2 py-0.5 rounded bg-purple-50 dark:bg-purple-950/40 text-purple-700 dark:text-purple-300">
                          Rspamd Milter
                        </span>
                      )}
                      {srv.antivirus_enabled && (
                        <span className="px-2 py-0.5 rounded bg-cyan-50 dark:bg-cyan-950/40 text-cyan-700 dark:text-cyan-300">
                          ClamAV Scanner
                        </span>
                      )}
                      {srv.dkim_enabled && (
                        <span className="px-2 py-0.5 rounded bg-indigo-50 dark:bg-indigo-950/40 text-indigo-700 dark:text-indigo-300">
                          DKIM RSA-2048
                        </span>
                      )}
                      {srv.spf_enabled && (
                        <span className="px-2 py-0.5 rounded bg-blue-50 dark:bg-blue-950/40 text-blue-700 dark:text-blue-300">
                          SPF Synthesizer
                        </span>
                      )}
                      {srv.dmarc_enabled && (
                        <span className="px-2 py-0.5 rounded bg-violet-50 dark:bg-violet-950/40 text-violet-700 dark:text-violet-300">
                          DMARC Enforcer
                        </span>
                      )}
                      {srv.webmail_enabled && (
                        <span className="px-2 py-0.5 rounded bg-amber-50 dark:bg-amber-950/40 text-amber-700 dark:text-amber-300">
                          Webmail Suite
                        </span>
                      )}
                      {srv.backup_enabled && (
                        <span className="px-2 py-0.5 rounded bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-300">
                          Automated Backups
                        </span>
                      )}
                    </div>

                    {/* Anti-Abuse & Rate Controls */}
                    <div className="text-[11px] text-slate-500 dark:text-slate-400 flex items-center justify-between border-t border-slate-100 dark:border-surface-800/80 pt-2.5">
                      <span>
                        Abuse Limits:{' '}
                        <strong className="text-slate-700 dark:text-slate-300">
                          {srv.rate_limit_per_mailbox_hr > 0 ? `${srv.rate_limit_per_mailbox_hr}/box/hr` : 'Uncapped'}
                        </strong>{' '}
                        | Fail Threshold:{' '}
                        <strong className="text-slate-700 dark:text-slate-300">{srv.auth_failure_threshold} fails</strong>
                      </span>
                      <span>
                        Domains:{' '}
                        <strong className="text-indigo-600 dark:text-indigo-400 font-bold">{srv.domain_count || 1}</strong> | Mailboxes:{' '}
                        <strong className="text-indigo-600 dark:text-indigo-400 font-bold">{srv.mailbox_count || 0}</strong>
                      </span>
                    </div>

                    {/* Node Actions */}
                    <div className="flex items-center justify-end gap-2 pt-2 border-t border-slate-100 dark:border-surface-800">
                      {srv.provisioning_logs && (
                        <button
                          onClick={() => setServerLogsModal(srv)}
                          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-surface-800 transition-all"
                        >
                          <FileText className="w-3.5 h-3.5 text-slate-400" />
                          <span>Provisioning Logs</span>
                        </button>
                      )}
                      <button
                        onClick={() => handleDeleteMailServer(srv.id, srv.name)}
                        className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-rose-600 dark:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-950/20 transition-all"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                        <span>Remove Node</span>
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Tab Content: Mailboxes */}
        {activeTab === 'mailboxes' && (
          <div className="space-y-4">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
              <div className="flex items-center gap-2 w-full max-w-sm bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-xl px-3.5 py-2 shadow-xs focus-within:border-indigo-500 focus-within:ring-2 focus-within:ring-indigo-500/20 transition-all">
                <Search className="w-4 h-4 text-slate-400 flex-shrink-0" />
                <input
                  type="text"
                  placeholder="Search mailboxes..."
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  className="w-full bg-transparent text-xs font-medium text-slate-900 dark:text-white placeholder:text-slate-400 focus:outline-none"
                />
                {search && (
                  <button onClick={() => setSearch('')} className="text-slate-400 hover:text-slate-600 p-0.5">
                    <X className="w-3.5 h-3.5" />
                  </button>
                )}
              </div>

              <div className="text-xs text-slate-600 dark:text-slate-400 font-medium">
                IMAP: <code className="text-indigo-600 dark:text-indigo-400 font-mono font-semibold">mail.yourdomain.com:993 (SSL)</code> |
                SMTP: <code className="text-indigo-600 dark:text-indigo-400 font-mono font-semibold">mail.yourdomain.com:587 (STARTTLS)</code>
              </div>
            </div>

            <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl overflow-hidden shadow-xs">
              <div className="overflow-x-auto">
                <table className="w-full text-left text-sm border-collapse">
                  <thead>
                    <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-surface-950 text-slate-700 dark:text-slate-300 text-[11px] font-bold uppercase tracking-wider">
                      <th className="px-6 py-4">Account / Email</th>
                      <th className="px-6 py-4">Display Name</th>
                      <th className="px-6 py-4">Storage Quota</th>
                      <th className="px-6 py-4">Status</th>
                      <th className="px-6 py-4 text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 dark:divide-surface-800/80">
                    {filteredMailboxes.length === 0 ? (
                      <tr>
                        <td colSpan={5} className="px-6 py-12 text-center text-slate-400 text-xs">
                          {mailboxes.length === 0
                            ? 'No email mailboxes configured yet. Click "Create Mailbox" to add your first account.'
                            : 'No mailboxes match your search filter.'}
                        </td>
                      </tr>
                    ) : (
                      filteredMailboxes.map((mb) => {
                        const usagePercent = Math.round(((mb.used_bytes || 0) / (mb.quota_bytes || 1)) * 100);
                        const usedMB = ((mb.used_bytes || 0) / (1024 * 1024)).toFixed(1);
                        const quotaGB = ((mb.quota_bytes || 5368709120) / (1024 * 1024 * 1024)).toFixed(0);

                        return (
                          <tr key={mb.id} className="hover:bg-slate-50/80 dark:hover:bg-surface-800/40 transition">
                            <td className="px-6 py-4 font-mono font-medium text-slate-900 dark:text-white flex items-center gap-2.5">
                              <div className="w-8 h-8 rounded-lg bg-indigo-50 dark:bg-indigo-500/10 border border-indigo-200 dark:border-indigo-500/20 flex items-center justify-center text-indigo-600 dark:text-indigo-400">
                                <Mail className="w-4 h-4" />
                              </div>
                              <span>{mb.email}</span>
                            </td>
                            <td className="px-6 py-4 text-slate-600 dark:text-slate-300">{mb.name}</td>
                            <td className="px-6 py-4">
                              <div className="w-44">
                                <div className="flex justify-between text-xs mb-1">
                                  <span className="text-slate-500">{usedMB} MB</span>
                                  <span className="text-slate-500 font-medium">{quotaGB} GB</span>
                                </div>
                                <div className="w-full h-1.5 bg-slate-200 dark:bg-surface-700 rounded-full overflow-hidden">
                                  <div
                                    className={`h-full rounded-full transition-all ${
                                      usagePercent > 80 ? 'bg-amber-500' : 'bg-indigo-600'
                                    }`}
                                    style={{ width: `${Math.max(usagePercent, 2)}%` }}
                                  />
                                </div>
                              </div>
                            </td>
                            <td className="px-6 py-4">
                              {mb.is_suspended ? (
                                <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-amber-50 dark:bg-amber-500/10 text-amber-700 dark:text-amber-400 border border-amber-200 dark:border-amber-500/20">
                                  <Lock className="w-3 h-3" />
                                  Suspended
                                </span>
                              ) : (
                                <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-emerald-50 dark:bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-500/20">
                                  <CheckCircle2 className="w-3 h-3" />
                                  Active
                                </span>
                              )}
                            </td>
                            <td className="px-6 py-4 text-right space-x-2">
                              <button
                                onClick={() => {
                                  setSelectedWebmailEmail(mb.email);
                                  setActiveTab('webmail');
                                }}
                                className="inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-indigo-50 dark:bg-indigo-600/15 hover:bg-indigo-100 dark:hover:bg-indigo-600/30 text-indigo-700 dark:text-indigo-300 text-xs font-semibold border border-indigo-200 dark:border-indigo-500/30 transition shadow-xs"
                                title={`Open Webmail Suite for ${mb.email}`}
                              >
                                <Inbox className="w-3.5 h-3.5 text-indigo-500 dark:text-indigo-400" />
                                <span>Webmail</span>
                              </button>
                              <button
                                onClick={() => handleTestMailbox(mb)}
                                className="p-1.5 rounded-lg hover:bg-slate-100 dark:hover:bg-surface-700 text-slate-500 hover:text-emerald-600 transition"
                                title="Test SMTP & IMAP Credentials"
                              >
                                <Radio className="w-4 h-4" />
                              </button>
                              <button
                                onClick={() => setShowClientSetupModal(mb)}
                                className="p-1.5 rounded-lg hover:bg-slate-100 dark:hover:bg-surface-700 text-slate-500 hover:text-indigo-600 transition"
                                title="Client Setup Parameters"
                              >
                                <Layers className="w-4 h-4" />
                              </button>
                              <button
                                onClick={() => setShowPasswordModal(mb)}
                                className="p-1.5 rounded-lg hover:bg-slate-100 dark:hover:bg-surface-700 text-slate-500 hover:text-slate-800 dark:hover:text-slate-200 transition"
                                title="Change Password"
                              >
                                <Key className="w-4 h-4" />
                              </button>
                              <button
                                onClick={() => handleToggleSuspendMailbox(mb)}
                                className={`p-1.5 rounded-lg transition ${
                                  mb.is_suspended
                                    ? 'hover:bg-emerald-50 dark:hover:bg-emerald-500/20 text-amber-600 hover:text-emerald-600'
                                    : 'hover:bg-amber-50 dark:hover:bg-amber-500/20 text-slate-400 hover:text-amber-600'
                                }`}
                                title={mb.is_suspended ? 'Resume Mailbox (Re-enable Access)' : 'Suspend Mailbox (Block Access)'}
                              >
                                {mb.is_suspended ? <UserCheck className="w-4 h-4" /> : <Lock className="w-4 h-4" />}
                              </button>
                              <button
                                onClick={() => handleDeleteMailbox(mb.id)}
                                className="p-1.5 rounded-lg hover:bg-rose-50 dark:hover:bg-red-500/20 text-slate-400 hover:text-rose-600 transition"
                                title="Delete Mailbox"
                              >
                                <Trash2 className="w-4 h-4" />
                              </button>
                            </td>
                          </tr>
                        );
                      })
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}

        {/* Tab Content: Webmail Client */}
        {activeTab === 'webmail' && (
          <div className="space-y-4">
            <WebmailClient
              mailboxes={mailboxes}
              initialSelectedEmail={selectedWebmailEmail}
              onBackToEmailSettings={() => setActiveTab('mailboxes')}
              showBackToSettings={true}
            />
          </div>
        )}

        {/* Tab Content: Domains */}
        {activeTab === 'domains' && (
          <div className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {filteredDomains.length === 0 ? (
                <div className="col-span-2 p-12 text-center bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl text-slate-400 text-xs space-y-3">
                  <Globe className="w-8 h-8 mx-auto text-slate-300 dark:text-slate-600 stroke-[1.5]" />
                  <p className="font-semibold text-slate-700 dark:text-slate-300 text-sm">No Email Domains Configured</p>
                  <p className="text-slate-500">
                    Add a domain to automatically generate 2048-bit RSA DKIM keys and configure Postfix virtual mailboxes.
                  </p>
                  <button
                    onClick={() => setShowAddDomainModal(true)}
                    className="inline-flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-semibold bg-indigo-600 text-white hover:bg-indigo-500"
                  >
                    <Plus className="w-4 h-4" />
                    <span>Add First Domain</span>
                  </button>
                </div>
              ) : (
                filteredDomains.map((d) => (
                  <div
                    key={d.id}
                    className="p-6 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 space-y-4 shadow-xs"
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <div className="w-10 h-10 rounded-xl bg-indigo-50 dark:bg-indigo-500/10 border border-indigo-200 dark:border-indigo-500/20 flex items-center justify-center text-indigo-600 dark:text-indigo-400">
                          <Globe className="w-5 h-5" />
                        </div>
                        <div>
                          <h3 className="font-bold text-slate-900 dark:text-white text-lg">{d.domain}</h3>
                          <p className="text-xs text-slate-500 font-mono">Mail Host: {d.mail_hostname}</p>
                        </div>
                      </div>
                      <span className="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium bg-emerald-50 dark:bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-500/20">
                        Active
                      </span>
                    </div>

                    <div className="grid grid-cols-3 gap-2 py-2 border-y border-slate-100 dark:border-surface-800 text-xs">
                      <div>
                        <p className="text-slate-500">Mailboxes</p>
                        <p className="text-slate-900 dark:text-white font-semibold mt-0.5">{d.mailbox_count}</p>
                      </div>
                      <div>
                        <p className="text-slate-500">Aliases</p>
                        <p className="text-slate-900 dark:text-white font-semibold mt-0.5">{d.alias_count}</p>
                      </div>
                      <div>
                        <p className="text-slate-500">DKIM Selector</p>
                        <p className="text-indigo-600 dark:text-indigo-400 font-mono font-semibold mt-0.5">
                          {d.dkim_selector}
                        </p>
                      </div>
                    </div>

                    <div className="flex flex-wrap gap-1.5 pt-1">
                      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[10px] font-semibold bg-emerald-50 dark:bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-500/20">
                        <CheckCircle2 className="w-3 h-3" /> DKIM (2048-bit RSA)
                      </span>
                      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[10px] font-semibold bg-blue-50 dark:bg-blue-500/10 text-blue-700 dark:text-blue-400 border border-blue-200 dark:border-blue-500/20">
                        <CheckCircle2 className="w-3 h-3" /> SPF Active
                      </span>
                      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[10px] font-semibold bg-purple-50 dark:bg-purple-500/10 text-purple-700 dark:text-purple-400 border border-purple-200 dark:border-purple-500/20">
                        <CheckCircle2 className="w-3 h-3" /> DMARC Policy
                      </span>
                    </div>

                    <div className="flex items-center justify-between pt-2 border-t border-slate-100 dark:border-surface-800">
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => openDNSModal(d)}
                          className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-semibold bg-indigo-50 dark:bg-indigo-500/10 text-indigo-700 dark:text-indigo-300 hover:bg-indigo-100 border border-indigo-200 dark:border-indigo-500/30 transition"
                        >
                          <FileText className="w-3.5 h-3.5" />
                          <span>DNS Records (SPF, DKIM, DMARC)</span>
                        </button>
                        <button
                          onClick={() => handleVerifyDNS(d.id)}
                          disabled={verifyingDNS}
                          className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-semibold bg-emerald-50 dark:bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 hover:bg-emerald-100 border border-emerald-200 dark:border-emerald-500/30 transition disabled:opacity-50"
                        >
                          <ShieldCheck className="w-3.5 h-3.5" />
                          <span>Verify Live DNS</span>
                        </button>
                      </div>
                      <button
                        onClick={() => handleDeleteDomain(d.id)}
                        className="p-1.5 rounded-lg text-slate-400 hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/20 transition"
                        title="Delete Domain"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>
        )}

        {/* Tab Content: Deliverability Health Audit */}
        {activeTab === 'health' && (
          <div className="space-y-4">
            <div className="p-6 rounded-2xl bg-gradient-to-r from-indigo-50/50 dark:from-indigo-950/40 via-white dark:via-surface-900 to-white dark:to-surface-900 border border-indigo-200 dark:border-indigo-500/20 flex flex-col md:flex-row items-center justify-between gap-6 shadow-xs">
              <div>
                <span className="px-2.5 py-1 rounded-full text-xs font-semibold bg-indigo-100 dark:bg-indigo-500/20 text-indigo-800 dark:text-indigo-300 border border-indigo-200 dark:border-indigo-500/30">
                  Deliverability Health Engine
                </span>
                <h2 className="text-xl font-bold text-slate-900 dark:text-white mt-2">
                  Zero Open Relay, Strict MTA Routing & RFC Compliance
                </h2>
                <p className="text-sm text-slate-600 dark:text-slate-300 mt-1 max-w-2xl">
                  Hostvra measures live DNS records, reverse DNS PTR, FCrDNS, TLS submission, and MTA relay restrictions to
                  prevent spoofing and guarantee high inbox deliverability.
                </p>
              </div>

              {/* Selector & Action */}
              <div className="flex items-center gap-3">
                <select
                  value={healthDomainId}
                  onChange={(e) => {
                    setHealthDomainId(e.target.value);
                    fetchHealthAudit(e.target.value);
                  }}
                  className="px-3.5 py-2 bg-white dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white font-medium focus:outline-none"
                >
                  {domains.map((d) => (
                    <option key={d.id} value={d.id}>
                      {d.domain}
                    </option>
                  ))}
                </select>

                <button
                  onClick={() => fetchHealthAudit(healthDomainId)}
                  disabled={healthLoading || !healthDomainId}
                  className="inline-flex items-center gap-2 px-3.5 py-2 rounded-xl text-xs font-semibold bg-indigo-600 hover:bg-indigo-500 text-white shadow-md shadow-indigo-600/20 transition disabled:opacity-50"
                >
                  <RefreshCw className={`w-3.5 h-3.5 ${healthLoading ? 'animate-spin' : ''}`} />
                  <span>{healthLoading ? 'Auditing...' : 'Run Audit'}</span>
                </button>
              </div>
            </div>

            {/* Health Score Card */}
            {healthData ? (
              <div className="space-y-4">
                <div className="p-6 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 flex flex-col sm:flex-row items-center justify-between gap-6 shadow-xs">
                  <div className="flex items-center gap-4">
                    <div
                      className={`w-20 h-20 rounded-2xl flex flex-col items-center justify-center border font-bold ${
                        healthData.score >= 90
                          ? 'bg-emerald-50 dark:bg-emerald-950/30 border-emerald-300 dark:border-emerald-500/30 text-emerald-600 dark:text-emerald-400'
                          : healthData.score >= 70
                          ? 'bg-amber-50 dark:bg-amber-950/30 border-amber-300 dark:border-amber-500/30 text-amber-600 dark:text-amber-400'
                          : 'bg-rose-50 dark:bg-rose-950/30 border-rose-300 dark:border-rose-500/30 text-rose-600 dark:text-rose-400'
                      }`}
                    >
                      <span className="text-3xl font-black">{healthData.score}</span>
                      <span className="text-[10px] uppercase tracking-wider font-mono">/ 100</span>
                    </div>

                    <div>
                      <div className="flex items-center gap-2">
                        <h3 className="text-lg font-bold text-slate-900 dark:text-white">Health Rating: {healthData.rating}</h3>
                        <span
                          className={`px-2 py-0.5 rounded text-[11px] font-semibold uppercase ${
                            healthData.score >= 90
                              ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-500/20 dark:text-emerald-300'
                              : healthData.score >= 70
                              ? 'bg-amber-100 text-amber-800 dark:bg-amber-500/20 dark:text-amber-300'
                              : 'bg-rose-100 text-rose-800 dark:bg-rose-500/20 dark:text-rose-300'
                          }`}
                        >
                          {healthData.score >= 90 ? 'Optimal' : healthData.score >= 70 ? 'Action Needed' : 'Critical'}
                        </span>
                      </div>
                      <p className="text-xs text-slate-500 mt-1">
                        Audited live against RFC 5321, RFC 7208 (SPF), RFC 6376 (DKIM), and RFC 7489 (DMARC).
                      </p>
                    </div>
                  </div>

                  <div className="flex flex-wrap gap-2 text-xs">
                    <span
                      className={`inline-flex items-center gap-1 px-2.5 py-1 rounded-full font-medium ${
                        healthData.open_relay_rejected
                          ? 'bg-emerald-50 dark:bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 border border-emerald-200'
                          : 'bg-rose-50 text-rose-700 border border-rose-200'
                      }`}
                    >
                      {healthData.open_relay_rejected ? <CheckCircle2 className="w-3.5 h-3.5" /> : <XCircle className="w-3.5 h-3.5" />}
                      <span>Zero Open Relay Guard</span>
                    </span>
                    <span
                      className={`inline-flex items-center gap-1 px-2.5 py-1 rounded-full font-medium ${
                        healthData.tls_valid
                          ? 'bg-emerald-50 dark:bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 border border-emerald-200'
                          : 'bg-rose-50 text-rose-700 border border-rose-200'
                      }`}
                    >
                      {healthData.tls_valid ? <CheckCircle2 className="w-3.5 h-3.5" /> : <XCircle className="w-3.5 h-3.5" />}
                      <span>TLS 587/465 Active</span>
                    </span>
                  </div>
                </div>

                {/* Deductions breakdown if any */}
                {healthData.deductions && healthData.deductions.length > 0 && (
                  <div className="p-5 rounded-2xl bg-amber-50/50 dark:bg-amber-950/20 border border-amber-200 dark:border-amber-500/30 space-y-3">
                    <h4 className="text-xs font-bold uppercase tracking-wider text-amber-800 dark:text-amber-400 flex items-center gap-2">
                      <AlertTriangle className="w-4 h-4" />
                      <span>Itemized Score Deductions & Remediation Plan</span>
                    </h4>
                    <div className="space-y-2 text-xs">
                      {healthData.deductions.map((ded, idx) => (
                        <div
                          key={idx}
                          className="p-3 rounded-xl bg-white dark:bg-surface-900 border border-amber-200 dark:border-amber-500/20 flex flex-col sm:flex-row sm:items-center justify-between gap-2"
                        >
                          <div>
                            <span className="font-semibold text-slate-900 dark:text-white">{ded.reason}</span>
                            <p className="text-slate-500 text-[11px] mt-0.5">Fix: {ded.remediation}</p>
                          </div>
                          <span className="px-2 py-0.5 rounded text-rose-600 dark:text-rose-400 font-bold bg-rose-50 dark:bg-rose-950/40 text-xs shrink-0">
                            -{ded.points} pts
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {/* Audit Checklist */}
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {healthData.checks.map((chk, idx) => (
                    <div
                      key={idx}
                      className="p-5 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 space-y-2 shadow-xs"
                    >
                      <div className="flex items-center justify-between">
                        <h4 className="font-semibold text-slate-900 dark:text-white text-sm flex items-center gap-2">
                          {chk.passed ? (
                            <CheckCircle2 className="w-4 h-4 text-emerald-500" />
                          ) : (
                            <XCircle className="w-4 h-4 text-rose-500" />
                          )}
                          <span>{chk.name}</span>
                        </h4>
                        <span
                          className={`px-2 py-0.5 rounded text-[11px] font-semibold uppercase ${
                            chk.passed
                              ? 'bg-emerald-50 dark:bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-500/20'
                              : 'bg-rose-50 dark:bg-rose-500/10 text-rose-700 dark:text-rose-400 border border-rose-200 dark:border-rose-500/20'
                          }`}
                        >
                          {chk.passed ? 'PASS' : 'FAIL'}
                        </span>
                      </div>
                      <p className="text-xs text-slate-500 dark:text-slate-400">{chk.detail}</p>
                      {!chk.passed && chk.remediation && (
                        <p className="text-[11px] text-amber-700 dark:text-amber-400 bg-amber-50 dark:bg-amber-950/30 p-2 rounded-lg font-mono">
                          Remedy: {chk.remediation}
                        </p>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            ) : (
              <div className="p-12 text-center bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl text-slate-400 text-xs">
                Select a domain and click &quot;Run Audit&quot; to test deliverability health.
              </div>
            )}
          </div>
        )}

        {/* Tab Content: SMTP & Client Settings */}
        {activeTab === 'smtp' && (
          <div className="space-y-4">
            <div className="p-6 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 space-y-4 shadow-xs">
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                <div>
                  <h3 className="text-base font-bold text-slate-900 dark:text-white">
                    Mail Client Configuration Parameters
                  </h3>
                  <p className="text-xs text-slate-500 mt-0.5">
                    Connect Microsoft Outlook, Apple Mail, Thunderbird, iPhone, or Android to Hostvra mailboxes.
                  </p>
                </div>

                <div className="flex items-center gap-2">
                  <label className="text-xs text-slate-500 font-semibold">Domain:</label>
                  <select
                    value={selectedSmtpDomain}
                    onChange={(e) => {
                      setSelectedSmtpDomain(e.target.value);
                      fetchSmtpSettings(e.target.value);
                    }}
                    className="px-3 py-1.5 bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white font-medium focus:outline-none"
                  >
                    {domains.map((d) => (
                      <option key={d.id} value={d.domain}>
                        {d.domain}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              {smtpSettings && (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 pt-2">
                  {/* Incoming IMAP */}
                  <div className="p-5 rounded-2xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 space-y-3">
                    <h4 className="text-xs font-bold uppercase tracking-wider text-indigo-600 dark:text-indigo-400 flex items-center gap-2">
                      <Inbox className="w-4 h-4" />
                      <span>Incoming Server (IMAP / POP3)</span>
                    </h4>

                    <div className="space-y-2 text-xs font-mono">
                      <div className="flex items-center justify-between p-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800">
                        <span className="text-slate-500">Server Host:</span>
                        <div className="flex items-center gap-2">
                          <span className="text-slate-900 dark:text-white font-semibold">
                            {smtpSettings.incoming_server}
                          </span>
                          <button
                            onClick={() => copyToClipboard(smtpSettings.incoming_server, 'imap_host')}
                            className="text-slate-400 hover:text-slate-600"
                          >
                            {copiedKey === 'imap_host' ? <Check className="w-3.5 h-3.5 text-emerald-500" /> : <Copy className="w-3.5 h-3.5" />}
                          </button>
                        </div>
                      </div>

                      <div className="flex items-center justify-between p-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800">
                        <span className="text-slate-500">IMAP Port (Recommended):</span>
                        <span className="text-indigo-600 dark:text-indigo-400 font-semibold">
                          {smtpSettings.incoming_imap_port} (SSL/TLS)
                        </span>
                      </div>

                      <div className="flex items-center justify-between p-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800">
                        <span className="text-slate-500">POP3 Port (Legacy):</span>
                        <span className="text-slate-700 dark:text-slate-300 font-semibold">
                          {smtpSettings.incoming_pop3_port} (SSL/TLS)
                        </span>
                      </div>

                      <div className="flex items-center justify-between p-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800">
                        <span className="text-slate-500">Authentication:</span>
                        <span className="text-slate-700 dark:text-slate-300 font-semibold">Normal Password (Plain / Dovecot)</span>
                      </div>
                    </div>
                  </div>

                  {/* Outgoing SMTP */}
                  <div className="p-5 rounded-2xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 space-y-3">
                    <h4 className="text-xs font-bold uppercase tracking-wider text-blue-600 dark:text-blue-400 flex items-center gap-2">
                      <Send className="w-4 h-4" />
                      <span>Outgoing Server (SMTP Submission)</span>
                    </h4>

                    <div className="space-y-2 text-xs font-mono">
                      <div className="flex items-center justify-between p-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800">
                        <span className="text-slate-500">Server Host:</span>
                        <div className="flex items-center gap-2">
                          <span className="text-slate-900 dark:text-white font-semibold">
                            {smtpSettings.outgoing_server}
                          </span>
                          <button
                            onClick={() => copyToClipboard(smtpSettings.outgoing_server, 'smtp_host')}
                            className="text-slate-400 hover:text-slate-600"
                          >
                            {copiedKey === 'smtp_host' ? <Check className="w-3.5 h-3.5 text-emerald-500" /> : <Copy className="w-3.5 h-3.5" />}
                          </button>
                        </div>
                      </div>

                      <div className="flex items-center justify-between p-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800">
                        <span className="text-slate-500">SMTP Submission (Standard):</span>
                        <span className="text-blue-600 dark:text-blue-400 font-semibold">
                          {smtpSettings.outgoing_smtp_submission_port} (STARTTLS)
                        </span>
                      </div>

                      <div className="flex items-center justify-between p-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800">
                        <span className="text-slate-500">SMTPS Port (Direct SSL):</span>
                        <span className="text-slate-700 dark:text-slate-300 font-semibold">
                          {smtpSettings.outgoing_smtp_ssl_port} (SSL/TLS)
                        </span>
                      </div>

                      <div className="flex items-center justify-between p-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800">
                        <span className="text-slate-500">Authentication:</span>
                        <span className="text-emerald-600 dark:text-emerald-400 font-semibold">Mandatory (Same as IMAP)</span>
                      </div>
                    </div>
                  </div>
                </div>
              )}

              {/* Automated Client Configuration Profiles */}
              <div className="p-5 rounded-2xl bg-gradient-to-r from-slate-50 to-indigo-50/30 dark:from-surface-950 dark:to-indigo-950/20 border border-slate-200 dark:border-surface-800 space-y-4">
                <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                  <div>
                    <h4 className="text-sm font-bold text-slate-900 dark:text-white flex items-center gap-2">
                      <Sparkles className="w-4 h-4 text-indigo-500" />
                      <span>Automated Mail Client Setup Profiles</span>
                    </h4>
                    <p className="text-xs text-slate-500 mt-0.5">
                      Download pre-configured device profiles and discovery XML files for {selectedSmtpDomain || 'your domain'}.
                    </p>
                  </div>
                  <button
                    onClick={() => setActiveTab('guide')}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-white dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-xs font-semibold text-indigo-600 dark:text-indigo-400 hover:bg-slate-50 dark:hover:bg-surface-700 shadow-xs transition"
                  >
                    <BookOpen className="w-3.5 h-3.5" />
                    <span>View 35-Topic User Guide</span>
                  </button>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                  <a
                    href={`/api/v1/email/clients/apple?domain=${selectedSmtpDomain || domains[0]?.domain || 'example.com'}`}
                    download
                    className="p-3.5 rounded-xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 hover:border-indigo-400 dark:hover:border-indigo-600 transition flex flex-col justify-between group shadow-xs"
                  >
                    <div>
                      <div className="flex items-center justify-between text-xs font-bold text-slate-900 dark:text-white">
                        <span>Apple iOS &amp; macOS</span>
                        <Download className="w-3.5 h-3.5 text-slate-400 group-hover:text-indigo-600 transition" />
                      </div>
                      <p className="text-[11px] text-slate-500 mt-1">
                        One-click <code className="font-mono text-indigo-500">.mobileconfig</code> payload profile.
                      </p>
                    </div>
                    <span className="text-[10px] font-bold text-indigo-600 dark:text-indigo-400 mt-2 block">
                      Download Profile &rarr;
                    </span>
                  </a>

                  <a
                    href={`/api/v1/email/clients/thunderbird?domain=${selectedSmtpDomain || domains[0]?.domain || 'example.com'}`}
                    download
                    className="p-3.5 rounded-xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 hover:border-indigo-400 dark:hover:border-indigo-600 transition flex flex-col justify-between group shadow-xs"
                  >
                    <div>
                      <div className="flex items-center justify-between text-xs font-bold text-slate-900 dark:text-white">
                        <span>Mozilla Thunderbird</span>
                        <Download className="w-3.5 h-3.5 text-slate-400 group-hover:text-indigo-600 transition" />
                      </div>
                      <p className="text-[11px] text-slate-500 mt-1">
                        Standard <code className="font-mono text-indigo-500">autoconfig.xml</code> payload.
                      </p>
                    </div>
                    <span className="text-[10px] font-bold text-indigo-600 dark:text-indigo-400 mt-2 block">
                      Download XML &rarr;
                    </span>
                  </a>

                  <a
                    href={`/api/v1/email/clients/outlook?domain=${selectedSmtpDomain || domains[0]?.domain || 'example.com'}`}
                    download
                    className="p-3.5 rounded-xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 hover:border-indigo-400 dark:hover:border-indigo-600 transition flex flex-col justify-between group shadow-xs"
                  >
                    <div>
                      <div className="flex items-center justify-between text-xs font-bold text-slate-900 dark:text-white">
                        <span>Microsoft Outlook</span>
                        <Download className="w-3.5 h-3.5 text-slate-400 group-hover:text-indigo-600 transition" />
                      </div>
                      <p className="text-[11px] text-slate-500 mt-1">
                        Exchange <code className="font-mono text-indigo-500">autodiscover.xml</code> payload.
                      </p>
                    </div>
                    <span className="text-[10px] font-bold text-indigo-600 dark:text-indigo-400 mt-2 block">
                      Download XML &rarr;
                    </span>
                  </a>
                </div>
              </div>

              {/* Client Setup Walkthroughs */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-3 pt-1">
                <div className="p-3.5 rounded-xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 space-y-1.5 shadow-xs">
                  <div className="flex items-center gap-2 font-bold text-xs text-slate-900 dark:text-white">
                    <Layers className="w-4 h-4 text-indigo-500" />
                    <span>Microsoft Outlook</span>
                  </div>
                  <p className="text-[11px] text-slate-500 leading-relaxed">
                    Choose <strong>Manual Setup &gt; IMAP</strong>. Set Incoming Host to <code className="font-mono font-bold text-slate-700 dark:text-slate-300">{smtpSettings?.incoming_server || 'mail.' + (selectedSmtpDomain || 'example.com')}</code> on port <strong>993 (SSL/TLS)</strong>, and Outgoing Host on port <strong>587 (STARTTLS)</strong>.
                  </p>
                </div>

                <div className="p-3.5 rounded-xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 space-y-1.5 shadow-xs">
                  <div className="flex items-center gap-2 font-bold text-xs text-slate-900 dark:text-white">
                    <Mail className="w-4 h-4 text-blue-500" />
                    <span>Mozilla Thunderbird</span>
                  </div>
                  <p className="text-[11px] text-slate-500 leading-relaxed">
                    Enter your email and password. Thunderbird auto-detects ports <strong>993</strong> and <strong>587</strong> via Hostvra's XML autoconfig endpoint.
                  </p>
                </div>

                <div className="p-3.5 rounded-xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 space-y-1.5 shadow-xs">
                  <div className="flex items-center gap-2 font-bold text-xs text-slate-900 dark:text-white">
                    <Cpu className="w-4 h-4 text-purple-500" />
                    <span>Apple Mail &amp; iOS</span>
                  </div>
                  <p className="text-[11px] text-slate-500 leading-relaxed">
                    Download the <code className="font-mono text-indigo-500">.mobileconfig</code> profile above or manually add an IMAP account with username as full email and SSL enabled.
                  </p>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Tab Content: Mail Queue */}
        {activeTab === 'queue' && (
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-base font-bold text-slate-900 dark:text-white">Postfix Outbound Mail Queue</h3>
                <p className="text-xs text-slate-500 mt-0.5">
                  Real-time queue contents monitored directly via <code className="font-mono">postqueue -p</code>.
                </p>
              </div>

              <div className="flex items-center gap-2">
                <button
                  onClick={fetchQueue}
                  disabled={queueLoading}
                  className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-white dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-xs font-semibold text-slate-700 dark:text-slate-200"
                >
                  <RefreshCw className={`w-3.5 h-3.5 ${queueLoading ? 'animate-spin' : ''}`} />
                  <span>Refresh</span>
                </button>

                <button
                  onClick={handleFlushQueue}
                  disabled={flushingQueue}
                  className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-xl text-xs font-semibold bg-indigo-600 hover:bg-indigo-500 text-white shadow-sm transition disabled:opacity-50"
                >
                  <RotateCw className={`w-3.5 h-3.5 ${flushingQueue ? 'animate-spin' : ''}`} />
                  <span>{flushingQueue ? 'Flushing Postfix...' : 'Flush Queue (postqueue -f)'}</span>
                </button>
              </div>
            </div>

            {queueLoading ? (
              <div className="p-12 text-center text-slate-400 text-xs">
                <RefreshCw className="w-5 h-5 animate-spin mx-auto text-indigo-500 mb-2" />
                <span>Querying Postfix spool...</span>
              </div>
            ) : queueItems.length === 0 ? (
              <div className="p-8 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 text-center space-y-4 shadow-xs">
                <div className="w-12 h-12 rounded-xl bg-emerald-50 dark:bg-emerald-500/10 border border-emerald-200 dark:border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400 mx-auto">
                  <Check className="w-6 h-6" />
                </div>
                <div>
                  <h3 className="text-lg font-bold text-slate-900 dark:text-white">Postfix Mail Queue is Empty</h3>
                  <p className="text-sm text-slate-500 max-w-md mx-auto mt-1">
                    All outgoing messages have been accepted by destination MTAs. There are no deferred, hold, or frozen emails.
                  </p>
                </div>
              </div>
            ) : (
              <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl overflow-hidden shadow-xs">
                <div className="overflow-x-auto">
                  <table className="w-full text-left text-sm border-collapse">
                    <thead>
                      <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-surface-950 text-slate-700 dark:text-slate-300 text-[11px] font-bold uppercase tracking-wider">
                        <th className="px-6 py-4">Queue ID</th>
                        <th className="px-6 py-4">Sender</th>
                        <th className="px-6 py-4">Recipients</th>
                        <th className="px-6 py-4">Arrival Time</th>
                        <th className="px-6 py-4">Reason / Deferral</th>
                        <th className="px-6 py-4 text-right">Action</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100 dark:divide-surface-800/80 font-mono text-xs">
                      {queueItems.map((item) => (
                        <tr key={item.id} className="hover:bg-slate-50 dark:hover:bg-surface-800/40">
                          <td className="px-6 py-4 font-bold text-indigo-600 dark:text-indigo-400">{item.id}</td>
                          <td className="px-6 py-4 text-slate-900 dark:text-white">{item.sender}</td>
                          <td className="px-6 py-4 text-slate-700 dark:text-slate-300">{item.recipients.join(', ')}</td>
                          <td className="px-6 py-4 text-slate-500">{item.arrival_time}</td>
                          <td className="px-6 py-4 text-amber-600 dark:text-amber-400">{item.reason || 'Deferred'}</td>
                          <td className="px-6 py-4 text-right">
                            <button
                              onClick={() => handleDeleteQueueItem(item.id)}
                              className="p-1.5 rounded-lg text-slate-400 hover:text-rose-600 transition"
                              title="Delete from Queue (postsuper -d)"
                            >
                              <Trash2 className="w-4 h-4" />
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </div>
        )}

        {/* Tab Content: Delivery Logs */}
        {activeTab === 'logs' && (
          <div className="space-y-4">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
              <div className="flex items-center gap-2 w-full max-w-sm bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-xl px-3.5 py-2 shadow-xs focus-within:border-indigo-500">
                <Search className="w-4 h-4 text-slate-400 flex-shrink-0" />
                <input
                  type="text"
                  placeholder="Filter logs by sender or recipient..."
                  value={logsSearch}
                  onChange={(e) => setLogsSearch(e.target.value)}
                  className="w-full bg-transparent text-xs font-medium text-slate-900 dark:text-white placeholder:text-slate-400 focus:outline-none"
                />
              </div>

              <div className="flex items-center gap-2">
                <select
                  value={logsFilterStatus}
                  onChange={(e) => setLogsFilterStatus(e.target.value)}
                  className="px-3 py-2 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-xl text-xs text-slate-700 dark:text-slate-200"
                >
                  <option value="all">All Delivery Statuses</option>
                  <option value="delivered">Delivered (250 OK)</option>
                  <option value="deferred">Deferred (421/450)</option>
                  <option value="rejected">Rejected (550/554)</option>
                  <option value="bounced">Bounced</option>
                </select>

                <button
                  onClick={fetchLogs}
                  disabled={logsLoading}
                  className="p-2 rounded-xl bg-white dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-slate-600 dark:text-slate-300"
                  title="Refresh Logs"
                >
                  <RefreshCw className={`w-3.5 h-3.5 ${logsLoading ? 'animate-spin' : ''}`} />
                </button>
              </div>
            </div>

            <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl overflow-hidden shadow-xs">
              <div className="overflow-x-auto">
                <table className="w-full text-left text-sm border-collapse">
                  <thead>
                    <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-surface-950 text-slate-700 dark:text-slate-300 text-[11px] font-bold uppercase tracking-wider">
                      <th className="px-6 py-4">Timestamp</th>
                      <th className="px-6 py-4">Sender</th>
                      <th className="px-6 py-4">Recipient</th>
                      <th className="px-6 py-4">Status</th>
                      <th className="px-6 py-4">Spam Score</th>
                      <th className="px-6 py-4">Message ID</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 dark:divide-surface-800/80 font-mono text-xs">
                    {filteredLogs.length === 0 ? (
                      <tr>
                        <td colSpan={6} className="px-6 py-12 text-center text-slate-400 text-xs">
                          {deliveryLogs.length === 0
                            ? 'No MTA delivery logs recorded yet.'
                            : 'No delivery logs match your filter criteria.'}
                        </td>
                      </tr>
                    ) : (
                      filteredLogs.map((log) => (
                        <tr key={log.id} className="hover:bg-slate-50 dark:hover:bg-surface-800/40 transition-colors">
                          <td className="px-6 py-4 text-slate-500">{new Date(log.timestamp).toLocaleString()}</td>
                          <td className="px-6 py-4 font-bold text-slate-900 dark:text-white">{log.sender}</td>
                          <td className="px-6 py-4 text-slate-700 dark:text-slate-300">{log.recipient}</td>
                          <td className="px-6 py-4">
                            <span
                              className={`px-2 py-0.5 rounded text-[11px] font-semibold ${
                                log.status.toLowerCase().includes('delivered') || log.status.includes('250')
                                  ? 'bg-emerald-50 dark:bg-emerald-500/10 text-emerald-700 dark:text-emerald-400'
                                  : log.status.toLowerCase().includes('deferred')
                                  ? 'bg-amber-50 dark:bg-amber-500/10 text-amber-700 dark:text-amber-400'
                                  : 'bg-rose-50 dark:bg-rose-500/10 text-rose-700 dark:text-rose-400'
                              }`}
                            >
                              {log.status}
                            </span>
                          </td>
                          <td className="px-6 py-4 text-slate-500">{log.spam_score.toFixed(2)}</td>
                          <td className="px-6 py-4 text-slate-400 truncate max-w-xs">{log.message_id || '-'}</td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}

        {/* Tab Content: Suppressions */}
        {activeTab === 'suppressions' && (
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-base font-bold text-slate-900 dark:text-white">Email Suppression List</h3>
                <p className="text-xs text-slate-500 mt-0.5">
                  Addresses blocked from delivery due to hard bounces, spam complaints, or manual suppression.
                </p>
              </div>
              <button
                onClick={fetchSuppressions}
                disabled={suppressionsLoading}
                className="p-2 rounded-xl bg-white dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-slate-600 dark:text-slate-300"
              >
                <RefreshCw className={`w-3.5 h-3.5 ${suppressionsLoading ? 'animate-spin' : ''}`} />
              </button>
            </div>

            <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl overflow-hidden shadow-xs">
              <div className="overflow-x-auto">
                <table className="w-full text-left text-sm border-collapse">
                  <thead>
                    <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-surface-950 text-slate-700 dark:text-slate-300 text-[11px] font-bold uppercase tracking-wider">
                      <th className="px-6 py-4">Suppressed Email</th>
                      <th className="px-6 py-4">Reason</th>
                      <th className="px-6 py-4">Details</th>
                      <th className="px-6 py-4">Suppressed Since</th>
                      <th className="px-6 py-4 text-right">Action</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 dark:divide-surface-800/80 font-mono text-xs">
                    {suppressions.length === 0 ? (
                      <tr>
                        <td colSpan={5} className="px-6 py-12 text-center text-slate-400 text-xs">
                          No email addresses are currently suppressed. Clean reputation!
                        </td>
                      </tr>
                    ) : (
                      suppressions.map((sup) => (
                        <tr key={sup.id} className="hover:bg-slate-50 dark:hover:bg-surface-800/40">
                          <td className="px-6 py-4 font-bold text-slate-900 dark:text-white">{sup.email}</td>
                          <td className="px-6 py-4">
                            <span className="px-2 py-0.5 rounded text-[11px] font-semibold bg-rose-50 text-rose-700 dark:bg-rose-500/10 dark:text-rose-400">
                              {sup.reason}
                            </span>
                          </td>
                          <td className="px-6 py-4 text-slate-500">{sup.details}</td>
                          <td className="px-6 py-4 text-slate-500">{new Date(sup.created_at).toLocaleDateString()}</td>
                          <td className="px-6 py-4 text-right">
                            <button
                              onClick={() => handleRemoveSuppression(sup.id)}
                              className="px-2.5 py-1 rounded-lg text-xs font-semibold text-indigo-600 dark:text-indigo-400 hover:bg-indigo-50 dark:hover:bg-indigo-950/40 transition"
                            >
                              Unsuppress
                            </button>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}

        {/* Tab Content: Services & Daemons */}
        {activeTab === 'services' && (
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-base font-bold text-slate-900 dark:text-white">Email Daemons & Services</h3>
                <p className="text-xs text-slate-500 mt-0.5">
                  Direct management of Postfix MTA, Dovecot IMAP, Rspamd Milter, and ClamAV Antivirus.
                </p>
              </div>
              <button
                onClick={fetchServices}
                disabled={servicesLoading}
                className="p-2 rounded-xl bg-white dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-slate-600 dark:text-slate-300"
              >
                <RefreshCw className={`w-3.5 h-3.5 ${servicesLoading ? 'animate-spin' : ''}`} />
              </button>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {services.map((svc) => (
                <div
                  key={svc.name}
                  className="p-6 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 space-y-4 shadow-xs"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-xl bg-indigo-50 dark:bg-indigo-500/10 border border-indigo-200 dark:border-indigo-500/20 flex items-center justify-center text-indigo-600 dark:text-indigo-400">
                        <Activity className="w-5 h-5" />
                      </div>
                      <div>
                        <h4 className="font-bold text-slate-900 dark:text-white text-base capitalize">{svc.name}</h4>
                        <p className="text-xs text-slate-500 font-mono">Version: {svc.version || 'installed'}</p>
                      </div>
                    </div>
                    <span
                      className={`inline-flex items-center px-2.5 py-1 rounded-full text-xs font-semibold ${
                        svc.status === 'running'
                          ? 'bg-emerald-50 dark:bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-500/20'
                          : 'bg-rose-50 text-rose-700 border border-rose-200'
                      }`}
                    >
                      {svc.status.toUpperCase()}
                    </span>
                  </div>

                  <div className="grid grid-cols-3 gap-2 py-2 border-y border-slate-100 dark:border-surface-800 text-xs">
                    <div>
                      <p className="text-slate-500">Active Ports</p>
                      <p className="text-slate-900 dark:text-white font-mono font-semibold mt-0.5">
                        {svc.active_ports && svc.active_ports.length > 0 ? svc.active_ports.join(', ') : 'IPC / Milter'}
                      </p>
                    </div>
                    <div>
                      <p className="text-slate-500">Memory</p>
                      <p className="text-slate-900 dark:text-white font-semibold mt-0.5">
                        {svc.memory_bytes ? `${(svc.memory_bytes / (1024 * 1024)).toFixed(0)} MB` : 'Dynamic'}
                      </p>
                    </div>
                    <div>
                      <p className="text-slate-500">Uptime</p>
                      <p className="text-slate-900 dark:text-white font-semibold mt-0.5">
                        {svc.uptime_seconds ? `${Math.floor(svc.uptime_seconds / 3600)}h` : 'Active'}
                      </p>
                    </div>
                  </div>

                  <div className="flex items-center justify-end gap-2 pt-1">
                    <button
                      onClick={() => handleServiceAction(svc.name, 'reload')}
                      disabled={serviceActionLoading === `${svc.name}-reload`}
                      className="px-3 py-1.5 rounded-lg text-xs font-semibold bg-slate-100 dark:bg-surface-800 hover:bg-slate-200 text-slate-700 dark:text-slate-200 transition"
                    >
                      {serviceActionLoading === `${svc.name}-reload` ? 'Reloading...' : 'Reload Config'}
                    </button>
                    <button
                      onClick={() => handleServiceAction(svc.name, 'restart')}
                      disabled={serviceActionLoading === `${svc.name}-restart`}
                      className="px-3 py-1.5 rounded-lg text-xs font-semibold bg-indigo-600 hover:bg-indigo-500 text-white shadow-xs transition"
                    >
                      {serviceActionLoading === `${svc.name}-restart` ? 'Restarting...' : 'Restart Service'}
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Tab Content: Live Send Test Email Tool */}
        {activeTab === 'tester' && (
          <div className="space-y-4">
            <div className="p-6 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 space-y-4 shadow-xs">
              <div>
                <h3 className="text-base font-bold text-slate-900 dark:text-white">Live SMTP Handshake Transmitter</h3>
                <p className="text-xs text-slate-500 mt-0.5">
                  Send a live test message and monitor the exact Postfix SMTP dialogue (EHLO, STARTTLS, AUTH, 250 OK).
                </p>
              </div>

              <form onSubmit={handleSendTestEmail} className="space-y-4 max-w-2xl">
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                      Sender Mailbox (From)
                    </label>
                    <select
                      value={testEmailFrom}
                      onChange={(e) => setTestEmailFrom(e.target.value)}
                      className="w-full px-3.5 py-2 bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white font-mono focus:outline-none"
                    >
                      {mailboxes.map((mb) => (
                        <option key={mb.id} value={mb.email}>
                          {mb.email}
                        </option>
                      ))}
                    </select>
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                      Recipient Email (To)
                    </label>
                    <input
                      type="email"
                      required
                      placeholder="check-auth@verifier.port25.com or user@gmail.com"
                      value={testEmailTo}
                      onChange={(e) => setTestEmailTo(e.target.value)}
                      className="w-full px-3.5 py-2 bg-white dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">Subject</label>
                  <input
                    type="text"
                    required
                    value={testEmailSubject}
                    onChange={(e) => setTestEmailSubject(e.target.value)}
                    className="w-full px-3.5 py-2 bg-white dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                    Message Body
                  </label>
                  <textarea
                    rows={4}
                    value={testEmailBody}
                    onChange={(e) => setTestEmailBody(e.target.value)}
                    className="w-full px-3.5 py-2 bg-white dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500 font-mono"
                  />
                </div>

                <button
                  type="submit"
                  disabled={sendingTestEmail || !testEmailFrom || !testEmailTo}
                  className="px-5 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-md shadow-indigo-600/25 flex items-center gap-2 transition disabled:opacity-50"
                >
                  <Send className="w-4 h-4" />
                  <span>{sendingTestEmail ? 'Transmitting Handshake...' : 'Transmit Test Email'}</span>
                </button>
              </form>

              {/* Live SMTP Transcript */}
              {testEmailResult && (
                <div className="pt-4 border-t border-slate-200 dark:border-surface-800 space-y-3">
                  <div className="flex items-center justify-between">
                    <h4 className="text-xs font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300 flex items-center gap-2">
                      <Terminal className="w-4 h-4 text-emerald-500" />
                      <span>SMTP Handshake Dialogue</span>
                    </h4>
                    <span
                      className={`px-2 py-0.5 rounded text-xs font-semibold ${
                        testEmailResult.success
                          ? 'bg-emerald-50 text-emerald-700 border border-emerald-200'
                          : 'bg-rose-50 text-rose-700 border border-rose-200'
                      }`}
                    >
                      {testEmailResult.success
                        ? `Delivered in ${testEmailResult.latency_ms}ms`
                        : 'Delivery Failed'}
                    </span>
                  </div>

                  <div className="p-4 rounded-xl bg-slate-950 border border-slate-800 text-emerald-400 font-mono text-xs space-y-1 overflow-x-auto max-h-72">
                    {testEmailResult.transcript.map((line, idx) => (
                      <p key={idx} className="whitespace-pre">
                        {line}
                      </p>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Tab Content: User Guide (35 Topics) */}
        {activeTab === 'guide' && (
          <EmailUserGuide
            currentDomain={selectedSmtpDomain || domains[0]?.domain || 'example.com'}
            mailHostname={smtpSettings?.incoming_server || `mail.${selectedSmtpDomain || domains[0]?.domain || 'example.com'}`}
            serverIPv4={mailServers[0]?.ipv4_address || '127.0.0.1'}
            imapPort={smtpSettings?.incoming_imap_port || 143}
            imapsPort={993}
            pop3Port={smtpSettings?.incoming_pop3_port || 110}
            pop3sPort={995}
            smtpSubmissionPort={smtpSettings?.outgoing_smtp_submission_port || 587}
            smtpsPort={smtpSettings?.outgoing_smtp_ssl_port || 465}
            storagePath={mailServers[0]?.storage_location || '/var/mail/vhosts'}
            onSelectTab={(tab) => setActiveTab(tab as any)}
          />
        )}

        {/* Modal: Add Email Domain */}
        {showAddDomainModal && (
          <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl w-full max-w-md p-6 space-y-5 shadow-2xl">
              <div className="flex items-center justify-between">
                <h3 className="text-lg font-bold text-slate-900 dark:text-white flex items-center gap-2">
                  <Globe className="w-5 h-5 text-indigo-500" />
                  <span>Add Email Domain</span>
                </h3>
                <button onClick={() => setShowAddDomainModal(false)} className="text-slate-400 hover:text-slate-700">
                  ✕
                </button>
              </div>

              <form onSubmit={handleAddDomain} className="space-y-4">
                <div>
                  <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">Domain Name</label>
                  <input
                    type="text"
                    required
                    placeholder="example.com"
                    value={newDomainName}
                    onChange={(e) => {
                      setNewDomainName(e.target.value);
                      if (!newDomainMailHost && e.target.value) {
                        setNewDomainMailHost(`mail.${e.target.value.toLowerCase().trim()}`);
                      }
                    }}
                    className="w-full px-3.5 py-2.5 bg-white dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-sm text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500 font-mono"
                  />
                  <p className="text-[11px] text-slate-500 mt-1">
                    Hostvra will automatically generate 2048-bit RSA DKIM keys, configure Postfix virtual mailboxes, and prepare SPF/DMARC records.
                  </p>
                </div>

                <div className="grid grid-cols-2 gap-3 pt-1">
                  <div>
                    <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">Mail Hostname</label>
                    <input
                      type="text"
                      placeholder={newDomainName ? `mail.${newDomainName}` : 'mail.example.com'}
                      value={newDomainMailHost}
                      onChange={(e) => setNewDomainMailHost(e.target.value)}
                      className="w-full px-3 py-2 bg-white dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500 font-mono"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">DKIM Selector</label>
                    <input
                      type="text"
                      value={newDomainDKIMSelector}
                      onChange={(e) => setNewDomainDKIMSelector(e.target.value)}
                      className="w-full px-3 py-2 bg-white dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500 font-mono"
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">Domain Storage Quota (GB)</label>
                  <input
                    type="number"
                    min={1}
                    max={5000}
                    value={newDomainStorageGB}
                    onChange={(e) => setNewDomainStorageGB(parseInt(e.target.value) || 50)}
                    className="w-full px-3 py-2 bg-white dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                  />
                </div>

                <div className="pt-3 flex justify-end gap-3">
                  <button
                    type="button"
                    onClick={() => setShowAddDomainModal(false)}
                    className="px-4 py-2 rounded-xl text-sm font-medium text-slate-600 dark:text-slate-300 hover:bg-slate-100"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={addingDomain}
                    className="px-4 py-2 rounded-xl text-sm font-medium bg-indigo-600 hover:bg-indigo-500 text-white shadow-lg shadow-indigo-600/20 disabled:opacity-50"
                  >
                    {addingDomain ? 'Generating DKIM...' : 'Add & Generate DKIM'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* Modal: Create Mailbox */}
        {showAddMailboxModal && (
          <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl w-full max-w-md p-6 space-y-5 shadow-2xl">
              <div className="flex items-center justify-between">
                <h3 className="text-lg font-bold text-slate-900 dark:text-white flex items-center gap-2">
                  <Inbox className="w-5 h-5 text-indigo-500" />
                  <span>Create Mailbox Account</span>
                </h3>
                <button onClick={() => setShowAddMailboxModal(false)} className="text-slate-400 hover:text-slate-700">
                  ✕
                </button>
              </div>

              {domains.length === 0 ? (
                <div className="p-6 bg-amber-50 dark:bg-amber-950/30 border border-amber-200 dark:border-amber-800/60 rounded-2xl text-center space-y-3">
                  <AlertTriangle className="w-10 h-10 text-amber-500 mx-auto" />
                  <h4 className="text-sm font-bold text-amber-900 dark:text-amber-200">No Email Domains Configured Yet</h4>
                  <p className="text-xs text-amber-700 dark:text-amber-300/90 leading-relaxed">
                    You need to add an email domain first before creating mailboxes. Hostvra will generate a 2048-bit RSA DKIM key and prepare DNS records for your domain.
                  </p>
                  <div className="pt-2 flex justify-center gap-3">
                    <button
                      type="button"
                      onClick={() => setShowAddMailboxModal(false)}
                      className="px-4 py-2 rounded-xl text-xs font-medium text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-surface-800"
                    >
                      Close
                    </button>
                    <button
                      type="button"
                      onClick={() => {
                        setShowAddMailboxModal(false);
                        setShowAddDomainModal(true);
                      }}
                      className="inline-flex items-center gap-2 px-4 py-2 bg-indigo-600 hover:bg-indigo-500 text-white rounded-xl text-xs font-semibold shadow-md shadow-indigo-600/20 transition-all"
                    >
                      <Plus className="w-3.5 h-3.5" />
                      <span>Add Email Domain First</span>
                    </button>
                  </div>
                </div>
              ) : (
                <form onSubmit={handleAddMailbox} className="space-y-4">
                  <div>
                    <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">Select Domain</label>
                    <select
                      value={newMailboxDomain}
                      onChange={(e) => setNewMailboxDomain(e.target.value)}
                      className="w-full px-3.5 py-2.5 bg-white dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-sm text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                    >
                      {domains.map((d) => (
                        <option key={d.id} value={d.id}>
                          @{d.domain} ({d.mail_hostname || `mail.${d.domain}`})
                        </option>
                      ))}
                    </select>
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">Username (Local-Part)</label>
                    <div className="flex items-center">
                      <input
                        type="text"
                        required
                        placeholder="info or support"
                        value={newLocalPart}
                        onChange={(e) => setNewLocalPart(e.target.value)}
                        className="flex-1 px-3.5 py-2.5 bg-white dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-l-xl text-sm text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500 font-mono"
                      />
                      <span className="px-3 py-2.5 bg-slate-100 dark:bg-surface-800 border border-l-0 border-slate-200 dark:border-surface-800 rounded-r-xl text-xs text-indigo-600 dark:text-indigo-400 font-mono font-semibold">
                        @{domains.find((d) => d.id === newMailboxDomain)?.domain || domains[0]?.domain || 'yourdomain.com'}
                      </span>
                    </div>
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">Display Name</label>
                    <input
                      type="text"
                      placeholder="e.g. Sales Department"
                      value={newMailboxName}
                      onChange={(e) => setNewMailboxName(e.target.value)}
                      className="w-full px-3.5 py-2.5 bg-white dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-sm text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                    />
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">Password</label>
                    <input
                      type="password"
                      required
                      placeholder="At least 8 characters"
                      value={newMailboxPass}
                      onChange={(e) => setNewMailboxPass(e.target.value)}
                      className="w-full px-3.5 py-2.5 bg-white dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-sm text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500 font-mono"
                    />
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                      Storage Quota (GB)
                    </label>
                    <input
                      type="number"
                      min={1}
                      max={100}
                      value={newMailboxQuotaGB}
                      onChange={(e) => setNewMailboxQuotaGB(parseInt(e.target.value) || 5)}
                      className="w-full px-3.5 py-2.5 bg-white dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-sm text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                    />
                  </div>

                  <div className="pt-3 flex justify-end gap-3">
                    <button
                      type="button"
                      onClick={() => setShowAddMailboxModal(false)}
                      className="px-4 py-2 rounded-xl text-sm font-medium text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-surface-800"
                    >
                      Cancel
                    </button>
                    <button
                      type="submit"
                      disabled={creatingMailbox}
                      className="px-4 py-2 rounded-xl text-sm font-medium bg-indigo-600 hover:bg-indigo-500 text-white shadow-lg shadow-indigo-600/20 disabled:opacity-50"
                    >
                      {creatingMailbox ? 'Creating...' : 'Create Mailbox'}
                    </button>
                  </div>
                </form>
              )}
            </div>
          </div>
        )}

        {/* Modal: Live Cryptographic DNS Records (SPF, DKIM, DMARC, MX) */}
        {showDNSModal && (
          <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl w-full max-w-3xl p-6 space-y-5 shadow-2xl max-h-[90vh] flex flex-col">
              <div className="flex items-center justify-between pb-3 border-b border-slate-100 dark:border-surface-800">
                <div>
                  <h3 className="text-lg font-bold text-slate-900 dark:text-white flex items-center gap-2">
                    <ShieldCheck className="w-5 h-5 text-emerald-500" />
                    <span>DNS Configuration & Cryptographic Keys for {showDNSModal.domain}</span>
                  </h3>
                  <p className="text-xs text-slate-500 mt-0.5">
                    Publish these records in Cloudflare, cPanel, Namecheap, or your registrar to pass SPF, DKIM (2048-bit), and DMARC.
                  </p>
                </div>
                <button
                  onClick={() => setShowDNSModal(null)}
                  className="text-slate-400 hover:text-slate-700 dark:hover:text-white p-1 rounded-lg"
                >
                  ✕
                </button>
              </div>

              {/* Action Toolbar */}
              <div className="flex flex-wrap items-center justify-between gap-2 p-3 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 text-xs">
                <div className="flex items-center gap-2">
                  <span className="font-semibold text-slate-700 dark:text-slate-300">Quick Actions:</span>
                  <button
                    onClick={() => copyAllZoneRecords(showDNSModal, domainDNSRecords)}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg font-semibold bg-indigo-600 hover:bg-indigo-500 text-white shadow-xs transition"
                  >
                    {copiedKey === 'zone_all' ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
                    <span>{copiedKey === 'zone_all' ? 'Copied Zone File!' : 'Copy All Records (BIND / Cloudflare)'}</span>
                  </button>
                </div>

                <div className="flex items-center gap-2">
                  <button
                    onClick={() => handleRegenerateDKIM(showDNSModal.id)}
                    disabled={regeneratingDKIM}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg font-medium bg-white dark:bg-surface-800 text-slate-700 dark:text-slate-200 hover:bg-slate-100 border border-slate-200 dark:border-surface-700 transition disabled:opacity-50"
                  >
                    <RotateCw className={`w-3.5 h-3.5 ${regeneratingDKIM ? 'animate-spin' : ''}`} />
                    <span>Regenerate DKIM</span>
                  </button>
                  <button
                    onClick={() => handleVerifyDNS(showDNSModal.id)}
                    disabled={verifyingDNS}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg font-medium bg-emerald-50 dark:bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 hover:bg-emerald-100 border border-emerald-200 dark:border-emerald-500/30 transition disabled:opacity-50"
                  >
                    <ShieldCheck className="w-3.5 h-3.5" />
                    <span>Verify Live Propagation</span>
                  </button>
                </div>
              </div>

              {/* Records Scrollable List */}
              <div className="space-y-3 overflow-y-auto pr-1 flex-1 font-mono text-xs">
                {loadingDNS ? (
                  <div className="p-8 text-center text-slate-400 space-y-2">
                    <RefreshCw className="w-6 h-6 mx-auto animate-spin text-indigo-500" />
                    <p className="font-sans text-xs">Loading cryptographically signed DKIM, SPF, DMARC & MX records...</p>
                  </div>
                ) : domainDNSRecords.length > 0 ? (
                  domainDNSRecords.map((rec, idx) => {
                    const copyHostKey = `host_${idx}`;
                    const copyValKey = `val_${idx}`;
                    return (
                      <div
                        key={idx}
                        className="p-3.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 space-y-2"
                      >
                        <div className="flex items-center justify-between font-sans">
                          <div className="flex items-center gap-2">
                            <span
                              className={`px-2 py-0.5 rounded text-[11px] font-bold uppercase tracking-wider ${
                                rec.record_type === 'MX'
                                  ? 'bg-indigo-100 dark:bg-indigo-500/20 text-indigo-700 dark:text-indigo-300'
                                  : rec.record_type === 'TXT'
                                  ? 'bg-emerald-100 dark:bg-emerald-500/20 text-emerald-700 dark:text-emerald-300'
                                  : rec.record_type === 'A'
                                  ? 'bg-sky-100 dark:bg-sky-500/20 text-sky-700 dark:text-sky-300'
                                  : 'bg-purple-100 dark:bg-purple-500/20 text-purple-700 dark:text-purple-300'
                              }`}
                            >
                              {rec.record_type} Record
                            </span>
                            <span className="text-xs text-slate-500">{rec.message}</span>
                          </div>
                          <span className="text-[11px] text-slate-400">TTL: 3600</span>
                        </div>

                        <div className="grid grid-cols-1 md:grid-cols-3 gap-2 pt-1">
                          <div className="p-2.5 rounded-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 flex items-center justify-between">
                            <div className="overflow-hidden">
                              <span className="font-sans text-[10px] text-slate-400 block uppercase">Host / Name</span>
                              <span className="text-slate-900 dark:text-white font-semibold truncate block">
                                {rec.host}
                              </span>
                            </div>
                            <button
                              onClick={() => copyToClipboard(rec.host, copyHostKey)}
                              className="ml-2 p-1 text-slate-400 hover:text-indigo-600 rounded"
                              title="Copy Host"
                            >
                              {copiedKey === copyHostKey ? (
                                <Check className="w-3.5 h-3.5 text-emerald-500" />
                              ) : (
                                <Copy className="w-3.5 h-3.5" />
                              )}
                            </button>
                          </div>

                          <div className="md:col-span-2 p-2.5 rounded-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 flex items-center justify-between">
                            <div className="overflow-hidden pr-2">
                              <span className="font-sans text-[10px] text-slate-400 block uppercase">Value / Target</span>
                              <span className="text-indigo-600 dark:text-indigo-400 font-semibold break-all text-[11px] block">
                                {rec.expected}
                              </span>
                            </div>
                            <button
                              onClick={() => copyToClipboard(rec.expected, copyValKey)}
                              className="p-1 text-slate-400 hover:text-indigo-600 rounded flex-shrink-0"
                              title="Copy Value"
                            >
                              {copiedKey === copyValKey ? (
                                <Check className="w-3.5 h-3.5 text-emerald-500" />
                              ) : (
                                <Copy className="w-3.5 h-3.5" />
                              )}
                            </button>
                          </div>
                        </div>
                      </div>
                    );
                  })
                ) : (
                  <div className="space-y-3 font-mono text-xs">
                    {/* Fallback Display */}
                    <div className="p-3.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 space-y-1">
                      <div className="flex justify-between items-center text-slate-500 font-sans">
                        <span className="font-semibold text-slate-900 dark:text-white">MX Record (Incoming Mail)</span>
                        <button
                          onClick={() => copyToClipboard(`10 ${showDNSModal.mail_hostname}.`, 'mx')}
                          className="hover:text-indigo-600 flex items-center gap-1 text-[11px]"
                        >
                          {copiedKey === 'mx' ? <Check className="w-3.5 h-3.5 text-emerald-500" /> : <Copy className="w-3.5 h-3.5" />}
                          <span>Copy</span>
                        </button>
                      </div>
                      <p className="text-indigo-600 dark:text-indigo-400">Host: @ | Priority: 10 | Target: 10 {showDNSModal.mail_hostname}.</p>
                    </div>

                    <div className="p-3.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 space-y-1">
                      <div className="flex justify-between items-center text-slate-500 font-sans">
                        <span className="font-semibold text-slate-900 dark:text-white">SPF Record (Sender Policy Framework)</span>
                        <button
                          onClick={() => copyToClipboard('v=spf1 mx a ~all', 'spf')}
                          className="hover:text-indigo-600 flex items-center gap-1 text-[11px]"
                        >
                          {copiedKey === 'spf' ? <Check className="w-3.5 h-3.5 text-emerald-500" /> : <Copy className="w-3.5 h-3.5" />}
                          <span>Copy</span>
                        </button>
                      </div>
                      <p className="text-indigo-600 dark:text-indigo-400">Type: TXT | Host: @ | Value: v=spf1 mx a ~all</p>
                    </div>

                    <div className="p-3.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 space-y-1">
                      <div className="flex justify-between items-center text-slate-500 font-sans">
                        <span className="font-semibold text-slate-900 dark:text-white">DMARC Policy Record</span>
                        <button
                          onClick={() => copyToClipboard(`v=DMARC1; p=quarantine; sp=quarantine; rua=mailto:dmarc@${showDNSModal.domain}`, 'dmarc')}
                          className="hover:text-indigo-600 flex items-center gap-1 text-[11px]"
                        >
                          {copiedKey === 'dmarc' ? <Check className="w-3.5 h-3.5 text-emerald-500" /> : <Copy className="w-3.5 h-3.5" />}
                          <span>Copy</span>
                        </button>
                      </div>
                      <p className="text-indigo-600 dark:text-indigo-400">Type: TXT | Host: _dmarc | Value: v=DMARC1; p=quarantine; sp=quarantine; rua=mailto:dmarc@{showDNSModal.domain}</p>
                    </div>
                  </div>
                )}
              </div>

              <div className="pt-3 flex justify-between items-center border-t border-slate-100 dark:border-surface-800 font-sans">
                <span className="text-xs text-slate-500">
                  Tip: Most DNS updates propagate worldwide within 2 to 15 minutes.
                </span>
                <button
                  onClick={() => setShowDNSModal(null)}
                  className="px-5 py-2 rounded-xl text-sm font-semibold bg-indigo-600 hover:bg-indigo-500 text-white shadow-md shadow-indigo-600/20 transition"
                >
                  Done
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Modal: Live DNS Verification Result */}
        {showVerifyDNSModal && (
          <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl w-full max-w-lg p-6 space-y-5 shadow-2xl">
              <div className="flex items-center justify-between">
                <h3 className="text-lg font-bold text-slate-900 dark:text-white flex items-center gap-2">
                  <ShieldCheck className="w-5 h-5 text-indigo-500" />
                  <span>Live DNS Query for {showVerifyDNSModal.domain}</span>
                </h3>
                <button onClick={() => setShowVerifyDNSModal(null)} className="text-slate-400 hover:text-slate-700">
                  ✕
                </button>
              </div>

              <div className="space-y-3 text-xs">
                <div className="p-3 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 flex items-center justify-between">
                  <span className="font-semibold text-slate-700 dark:text-slate-300">MX Record:</span>
                  <span
                    className={`px-2 py-0.5 rounded font-semibold ${
                      showVerifyDNSModal.mx_valid
                        ? 'bg-emerald-50 text-emerald-700'
                        : 'bg-rose-50 text-rose-700'
                    }`}
                  >
                    {showVerifyDNSModal.mx_valid ? 'Valid (Incoming Active)' : 'Not Pointed'}
                  </span>
                </div>

                <div className="p-3 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 flex items-center justify-between">
                  <span className="font-semibold text-slate-700 dark:text-slate-300">SPF Record (TXT):</span>
                  <span
                    className={`px-2 py-0.5 rounded font-semibold ${
                      showVerifyDNSModal.spf_valid
                        ? 'bg-emerald-50 text-emerald-700'
                        : 'bg-rose-50 text-rose-700'
                    }`}
                  >
                    {showVerifyDNSModal.spf_valid ? 'Valid (RFC 7208)' : 'Missing or Duplicate'}
                  </span>
                </div>

                <div className="p-3 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 flex items-center justify-between">
                  <span className="font-semibold text-slate-700 dark:text-slate-300">DKIM Record:</span>
                  <span
                    className={`px-2 py-0.5 rounded font-semibold ${
                      showVerifyDNSModal.dkim_valid
                        ? 'bg-emerald-50 text-emerald-700'
                        : 'bg-amber-50 text-amber-700'
                    }`}
                  >
                    {showVerifyDNSModal.dkim_valid ? 'Active (RSA-2048)' : 'Pending DNS Propagation'}
                  </span>
                </div>

                <div className="p-3 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 flex items-center justify-between">
                  <span className="font-semibold text-slate-700 dark:text-slate-300">DMARC Policy:</span>
                  <span
                    className={`px-2 py-0.5 rounded font-semibold ${
                      showVerifyDNSModal.dmarc_valid
                        ? 'bg-emerald-50 text-emerald-700'
                        : 'bg-amber-50 text-amber-700'
                    }`}
                  >
                    {showVerifyDNSModal.dmarc_valid ? 'Configured' : 'Missing _dmarc TXT'}
                  </span>
                </div>

                {showVerifyDNSModal.issues && showVerifyDNSModal.issues.length > 0 && (
                  <div className="p-3 rounded-xl bg-amber-50 dark:bg-amber-950/20 border border-amber-200 text-amber-800 dark:text-amber-400 space-y-1">
                    <p className="font-semibold">Detected Issues:</p>
                    {showVerifyDNSModal.issues.map((iss, i) => (
                      <p key={i}>• {iss}</p>
                    ))}
                  </div>
                )}
              </div>

              <div className="pt-2 flex justify-end">
                <button
                  onClick={() => setShowVerifyDNSModal(null)}
                  className="px-4 py-2 rounded-xl text-xs font-semibold bg-indigo-600 text-white"
                >
                  Close
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Modal: Test Mailbox Credentials */}
        {showTestMailboxModal && (
          <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl w-full max-w-md p-6 space-y-5 shadow-2xl">
              <div className="flex items-center justify-between">
                <h3 className="text-base font-bold text-slate-900 dark:text-white flex items-center gap-2">
                  <Radio className="w-5 h-5 text-indigo-500" />
                  <span>Credential Authentication Probe</span>
                </h3>
                <button onClick={() => setShowTestMailboxModal(null)} className="text-slate-400 hover:text-slate-700">
                  ✕
                </button>
              </div>

              <div className="space-y-3 text-xs">
                <p className="text-slate-600 dark:text-slate-300">
                  Testing live Dovecot IMAP and Postfix SMTP authentication for{' '}
                  <span className="font-mono font-bold text-slate-900 dark:text-white">{showTestMailboxModal.email}</span>.
                </p>

                {testingMailbox ? (
                  <div className="p-6 text-center text-slate-400 space-y-2">
                    <RefreshCw className="w-5 h-5 animate-spin mx-auto text-indigo-500" />
                    <p>Executing live SASL & IMAP handshakes...</p>
                  </div>
                ) : mailboxTestResult ? (
                  <div className="space-y-2">
                    <div className="p-3 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 flex items-center justify-between">
                      <span className="font-semibold text-slate-700 dark:text-slate-300">SMTP Auth (Port 587):</span>
                      <span
                        className={`px-2 py-0.5 rounded font-semibold ${
                          mailboxTestResult.smtp_auth_passed
                            ? 'bg-emerald-50 text-emerald-700'
                            : 'bg-rose-50 text-rose-700'
                        }`}
                      >
                        {mailboxTestResult.smtp_auth_passed
                          ? `Passed (${mailboxTestResult.smtp_latency_ms}ms)`
                          : 'Auth Failed'}
                      </span>
                    </div>

                    <div className="p-3 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 flex items-center justify-between">
                      <span className="font-semibold text-slate-700 dark:text-slate-300">IMAP Auth (Port 993):</span>
                      <span
                        className={`px-2 py-0.5 rounded font-semibold ${
                          mailboxTestResult.imap_auth_passed
                            ? 'bg-emerald-50 text-emerald-700'
                            : 'bg-rose-50 text-rose-700'
                        }`}
                      >
                        {mailboxTestResult.imap_auth_passed
                          ? `Passed (${mailboxTestResult.imap_latency_ms}ms)`
                          : 'Auth Failed'}
                      </span>
                    </div>

                    <p className="text-[11px] text-slate-500 p-2 bg-slate-50 dark:bg-surface-950 rounded-lg font-mono">
                      {mailboxTestResult.details}
                    </p>
                  </div>
                ) : null}
              </div>

              <div className="pt-2 flex justify-end">
                <button
                  onClick={() => setShowTestMailboxModal(null)}
                  className="px-4 py-2 rounded-xl text-xs font-semibold bg-indigo-600 text-white"
                >
                  Done
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Modal: Client Setup Guide */}
        {showClientSetupModal && (
          <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl w-full max-w-lg p-6 space-y-5 shadow-2xl">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-base font-bold text-slate-900 dark:text-white">
                    Mail Client Settings ({showClientSetupModal.email})
                  </h3>
                  <p className="text-xs text-slate-500">
                    Use these values in Microsoft Outlook, Apple Mail, or Thunderbird.
                  </p>
                </div>
                <button onClick={() => setShowClientSetupModal(null)} className="text-slate-400 hover:text-slate-700">
                  ✕
                </button>
              </div>

              <div className="space-y-2 text-xs font-mono">
                <div className="p-2.5 rounded-lg bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 flex items-center justify-between">
                  <span className="text-slate-500">Username:</span>
                  <span className="text-slate-900 dark:text-white font-semibold">{showClientSetupModal.email}</span>
                </div>
                <div className="p-2.5 rounded-lg bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 flex items-center justify-between">
                  <span className="text-slate-500">IMAP Server:</span>
                  <span className="text-indigo-600 dark:text-indigo-400 font-semibold">
                    mail.{showClientSetupModal.email.split('@')[1]} (Port 993 SSL)
                  </span>
                </div>
                <div className="p-2.5 rounded-lg bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 flex items-center justify-between">
                  <span className="text-slate-500">SMTP Server:</span>
                  <span className="text-blue-600 dark:text-blue-400 font-semibold">
                    mail.{showClientSetupModal.email.split('@')[1]} (Port 587 STARTTLS)
                  </span>
                </div>
              </div>

              <div className="pt-2 flex justify-end">
                <button
                  onClick={() => setShowClientSetupModal(null)}
                  className="px-4 py-2 rounded-xl text-xs font-semibold bg-indigo-600 text-white"
                >
                  Close
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Modal: Password Reset */}
        {showPasswordModal && (
          <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl w-full max-w-md p-6 space-y-5 shadow-2xl">
              <div className="flex items-center justify-between">
                <h3 className="text-lg font-bold text-slate-900 dark:text-white flex items-center gap-2">
                  <Key className="w-5 h-5 text-indigo-500" />
                  <span>Reset Mailbox Password</span>
                </h3>
                <button onClick={() => setShowPasswordModal(null)} className="text-slate-400 hover:text-slate-700">
                  ✕
                </button>
              </div>

              <div className="space-y-4">
                <p className="text-sm text-slate-600 dark:text-slate-300">
                  Update password for <span className="font-mono text-slate-900 dark:text-white font-semibold">{showPasswordModal.email}</span>
                </p>

                <div>
                  <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">New Password</label>
                  <input
                    type="password"
                    placeholder="Enter new strong password"
                    value={newPassword}
                    onChange={(e) => setNewPassword(e.target.value)}
                    className="w-full px-3.5 py-2.5 bg-white dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-sm text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                  />
                  <p className="text-[11px] text-slate-500 mt-1">
                    Password will be encrypted using Dovecot SHA512-CRYPT scheme.
                  </p>
                </div>

                <div className="pt-2 flex justify-end gap-3">
                  <button
                    onClick={() => setShowPasswordModal(null)}
                    className="px-4 py-2 rounded-xl text-sm font-medium text-slate-600 dark:text-slate-300 hover:bg-slate-100"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={handleSavePassword}
                    className="px-4 py-2 rounded-xl text-sm font-medium bg-indigo-600 hover:bg-indigo-500 text-white shadow-lg shadow-indigo-600/20"
                  >
                    Save Password
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Modal: Create Mail Server Wizard */}
        {showCreateServerModal && (
          <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-sm flex items-center justify-center p-4 overflow-y-auto">
            <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl w-full max-w-3xl max-h-[90vh] flex flex-col shadow-2xl overflow-hidden my-auto">
              {/* Modal Header */}
              <div className="p-5 border-b border-slate-200 dark:border-surface-800 flex items-center justify-between bg-slate-50/50 dark:bg-surface-950/50">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-xl bg-indigo-50 dark:bg-indigo-950/40 border border-indigo-200 dark:border-indigo-800 flex items-center justify-center text-indigo-600 dark:text-indigo-400">
                    <Server className="w-5 h-5" />
                  </div>
                  <div>
                    <h3 className="text-base font-bold text-slate-900 dark:text-white">Provision Enterprise Mail Server</h3>
                    <p className="text-xs text-slate-500">Autonomous Postfix + Dovecot + Rspamd node orchestration</p>
                  </div>
                </div>
                <button
                  onClick={() => setShowCreateServerModal(false)}
                  className="text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 p-1"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              {/* Wizard Tabs Navigation */}
              <div className="flex items-center gap-1 px-5 pt-3 border-b border-slate-200 dark:border-surface-800 bg-slate-50/20 dark:bg-surface-950/20 overflow-x-auto text-xs font-semibold">
                <button
                  type="button"
                  onClick={() => setServerWizardTab('general')}
                  className={`px-3 py-2 border-b-2 transition-all whitespace-nowrap ${
                    serverWizardTab === 'general'
                      ? 'border-indigo-600 text-indigo-600 dark:text-indigo-400'
                      : 'border-transparent text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
                  }`}
                >
                  1. General Settings
                </button>
                <button
                  type="button"
                  onClick={() => setServerWizardTab('storage_ports')}
                  className={`px-3 py-2 border-b-2 transition-all whitespace-nowrap ${
                    serverWizardTab === 'storage_ports'
                      ? 'border-indigo-600 text-indigo-600 dark:text-indigo-400'
                      : 'border-transparent text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
                  }`}
                >
                  2. Storage & Ports
                </button>
                <button
                  type="button"
                  onClick={() => setServerWizardTab('security')}
                  className={`px-3 py-2 border-b-2 transition-all whitespace-nowrap ${
                    serverWizardTab === 'security'
                      ? 'border-indigo-600 text-indigo-600 dark:text-indigo-400'
                      : 'border-transparent text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
                  }`}
                >
                  3. Security & Daemons
                </button>
                <button
                  type="button"
                  onClick={() => setServerWizardTab('ratelimits')}
                  className={`px-3 py-2 border-b-2 transition-all whitespace-nowrap ${
                    serverWizardTab === 'ratelimits'
                      ? 'border-indigo-600 text-indigo-600 dark:text-indigo-400'
                      : 'border-transparent text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
                  }`}
                >
                  4. Rate Limits & Anti-Abuse
                </button>
                <button
                  type="button"
                  onClick={() => setServerWizardTab('preflight')}
                  className={`px-3 py-2 border-b-2 transition-all whitespace-nowrap ${
                    serverWizardTab === 'preflight'
                      ? 'border-indigo-600 text-indigo-600 dark:text-indigo-400'
                      : 'border-transparent text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
                  }`}
                >
                  5. Preflight Validation
                  {preflightResult && (
                    <span
                      className={`ml-1.5 px-1.5 py-0.2 rounded-full text-[10px] font-bold ${
                        preflightResult.passed ? 'bg-emerald-100 text-emerald-800' : 'bg-rose-100 text-rose-800'
                      }`}
                    >
                      {preflightResult.passed ? 'PASS' : 'WARN'}
                    </span>
                  )}
                </button>
              </div>

              {/* Wizard Content */}
              <form onSubmit={handleCreateMailServer} className="flex-1 overflow-y-auto p-6 space-y-5">
                {serverCreationError && (
                  <div className="p-3.5 rounded-xl bg-rose-50 dark:bg-rose-950/30 border border-rose-200 dark:border-rose-900/50 flex items-start gap-2.5 text-xs text-rose-800 dark:text-rose-300">
                    <AlertTriangle className="w-4 h-4 flex-shrink-0 mt-0.5" />
                    <span>{serverCreationError}</span>
                  </div>
                )}

                {/* Tab 1: General */}
                {serverWizardTab === 'general' && (
                  <div className="space-y-4">
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                      <div>
                        <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                          Node Display Name
                        </label>
                        <input
                          type="text"
                          value={serverName}
                          onChange={(e) => setServerName(e.target.value)}
                          placeholder="e.g. Primary Mail Node"
                          required
                          className="w-full px-3.5 py-2.5 bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                        />
                      </div>
                      <div>
                        <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                          Mail Server Hostname (FQDN)
                        </label>
                        <input
                          type="text"
                          value={serverHostname}
                          onChange={(e) => setServerHostname(e.target.value)}
                          placeholder="e.g. mail.example.com"
                          required
                          className="w-full px-3.5 py-2.5 bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs font-mono text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                        />
                        <p className="text-[11px] text-slate-400 mt-1">
                          Should match the reverse DNS (PTR) record of the server IP.
                        </p>
                      </div>
                    </div>

                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                      <div>
                        <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                          Primary Mail Domain
                        </label>
                        <input
                          type="text"
                          value={serverPrimaryDomain}
                          onChange={(e) => setServerPrimaryDomain(e.target.value)}
                          placeholder="e.g. example.com"
                          required
                          className="w-full px-3.5 py-2.5 bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs font-mono text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                        />
                      </div>
                      <div>
                        <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                          Server IPv4 Address
                        </label>
                        <input
                          type="text"
                          value={serverIPv4}
                          onChange={(e) => setServerIPv4(e.target.value)}
                          placeholder="e.g. 198.51.100.1 or 127.0.0.1"
                          required
                          className="w-full px-3.5 py-2.5 bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs font-mono text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                        />
                      </div>
                    </div>

                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                      <div>
                        <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                          Server IPv6 Address (Optional)
                        </label>
                        <input
                          type="text"
                          value={serverIPv6}
                          onChange={(e) => setServerIPv6(e.target.value)}
                          placeholder="e.g. 2001:db8::1"
                          className="w-full px-3.5 py-2.5 bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs font-mono text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                        />
                      </div>
                      <div>
                        <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                          Server Timezone
                        </label>
                        <input
                          type="text"
                          value={serverTimezone}
                          onChange={(e) => setServerTimezone(e.target.value)}
                          placeholder="e.g. UTC"
                          className="w-full px-3.5 py-2.5 bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                        />
                      </div>
                    </div>

                    <div>
                      <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                        Additional Mail Domains (One per line or comma-separated)
                      </label>
                      <textarea
                        value={serverAdditionalDomains}
                        onChange={(e) => setServerAdditionalDomains(e.target.value)}
                        placeholder="app.example.com&#10;sales.example.com"
                        rows={2}
                        className="w-full px-3.5 py-2 bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs font-mono text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                      />
                    </div>
                  </div>
                )}

                {/* Tab 2: Storage & Ports */}
                {serverWizardTab === 'storage_ports' && (
                  <div className="space-y-4">
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                      <div>
                        <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                          Storage Location (Maildir Base)
                        </label>
                        <input
                          type="text"
                          value={serverStorageLocation}
                          onChange={(e) => setServerStorageLocation(e.target.value)}
                          placeholder="/var/mail/vhosts"
                          required
                          className="w-full px-3.5 py-2.5 bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs font-mono text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                        />
                        <p className="text-[11px] text-slate-400 mt-1">Owned by vmail:vmail (UID 5000:5000)</p>
                      </div>
                      <div>
                        <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                          Mailbox Storage Pool Limit (GB)
                        </label>
                        <input
                          type="number"
                          value={serverMailboxStorageLimitGB}
                          onChange={(e) => setServerMailboxStorageLimitGB(Number(e.target.value))}
                          min={1}
                          required
                          className="w-full px-3.5 py-2.5 bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                        />
                      </div>
                    </div>

                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                      <div>
                        <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                          Max Individual Mailbox Size (GB)
                        </label>
                        <input
                          type="number"
                          value={serverMaxMailboxSizeGB}
                          onChange={(e) => setServerMaxMailboxSizeGB(Number(e.target.value))}
                          min={1}
                          required
                          className="w-full px-3.5 py-2.5 bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                        />
                      </div>
                      <div>
                        <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                          Max Message & Attachment Size (MB)
                        </label>
                        <input
                          type="number"
                          value={serverMaxAttachmentSizeMB}
                          onChange={(e) => setServerMaxAttachmentSizeMB(Number(e.target.value))}
                          min={5}
                          required
                          className="w-full px-3.5 py-2.5 bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                        />
                        <p className="text-[11px] text-slate-400 mt-1">Configured in Postfix message_size_limit</p>
                      </div>
                    </div>

                    <div className="p-3.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 space-y-3">
                      <h4 className="text-xs font-bold text-slate-900 dark:text-white">Daemon Listening Ports</h4>
                      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs font-mono">
                        <div>
                          <label className="block text-[11px] text-slate-500 mb-1">SMTP Port</label>
                          <input
                            type="number"
                            value={serverSmtpPort}
                            onChange={(e) => setServerSmtpPort(Number(e.target.value))}
                            className="w-full px-2.5 py-1.5 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-lg text-xs"
                          />
                        </div>
                        <div>
                          <label className="block text-[11px] text-slate-500 mb-1">Submission (STARTTLS)</label>
                          <input
                            type="number"
                            value={serverSmtpSubmissionPort}
                            onChange={(e) => setServerSmtpSubmissionPort(Number(e.target.value))}
                            className="w-full px-2.5 py-1.5 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-lg text-xs"
                          />
                        </div>
                        <div>
                          <label className="block text-[11px] text-slate-500 mb-1">SMTPS (TLS)</label>
                          <input
                            type="number"
                            value={serverSmtpsPort}
                            onChange={(e) => setServerSmtpsPort(Number(e.target.value))}
                            className="w-full px-2.5 py-1.5 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-lg text-xs"
                          />
                        </div>
                        <div>
                          <label className="block text-[11px] text-slate-500 mb-1">IMAP (143)</label>
                          <input
                            type="number"
                            value={serverImapPort}
                            onChange={(e) => setServerImapPort(Number(e.target.value))}
                            className="w-full px-2.5 py-1.5 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-lg text-xs"
                          />
                        </div>
                        <div>
                          <label className="block text-[11px] text-slate-500 mb-1">IMAPS (993)</label>
                          <input
                            type="number"
                            value={serverImapsPort}
                            onChange={(e) => setServerImapsPort(Number(e.target.value))}
                            className="w-full px-2.5 py-1.5 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-lg text-xs"
                          />
                        </div>
                        <div>
                          <label className="block text-[11px] text-slate-500 mb-1">POP3 (110)</label>
                          <input
                            type="number"
                            value={serverPop3Port}
                            onChange={(e) => setServerPop3Port(Number(e.target.value))}
                            className="w-full px-2.5 py-1.5 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-lg text-xs"
                          />
                        </div>
                        <div>
                          <label className="block text-[11px] text-slate-500 mb-1">POP3S (995)</label>
                          <input
                            type="number"
                            value={serverPop3sPort}
                            onChange={(e) => setServerPop3sPort(Number(e.target.value))}
                            className="w-full px-2.5 py-1.5 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-lg text-xs"
                          />
                        </div>
                      </div>
                    </div>
                  </div>
                )}

                {/* Tab 3: Security & Daemons */}
                {serverWizardTab === 'security' && (
                  <div className="space-y-3">
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
                      <label className="p-3 rounded-xl border border-slate-200 dark:border-surface-800 bg-slate-50/50 dark:bg-surface-950/50 flex items-start gap-3 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={serverTlsEnabled}
                          onChange={(e) => setServerTlsEnabled(e.target.checked)}
                          className="mt-1 rounded text-indigo-600 focus:ring-indigo-500"
                        />
                        <div>
                          <span className="font-bold text-slate-900 dark:text-white block">TLS / SSL Encryption</span>
                          <span className="text-slate-500 text-[11px]">Enforce opportunistic STARTTLS and mandatory TLS for IMAPS/SMTPS</span>
                        </div>
                      </label>

                      <label className="p-3 rounded-xl border border-slate-200 dark:border-surface-800 bg-slate-50/50 dark:bg-surface-950/50 flex items-start gap-3 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={serverSpamFilterEnabled}
                          onChange={(e) => setServerSpamFilterEnabled(e.target.checked)}
                          className="mt-1 rounded text-indigo-600 focus:ring-indigo-500"
                        />
                        <div>
                          <span className="font-bold text-slate-900 dark:text-white block">Rspamd Spam Filtering</span>
                          <span className="text-slate-500 text-[11px]">Bayesian classification, greylisting, and milter integration</span>
                        </div>
                      </label>

                      <label className="p-3 rounded-xl border border-slate-200 dark:border-surface-800 bg-slate-50/50 dark:bg-surface-950/50 flex items-start gap-3 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={serverDkimEnabled}
                          onChange={(e) => setServerDkimEnabled(e.target.checked)}
                          className="mt-1 rounded text-indigo-600 focus:ring-indigo-500"
                        />
                        <div>
                          <span className="font-bold text-slate-900 dark:text-white block">DKIM RSA-2048 Signing</span>
                          <span className="text-slate-500 text-[11px]">Automated key generation and milter header signing</span>
                        </div>
                      </label>

                      <label className="p-3 rounded-xl border border-slate-200 dark:border-surface-800 bg-slate-50/50 dark:bg-surface-950/50 flex items-start gap-3 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={serverSpfEnabled}
                          onChange={(e) => setServerSpfEnabled(e.target.checked)}
                          className="mt-1 rounded text-indigo-600 focus:ring-indigo-500"
                        />
                        <div>
                          <span className="font-bold text-slate-900 dark:text-white block">SPF Record Provisioning</span>
                          <span className="text-slate-500 text-[11px]">Synthesize strict SPF records with matching server IP</span>
                        </div>
                      </label>

                      <label className="p-3 rounded-xl border border-slate-200 dark:border-surface-800 bg-slate-50/50 dark:bg-surface-950/50 flex items-start gap-3 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={serverDmarcEnabled}
                          onChange={(e) => setServerDmarcEnabled(e.target.checked)}
                          className="mt-1 rounded text-indigo-600 focus:ring-indigo-500"
                        />
                        <div>
                          <span className="font-bold text-slate-900 dark:text-white block">DMARC Policy Enforcement</span>
                          <span className="text-slate-500 text-[11px]">Deploy _dmarc TXT with quarantine/reject deliverability alignment</span>
                        </div>
                      </label>

                      <label className="p-3 rounded-xl border border-slate-200 dark:border-surface-800 bg-slate-50/50 dark:bg-surface-950/50 flex items-start gap-3 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={serverWebmailEnabled}
                          onChange={(e) => setServerWebmailEnabled(e.target.checked)}
                          className="mt-1 rounded text-indigo-600 focus:ring-indigo-500"
                        />
                        <div>
                          <span className="font-bold text-slate-900 dark:text-white block">Roundcube Webmail</span>
                          <span className="text-slate-500 text-[11px]">Browser-based webmail client configured for this mail node</span>
                        </div>
                      </label>

                      <label className="p-3 rounded-xl border border-slate-200 dark:border-surface-800 bg-slate-50/50 dark:bg-surface-950/50 flex items-start gap-3 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={serverAntivirusEnabled}
                          onChange={(e) => setServerAntivirusEnabled(e.target.checked)}
                          className="mt-1 rounded text-indigo-600 focus:ring-indigo-500"
                        />
                        <div>
                          <span className="font-bold text-slate-900 dark:text-white block">ClamAV Antivirus</span>
                          <span className="text-slate-500 text-[11px]">Attachment virus scanning (Note: requires 1.5GB+ RAM)</span>
                        </div>
                      </label>

                      <label className="p-3 rounded-xl border border-slate-200 dark:border-surface-800 bg-slate-50/50 dark:bg-surface-950/50 flex items-start gap-3 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={serverAutoSslEnabled}
                          onChange={(e) => setServerAutoSslEnabled(e.target.checked)}
                          className="mt-1 rounded text-indigo-600 focus:ring-indigo-500"
                        />
                        <div>
                          <span className="font-bold text-slate-900 dark:text-white block">Auto-SSL (Let's Encrypt)</span>
                          <span className="text-slate-500 text-[11px]">Automated certificate issuance and 90-day renewal cycle</span>
                        </div>
                      </label>
                    </div>
                  </div>
                )}

                {/* Tab 4: Rate Limits & Anti-Abuse */}
                {serverWizardTab === 'ratelimits' && (
                  <div className="space-y-4">
                    <div className="p-3.5 rounded-xl bg-purple-50 dark:bg-purple-950/30 border border-purple-200 dark:border-purple-900/50 text-xs text-purple-900 dark:text-purple-300">
                      <p className="font-bold mb-1">Zero Arbitrary Daily Cap Policy</p>
                      <p className="text-[11px] leading-relaxed">
                        Hostvra does not limit your daily email volume. The rate limits below serve strictly as defensive abuse and spam controls to protect your server IP from reputation damage and blacklist listing.
                      </p>
                    </div>

                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                      <div>
                        <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                          Per-Mailbox Rate Limit (emails/hour)
                        </label>
                        <input
                          type="number"
                          value={serverRateLimitMailboxHr}
                          onChange={(e) => setServerRateLimitMailboxHr(Number(e.target.value))}
                          min={0}
                          className="w-full px-3.5 py-2.5 bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                        />
                        <p className="text-[11px] text-slate-400 mt-1">Set to 0 for unlimited</p>
                      </div>
                      <div>
                        <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                          Per-Domain Rate Limit (emails/hour)
                        </label>
                        <input
                          type="number"
                          value={serverRateLimitDomainHr}
                          onChange={(e) => setServerRateLimitDomainHr(Number(e.target.value))}
                          min={0}
                          className="w-full px-3.5 py-2.5 bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                        />
                        <p className="text-[11px] text-slate-400 mt-1">Set to 0 for unlimited</p>
                      </div>
                    </div>

                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                      <div>
                        <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                          Per-Connecting-IP Rate Limit (emails/hour)
                        </label>
                        <input
                          type="number"
                          value={serverRateLimitIpHr}
                          onChange={(e) => setServerRateLimitIpHr(Number(e.target.value))}
                          min={0}
                          className="w-full px-3.5 py-2.5 bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                        />
                        <p className="text-[11px] text-slate-400 mt-1">Set to 0 for unlimited</p>
                      </div>
                      <div>
                        <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                          SMTP Auth Failure Ban Threshold
                        </label>
                        <input
                          type="number"
                          value={serverAuthFailureThreshold}
                          onChange={(e) => setServerAuthFailureThreshold(Number(e.target.value))}
                          min={3}
                          max={100}
                          className="w-full px-3.5 py-2.5 bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                        />
                        <p className="text-[11px] text-slate-400 mt-1">Number of failed password attempts before temporary IP ban</p>
                      </div>
                    </div>
                  </div>
                )}

                {/* Tab 5: Preflight Validation */}
                {serverWizardTab === 'preflight' && (
                  <div className="space-y-4">
                    <div className="flex items-center justify-between p-3.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800">
                      <div>
                        <h4 className="text-xs font-bold text-slate-900 dark:text-white">Hostvra Agent System Preflight</h4>
                        <p className="text-[11px] text-slate-500">
                          Validates EUID privileges, listening ports, conflicting MTAs, and storage writeability.
                        </p>
                      </div>
                      <button
                        type="button"
                        onClick={handleRunPreflight}
                        disabled={preflightLoading}
                        className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold bg-indigo-600 hover:bg-indigo-500 text-white shadow-xs transition-all"
                      >
                        <RefreshCw className={`w-3.5 h-3.5 ${preflightLoading ? 'animate-spin' : ''}`} />
                        <span>{preflightLoading ? 'Checking...' : 'Run Preflight Check'}</span>
                      </button>
                    </div>

                    {preflightResult ? (
                      <div className="space-y-2">
                        <div className="flex items-center justify-between text-xs px-1">
                          <span className="font-semibold text-slate-700 dark:text-slate-300">
                            Results for {preflightResult.hostname} ({preflightResult.checked_at})
                          </span>
                          <span
                            className={`px-2 py-0.5 rounded text-[10px] font-bold ${
                              preflightResult.passed ? 'bg-emerald-50 text-emerald-700' : 'bg-rose-50 text-rose-700'
                            }`}
                          >
                            {preflightResult.passed ? 'ALL CHECKS SATISFIED' : 'SYSTEM ISSUES DETECTED'}
                          </span>
                        </div>
                        <div className="divide-y divide-slate-100 dark:divide-surface-800/80 border border-slate-200 dark:border-surface-800 rounded-xl overflow-hidden text-xs">
                          {preflightResult.checks.map((chk, i) => (
                            <div key={i} className="p-3 bg-white dark:bg-surface-900 flex items-start justify-between gap-3">
                              <div className="space-y-0.5">
                                <div className="flex items-center gap-2">
                                  <span className="font-semibold text-slate-900 dark:text-white">{chk.name}</span>
                                  <span className="text-[10px] text-slate-400 uppercase font-mono">[{chk.category}]</span>
                                </div>
                                <p className="text-[11px] text-slate-800 dark:text-slate-200 font-medium">{chk.message}</p>
                                {chk.details && <p className="text-[10px] font-mono text-slate-700 dark:text-slate-300">{chk.details}</p>}
                              </div>
                              <span
                                className={`px-2 py-0.5 rounded text-[10px] font-bold ${
                                  chk.status === 'passed'
                                    ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-400'
                                    : chk.status === 'warning'
                                    ? 'bg-amber-50 text-amber-700 dark:bg-amber-950/40 dark:text-amber-400'
                                    : 'bg-rose-50 text-rose-700 dark:bg-rose-950/40 dark:text-rose-400'
                                }`}
                              >
                                {chk.status.toUpperCase()}
                              </span>
                            </div>
                          ))}
                        </div>
                      </div>
                    ) : (
                      <div className="p-8 text-center border border-dashed border-slate-200 dark:border-surface-800 rounded-xl space-y-2">
                        <Activity className="w-6 h-6 text-slate-400 mx-auto" />
                        <p className="text-xs text-slate-600 dark:text-slate-300 font-semibold">Preflight check has not been executed yet.</p>
                        <p className="text-[11px] text-slate-400 max-w-sm mx-auto">
                          Click "Run Preflight Check" above to test real port binding permissions, storage access, and ensure no conflicting sendmail or exim daemons are running.
                        </p>
                      </div>
                    )}
                  </div>
                )}

                {/* Modal Footer Controls */}
                <div className="pt-4 border-t border-slate-200 dark:border-surface-800 flex items-center justify-between">
                  <button
                    type="button"
                    onClick={() => setShowCreateServerModal(false)}
                    className="px-4 py-2 rounded-xl text-xs font-medium text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-surface-800"
                  >
                    Cancel
                  </button>

                  <div className="flex items-center gap-2">
                    {serverWizardTab !== 'preflight' ? (
                      <button
                        type="button"
                        onClick={() => {
                          if (serverWizardTab === 'general') setServerWizardTab('storage_ports');
                          else if (serverWizardTab === 'storage_ports') setServerWizardTab('security');
                          else if (serverWizardTab === 'security') setServerWizardTab('ratelimits');
                          else if (serverWizardTab === 'ratelimits') setServerWizardTab('preflight');
                        }}
                        className="px-4 py-2 rounded-xl text-xs font-semibold bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-200 hover:bg-slate-200 dark:hover:bg-surface-700"
                      >
                        Next Step →
                      </button>
                    ) : null}

                    <button
                      type="submit"
                      disabled={creatingServer}
                      className="flex items-center gap-1.5 px-5 py-2 rounded-xl text-xs font-semibold bg-indigo-600 hover:bg-indigo-500 text-white shadow-lg shadow-indigo-600/20 transition-all disabled:opacity-50"
                    >
                      {creatingServer ? (
                        <>
                          <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                          <span>Provisioning Node...</span>
                        </>
                      ) : (
                        <>
                          <Server className="w-3.5 h-3.5" />
                          <span>Deploy Mail Server Node</span>
                        </>
                      )}
                    </button>
                  </div>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* Modal: View Server Provisioning Logs */}
        {serverLogsModal && (
          <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl w-full max-w-2xl p-6 space-y-4 shadow-2xl">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-base font-bold text-slate-900 dark:text-white flex items-center gap-2">
                    <FileText className="w-4 h-4 text-indigo-500" />
                    <span>Provisioning Audit Logs</span>
                  </h3>
                  <p className="text-xs font-mono text-slate-500 mt-0.5">{serverLogsModal.hostname} ({serverLogsModal.id})</p>
                </div>
                <button onClick={() => setServerLogsModal(null)} className="text-slate-400 hover:text-slate-700 p-1">
                  <X className="w-5 h-5" />
                </button>
              </div>

              <pre
                className="font-mono text-xs p-4 rounded-xl overflow-x-auto max-h-[60vh] whitespace-pre-wrap leading-relaxed border border-slate-300 dark:border-surface-700 font-semibold audit-logs-pre text-black"
                style={{ color: '#000000', backgroundColor: '#F8FAFC' }}
              >
                {serverLogsModal.provisioning_logs || 'No provisioning logs recorded for this mail node.'}
              </pre>

              <div className="pt-2 flex justify-end">
                <button
                  onClick={() => setServerLogsModal(null)}
                  className="px-4 py-2 rounded-xl text-xs font-semibold bg-indigo-600 text-white hover:bg-indigo-500"
                >
                  Close
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
