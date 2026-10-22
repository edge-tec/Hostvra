'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Send,
  ShieldCheck,
  ShieldAlert,
  Server,
  RefreshCw,
  Plus,
  Trash2,
  Edit2,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  ExternalLink,
  Copy,
  Check,
  Eye,
  EyeOff,
  Zap,
  Activity,
  Layers,
  Globe,
  Radio,
  FileText,
  Lock,
  ArrowRight,
  ChevronRight,
  Info,
} from 'lucide-react';
import {
  EmailDomain,
  EmailSMTPRelay,
  RelayConnectionTestResult,
  DeliverabilityAuditResult,
  listEmailRelays,
  createEmailRelay,
  updateEmailRelay,
  deleteEmailRelay,
  testEmailRelay,
  verifyDomainDeliverability,
} from '@/lib/api';

interface SmtpRelayStudioProps {
  domains: EmailDomain[];
  showToast?: (msg: string) => void;
}

interface ProviderPreset {
  id: string;
  name: string;
  host: string;
  port: number;
  encryption: 'starttls' | 'ssl' | 'none';
  authType: 'plain' | 'login' | 'cram-md5';
  usernameHint: string;
  description: string;
  badgeColor: string;
}

const PROVIDER_PRESETS: ProviderPreset[] = [
  {
    id: 'ses',
    name: 'Amazon SES',
    host: 'email-smtp.us-east-1.amazonaws.com',
    port: 587,
    encryption: 'starttls',
    authType: 'login',
    usernameHint: 'AKIAIOSFODNN7EXAMPLE (SES SMTP Username)',
    description: 'High-volume cloud deliverability via AWS SES relay',
    badgeColor: 'bg-amber-500/10 text-amber-500 border-amber-500/20',
  },
  {
    id: 'sendgrid',
    name: 'SendGrid',
    host: 'smtp.sendgrid.net',
    port: 587,
    encryption: 'starttls',
    authType: 'plain',
    usernameHint: 'apikey (literal string "apikey")',
    description: 'Twilio SendGrid transactional relay engine',
    badgeColor: 'bg-blue-500/10 text-blue-500 border-blue-500/20',
  },
  {
    id: 'mailgun',
    name: 'Mailgun',
    host: 'smtp.mailgun.org',
    port: 587,
    encryption: 'starttls',
    authType: 'plain',
    usernameHint: 'postmaster@yourdomain.com',
    description: 'Mailgun SMTP relay with granular webhook tracking',
    badgeColor: 'bg-rose-500/10 text-rose-500 border-rose-500/20',
  },
  {
    id: 'brevo',
    name: 'Brevo (Sendinblue)',
    host: 'smtp-relay.brevo.com',
    port: 587,
    encryption: 'starttls',
    authType: 'login',
    usernameHint: 'Account email or master SMTP login',
    description: 'Brevo transactional and marketing relay',
    badgeColor: 'bg-emerald-500/10 text-emerald-500 border-emerald-500/20',
  },
  {
    id: 'postmark',
    name: 'Postmark',
    host: 'smtp.postmarkapp.com',
    port: 587,
    encryption: 'starttls',
    authType: 'plain',
    usernameHint: 'Server API Token UUID',
    description: 'Ultra-fast inbox placement dedicated to transactional email',
    badgeColor: 'bg-yellow-500/10 text-yellow-500 border-yellow-500/20',
  },
  {
    id: 'custom',
    name: 'Custom SMTP Server',
    host: '',
    port: 587,
    encryption: 'starttls',
    authType: 'plain',
    usernameHint: 'Custom SMTP username or email',
    description: 'Any external Postfix, Exim, or corporate gateway',
    badgeColor: 'bg-purple-500/10 text-purple-500 border-purple-500/20',
  },
];

export function SmtpRelayStudio({ domains, showToast }: SmtpRelayStudioProps) {
  const [relays, setRelays] = useState<EmailSMTPRelay[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedDomainFilter, setSelectedDomainFilter] = useState<string>('all');
  const [copiedKey, setCopiedKey] = useState<string | null>(null);

  // Modal states
  const [showModal, setShowModal] = useState(false);
  const [editingRelay, setEditingRelay] = useState<EmailSMTPRelay | null>(null);
  const [saving, setSaving] = useState(false);
  const [showPassword, setShowPassword] = useState(false);

  // Form fields
  const [selectedPreset, setSelectedPreset] = useState<string>('ses');
  const [formDomainId, setFormDomainId] = useState<string>('');
  const [formName, setFormName] = useState<string>('');
  const [formHost, setFormHost] = useState<string>('');
  const [formPort, setFormPort] = useState<number>(587);
  const [formUsername, setFormUsername] = useState<string>('');
  const [formPassword, setFormPassword] = useState<string>('');
  const [formAuthType, setFormAuthType] = useState<string>('login');
  const [formEncryption, setFormEncryption] = useState<string>('starttls');
  const [formSenderDomain, setFormSenderDomain] = useState<string>('');
  const [formRateLimitSec, setFormRateLimitSec] = useState<number>(20);
  const [formRateLimitDay, setFormRateLimitDay] = useState<number>(50000);
  const [formTestOnSave, setFormTestOnSave] = useState<boolean>(true);

  // Probe testing modal
  const [probeLoading, setProbeLoading] = useState<string | null>(null);
  const [probeResult, setProbeResult] = useState<{ relayName: string; result: RelayConnectionTestResult } | null>(null);

  // Deliverability DNS Audit modal
  const [auditLoading, setAuditLoading] = useState<string | null>(null);
  const [auditResult, setAuditResult] = useState<{ relayName: string; result: DeliverabilityAuditResult } | null>(null);

  const toast = useCallback((msg: string) => {
    if (showToast) {
      showToast(msg);
    } else {
      alert(msg);
    }
  }, [showToast]);

  const copyToClipboard = (text: string, key: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(key);
    setTimeout(() => setCopiedKey(null), 2000);
    toast('Copied to clipboard');
  };

  const fetchRelays = useCallback(async () => {
    setLoading(true);
    try {
      const domainId = selectedDomainFilter !== 'all' ? selectedDomainFilter : undefined;
      const res = await listEmailRelays(domainId);
      if (res.data) {
        setRelays(res.data);
      }
    } catch (err: any) {
      console.error('Failed to fetch SMTP relays', err);
      toast('Failed to load SMTP relays');
    } finally {
      setLoading(false);
    }
  }, [selectedDomainFilter, toast]);

  useEffect(() => {
    fetchRelays();
  }, [fetchRelays]);

  // Handle Preset Selection
  const applyPreset = (presetId: string) => {
    setSelectedPreset(presetId);
    const p = PROVIDER_PRESETS.find((x) => x.id === presetId);
    if (!p) return;

    if (presetId !== 'custom') {
      setFormHost(p.host);
      setFormPort(p.port);
      setFormEncryption(p.encryption);
      setFormAuthType(p.authType);
      if (!editingRelay) {
        setFormName(`${p.name} Relay`);
      }
    }
  };

  const openCreateModal = () => {
    setEditingRelay(null);
    setSelectedPreset('ses');
    const p = PROVIDER_PRESETS[0];
    setFormDomainId(domains[0]?.id || '');
    setFormSenderDomain(domains[0]?.domain || '');
    setFormName('Amazon SES Relay');
    setFormHost(p.host);
    setFormPort(p.port);
    setFormUsername('');
    setFormPassword('');
    setFormAuthType(p.authType);
    setFormEncryption(p.encryption);
    setFormRateLimitSec(20);
    setFormRateLimitDay(50000);
    setFormTestOnSave(true);
    setShowPassword(false);
    setShowModal(true);
  };

  const openEditModal = (relay: EmailSMTPRelay) => {
    setEditingRelay(relay);
    const matched = PROVIDER_PRESETS.find((x) => x.id === relay.provider) || PROVIDER_PRESETS.find((x) => x.id === 'custom')!;
    setSelectedPreset(matched.id);
    setFormDomainId(relay.domain_id || '');
    setFormSenderDomain(relay.sender_domain || '');
    setFormName(relay.name);
    setFormHost(relay.host);
    setFormPort(relay.port);
    setFormUsername(relay.username);
    setFormPassword(relay.password_encrypted || '');
    setFormAuthType(relay.auth_type || 'plain');
    setFormEncryption(relay.encryption || 'starttls');
    setFormRateLimitSec(relay.rate_limit_per_second || 20);
    setFormRateLimitDay(relay.rate_limit_per_day || 50000);
    setFormTestOnSave(false);
    setShowPassword(false);
    setShowModal(true);
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formHost.trim()) {
      toast('Relay host is required');
      return;
    }
    if (!formSenderDomain.trim()) {
      toast('Sender domain is required');
      return;
    }

    setSaving(true);
    try {
      if (editingRelay) {
        const payload: any = {
          domain_id: formDomainId || undefined,
          name: formName,
          provider: selectedPreset,
          host: formHost.trim(),
          port: Number(formPort),
          username: formUsername.trim(),
          auth_type: formAuthType,
          encryption: formEncryption,
          sender_domain: formSenderDomain.trim().toLowerCase(),
          rate_limit_per_second: Number(formRateLimitSec),
          rate_limit_per_day: Number(formRateLimitDay),
        };
        if (formPassword && formPassword !== '••••••••') {
          payload.password = formPassword;
        }
        const res = await updateEmailRelay(editingRelay.id, payload);
        if (res.data) {
          toast('SMTP relay updated successfully');
          setShowModal(false);
          fetchRelays();
        }
      } else {
        const payload = {
          domain_id: formDomainId || undefined,
          name: formName,
          provider: selectedPreset,
          host: formHost.trim(),
          port: Number(formPort),
          username: formUsername.trim(),
          password: formPassword,
          auth_type: formAuthType,
          encryption: formEncryption,
          sender_domain: formSenderDomain.trim().toLowerCase(),
          rate_limit_per_second: Number(formRateLimitSec),
          rate_limit_per_day: Number(formRateLimitDay),
          test_on_save: formTestOnSave,
        };
        const res = await createEmailRelay(payload);
        if (res.data) {
          toast('SMTP relay created successfully');
          setShowModal(false);
          fetchRelays();
        }
      }
    } catch (err: any) {
      console.error('Failed to save relay', err);
      toast(err.message || 'Failed to save SMTP relay');
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (relay: EmailSMTPRelay) => {
    if (!confirm(`Are you sure you want to remove relay "${relay.name}"? Postfix will revert to direct MX routing for ${relay.sender_domain}.`)) {
      return;
    }
    try {
      await deleteEmailRelay(relay.id);
      toast(`Relay "${relay.name}" deleted`);
      fetchRelays();
    } catch (err: any) {
      toast(err.message || 'Failed to delete relay');
    }
  };

  const handleTestProbe = async (relay: EmailSMTPRelay) => {
    setProbeLoading(relay.id);
    try {
      const res = await testEmailRelay(relay.id);
      if (res.data) {
        setProbeResult({
          relayName: relay.name,
          result: res.data,
        });
        fetchRelays();
      }
    } catch (err: any) {
      toast(err.message || 'Probe connection test failed');
    } finally {
      setProbeLoading(null);
    }
  };

  const handleVerifyDns = async (relay: EmailSMTPRelay) => {
    setAuditLoading(relay.id);
    try {
      const res = await verifyDomainDeliverability(relay.id);
      if (res.data) {
        setAuditResult({
          relayName: relay.name,
          result: res.data,
        });
        fetchRelays();
      }
    } catch (err: any) {
      toast(err.message || 'DNS Deliverability verification failed');
    } finally {
      setAuditLoading(null);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header Banner */}
      <div className="p-6 rounded-2xl bg-gradient-to-r from-blue-900/40 via-indigo-900/30 to-purple-900/30 border border-indigo-500/20 backdrop-blur-md shadow-xs">
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div className="space-y-1.5">
            <div className="inline-flex items-center gap-2 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-indigo-500/20 text-indigo-300 border border-indigo-500/30">
              <Zap className="w-3.5 h-3.5 text-indigo-400" />
              <span>Enterprise Delivery Hub</span>
            </div>
            <h2 className="text-xl font-bold text-white tracking-tight">External SMTP Relays &amp; Deliverability</h2>
            <p className="text-xs text-slate-300 max-w-2xl">
              Route outgoing mail via enterprise SMTP relays (Amazon SES, SendGrid, Mailgun, Brevo, Postmark) with real-time TLS handshake verification, password encryption, and authoritative live DNS checks.
            </p>
          </div>

          <div className="flex items-center gap-3">
            <button
              onClick={fetchRelays}
              disabled={loading}
              className="p-2.5 rounded-xl bg-white/10 hover:bg-white/15 text-white border border-white/10 transition shadow-xs disabled:opacity-50"
              title="Refresh relays"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            </button>
            <button
              onClick={openCreateModal}
              className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-semibold text-xs transition shadow-sm"
            >
              <Plus className="w-4 h-4" />
              <span>Configure SMTP Relay</span>
            </button>
          </div>
        </div>

        {/* Filter Bar */}
        <div className="mt-6 pt-4 border-t border-white/10 flex flex-wrap items-center gap-3">
          <span className="text-xs font-semibold text-slate-300">Filter Domain:</span>
          <select
            value={selectedDomainFilter}
            onChange={(e) => setSelectedDomainFilter(e.target.value)}
            className="px-3 py-1.5 bg-slate-900/80 border border-slate-700 rounded-xl text-xs text-white focus:outline-none focus:border-indigo-400"
          >
            <option value="all">All Domains &amp; Global Relays</option>
            {domains.map((d) => (
              <option key={d.id} value={d.id}>
                {d.domain}
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* Relays Cards List */}
      {loading ? (
        <div className="p-12 text-center rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800">
          <RefreshCw className="w-8 h-8 text-indigo-500 animate-spin mx-auto mb-3" />
          <p className="text-sm font-medium text-slate-600 dark:text-slate-400">Loading configured SMTP relays...</p>
        </div>
      ) : relays.length === 0 ? (
        <div className="p-12 text-center rounded-2xl bg-white dark:bg-surface-900 border border-dashed border-slate-300 dark:border-surface-700 space-y-4">
          <div className="w-14 h-14 mx-auto rounded-2xl bg-indigo-500/10 flex items-center justify-center text-indigo-500">
            <Send className="w-7 h-7" />
          </div>
          <div className="space-y-1">
            <h3 className="text-base font-bold text-slate-900 dark:text-white">No External SMTP Relays Configured</h3>
            <p className="text-xs text-slate-500 max-w-md mx-auto">
              Your server is currently transmitting mail directly via local Postfix MX records. Configure an external relay like Amazon SES or SendGrid to bypass cloud port 25 restrictions and boost inbox placement.
            </p>
          </div>
          <button
            onClick={openCreateModal}
            className="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-semibold text-xs shadow-xs transition"
          >
            <Plus className="w-4 h-4" />
            <span>Add First SMTP Relay</span>
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {relays.map((relay) => {
            const preset = PROVIDER_PRESETS.find((p) => p.id === relay.provider) || PROVIDER_PRESETS[5];
            const isDomainSpecific = Boolean(relay.domain_id);
            const domainName = domains.find((d) => d.id === relay.domain_id)?.domain || relay.sender_domain || 'Global';

            return (
              <div
                key={relay.id}
                className="p-5 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 space-y-4 shadow-xs hover:border-indigo-500/40 transition"
              >
                {/* Relay Header */}
                <div className="flex items-start justify-between gap-3">
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <span className={`px-2 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-wider border ${preset.badgeColor}`}>
                        {preset.name}
                      </span>
                      {isDomainSpecific ? (
                        <span className="px-2 py-0.5 rounded-full text-[10px] font-medium bg-slate-100 dark:bg-surface-800 text-slate-600 dark:text-slate-300 border border-slate-200 dark:border-surface-700">
                          Domain: {domainName}
                        </span>
                      ) : (
                        <span className="px-2 py-0.5 rounded-full text-[10px] font-medium bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20">
                          Server-wide Default
                        </span>
                      )}
                    </div>
                    <h3 className="text-base font-bold text-slate-900 dark:text-white flex items-center gap-1.5">
                      <span>{relay.name}</span>
                    </h3>
                  </div>

                  <div className="flex items-center gap-1">
                    <button
                      onClick={() => openEditModal(relay)}
                      className="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-surface-800 transition"
                      title="Edit Relay"
                    >
                      <Edit2 className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => handleDelete(relay)}
                      className="p-1.5 rounded-lg text-slate-400 hover:text-rose-500 hover:bg-rose-50 dark:hover:bg-rose-950/30 transition"
                      title="Delete Relay"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                </div>

                {/* Connection Specs */}
                <div className="grid grid-cols-2 gap-2 text-xs font-mono">
                  <div className="p-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200/80 dark:border-surface-800">
                    <span className="text-[10px] font-sans font-semibold text-slate-400 uppercase tracking-wider block">Relay Endpoint</span>
                    <span className="text-slate-800 dark:text-slate-200 font-bold truncate block">
                      {relay.host}:{relay.port}
                    </span>
                    <span className="text-[10px] text-indigo-500 font-sans uppercase font-medium">
                      {relay.encryption.toUpperCase()} &bull; {relay.auth_type.toUpperCase()}
                    </span>
                  </div>

                  <div className="p-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200/80 dark:border-surface-800">
                    <span className="text-[10px] font-sans font-semibold text-slate-400 uppercase tracking-wider block">Sender Domain</span>
                    <span className="text-slate-800 dark:text-slate-200 font-bold truncate block">
                      @{relay.sender_domain || '*'}
                    </span>
                    <span className="text-[10px] text-slate-400 font-sans">
                      {relay.rate_limit_per_second ? `${relay.rate_limit_per_second} msg/sec` : 'Unmetered'}
                    </span>
                  </div>
                </div>

                {/* Verification & Test Status Badges */}
                <div className="flex flex-wrap items-center justify-between gap-2 pt-2 border-t border-slate-100 dark:border-surface-800 text-xs">
                  <div className="flex items-center gap-2">
                    {relay.is_verified ? (
                      <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-emerald-600 dark:text-emerald-400 bg-emerald-500/10 px-2 py-0.5 rounded-full border border-emerald-500/20">
                        <ShieldCheck className="w-3.5 h-3.5" />
                        <span>DNS Verified</span>
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-amber-600 dark:text-amber-400 bg-amber-500/10 px-2 py-0.5 rounded-full border border-amber-500/20">
                        <AlertTriangle className="w-3.5 h-3.5" />
                        <span>DNS Pending</span>
                      </span>
                    )}

                    {relay.last_test_status === 'success' ? (
                      <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-emerald-600 dark:text-emerald-400">
                        <CheckCircle2 className="w-3.5 h-3.5" />
                        <span>Probe OK</span>
                      </span>
                    ) : relay.last_test_status === 'failed' ? (
                      <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-rose-500" title={relay.last_test_error}>
                        <XCircle className="w-3.5 h-3.5" />
                        <span className="truncate max-w-[120px]">Probe Failed</span>
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1 text-[11px] text-slate-400">
                        <span>Untested</span>
                      </span>
                    )}
                  </div>

                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => handleTestProbe(relay)}
                      disabled={probeLoading === relay.id}
                      className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg bg-slate-100 dark:bg-surface-800 hover:bg-slate-200 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-300 font-semibold text-[11px] transition disabled:opacity-50"
                    >
                      <Activity className={`w-3.5 h-3.5 text-indigo-500 ${probeLoading === relay.id ? 'animate-spin' : ''}`} />
                      <span>{probeLoading === relay.id ? 'Probing...' : 'Test Probe'}</span>
                    </button>

                    <button
                      onClick={() => handleVerifyDns(relay)}
                      disabled={auditLoading === relay.id}
                      className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg bg-indigo-50 dark:bg-indigo-950/40 hover:bg-indigo-100 dark:hover:bg-indigo-900/40 text-indigo-600 dark:text-indigo-400 font-semibold text-[11px] transition border border-indigo-200 dark:border-indigo-800 disabled:opacity-50"
                    >
                      <Globe className={`w-3.5 h-3.5 ${auditLoading === relay.id ? 'animate-spin' : ''}`} />
                      <span>{auditLoading === relay.id ? 'Verifying...' : 'DNS Audit'}</span>
                    </button>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* ==================================================================== */}
      {/* ADD / EDIT RELAY MODAL                                               */}
      {/* ==================================================================== */}
      {showModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs">
          <div className="w-full max-w-2xl bg-white dark:bg-surface-900 rounded-2xl border border-slate-200 dark:border-surface-800 shadow-2xl overflow-hidden flex flex-col max-h-[90vh]">
            <div className="px-6 py-4 border-b border-slate-200 dark:border-surface-800 flex items-center justify-between">
              <div>
                <h3 className="text-base font-bold text-slate-900 dark:text-white">
                  {editingRelay ? 'Modify SMTP Relay Gateway' : 'Configure New SMTP Relay Gateway'}
                </h3>
                <p className="text-xs text-slate-500 mt-0.5">
                  Route outgoing Postfix traffic through certified high-reputation delivery networks.
                </p>
              </div>
              <button
                onClick={() => setShowModal(false)}
                className="p-1 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-white"
              >
                &times;
              </button>
            </div>

            <form onSubmit={handleSave} className="p-6 space-y-5 overflow-y-auto">
              {/* Preset Selector */}
              <div className="space-y-2">
                <label className="text-xs font-bold text-slate-700 dark:text-slate-300 uppercase tracking-wider block">
                  Select Relay Provider Preset
                </label>
                <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
                  {PROVIDER_PRESETS.map((p) => {
                    const isSelected = selectedPreset === p.id;
                    return (
                      <button
                        key={p.id}
                        type="button"
                        onClick={() => applyPreset(p.id)}
                        className={`p-3 rounded-xl border text-left transition flex flex-col justify-between ${
                          isSelected
                            ? 'bg-indigo-50/50 dark:bg-indigo-950/30 border-indigo-500 ring-2 ring-indigo-500/20'
                            : 'bg-slate-50 dark:bg-surface-950 border-slate-200 dark:border-surface-800 hover:border-slate-300 dark:hover:border-surface-700'
                        }`}
                      >
                        <div className="flex items-center justify-between w-full">
                          <span className="text-xs font-bold text-slate-900 dark:text-white">{p.name}</span>
                          {isSelected && <Check className="w-3.5 h-3.5 text-indigo-500" />}
                        </div>
                        <span className="text-[10px] text-slate-500 mt-1 line-clamp-1">{p.description}</span>
                      </button>
                    );
                  })}
                </div>
              </div>

              {/* General Config */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div className="space-y-1.5">
                  <label className="text-xs font-semibold text-slate-700 dark:text-slate-300">Relay Name</label>
                  <input
                    type="text"
                    required
                    value={formName}
                    onChange={(e) => setFormName(e.target.value)}
                    placeholder="e.g. Amazon SES Primary"
                    className="w-full px-3 py-2 bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                  />
                </div>

                <div className="space-y-1.5">
                  <label className="text-xs font-semibold text-slate-700 dark:text-slate-300">Target Domain Association</label>
                  <select
                    value={formDomainId}
                    onChange={(e) => {
                      setFormDomainId(e.target.value);
                      const matched = domains.find((d) => d.id === e.target.value);
                      if (matched) {
                        setFormSenderDomain(matched.domain);
                      }
                    }}
                    className="w-full px-3 py-2 bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                  >
                    <option value="">Server-Wide (Fallback Default)</option>
                    {domains.map((d) => (
                      <option key={d.id} value={d.id}>
                        {d.domain}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="space-y-1.5">
                  <label className="text-xs font-semibold text-slate-700 dark:text-slate-300">Sender Domain (Envelop From)</label>
                  <input
                    type="text"
                    required
                    value={formSenderDomain}
                    onChange={(e) => setFormSenderDomain(e.target.value)}
                    placeholder="e.g. yourcompany.com"
                    className="w-full px-3 py-2 bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                  />
                </div>

                <div className="space-y-1.5">
                  <label className="text-xs font-semibold text-slate-700 dark:text-slate-300">Relay Host / Gateway</label>
                  <input
                    type="text"
                    required
                    value={formHost}
                    onChange={(e) => setFormHost(e.target.value)}
                    placeholder="e.g. email-smtp.us-east-1.amazonaws.com"
                    className="w-full px-3 py-2 bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white font-mono focus:outline-none focus:border-indigo-500"
                  />
                </div>

                <div className="space-y-1.5">
                  <label className="text-xs font-semibold text-slate-700 dark:text-slate-300">SMTP Port</label>
                  <select
                    value={formPort}
                    onChange={(e) => setFormPort(Number(e.target.value))}
                    className="w-full px-3 py-2 bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                  >
                    <option value={587}>Port 587 (Standard Submission STARTTLS)</option>
                    <option value={465}>Port 465 (Direct SSL/TLS Wrap)</option>
                    <option value={2525}>Port 2525 (Alternative Cloud Port)</option>
                    <option value={25}>Port 25 (Direct Inbound)</option>
                  </select>
                </div>

                <div className="space-y-1.5">
                  <label className="text-xs font-semibold text-slate-700 dark:text-slate-300">TLS Encryption Mode</label>
                  <select
                    value={formEncryption}
                    onChange={(e) => setFormEncryption(e.target.value)}
                    className="w-full px-3 py-2 bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                  >
                    <option value="starttls">STARTTLS (Required)</option>
                    <option value="ssl">SSL / TLS Direct Wrapper</option>
                    <option value="none">None (Plaintext - Not Recommended)</option>
                  </select>
                </div>
              </div>

              {/* Authentication Credentials */}
              <div className="p-4 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 space-y-3">
                <h4 className="text-xs font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300 flex items-center gap-1.5">
                  <Lock className="w-3.5 h-3.5 text-indigo-500" />
                  <span>SMTP Authentication Credentials</span>
                </h4>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  <div className="space-y-1">
                    <label className="text-xs font-medium text-slate-600 dark:text-slate-400">Username / API Key</label>
                    <input
                      type="text"
                      value={formUsername}
                      onChange={(e) => setFormUsername(e.target.value)}
                      placeholder={PROVIDER_PRESETS.find((p) => p.id === selectedPreset)?.usernameHint || 'Username'}
                      className="w-full px-3 py-2 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white font-mono focus:outline-none focus:border-indigo-500"
                    />
                  </div>

                  <div className="space-y-1">
                    <div className="flex items-center justify-between">
                      <label className="text-xs font-medium text-slate-600 dark:text-slate-400">Password / Secret</label>
                      <button
                        type="button"
                        onClick={() => setShowPassword(!showPassword)}
                        className="text-[10px] text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 inline-flex items-center gap-1"
                      >
                        {showPassword ? <EyeOff className="w-3 h-3" /> : <Eye className="w-3 h-3" />}
                        <span>{showPassword ? 'Hide' : 'Reveal'}</span>
                      </button>
                    </div>
                    <input
                      type={showPassword ? 'text' : 'password'}
                      value={formPassword}
                      onChange={(e) => setFormPassword(e.target.value)}
                      placeholder={editingRelay ? '•••••••• (Leave blank to keep unchanged)' : 'Enter SMTP password or API token'}
                      className="w-full px-3 py-2 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white font-mono focus:outline-none focus:border-indigo-500"
                    />
                  </div>
                </div>

                <div className="flex items-center gap-4 pt-1">
                  <label className="text-xs font-medium text-slate-600 dark:text-slate-400">Auth Method:</label>
                  <label className="inline-flex items-center gap-1.5 text-xs text-slate-700 dark:text-slate-300 cursor-pointer">
                    <input
                      type="radio"
                      name="authType"
                      value="login"
                      checked={formAuthType === 'login'}
                      onChange={() => setFormAuthType('login')}
                      className="text-indigo-600 focus:ring-0"
                    />
                    <span>LOGIN</span>
                  </label>
                  <label className="inline-flex items-center gap-1.5 text-xs text-slate-700 dark:text-slate-300 cursor-pointer">
                    <input
                      type="radio"
                      name="authType"
                      value="plain"
                      checked={formAuthType === 'plain'}
                      onChange={() => setFormAuthType('plain')}
                      className="text-indigo-600 focus:ring-0"
                    />
                    <span>PLAIN</span>
                  </label>
                  <label className="inline-flex items-center gap-1.5 text-xs text-slate-700 dark:text-slate-300 cursor-pointer">
                    <input
                      type="radio"
                      name="authType"
                      value="cram-md5"
                      checked={formAuthType === 'cram-md5'}
                      onChange={() => setFormAuthType('cram-md5')}
                      className="text-indigo-600 focus:ring-0"
                    />
                    <span>CRAM-MD5</span>
                  </label>
                </div>
              </div>

              {/* Rate Limits & Probe on save */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div className="space-y-1.5">
                  <label className="text-xs font-semibold text-slate-700 dark:text-slate-300">Max Messages / Second</label>
                  <input
                    type="number"
                    min={1}
                    max={1000}
                    value={formRateLimitSec}
                    onChange={(e) => setFormRateLimitSec(Number(e.target.value))}
                    className="w-full px-3 py-2 bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                  />
                </div>

                <div className="space-y-1.5">
                  <label className="text-xs font-semibold text-slate-700 dark:text-slate-300">Daily Message Quota</label>
                  <input
                    type="number"
                    min={100}
                    max={10000000}
                    value={formRateLimitDay}
                    onChange={(e) => setFormRateLimitDay(Number(e.target.value))}
                    className="w-full px-3 py-2 bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                  />
                </div>
              </div>

              {!editingRelay && (
                <label className="flex items-center gap-2 cursor-pointer p-3 rounded-xl bg-indigo-50/50 dark:bg-indigo-950/20 border border-indigo-200/50 dark:border-indigo-800/40">
                  <input
                    type="checkbox"
                    checked={formTestOnSave}
                    onChange={(e) => setFormTestOnSave(e.target.checked)}
                    className="rounded text-indigo-600 focus:ring-0"
                  />
                  <span className="text-xs font-medium text-indigo-900 dark:text-indigo-300">
                    Execute real-time TLS handshake &amp; AUTH test probe immediately upon saving
                  </span>
                </label>
              )}

              {/* Modal Actions */}
              <div className="flex items-center justify-end gap-3 pt-3 border-t border-slate-200 dark:border-surface-800">
                <button
                  type="button"
                  onClick={() => setShowModal(false)}
                  className="px-4 py-2 rounded-xl border border-slate-200 dark:border-surface-800 text-xs font-semibold text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-surface-800 transition"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={saving}
                  className="inline-flex items-center gap-2 px-5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-semibold text-xs transition shadow-sm disabled:opacity-50"
                >
                  {saving && <RefreshCw className="w-3.5 h-3.5 animate-spin" />}
                  <span>{saving ? 'Saving Gateway...' : editingRelay ? 'Save Changes' : 'Establish Relay'}</span>
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* ==================================================================== */}
      {/* CONNECTION PROBE MODAL                                               */}
      {/* ==================================================================== */}
      {probeResult && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs">
          <div className="w-full max-w-xl bg-white dark:bg-surface-900 rounded-2xl border border-slate-200 dark:border-surface-800 shadow-2xl overflow-hidden flex flex-col">
            <div className="px-6 py-4 border-b border-slate-200 dark:border-surface-800 flex items-center justify-between">
              <div className="flex items-center gap-2">
                {probeResult.result.success ? (
                  <CheckCircle2 className="w-5 h-5 text-emerald-500" />
                ) : (
                  <XCircle className="w-5 h-5 text-rose-500" />
                )}
                <div>
                  <h3 className="text-base font-bold text-slate-900 dark:text-white">
                    {probeResult.result.success ? 'Probe Verified: ' : 'Probe Failed: '}
                    {probeResult.relayName}
                  </h3>
                  <p className="text-xs text-slate-500">Live TLS handshake and SMTP authentication trace</p>
                </div>
              </div>
              <button onClick={() => setProbeResult(null)} className="p-1 rounded-lg text-slate-400 hover:text-white">
                &times;
              </button>
            </div>

            <div className="p-6 space-y-4 text-xs font-mono">
              <div className="grid grid-cols-2 gap-2 font-sans">
                <div className="p-3 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800">
                  <span className="text-[10px] text-slate-400 block font-semibold uppercase">Roundtrip Latency</span>
                  <span className="text-base font-bold text-slate-900 dark:text-white font-mono">
                    {probeResult.result.latency_ms} ms
                  </span>
                </div>
                <div className="p-3 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800">
                  <span className="text-[10px] text-slate-400 block font-semibold uppercase">SMTP Authentication</span>
                  <span
                    className={`text-base font-bold font-mono ${
                      probeResult.result.auth_succeeded ? 'text-emerald-500' : 'text-rose-500'
                    }`}
                  >
                    {probeResult.result.auth_succeeded ? 'AUTHENTICATED' : 'AUTH FAILED'}
                  </span>
                </div>
              </div>

              {probeResult.result.tls_version && (
                <div className="p-3 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 space-y-1">
                  <div className="flex items-center justify-between">
                    <span className="text-slate-500">Cipher Protocol:</span>
                    <span className="text-indigo-400 font-bold">{probeResult.result.tls_version}</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-slate-500">Cipher Suite:</span>
                    <span className="text-slate-700 dark:text-slate-300 font-semibold">{probeResult.result.tls_cipher}</span>
                  </div>
                  {probeResult.result.cert_issuer && (
                    <div className="flex items-center justify-between pt-1 border-t border-slate-200 dark:border-surface-800">
                      <span className="text-slate-500">SSL Certificate Issuer:</span>
                      <span className="text-slate-700 dark:text-slate-300 truncate max-w-[280px]">
                        {probeResult.result.cert_issuer}
                      </span>
                    </div>
                  )}
                  {probeResult.result.cert_expiry && (
                    <div className="flex items-center justify-between">
                      <span className="text-slate-500">Certificate Expires:</span>
                      <span className="text-emerald-500">{new Date(probeResult.result.cert_expiry).toLocaleDateString()}</span>
                    </div>
                  )}
                </div>
              )}

              {/* Handshake Trace Log */}
              {probeResult.result.trace && probeResult.result.trace.length > 0 && (
                <div className="space-y-1.5">
                  <span className="text-[11px] font-sans font-bold text-slate-700 dark:text-slate-300 uppercase tracking-wider block">
                    SMTP Handshake Trace Log
                  </span>
                  <div className="p-3 rounded-xl bg-slate-900 text-slate-200 overflow-x-auto max-h-48 space-y-1 border border-slate-800 text-[11px]">
                    {probeResult.result.trace.map((step, idx) => (
                      <div key={idx} className="flex items-start gap-2">
                        <span className="text-slate-600 select-none">{idx + 1}.</span>
                        <span className={step.includes('Failed') ? 'text-rose-400' : 'text-emerald-400'}>{step}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {probeResult.result.error && (
                <div className="p-3 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-500 text-xs font-sans">
                  <strong>Error: </strong> {probeResult.result.error}
                </div>
              )}
            </div>

            <div className="px-6 py-3 border-t border-slate-200 dark:border-surface-800 flex justify-end">
              <button
                onClick={() => setProbeResult(null)}
                className="px-4 py-2 rounded-xl bg-slate-100 dark:bg-surface-800 text-xs font-semibold text-slate-800 dark:text-slate-200 hover:bg-slate-200 dark:hover:bg-surface-700 transition"
              >
                Close Trace
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ==================================================================== */}
      {/* DELIVERABILITY DNS AUDIT MODAL                                       */}
      {/* ==================================================================== */}
      {auditResult && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs">
          <div className="w-full max-w-3xl bg-white dark:bg-surface-900 rounded-2xl border border-slate-200 dark:border-surface-800 shadow-2xl overflow-hidden flex flex-col max-h-[90vh]">
            <div className="px-6 py-4 border-b border-slate-200 dark:border-surface-800 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Globe className="w-5 h-5 text-indigo-500" />
                <div>
                  <h3 className="text-base font-bold text-slate-900 dark:text-white">
                    Deliverability DNS Audit: {auditResult.result.domain}
                  </h3>
                  <p className="text-xs text-slate-500">
                    Live authoritative DNS query verification for {auditResult.result.provider.toUpperCase()} relay
                  </p>
                </div>
              </div>
              <button onClick={() => setAuditResult(null)} className="p-1 rounded-lg text-slate-400 hover:text-white">
                &times;
              </button>
            </div>

            <div className="p-6 space-y-5 overflow-y-auto">
              {/* Overall Verification Status */}
              <div
                className={`p-4 rounded-xl border flex items-center justify-between ${
                  auditResult.result.is_fully_verified
                    ? 'bg-emerald-500/10 border-emerald-500/30 text-emerald-600 dark:text-emerald-400'
                    : 'bg-amber-500/10 border-amber-500/30 text-amber-600 dark:text-amber-400'
                }`}
              >
                <div className="flex items-center gap-3">
                  {auditResult.result.is_fully_verified ? (
                    <ShieldCheck className="w-6 h-6 text-emerald-500" />
                  ) : (
                    <ShieldAlert className="w-6 h-6 text-amber-500" />
                  )}
                  <div>
                    <h4 className="text-sm font-bold">
                      {auditResult.result.is_fully_verified
                        ? 'Domain Authenticated for Relaying'
                        : 'DNS Records Incomplete or Not Propagated'}
                    </h4>
                    <p className="text-xs opacity-90">
                      {auditResult.result.is_fully_verified
                        ? 'SPF, DKIM, DMARC, and MX records are correctly configured.'
                        : 'One or more required DNS records are missing or need updating in your DNS zone.'}
                    </p>
                  </div>
                </div>
              </div>

              {/* Records Breakdown */}
              <div className="space-y-3">
                <h4 className="text-xs font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300">
                  DNS Record Validations
                </h4>
                {auditResult.result.records.map((rec, idx) => {
                  const isValid = rec.status === 'valid';
                  const isWarning = rec.status === 'warning';

                  return (
                    <div
                      key={idx}
                      className="p-4 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 space-y-2"
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <span className="font-mono font-bold text-xs px-2 py-0.5 rounded-md bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 text-slate-800 dark:text-slate-200">
                            {rec.record_type}
                          </span>
                          <span className="text-xs font-semibold text-slate-700 dark:text-slate-300">{rec.domain}</span>
                        </div>
                        <span
                          className={`text-xs font-semibold px-2 py-0.5 rounded-full ${
                            isValid
                              ? 'bg-emerald-500/10 text-emerald-500'
                              : isWarning
                              ? 'bg-amber-500/10 text-amber-500'
                              : 'bg-rose-500/10 text-rose-500'
                          }`}
                        >
                          {rec.status.toUpperCase()}
                        </span>
                      </div>

                      {/* Instructions */}
                      <p className="text-xs text-slate-500">{rec.instructions}</p>

                      {/* Expected Value & Copy Button */}
                      {rec.expected_value && (
                        <div className="p-2.5 rounded-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 flex items-center justify-between text-xs font-mono">
                          <div className="truncate mr-2">
                            <span className="text-slate-400 text-[10px] uppercase font-sans font-semibold block">
                              Recommended DNS Record Content
                            </span>
                            <span className="text-indigo-600 dark:text-indigo-400 font-semibold">{rec.expected_value}</span>
                          </div>
                          <button
                            onClick={() => copyToClipboard(rec.expected_value, `rec_${idx}`)}
                            className="p-1.5 rounded-md hover:bg-slate-100 dark:hover:bg-surface-800 text-slate-400 hover:text-slate-600 dark:hover:text-white transition shrink-0"
                            title="Copy Value"
                          >
                            {copiedKey === `rec_${idx}` ? (
                              <Check className="w-4 h-4 text-emerald-500" />
                            ) : (
                              <Copy className="w-4 h-4" />
                            )}
                          </button>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>

              {/* Recommendations */}
              {auditResult.result.recommendations && auditResult.result.recommendations.length > 0 && (
                <div className="p-4 rounded-xl bg-indigo-50/50 dark:bg-indigo-950/20 border border-indigo-200 dark:border-indigo-800/40 space-y-2">
                  <h4 className="text-xs font-bold uppercase tracking-wider text-indigo-700 dark:text-indigo-300 flex items-center gap-1.5">
                    <Info className="w-4 h-4" />
                    <span>Actionable Recommendations</span>
                  </h4>
                  <ul className="text-xs text-indigo-900 dark:text-indigo-200 space-y-1 list-disc list-inside">
                    {auditResult.result.recommendations.map((rec, i) => (
                      <li key={i}>{rec}</li>
                    ))}
                  </ul>
                </div>
              )}
            </div>

            <div className="px-6 py-3 border-t border-slate-200 dark:border-surface-800 flex justify-end">
              <button
                onClick={() => setAuditResult(null)}
                className="px-4 py-2 rounded-xl bg-slate-100 dark:bg-surface-800 text-xs font-semibold text-slate-800 dark:text-slate-200 hover:bg-slate-200 dark:hover:bg-surface-700 transition"
              >
                Close Audit
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
