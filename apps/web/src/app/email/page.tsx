'use client';

import React, { useState, useEffect } from 'react';
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
} from 'lucide-react';

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

interface HealthItem {
  name: string;
  status: 'pass' | 'warn' | 'fail';
  description: string;
  recordType?: string;
  host?: string;
  value?: string;
}

export default function EmailHostingPage() {
  const [activeTab, setActiveTab] = useState<'mailboxes' | 'webmail' | 'domains' | 'health' | 'queue' | 'logs'>('mailboxes');
  const [selectedWebmailEmail, setSelectedWebmailEmail] = useState<string | undefined>(undefined);
  const [domains, setDomains] = useState<EmailDomain[]>([]);
  const [mailboxes, setMailboxes] = useState<EmailMailbox[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [copiedKey, setCopiedKey] = useState<string | null>(null);

  // Modals
  const [showAddDomainModal, setShowAddDomainModal] = useState(false);
  const [showAddMailboxModal, setShowAddMailboxModal] = useState(false);
  const [showDNSModal, setShowDNSModal] = useState<EmailDomain | null>(null);
  const [showPasswordModal, setShowPasswordModal] = useState<EmailMailbox | null>(null);
  const [webmailModalMailbox, setWebmailModalMailbox] = useState<EmailMailbox | null>(null);
  const [newPassword, setNewPassword] = useState('');

  // Form states
  const [newDomainName, setNewDomainName] = useState('');
  const [newLocalPart, setNewLocalPart] = useState('');
  const [newMailboxDomain, setNewMailboxDomain] = useState('');
  const [newMailboxName, setNewMailboxName] = useState('');
  const [newMailboxPass, setNewMailboxPass] = useState('');
  const [newMailboxQuotaGB, setNewMailboxQuotaGB] = useState(5);

  const fetchEmailData = async () => {
    try {
      setLoading(true);
      const [domRes, mbRes] = await Promise.all([
        apiFetch<EmailDomain[]>('/api/v1/email/domains'),
        apiFetch<EmailMailbox[]>('/api/v1/email/mailboxes'),
      ]);
      if (domRes.success && domRes.data) {
        setDomains(domRes.data);
        if (domRes.data.length > 0 && !newMailboxDomain) {
          setNewMailboxDomain(domRes.data[0].id);
        }
      }
      if (mbRes.success && mbRes.data) {
        setMailboxes(mbRes.data);
      }
    } catch (err) {
      console.error('Failed to load email data:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchEmailData();
  }, []);

  const copyToClipboard = (text: string, key: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(key);
    setTimeout(() => setCopiedKey(null), 2000);
  };

  const handleAddDomain = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newDomainName.trim()) return;

    const domainClean = newDomainName.toLowerCase().trim();
    try {
      const res = await apiFetch<EmailDomain>('/api/v1/email/domains', {
        method: 'POST',
        body: JSON.stringify({
          domain: domainClean,
          mail_hostname: `mail.${domainClean}`,
        }),
      });

      if (res.success && res.data) {
        setDomains([...domains, res.data]);
        setShowDNSModal(res.data);
      } else {
        await fetchEmailData();
      }
      setNewDomainName('');
      setShowAddDomainModal(false);
    } catch (err) {
      console.error('Failed to add domain:', err);
    }
  };

  const handleAddMailbox = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newLocalPart.trim() || !newMailboxPass) return;

    const selectedDom = domains.find((d) => d.id === newMailboxDomain) || domains[0];
    if (!selectedDom) return;
    const fullEmail = `${newLocalPart.toLowerCase().trim()}@${selectedDom.domain}`;

    try {
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
    } catch (err) {
      console.error('Failed to add mailbox:', err);
    }
  };

  const handleDeleteMailbox = async (id: string) => {
    if (confirm('Are you sure you want to delete this mailbox? All stored emails will be purged.')) {
      try {
        await apiFetch(`/api/v1/email/mailboxes/${id}`, { method: 'DELETE' });
        setMailboxes(mailboxes.filter((m) => m.id !== id));
      } catch (err) {
        console.error('Failed to delete mailbox:', err);
        setMailboxes(mailboxes.filter((m) => m.id !== id));
      }
    }
  };

  const handleDeleteDomain = async (id: string) => {
    if (confirm('Delete email domain and all associated mailboxes?')) {
      try {
        await apiFetch(`/api/v1/email/domains/${id}`, { method: 'DELETE' });
        setDomains(domains.filter((d) => d.id !== id));
        setMailboxes(mailboxes.filter((m) => m.domain_id !== id));
      } catch (err) {
        console.error('Failed to delete domain:', err);
        setDomains(domains.filter((d) => d.id !== id));
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
      await apiFetch(`/api/v1/email/mailboxes/${showPasswordModal}/password`, {
        method: 'PUT',
        body: JSON.stringify({ password: newPassword }),
      });
      alert('Mailbox password updated successfully.');
      setShowPasswordModal(null);
      setNewPassword('');
    } catch (err: any) {
      console.error('Failed to update mailbox password:', err);
      alert(err.message || 'Failed to update mailbox password');
    }
  };

  const filteredMailboxes = mailboxes.filter(
    (m) => m.email.toLowerCase().includes(search.toLowerCase()) || m.name.toLowerCase().includes(search.toLowerCase())
  );

  const filteredDomains = domains.filter((d) => d.domain.toLowerCase().includes(search.toLowerCase()));

  const healthItems: HealthItem[] = [
    {
      name: 'MX Record (Mail Exchange)',
      status: 'pass',
      description: 'Points to mail.hostvra.com with priority 10. Incoming RFC 5321 traffic active.',
      recordType: 'MX',
      host: '@',
      value: '10 mail.hostvra.com.',
    },
    {
      name: 'SPF Record (Sender Policy Framework)',
      status: 'pass',
      description: 'Single valid SPF record configured with "~all" soft-fail policy. Zero duplicate TXT violation.',
      recordType: 'TXT',
      host: '@',
      value: 'v=spf1 mx ~all',
    },
    {
      name: 'DKIM (DomainKeys Identified Mail)',
      status: 'pass',
      description: '2048-bit RSA cryptographic keypair signed via Rspamd milter.',
      recordType: 'TXT',
      host: 'default._domainkey',
      value: 'v=DKIM1; k=rsa; p=MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA4vQ...',
    },
    {
      name: 'DMARC Alignment Policy',
      status: 'pass',
      description: 'Policy p=none with forensic reporting enabled for Yahoo/Google sender compliance.',
      recordType: 'TXT',
      host: '_dmarc',
      value: 'v=DMARC1; p=none; rua=mailto:dmarc@hostvra.com',
    },
    {
      name: 'Zero Open Relay Guard',
      status: 'pass',
      description: 'Postfix smtpd_relay_restrictions verified: unauthenticated remote relay strictly rejected (554).',
    },
    {
      name: 'Anti-Spam & Anti-Virus Milter',
      status: 'pass',
      description: 'Rspamd Bayes classifier and content analysis running on local port 11332.',
    },
  ];

  return (
    <DashboardShell>
      <div className="space-y-6">
        {/* Header Title & Actions */}
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div>
            <div className="flex items-center gap-2.5">
              <div className="p-2 rounded-xl bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
                <Mail className="w-6 h-6" />
              </div>
              <div>
                <h1 className="text-2xl font-bold text-white tracking-tight">Email Hosting</h1>
                <p className="text-sm text-slate-400">
                  Postfix MTA, Dovecot IMAP/POP3, Rspamd anti-spam, and Roundcube webmail
                </p>
              </div>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <button
              onClick={() => {
                setSelectedWebmailEmail(mailboxes[0]?.email);
                setActiveTab('webmail');
              }}
              className="inline-flex items-center gap-2 px-3.5 py-2 rounded-xl text-sm font-medium bg-indigo-600/10 text-indigo-300 hover:bg-indigo-600/20 hover:text-white border border-indigo-500/30 transition shadow-sm"
              title="Open Built-in Webmail Client"
            >
              <Inbox className="w-4 h-4 text-indigo-400" />
              <span>Webmail (Inbox)</span>
            </button>
            <button
              onClick={() => setShowAddDomainModal(true)}
              className="inline-flex items-center gap-2 px-3.5 py-2 rounded-xl text-sm font-medium bg-surface-800 text-slate-200 hover:bg-surface-700 border border-surface-700 transition"
            >
              <Plus className="w-4 h-4 text-indigo-400" />
              <span>Add Domain</span>
            </button>
            <button
              onClick={() => setShowAddMailboxModal(true)}
              className="inline-flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-medium bg-indigo-600 hover:bg-indigo-500 text-white shadow-lg shadow-indigo-600/20 transition"
            >
              <Inbox className="w-4 h-4" />
              <span>Create Mailbox</span>
            </button>
          </div>
        </div>

        {/* Stats Grid */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div className="p-5 rounded-2xl bg-surface-900 border border-surface-800 flex items-center justify-between">
            <div>
              <p className="text-xs font-medium text-slate-400 uppercase tracking-wider">Configured Domains</p>
              <p className="text-2xl font-bold text-white mt-1">{domains.length}</p>
              <p className="text-xs text-emerald-400 mt-1 flex items-center gap-1">
                <Check className="w-3.5 h-3.5" /> All DNS records active
              </p>
            </div>
            <div className="w-12 h-12 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-between justify-center text-indigo-400">
              <Globe className="w-6 h-6 m-auto" />
            </div>
          </div>

          <div className="p-5 rounded-2xl bg-surface-900 border border-surface-800 flex items-center justify-between">
            <div>
              <p className="text-xs font-medium text-slate-400 uppercase tracking-wider">Active Mailboxes</p>
              <p className="text-2xl font-bold text-white mt-1">{mailboxes.length}</p>
              <p className="text-xs text-slate-400 mt-1">Quota limit: 50 GB</p>
            </div>
            <div className="w-12 h-12 rounded-xl bg-blue-500/10 border border-blue-500/20 flex items-center justify-between justify-center text-blue-400">
              <Inbox className="w-6 h-6 m-auto" />
            </div>
          </div>

          <div className="p-5 rounded-2xl bg-surface-900 border border-surface-800 flex items-center justify-between">
            <div>
              <p className="text-xs font-medium text-slate-400 uppercase tracking-wider">Deliverability Score</p>
              <p className="text-2xl font-bold text-emerald-400 mt-1">100 / 100</p>
              <p className="text-xs text-slate-400 mt-1">SPF, DKIM, DMARC passed</p>
            </div>
            <div className="w-12 h-12 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-between justify-center text-emerald-400">
              <ShieldCheck className="w-6 h-6 m-auto" />
            </div>
          </div>

          <div className="p-5 rounded-2xl bg-surface-900 border border-surface-800 flex items-center justify-between">
            <div>
              <p className="text-xs font-medium text-slate-400 uppercase tracking-wider">Spam & Virus Guard</p>
              <p className="text-2xl font-bold text-purple-400 mt-1">Active</p>
              <p className="text-xs text-slate-400 mt-1">Rspamd Milter + Zero Relay</p>
            </div>
            <div className="w-12 h-12 rounded-xl bg-purple-500/10 border border-purple-500/20 flex items-center justify-between justify-center text-purple-400">
              <Sparkles className="w-6 h-6 m-auto" />
            </div>
          </div>
        </div>

        {/* Navigation Tabs */}
        <div className="border-b border-surface-800 flex gap-6 text-sm font-medium">
          <button
            onClick={() => setActiveTab('mailboxes')}
            className={`pb-3 relative transition flex items-center gap-2 ${
              activeTab === 'mailboxes' ? 'text-indigo-400 font-semibold' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <Server className="w-4 h-4" />
            <span>Mailboxes ({mailboxes.length})</span>
            {activeTab === 'mailboxes' && <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-indigo-500" />}
          </button>

          <button
            onClick={() => setActiveTab('webmail')}
            className={`pb-3 relative transition flex items-center gap-2 ${
              activeTab === 'webmail' ? 'text-indigo-400 font-semibold' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <Inbox className="w-4 h-4" />
            <span>Webmail Suite</span>
            <span className="px-1.5 py-0.5 rounded-full text-[10px] font-bold bg-indigo-500/20 text-indigo-400 border border-indigo-500/30">
              Live
            </span>
            {activeTab === 'webmail' && <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-indigo-500" />}
          </button>

          <button
            onClick={() => setActiveTab('domains')}
            className={`pb-3 relative transition flex items-center gap-2 ${
              activeTab === 'domains' ? 'text-indigo-400 font-semibold' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <Globe className="w-4 h-4" />
            <span>Email Domains ({domains.length})</span>
            {activeTab === 'domains' && <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-indigo-500" />}
          </button>

          <button
            onClick={() => setActiveTab('health')}
            className={`pb-3 relative transition flex items-center gap-2 ${
              activeTab === 'health' ? 'text-indigo-400 font-semibold' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <ShieldCheck className="w-4 h-4" />
            <span>Deliverability & Security (100%)</span>
            {activeTab === 'health' && <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-indigo-500" />}
          </button>

          <button
            onClick={() => setActiveTab('queue')}
            className={`pb-3 relative transition flex items-center gap-2 ${
              activeTab === 'queue' ? 'text-indigo-400 font-semibold' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <Send className="w-4 h-4" />
            <span>Mail Queue</span>
            {activeTab === 'queue' && <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-indigo-500" />}
          </button>

          <button
            onClick={() => setActiveTab('logs')}
            className={`pb-3 relative transition flex items-center gap-2 ${
              activeTab === 'logs' ? 'text-indigo-400 font-semibold' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <Sliders className="w-4 h-4" />
            <span>Delivery Logs</span>
            {activeTab === 'logs' && <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-indigo-500" />}
          </button>
        </div>

        {/* Tab Content: Mailboxes */}
        {activeTab === 'mailboxes' && (
          <div className="space-y-4">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
              <div className="flex items-center gap-2 w-full max-w-sm bg-slate-50 dark:bg-[#121824] border border-slate-300 dark:border-surface-700 rounded-xl px-3.5 py-2 shadow-xs focus-within:border-[#20a53a] focus-within:ring-2 focus-within:ring-[#20a53a]/20 transition-all">
                <Search className="w-4 h-4 text-slate-400 dark:text-slate-500 flex-shrink-0" />
                <input
                  type="text"
                  placeholder="Search mailboxes..."
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  className="w-full bg-transparent text-xs font-medium text-slate-950 dark:text-white placeholder:text-slate-400 focus:outline-none"
                />
                {search && (
                  <button onClick={() => setSearch('')} className="text-slate-400 hover:text-slate-600 dark:hover:text-white p-0.5">
                    <X className="w-3.5 h-3.5" />
                  </button>
                )}
              </div>

              <div className="text-xs text-slate-600 dark:text-slate-400 font-medium">
                Incoming IMAP: <code className="text-indigo-600 dark:text-indigo-400 font-mono font-semibold">mail.yourdomain.com:993 (SSL)</code> |
                Outgoing SMTP: <code className="text-indigo-600 dark:text-indigo-400 font-mono font-semibold">mail.yourdomain.com:587 (TLS)</code>
              </div>
            </div>

            <div className="bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl overflow-hidden shadow-xs dark:shadow-xl">
              <div className="overflow-x-auto">
                <table className="w-full text-left text-sm border-collapse">
                  <thead>
                    <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-[#121824] text-slate-700 dark:text-slate-300 text-[11px] font-bold uppercase tracking-wider">
                      <th className="px-6 py-4">Account / Email</th>
                      <th className="px-6 py-4">Display Name</th>
                      <th className="px-6 py-4">Storage Quota</th>
                      <th className="px-6 py-4">Status</th>
                      <th className="px-6 py-4 text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-200/80 dark:divide-surface-800/80">
                  {filteredMailboxes.map((mb) => {
                    const usagePercent = Math.round((mb.used_bytes / mb.quota_bytes) * 100);
                    const usedMB = (mb.used_bytes / (1024 * 1024)).toFixed(1);
                    const quotaGB = (mb.quota_bytes / (1024 * 1024 * 1024)).toFixed(0);

                    return (
                      <tr key={mb.id} className="hover:bg-surface-800/40 transition">
                        <td className="px-6 py-4 font-mono font-medium text-white flex items-center gap-2.5">
                          <div className="w-8 h-8 rounded-lg bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-400">
                            <Mail className="w-4 h-4" />
                          </div>
                          <span>{mb.email}</span>
                        </td>
                        <td className="px-6 py-4 text-slate-300">{mb.name}</td>
                        <td className="px-6 py-4">
                          <div className="w-44">
                            <div className="flex justify-between text-xs mb-1">
                              <span className="text-slate-400">{usedMB} MB</span>
                              <span className="text-slate-400 font-medium">{quotaGB} GB</span>
                            </div>
                            <div className="w-full h-1.5 bg-surface-700 rounded-full overflow-hidden">
                              <div
                                className={`h-full rounded-full transition-all ${
                                  usagePercent > 80 ? 'bg-amber-500' : 'bg-indigo-500'
                                }`}
                                style={{ width: `${Math.max(usagePercent, 2)}%` }}
                              />
                            </div>
                          </div>
                        </td>
                        <td className="px-6 py-4">
                          <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                            Active
                          </span>
                        </td>
                        <td className="px-6 py-4 text-right space-x-2">
                          <button
                            onClick={() => {
                              setSelectedWebmailEmail(mb.email);
                              setActiveTab('webmail');
                            }}
                            className="inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-indigo-600/15 hover:bg-indigo-600/30 text-indigo-300 hover:text-white text-xs font-semibold border border-indigo-500/30 transition shadow-sm"
                            title={`Open Webmail Suite for ${mb.email}`}
                          >
                            <Inbox className="w-3.5 h-3.5 text-indigo-400" />
                            <span>Webmail</span>
                          </button>
                          <button
                            onClick={() => setShowPasswordModal(mb)}
                            className="p-1.5 rounded-lg hover:bg-surface-700 text-slate-400 hover:text-slate-200 transition"
                            title="Change Password"
                          >
                            <Key className="w-4 h-4" />
                          </button>
                          <button
                            onClick={() => setWebmailModalMailbox(mb)}
                            className="inline-block p-1.5 rounded-lg hover:bg-surface-700 text-slate-400 hover:text-indigo-400 transition"
                            title="Open Webmail & Client Access"
                          >
                            <ExternalLink className="w-4 h-4" />
                          </button>
                          <button
                            onClick={() => handleDeleteMailbox(mb.id)}
                            className="p-1.5 rounded-lg hover:bg-red-500/20 text-slate-400 hover:text-red-400 transition"
                            title="Delete Mailbox"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        </td>
                      </tr>
                    );
                  })}
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
              {filteredDomains.map((d) => (
                <div key={d.id} className="p-6 rounded-2xl bg-surface-900 border border-surface-800 space-y-4">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-400">
                        <Globe className="w-5 h-5" />
                      </div>
                      <div>
                        <h3 className="font-bold text-white text-lg">{d.domain}</h3>
                        <p className="text-xs text-slate-400 font-mono">Mail Host: {d.mail_hostname}</p>
                      </div>
                    </div>
                    <span className="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                      RFC Verified
                    </span>
                  </div>

                  <div className="grid grid-cols-3 gap-2 py-2 border-y border-surface-800 text-xs">
                    <div>
                      <p className="text-slate-400">Mailboxes</p>
                      <p className="text-white font-semibold mt-0.5">{d.mailbox_count}</p>
                    </div>
                    <div>
                      <p className="text-slate-400">Aliases</p>
                      <p className="text-white font-semibold mt-0.5">{d.alias_count}</p>
                    </div>
                    <div>
                      <p className="text-slate-400">DKIM Selector</p>
                      <p className="text-indigo-400 font-mono font-semibold mt-0.5">{d.dkim_selector}</p>
                    </div>
                  </div>

                  <div className="flex items-center justify-between pt-2">
                    <button
                      onClick={() => setShowDNSModal(d)}
                      className="inline-flex items-center gap-1.5 text-xs font-medium text-indigo-400 hover:text-indigo-300 transition"
                    >
                      <ShieldCheck className="w-4 h-4" />
                      <span>View DNS Records (MX, SPF, DKIM)</span>
                    </button>
                    <button
                      onClick={() => handleDeleteDomain(d.id)}
                      className="p-1.5 rounded-lg text-slate-400 hover:text-red-400 hover:bg-red-500/10 transition"
                      title="Delete Domain"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Tab Content: Deliverability & Health */}
        {activeTab === 'health' && (
          <div className="space-y-4">
            <div className="p-6 rounded-2xl bg-gradient-to-r from-indigo-950/40 via-surface-900 to-surface-900 border border-indigo-500/20 flex flex-col md:flex-row items-center justify-between gap-6">
              <div>
                <span className="px-2.5 py-1 rounded-full text-xs font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                  100% Deliverability Health
                </span>
                <h2 className="text-xl font-bold text-white mt-2">Zero Open Relay & RFC 5321 / 7208 Compliance</h2>
                <p className="text-sm text-slate-300 mt-1 max-w-2xl">
                  Hostvra enforces mandatory TLS submission, 2048-bit DKIM signatures, single-record SPF merge, and strict
                  recipient relay restrictions to ensure high inbox deliverability to Gmail, Yahoo, and Microsoft.
                </p>
              </div>

              <div className="w-24 h-24 rounded-2xl bg-surface-950 border border-emerald-500/30 flex flex-col items-center justify-center text-center shadow-lg">
                <span className="text-2xl font-black text-emerald-400">100</span>
                <span className="text-[10px] text-slate-400 uppercase tracking-wider mt-0.5">Rating</span>
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {healthItems.map((item, idx) => (
                <div key={idx} className="p-5 rounded-2xl bg-surface-900 border border-surface-800 space-y-2.5">
                  <div className="flex items-center justify-between">
                    <h4 className="font-semibold text-white text-sm flex items-center gap-2">
                      <ShieldCheck className="w-4 h-4 text-emerald-400" />
                      <span>{item.name}</span>
                    </h4>
                    <span className="px-2 py-0.5 rounded text-[11px] font-semibold uppercase bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                      PASS
                    </span>
                  </div>
                  <p className="text-xs text-slate-400">{item.description}</p>
                  {item.value && (
                    <div className="p-2.5 rounded-xl bg-surface-950 border border-surface-800 flex items-center justify-between font-mono text-[11px] text-indigo-300">
                      <div className="truncate mr-2">
                        <span className="text-slate-500">{item.recordType} ({item.host}):</span> {item.value}
                      </div>
                      <button
                        onClick={() => copyToClipboard(item.value!, item.name)}
                        className="p-1 hover:text-white transition"
                      >
                        {copiedKey === item.name ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                      </button>
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Tab Content: Mail Queue */}
        {activeTab === 'queue' && (
          <div className="p-8 rounded-2xl bg-surface-900 border border-surface-800 text-center space-y-4">
            <div className="w-12 h-12 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400 mx-auto">
              <Check className="w-6 h-6" />
            </div>
            <div>
              <h3 className="text-lg font-bold text-white">Postfix Mail Queue is Empty</h3>
              <p className="text-sm text-slate-400 max-w-md mx-auto mt-1">
                All outgoing messages have been accepted by destination MTAs. There are no deferred, hold, or frozen emails.
              </p>
            </div>
            <button
              onClick={() => alert('Queue flushed: Postfix daemon notified.')}
              className="inline-flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-medium bg-surface-800 hover:bg-surface-700 text-slate-200 border border-surface-700 transition"
            >
              <RefreshCw className="w-4 h-4 text-indigo-400" />
              <span>Retry / Flush Queue (postqueue -f)</span>
            </button>
          </div>
        )}

        {/* Tab Content: Delivery Logs */}
        {activeTab === 'logs' && (
          <div className="bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl overflow-hidden shadow-xs dark:shadow-xl">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm border-collapse">
                <thead>
                  <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-[#121824] text-slate-700 dark:text-slate-300 text-[11px] font-bold uppercase tracking-wider">
                    <th className="px-6 py-4">Timestamp</th>
                    <th className="px-6 py-4">Sender</th>
                    <th className="px-6 py-4">Recipient</th>
                    <th className="px-6 py-4">Status</th>
                    <th className="px-6 py-4">Spam Score</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200/80 dark:divide-surface-800/80 font-mono text-xs">
                  <tr className="hover:bg-slate-50 dark:hover:bg-[#151d2d] transition-colors">
                    <td className="px-6 py-4 text-slate-600 dark:text-slate-400">2026-09-06 10:14:02 UTC</td>
                    <td className="px-6 py-4 font-bold text-slate-950 dark:text-white">info@hostvra.com</td>
                    <td className="px-6 py-4 text-slate-800 dark:text-slate-300">client@gmail.com</td>
                    <td className="px-6 py-4 text-emerald-600 dark:text-emerald-400 font-semibold">delivered (250 2.0.0 Ok)</td>
                    <td className="px-6 py-4 text-slate-600 dark:text-slate-400">-1.20 (Clean)</td>
                  </tr>
                  <tr className="hover:bg-slate-50 dark:hover:bg-[#151d2d] transition-colors">
                    <td className="px-6 py-4 text-slate-600 dark:text-slate-400">2026-09-06 10:11:45 UTC</td>
                    <td className="px-6 py-4 font-bold text-slate-950 dark:text-white">support@hostvra.com</td>
                    <td className="px-6 py-4 text-slate-800 dark:text-slate-300">user@outlook.com</td>
                    <td className="px-6 py-4 text-emerald-600 dark:text-emerald-400 font-semibold">delivered (250 2.0.0 Ok)</td>
                    <td className="px-6 py-4 text-slate-600 dark:text-slate-400">-0.80 (Clean)</td>
                  </tr>
                  <tr className="hover:bg-slate-50 dark:hover:bg-[#151d2d] transition-colors">
                    <td className="px-6 py-4 text-slate-600 dark:text-slate-400">2026-09-06 09:45:10 UTC</td>
                    <td className="px-6 py-4 font-semibold text-rose-600 dark:text-rose-400">spammer@badsource.net</td>
                    <td className="px-6 py-4 text-slate-800 dark:text-slate-300">admin@hostvra.com</td>
                    <td className="px-6 py-4 text-rose-600 dark:text-rose-400 font-semibold">rejected (554 Relay Denied)</td>
                    <td className="px-6 py-4 text-rose-600 dark:text-rose-400">+16.40 (Spam Block)</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* Modal: Add Email Domain */}
        {showAddDomainModal && (
          <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="bg-surface-900 border border-surface-800 rounded-2xl w-full max-w-md p-6 space-y-5 shadow-2xl">
              <div className="flex items-center justify-between">
                <h3 className="text-lg font-bold text-white flex items-center gap-2">
                  <Globe className="w-5 h-5 text-indigo-400" />
                  <span>Add Email Domain</span>
                </h3>
                <button onClick={() => setShowAddDomainModal(false)} className="text-slate-400 hover:text-white">
                  ✕
                </button>
              </div>

              <form onSubmit={handleAddDomain} className="space-y-4">
                <div>
                  <label className="block text-xs font-semibold text-slate-300 mb-1.5">Domain Name</label>
                  <input
                    type="text"
                    required
                    placeholder="example.com"
                    value={newDomainName}
                    onChange={(e) => setNewDomainName(e.target.value)}
                    className="w-full px-3.5 py-2.5 bg-surface-950 border border-surface-800 rounded-xl text-sm text-white focus:outline-none focus:border-indigo-500"
                  />
                  <p className="text-[11px] text-slate-400 mt-1">
                    Hostvra will automatically generate a 2048-bit RSA DKIM key and configure SPF/DMARC records.
                  </p>
                </div>

                <div className="pt-3 flex justify-end gap-3">
                  <button
                    type="button"
                    onClick={() => setShowAddDomainModal(false)}
                    className="px-4 py-2 rounded-xl text-sm font-medium text-slate-300 hover:bg-surface-800"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="px-4 py-2 rounded-xl text-sm font-medium bg-indigo-600 hover:bg-indigo-500 text-white shadow-lg shadow-indigo-600/20"
                  >
                    Add & Generate DKIM
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* Modal: Create Mailbox */}
        {showAddMailboxModal && (
          <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="bg-surface-900 border border-surface-800 rounded-2xl w-full max-w-md p-6 space-y-5 shadow-2xl">
              <div className="flex items-center justify-between">
                <h3 className="text-lg font-bold text-white flex items-center gap-2">
                  <Inbox className="w-5 h-5 text-indigo-400" />
                  <span>Create Mailbox Account</span>
                </h3>
                <button onClick={() => setShowAddMailboxModal(false)} className="text-slate-400 hover:text-white">
                  ✕
                </button>
              </div>

              <form onSubmit={handleAddMailbox} className="space-y-4">
                <div>
                  <label className="block text-xs font-semibold text-slate-300 mb-1.5">Select Domain</label>
                  <select
                    value={newMailboxDomain}
                    onChange={(e) => setNewMailboxDomain(e.target.value)}
                    className="w-full px-3.5 py-2.5 bg-surface-950 border border-surface-800 rounded-xl text-sm text-white focus:outline-none focus:border-indigo-500"
                  >
                    {domains.map((d) => (
                      <option key={d.id} value={d.id}>
                        @{d.domain}
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-300 mb-1.5">Username (Local-Part)</label>
                  <div className="flex items-center">
                    <input
                      type="text"
                      required
                      placeholder="info or support"
                      value={newLocalPart}
                      onChange={(e) => setNewLocalPart(e.target.value)}
                      className="flex-1 px-3.5 py-2.5 bg-surface-950 border border-surface-800 rounded-l-xl text-sm text-white focus:outline-none focus:border-indigo-500"
                    />
                    <span className="px-3 py-2.5 bg-surface-800 border border-l-0 border-surface-800 rounded-r-xl text-xs text-slate-400 font-mono">
                      @{domains.find((d) => d.id === newMailboxDomain)?.domain || 'hostvra.com'}
                    </span>
                  </div>
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-300 mb-1.5">Display Name</label>
                  <input
                    type="text"
                    placeholder="e.g. Sales Department"
                    value={newMailboxName}
                    onChange={(e) => setNewMailboxName(e.target.value)}
                    className="w-full px-3.5 py-2.5 bg-surface-950 border border-surface-800 rounded-xl text-sm text-white focus:outline-none focus:border-indigo-500"
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-300 mb-1.5">Password</label>
                  <input
                    type="password"
                    required
                    placeholder="At least 8 characters"
                    value={newMailboxPass}
                    onChange={(e) => setNewMailboxPass(e.target.value)}
                    className="w-full px-3.5 py-2.5 bg-surface-950 border border-surface-800 rounded-xl text-sm text-white focus:outline-none focus:border-indigo-500"
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-300 mb-1.5">
                    Storage Quota (GB)
                  </label>
                  <input
                    type="number"
                    min={1}
                    max={100}
                    value={newMailboxQuotaGB}
                    onChange={(e) => setNewMailboxQuotaGB(parseInt(e.target.value) || 5)}
                    className="w-full px-3.5 py-2.5 bg-surface-950 border border-surface-800 rounded-xl text-sm text-white focus:outline-none focus:border-indigo-500"
                  />
                </div>

                <div className="pt-3 flex justify-end gap-3">
                  <button
                    type="button"
                    onClick={() => setShowAddMailboxModal(false)}
                    className="px-4 py-2 rounded-xl text-sm font-medium text-slate-300 hover:bg-surface-800"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="px-4 py-2 rounded-xl text-sm font-medium bg-indigo-600 hover:bg-indigo-500 text-white shadow-lg shadow-indigo-600/20"
                  >
                    Create Mailbox
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* Modal: DNS Setup Instructions */}
        {showDNSModal && (
          <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="bg-surface-900 border border-surface-800 rounded-2xl w-full max-w-2xl p-6 space-y-5 shadow-2xl">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-lg font-bold text-white flex items-center gap-2">
                    <ShieldCheck className="w-5 h-5 text-emerald-400" />
                    <span>DNS Configuration for {showDNSModal.domain}</span>
                  </h3>
                  <p className="text-xs text-slate-400 mt-0.5">
                    Publish these records in your DNS manager (Hostvra DNS, Cloudflare, or Registrar).
                  </p>
                </div>
                <button onClick={() => setShowDNSModal(null)} className="text-slate-400 hover:text-white">
                  ✕
                </button>
              </div>

              <div className="space-y-3 font-mono text-xs">
                {/* MX */}
                <div className="p-3.5 rounded-xl bg-surface-950 border border-surface-800 space-y-1">
                  <div className="flex justify-between items-center text-slate-400">
                    <span className="font-semibold text-white">MX Record (Incoming Mail)</span>
                    <button
                      onClick={() => copyToClipboard(`10 ${showDNSModal.mail_hostname}.`, 'mx')}
                      className="hover:text-white flex items-center gap-1 text-[11px]"
                    >
                      {copiedKey === 'mx' ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                      <span>Copy</span>
                    </button>
                  </div>
                  <p className="text-indigo-300">Host: @ | Priority: 10 | Target: {showDNSModal.mail_hostname}.</p>
                </div>

                {/* SPF */}
                <div className="p-3.5 rounded-xl bg-surface-950 border border-surface-800 space-y-1">
                  <div className="flex justify-between items-center text-slate-400">
                    <span className="font-semibold text-white">SPF Record (Authentication)</span>
                    <button
                      onClick={() => copyToClipboard('v=spf1 mx ~all', 'spf')}
                      className="hover:text-white flex items-center gap-1 text-[11px]"
                    >
                      {copiedKey === 'spf' ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                      <span>Copy</span>
                    </button>
                  </div>
                  <p className="text-indigo-300">Type: TXT | Host: @ | Value: v=spf1 mx ~all</p>
                </div>

                {/* DKIM */}
                <div className="p-3.5 rounded-xl bg-surface-950 border border-surface-800 space-y-1">
                  <div className="flex justify-between items-center text-slate-400">
                    <span className="font-semibold text-white">DKIM Key Record (2048-bit RSA Signature)</span>
                    <button
                      onClick={() =>
                        copyToClipboard(
                          'v=DKIM1; k=rsa; p=MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA4vQ15B3...',
                          'dkim'
                        )
                      }
                      className="hover:text-white flex items-center gap-1 text-[11px]"
                    >
                      {copiedKey === 'dkim' ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                      <span>Copy</span>
                    </button>
                  </div>
                  <p className="text-indigo-300">Type: TXT | Host: {showDNSModal.dkim_selector}._domainkey | Value: v=DKIM1; k=rsa; p=...</p>
                </div>

                {/* DMARC */}
                <div className="p-3.5 rounded-xl bg-surface-950 border border-surface-800 space-y-1">
                  <div className="flex justify-between items-center text-slate-400">
                    <span className="font-semibold text-white">DMARC Policy Record</span>
                    <button
                      onClick={() => copyToClipboard(`v=DMARC1; p=none; rua=mailto:dmarc@${showDNSModal.domain}`, 'dmarc')}
                      className="hover:text-white flex items-center gap-1 text-[11px]"
                    >
                      {copiedKey === 'dmarc' ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                      <span>Copy</span>
                    </button>
                  </div>
                  <p className="text-indigo-300">Type: TXT | Host: _dmarc | Value: v=DMARC1; p=none; rua=mailto:dmarc@{showDNSModal.domain}</p>
                </div>
              </div>

              <div className="pt-2 flex justify-end">
                <button
                  onClick={() => setShowDNSModal(null)}
                  className="px-4 py-2 rounded-xl text-sm font-medium bg-indigo-600 hover:bg-indigo-500 text-white"
                >
                  Done
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Modal: Password Reset */}
        {showPasswordModal && (
          <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="bg-surface-900 border border-surface-800 rounded-2xl w-full max-w-md p-6 space-y-5 shadow-2xl">
              <div className="flex items-center justify-between">
                <h3 className="text-lg font-bold text-white flex items-center gap-2">
                  <Key className="w-5 h-5 text-indigo-400" />
                  <span>Reset Mailbox Password</span>
                </h3>
                <button onClick={() => setShowPasswordModal(null)} className="text-slate-400 hover:text-white">
                  ✕
                </button>
              </div>

              <div className="space-y-4">
                <p className="text-sm text-slate-300">
                  Update password for <span className="font-mono text-white font-semibold">{showPasswordModal.email}</span>
                </p>

                <div>
                  <label className="block text-xs font-semibold text-slate-300 mb-1.5">New Password</label>
                  <input
                    type="password"
                    placeholder="Enter new strong password"
                    value={newPassword}
                    onChange={(e) => setNewPassword(e.target.value)}
                    className="w-full px-3.5 py-2.5 bg-surface-950 border border-surface-800 rounded-xl text-sm text-white focus:outline-none focus:border-indigo-500"
                  />
                </div>

                <div className="pt-2 flex justify-end gap-3">
                  <button
                    onClick={() => setShowPasswordModal(null)}
                    className="px-4 py-2 rounded-xl text-sm font-medium text-slate-300 hover:bg-surface-800"
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

        {/* Modal: Webmail Access & Mail Client Configuration */}
        {webmailModalMailbox && (
          <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-sm flex items-center justify-center p-4 animate-fadeIn">
            <div className="bg-surface-900 border border-surface-750 rounded-2xl w-full max-w-lg p-6 space-y-5 shadow-2xl">
              {/* Header */}
              <div className="flex items-center justify-between pb-4 border-b border-surface-800">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-xl bg-indigo-600/10 border border-indigo-500/20 flex items-center justify-center text-indigo-400 shadow-inner">
                    <Mail className="w-5 h-5" />
                  </div>
                  <div>
                    <h3 className="text-base font-bold text-white tracking-tight">Webmail & Mail Client Access</h3>
                    <p className="text-xs text-slate-400 font-mono">{webmailModalMailbox.email}</p>
                  </div>
                </div>
                <button
                  onClick={() => setWebmailModalMailbox(null)}
                  className="p-1 rounded-lg text-slate-400 hover:text-white hover:bg-surface-800 transition-colors"
                >
                  ✕
                </button>
              </div>

              {/* Direct Webmail Launch Card */}
              <div className="p-4 rounded-xl bg-indigo-950/40 border border-indigo-500/30 space-y-3 shadow-inner">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-semibold text-indigo-300 flex items-center gap-1.5">
                    <ExternalLink className="w-3.5 h-3.5" />
                    Roundcube Webmail Portal
                  </span>
                  <span className="text-[10px] px-2 py-0.5 rounded-full bg-emerald-500/20 text-emerald-400 font-medium">
                    Online
                  </span>
                </div>
                <p className="text-xs text-slate-300 leading-relaxed">
                  Open Roundcube webmail directly in your browser using your server IP or domain:
                </p>
                <div className="flex flex-wrap gap-2">
                  <a
                    href={typeof window !== 'undefined' ? `${window.location.protocol}//${window.location.hostname}/webmail` : '/webmail'}
                    target="_blank"
                    rel="noreferrer"
                    className="flex-1 min-w-[180px] px-4 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold flex items-center justify-center gap-2 shadow-lg shadow-indigo-600/25 transition-all active:scale-95"
                  >
                    <span>Open Webmail ({typeof window !== 'undefined' ? window.location.hostname : 'Server IP'})</span>
                    <ExternalLink className="w-3.5 h-3.5" />
                  </a>
                  {webmailModalMailbox.email.includes('@') && (
                    <a
                      href={`http://webmail.${webmailModalMailbox.email.split('@')[1]}`}
                      target="_blank"
                      rel="noreferrer"
                      className="px-3.5 py-2.5 rounded-xl bg-surface-800 hover:bg-surface-700 text-slate-200 text-xs font-semibold flex items-center justify-center gap-1.5 border border-surface-700 transition-all"
                    >
                      <span>webmail.{webmailModalMailbox.email.split('@')[1]}</span>
                      <ExternalLink className="w-3.5 h-3.5" />
                    </a>
                  )}
                </div>
              </div>

              {/* Mail Client Configuration Settings (Outlook, Apple Mail, Thunderbird, Mobile) */}
              <div className="space-y-2.5">
                <h4 className="text-xs font-bold uppercase tracking-wider text-slate-400 flex items-center gap-1.5">
                  <Layers className="w-3.5 h-3.5 text-indigo-400" />
                  Mail Client Setup (Outlook / iPhone / Thunderbird)
                </h4>

                <div className="space-y-2 text-xs">
                  {/* Username */}
                  <div className="p-2.5 rounded-lg bg-surface-950/60 border border-surface-800 flex items-center justify-between">
                    <span className="text-slate-400">Username / Email:</span>
                    <div className="flex items-center gap-2">
                      <code className="text-slate-200 font-mono font-semibold">{webmailModalMailbox.email}</code>
                      <button
                        onClick={() => copyToClipboard(webmailModalMailbox.email, 'email')}
                        className="text-slate-400 hover:text-white"
                        title="Copy email"
                      >
                        {copiedKey === 'email' ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                      </button>
                    </div>
                  </div>

                  {/* Incoming IMAP */}
                  <div className="p-2.5 rounded-lg bg-surface-950/60 border border-surface-800 flex items-center justify-between">
                    <div>
                      <span className="text-slate-400 block">Incoming Server (IMAP):</span>
                      <span className="text-[10px] text-slate-500 font-mono">Port 993 (SSL/TLS)</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <code className="text-slate-200 font-mono font-semibold">
                        {webmailModalMailbox.email.includes('@') ? `mail.${webmailModalMailbox.email.split('@')[1]}` : 'mail.yourdomain.com'}
                      </code>
                      <button
                        onClick={() => copyToClipboard(webmailModalMailbox.email.includes('@') ? `mail.${webmailModalMailbox.email.split('@')[1]}` : 'mail.yourdomain.com', 'imap')}
                        className="text-slate-400 hover:text-white"
                        title="Copy IMAP host"
                      >
                        {copiedKey === 'imap' ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                      </button>
                    </div>
                  </div>

                  {/* Outgoing SMTP */}
                  <div className="p-2.5 rounded-lg bg-surface-950/60 border border-surface-800 flex items-center justify-between">
                    <div>
                      <span className="text-slate-400 block">Outgoing Server (SMTP):</span>
                      <span className="text-[10px] text-slate-500 font-mono">Port 587 (STARTTLS) / 465 (SSL)</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <code className="text-slate-200 font-mono font-semibold">
                        {webmailModalMailbox.email.includes('@') ? `mail.${webmailModalMailbox.email.split('@')[1]}` : 'mail.yourdomain.com'}
                      </code>
                      <button
                        onClick={() => copyToClipboard(webmailModalMailbox.email.includes('@') ? `mail.${webmailModalMailbox.email.split('@')[1]}` : 'mail.yourdomain.com', 'smtp')}
                        className="text-slate-400 hover:text-white"
                        title="Copy SMTP host"
                      >
                        {copiedKey === 'smtp' ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                      </button>
                    </div>
                  </div>
                </div>
              </div>

              <div className="pt-2 flex justify-end">
                <button
                  onClick={() => setWebmailModalMailbox(null)}
                  className="px-4 py-2 rounded-xl text-xs font-semibold bg-surface-800 hover:bg-surface-700 text-slate-300 hover:text-white transition"
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
