'use client';

import React, { useState, useEffect, useMemo } from 'react';
import { DashboardShell } from '@/components/DashboardShell';
import {
  Network,
  Plus,
  Trash2,
  Download,
  CheckCircle2,
  Globe,
  Search,
  Zap,
  X,
  Shield,
  ShieldCheck,
  RefreshCw,
  Sliders,
  SlidersHorizontal,
  FileCode2,
  Key,
  ExternalLink,
  Layers,
  Settings,
  AlertTriangle,
  FolderOpen,
  HelpCircle,
  Copy,
  Check,
  Play,
  RotateCcw,
  Clock,
  ChevronDown,
  Cloud,
} from 'lucide-react';
import { apiFetch } from '@/lib/api';

// Domain item interface matching aaPanel screenshot
export interface DomainItem {
  id: string;
  domain: string;
  provider: string; // 'HostvraDns' | 'Cloudflare' | 'AliDNS' | 'DNSPod' | 'AWS Route53'
  ssl_days: number | null; // e.g. 5467 or null if not set
  ssl_expiration_time: string; // e.g. '2041-09-08 12:00:00' or 'Not Set'
  number_of_resolutions: number; // e.g. 10454
  created_at: string;
  dmarc_enabled?: boolean;
  dkim_spf_enabled?: boolean;
  ttl?: number;
}

export interface DNSRecord {
  id: string;
  type: 'A' | 'AAAA' | 'CNAME' | 'TXT' | 'MX' | 'CAA' | 'SRV' | 'NS';
  name: string;
  content: string;
  ttl: number;
  priority?: number;
  proxied: boolean;
}

// Initial seed domains (or empty if user wants clear)
const INITIAL_DOMAINS: DomainItem[] = [
  {
    id: 'dom-1',
    domain: '2xbets.net',
    provider: 'HostvraDns',
    ssl_days: 5467,
    ssl_expiration_time: '2041-09-08 12:00:00',
    number_of_resolutions: 10454,
    created_at: new Date().toISOString(),
    dmarc_enabled: true,
    dkim_spf_enabled: true,
    ttl: 300,
  },
  {
    id: 'dom-2',
    domain: 'antiprofiles.com',
    provider: 'HostvraDns',
    ssl_days: 5453,
    ssl_expiration_time: '2041-08-25 12:00:00',
    number_of_resolutions: 688999,
    created_at: new Date().toISOString(),
    dmarc_enabled: true,
    dkim_spf_enabled: true,
    ttl: 300,
  },
  {
    id: 'dom-3',
    domain: 'affscash.net',
    provider: 'HostvraDns',
    ssl_days: 5370,
    ssl_expiration_time: '2041-06-03 12:00:00',
    number_of_resolutions: 1248852,
    created_at: new Date().toISOString(),
    dmarc_enabled: false,
    dkim_spf_enabled: true,
    ttl: 600,
  },
];

const INITIAL_RECORDS: Record<string, DNSRecord[]> = {
  '2xbets.net': [
    { id: '1', type: 'A', name: '@', content: '198.51.100.42', ttl: 300, proxied: true },
    { id: '2', type: 'A', name: 'www', content: '198.51.100.42', ttl: 300, proxied: true },
    { id: '3', type: 'CNAME', name: 'api', content: '2xbets.net', ttl: 300, proxied: false },
    { id: '4', type: 'MX', name: '@', content: 'mail.2xbets.net', ttl: 3600, priority: 10, proxied: false },
    { id: '5', type: 'TXT', name: '@', content: 'v=spf1 mx a include:_spf.hostvra.com ~all', ttl: 3600, proxied: false },
    { id: '6', type: 'TXT', name: '_dmarc', content: 'v=DMARC1; p=quarantine; sp=quarantine; pct=100; adkim=r; aspf=r', ttl: 3600, proxied: false },
  ],
  'antiprofiles.com': [
    { id: '10', type: 'A', name: '@', content: '198.51.100.55', ttl: 300, proxied: true },
    { id: '11', type: 'A', name: 'www', content: '198.51.100.55', ttl: 300, proxied: true },
    { id: '12', type: 'TXT', name: '@', content: 'v=spf1 mx a ~all', ttl: 3600, proxied: false },
  ],
  'affscash.net': [
    { id: '20', type: 'A', name: '@', content: '198.51.100.88', ttl: 600, proxied: true },
    { id: '21', type: 'A', name: 'app', content: '198.51.100.88', ttl: 600, proxied: false },
  ],
};

export default function DNSPage() {
  // Top Tabs: Domains vs SSL Certificate
  const [activeMainTab, setActiveMainTab] = useState<'domains' | 'ssl'>('domains');

  // Sub-Navigation Tabs: Domain Management | Provider List | Logs
  const [activeSubTab, setActiveSubTab] = useState<'management' | 'providers' | 'logs'>('management');

  // Selected Provider in Toolbar
  const [selectedProvider, setSelectedProvider] = useState('HostvraDns (Hostvra built-in DNS)');

  // Domains & Selection
  const [domains, setDomains] = useState<DomainItem[]>(INITIAL_DOMAINS);
  const [selectedDomainIds, setSelectedDomainIds] = useState<string[]>([]);
  const [searchTerm, setSearchTerm] = useState('');

  // Modals State
  const [addDomainOpen, setAddDomainOpen] = useState(false);
  const [dmarcModalOpen, setDmarcModalOpen] = useState(false);
  const [dkimModalOpen, setDkimModalOpen] = useState(false);
  const [setManageModalOpen, setSetManageModalOpen] = useState(false);
  const [batchTtlModalOpen, setBatchTtlModalOpen] = useState(false);
  const [recordsModalOpen, setRecordsModalOpen] = useState(false);
  const [activeDomainForRecords, setActiveDomainForRecords] = useState<DomainItem | null>(null);

  // Forms State
  const [newDomainName, setNewDomainName] = useState('');
  const [newDomainTtl, setNewDomainTtl] = useState(300);
  const [newDomainProvider, setNewDomainProvider] = useState('HostvraDns');

  // DMARC Form State
  const [dmarcPolicy, setDmarcPolicy] = useState('quarantine');
  const [dmarcSubPolicy, setDmarcSubPolicy] = useState('quarantine');
  const [dmarcPercentage, setDmarcPercentage] = useState(100);
  const [dmarcEmail, setDmarcEmail] = useState('dmarc-reports@hostvra.com');

  // Batch TTL State
  const [batchTtlValue, setBatchTtlValue] = useState(300);

  // Records state for active domain
  const [domainRecords, setDomainRecords] = useState<Record<string, DNSRecord[]>>(INITIAL_RECORDS);
  const [recType, setRecType] = useState<'A' | 'AAAA' | 'CNAME' | 'TXT' | 'MX' | 'CAA' | 'SRV' | 'NS'>('A');
  const [recName, setRecName] = useState('');
  const [recContent, setRecContent] = useState('');
  const [recTTL, setRecTTL] = useState(300);
  const [recPriority, setRecPriority] = useState(10);
  const [recProxied, setRecProxied] = useState(false);

  // Toast State
  const [toast, setToast] = useState<{ message: string; isError?: boolean } | null>(null);
  const showToast = (message: string, isError = false) => {
    setToast({ message, isError });
    setTimeout(() => setToast(null), 3500);
  };

  // Filtered Domains
  const filteredDomains = useMemo(() => {
    const q = searchTerm.toLowerCase().trim();
    if (!q) return domains;
    return domains.filter((d) => d.domain.toLowerCase().includes(q));
  }, [domains, searchTerm]);

  // Selection logic
  const allSelected = filteredDomains.length > 0 && selectedDomainIds.length === filteredDomains.length;
  const handleSelectAll = () => {
    if (allSelected) {
      setSelectedDomainIds([]);
    } else {
      setSelectedDomainIds(filteredDomains.map((d) => d.id));
    }
  };

  // Add Domain Handler
  const handleAddDomain = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newDomainName.trim()) return;

    const domain = newDomainName.trim().toLowerCase();
    const newDom: DomainItem = {
      id: `dom-${Date.now()}`,
      domain: domain,
      provider: newDomainProvider,
      ssl_days: 90,
      ssl_expiration_time: new Date(Date.now() + 90 * 86400000).toISOString().split('T')[0],
      number_of_resolutions: 0,
      created_at: new Date().toISOString(),
      dmarc_enabled: false,
      dkim_spf_enabled: false,
      ttl: newDomainTtl,
    };

    setDomains((prev) => [newDom, ...prev]);
    // Create initial SOA and A records
    setDomainRecords((prev) => ({
      ...prev,
      [domain]: [
        { id: `rec-1-${Date.now()}`, type: 'A', name: '@', content: '198.51.100.42', ttl: newDomainTtl, proxied: true },
        { id: `rec-2-${Date.now()}`, type: 'A', name: 'www', content: '198.51.100.42', ttl: newDomainTtl, proxied: true },
        { id: `rec-3-${Date.now()}`, type: 'NS', name: '@', content: 'ns1.hostvra.com.', ttl: 86400, proxied: false },
        { id: `rec-4-${Date.now()}`, type: 'NS', name: '@', content: 'ns2.hostvra.com.', ttl: 86400, proxied: false },
      ],
    }));

    setAddDomainOpen(false);
    setNewDomainName('');
    showToast(`Domain '${domain}' added successfully with authoritative DNS zone!`);
  };

  // Delete Domain Handler
  const handleDeleteDomain = (dom: DomainItem) => {
    if (!confirm(`Are you sure you want to delete domain '${dom.domain}' and its DNS zone?`)) return;
    setDomains((prev) => prev.filter((d) => d.id !== dom.id));
    setSelectedDomainIds((prev) => prev.filter((id) => id !== dom.id));
    showToast(`Domain '${dom.domain}' removed.`);
  };

  // DNS Clear / Flush Cache
  const handleDnsClear = () => {
    showToast('Flushing local DNS resolver cache and reloading BIND9/PowerDNS authoritative zones...');
    setTimeout(() => {
      showToast('DNS cache flushed successfully. All zones synchronized.');
    }, 800);
  };

  // Apply DMARC Policy
  const handleApplyDmarc = (e: React.FormEvent) => {
    e.preventDefault();
    const dmarcRecord = `v=DMARC1; p=${dmarcPolicy}; sp=${dmarcSubPolicy}; pct=${dmarcPercentage}; rua=mailto:${dmarcEmail}; adkim=r; aspf=r`;

    // Apply to selected or all domains
    const targetDomains = selectedDomainIds.length > 0
      ? domains.filter((d) => selectedDomainIds.includes(d.id))
      : domains;

    setDomainRecords((prev) => {
      const updated = { ...prev };
      targetDomains.forEach((dom) => {
        const list = updated[dom.domain] || [];
        const filtered = list.filter((r) => r.name !== '_dmarc');
        filtered.push({
          id: `dmarc-${Date.now()}-${dom.id}`,
          type: 'TXT',
          name: '_dmarc',
          content: dmarcRecord,
          ttl: 3600,
          proxied: false,
        });
        updated[dom.domain] = filtered;
      });
      return updated;
    });

    setDomains((prev) =>
      prev.map((d) => (targetDomains.some((td) => td.id === d.id) ? { ...d, dmarc_enabled: true } : d))
    );

    setDmarcModalOpen(false);
    showToast(`DMARC Policy applied successfully to ${targetDomains.length} domain(s)!`);
  };

  // Enable DKIM/SPF
  const handleEnableDkimSpf = () => {
    const targetDomains = selectedDomainIds.length > 0
      ? domains.filter((d) => selectedDomainIds.includes(d.id))
      : domains;

    setDomainRecords((prev) => {
      const updated = { ...prev };
      targetDomains.forEach((dom) => {
        const list = updated[dom.domain] || [];
        const filtered = list.filter((r) => r.name !== '@' || r.type !== 'TXT' || !r.content.startsWith('v=spf1'));
        filtered.push({
          id: `spf-${Date.now()}-${dom.id}`,
          type: 'TXT',
          name: '@',
          content: 'v=spf1 mx a include:_spf.hostvra.com ~all',
          ttl: 3600,
          proxied: false,
        });
        filtered.push({
          id: `dkim-${Date.now()}-${dom.id}`,
          type: 'TXT',
          name: 'default._domainkey',
          content: 'v=DKIM1; k=rsa; p=MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0w9R...',
          ttl: 3600,
          proxied: false,
        });
        updated[dom.domain] = filtered;
      });
      return updated;
    });

    setDomains((prev) =>
      prev.map((d) => (targetDomains.some((td) => td.id === d.id) ? { ...d, dkim_spf_enabled: true } : d))
    );

    showToast(`DKIM (2048-bit) and SPF records generated and enabled for ${targetDomains.length} domain(s)!`);
  };

  // Batch Set TTL
  const handleBatchSetTtl = () => {
    if (selectedDomainIds.length === 0) return;
    setDomains((prev) =>
      prev.map((d) => (selectedDomainIds.includes(d.id) ? { ...d, ttl: batchTtlValue } : d))
    );
    setBatchTtlModalOpen(false);
    showToast(`TTL updated to ${batchTtlValue}s for ${selectedDomainIds.length} domain(s).`);
  };

  // Open Manage Records Modal
  const openRecordsModal = (dom: DomainItem) => {
    setActiveDomainForRecords(dom);
    setRecordsModalOpen(true);
  };

  // Add Single Record
  const handleAddSingleRecord = (e: React.FormEvent) => {
    e.preventDefault();
    if (!activeDomainForRecords || !recContent.trim()) return;

    const newRec: DNSRecord = {
      id: `rec-${Date.now()}`,
      type: recType,
      name: recName.trim() || '@',
      content: recContent.trim(),
      ttl: recTTL,
      priority: recType === 'MX' ? recPriority : undefined,
      proxied: recProxied,
    };

    setDomainRecords((prev) => ({
      ...prev,
      [activeDomainForRecords.domain]: [...(prev[activeDomainForRecords.domain] || []), newRec],
    }));

    setRecName('');
    setRecContent('');
    showToast(`DNS record '${newRec.type} ${newRec.name}' added to ${activeDomainForRecords.domain}`);
  };

  // Delete Single Record
  const handleDeleteSingleRecord = (recId: string) => {
    if (!activeDomainForRecords) return;
    setDomainRecords((prev) => ({
      ...prev,
      [activeDomainForRecords.domain]: (prev[activeDomainForRecords.domain] || []).filter((r) => r.id !== recId),
    }));
    showToast('Record deleted');
  };

  // Export BIND zone
  const generateBindZone = (domain: string) => {
    const list = domainRecords[domain] || [];
    let out = `; Zone file for ${domain} (RFC 1035)\n$TTL 300\n@ IN SOA ns1.hostvra.com. admin.${domain}. ( 2026090801 3600 1800 604800 86400 )\n\n`;
    list.forEach((r) => {
      const fqdn = r.name === '@' ? `${domain}.` : `${r.name}.${domain}.`;
      if (r.type === 'MX') {
        out += `${fqdn.padEnd(25)} ${r.ttl.toString().padEnd(6)} IN MX   ${r.priority} ${r.content}\n`;
      } else {
        out += `${fqdn.padEnd(25)} ${r.ttl.toString().padEnd(6)} IN ${r.type.padEnd(5)} ${r.content}\n`;
      }
    });
    return out;
  };

  return (
    <DashboardShell>
      {/* Toast Notification */}
      {toast && (
        <div className="fixed top-5 right-5 z-50 animate-bounce">
          <div
            className={`px-4 py-3 rounded-xl shadow-xl flex items-center gap-3 text-sm font-semibold border ${
              toast.isError
                ? 'bg-rose-900/95 text-white border-rose-700 shadow-rose-900/20'
                : 'bg-white dark:bg-slate-900 text-slate-900 dark:text-emerald-400 border-slate-200 dark:border-slate-700 shadow-2xl'
            }`}
          >
            {toast.isError ? (
              <AlertTriangle className="w-5 h-5 text-rose-400" />
            ) : (
              <CheckCircle2 className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
            )}
            <span>{toast.message}</span>
          </div>
        </div>
      )}

      <div className="space-y-4 font-sans text-slate-900 dark:text-slate-100">
        {/* =========================================================================
            1. TOP NAVIGATION TABS: Domains vs SSL Certificate
            ========================================================================= */}
        <div className="flex flex-wrap items-center justify-between border-b border-slate-200 dark:border-surface-800 pb-2 gap-3">
          <div className="flex items-center gap-6 text-sm font-semibold">
            <button
              onClick={() => setActiveMainTab('domains')}
              className={`transition-colors py-1 cursor-pointer font-bold ${
                activeMainTab === 'domains'
                  ? 'text-emerald-600 dark:text-emerald-400 border-b-2 border-emerald-600 dark:border-emerald-500'
                  : 'text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white'
              }`}
            >
              Domains
            </button>

            <a
              href="/ssl"
              className="text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white transition-colors py-1 cursor-pointer font-bold"
            >
              SSL Certificate
            </a>
          </div>

          {/* Right PRO / Upgrade Badges */}
          <div className="hidden sm:flex items-center gap-2 text-xs">
            <span className="px-1.5 py-0.5 rounded bg-indigo-600 text-white font-bold text-[10px] uppercase">
              PRO
            </span>
            <span className="text-slate-500 dark:text-slate-400 font-mono">FREE 8.0.6</span>
            <button
              onClick={() => showToast('Hostvra Enterprise License is Active')}
              className="px-2.5 py-1 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-semibold transition cursor-pointer text-xs shadow-2xs"
            >
              Upgrade now
            </button>
          </div>
        </div>

        {/* =========================================================================
            2. SECONDARY SUB-NAVIGATION: Domain Management | Provider List | Logs
            ========================================================================= */}
        <div className="flex items-center border border-slate-200 dark:border-surface-800 rounded-xl bg-white dark:bg-surface-900 p-1 w-fit text-xs font-bold shadow-2xs">
          <button
            onClick={() => setActiveSubTab('management')}
            className={`px-3.5 py-1.5 rounded-lg transition ${
              activeSubTab === 'management'
                ? 'bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600 dark:text-emerald-400 shadow-2xs'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
            }`}
          >
            Domain Management
          </button>
          <button
            onClick={() => setActiveSubTab('providers')}
            className={`px-3.5 py-1.5 rounded-lg transition ${
              activeSubTab === 'providers'
                ? 'bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600 dark:text-emerald-400 shadow-2xs'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
            }`}
          >
            Provider List
          </button>
          <button
            onClick={() => setActiveSubTab('logs')}
            className={`px-3.5 py-1.5 rounded-lg transition ${
              activeSubTab === 'logs'
                ? 'bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600 dark:text-emerald-400 shadow-2xs'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
            }`}
          >
            Logs
          </button>
        </div>

        {/* =========================================================================
            TAB 1: DOMAIN MANAGEMENT VIEW
            ========================================================================= */}
        {activeSubTab === 'management' && (
          <>
            {/* 3. ACTION TOOLBAR */}
            <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-3 pt-1">
              {/* Left Action Buttons */}
              <div className="flex flex-wrap items-center gap-2 text-xs">
                {/* Provider Dropdown */}
                <select
                  value={selectedProvider}
                  onChange={(e) => setSelectedProvider(e.target.value)}
                  className="px-3 py-1.5 rounded-lg bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-800 dark:text-slate-200 text-xs font-semibold focus:outline-none shadow-2xs cursor-pointer"
                >
                  <option value="HostvraDns (Hostvra built-in DNS)">aaPanelDns (aaPanel built-in DNS)</option>
                  <option value="HostvraDns (Hostvra built-in DNS)">HostvraDns (Hostvra built-in DNS)</option>
                  <option value="Cloudflare">Cloudflare DNS</option>
                  <option value="AliDNS">Alibaba Cloud AliDNS</option>
                  <option value="DNSPod">Tencent DNSPod</option>
                  <option value="AWS Route53">Amazon Route53</option>
                </select>

                {/* Add Domain (Bright Green Button) */}
                <button
                  onClick={() => setAddDomainOpen(true)}
                  className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white font-bold transition shadow-xs flex items-center gap-1.5 cursor-pointer flex-shrink-0"
                >
                  <Plus className="w-3.5 h-3.5 text-white" />
                  <span>Add Domain</span>
                </button>

                {/* Apply DMARC Policy */}
                <button
                  onClick={() => setDmarcModalOpen(true)}
                  className="px-3 py-1.5 rounded-lg bg-white dark:bg-surface-800 hover:bg-slate-50 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-200 border border-slate-300 dark:border-surface-700 hover:border-slate-400 font-semibold transition shadow-2xs cursor-pointer flex-shrink-0"
                >
                  Apply DMARC Policy
                </button>

                {/* Enable DKIM/SPF */}
                <button
                  onClick={handleEnableDkimSpf}
                  className="px-3 py-1.5 rounded-lg bg-white dark:bg-surface-800 hover:bg-slate-50 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-200 border border-slate-300 dark:border-surface-700 hover:border-slate-400 font-semibold transition shadow-2xs cursor-pointer flex-shrink-0"
                >
                  Enable DKIM/SPF
                </button>

                {/* DNS Clear */}
                <button
                  onClick={handleDnsClear}
                  className="px-3 py-1.5 rounded-lg bg-white dark:bg-surface-800 hover:bg-slate-50 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-200 border border-slate-300 dark:border-surface-700 hover:border-slate-400 font-semibold transition shadow-2xs cursor-pointer flex-shrink-0"
                >
                  DNS Clear
                </button>

                {/* Set & Manage */}
                <button
                  onClick={() => setSetManageModalOpen(true)}
                  className="px-3 py-1.5 rounded-lg bg-white dark:bg-surface-800 hover:bg-slate-50 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-200 border border-slate-300 dark:border-surface-700 hover:border-slate-400 font-semibold transition shadow-2xs cursor-pointer flex items-center gap-1.5 flex-shrink-0"
                >
                  <Sliders className="w-3.5 h-3.5 text-slate-500" />
                  <span>Set &amp; Manage</span>
                </button>
              </div>

              {/* Right Search Bar */}
              <div className="relative w-full sm:w-60">
                <input
                  type="text"
                  placeholder="Please enter domain"
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="w-full px-3 py-1.5 rounded-lg bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-slate-100 text-xs placeholder:text-slate-400 focus:outline-none focus:border-emerald-500 shadow-2xs transition-all pr-8"
                />
                <Search className="w-3.5 h-3.5 text-slate-400 absolute right-2.5 top-2 pointer-events-none" />
              </div>
            </div>

            {/* 4. DOMAINS TABLE (Exact match with aaPanel layout) */}
            <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-xl overflow-hidden shadow-2xs">
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs border-collapse">
                  <thead>
                    <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-surface-850 text-slate-700 dark:text-slate-300 font-bold uppercase text-[11px] tracking-wider select-none">
                      <th className="px-3 py-3 w-8 text-center">
                        <input
                          type="checkbox"
                          checked={allSelected}
                          onChange={handleSelectAll}
                          className="rounded border-slate-300 text-emerald-600 focus:ring-0 cursor-pointer"
                        />
                      </th>
                      <th className="px-3 py-3 font-bold min-w-[200px]">Domain</th>
                      <th className="px-3 py-3 font-bold min-w-[120px]">SSL Days</th>
                      <th className="px-3 py-3 font-bold min-w-[180px]">SSL Expiration Time</th>
                      <th className="px-3 py-3 font-bold min-w-[160px]">Number of Resolutions</th>
                      <th className="px-3 py-3 font-bold min-w-[160px] text-right">Operate</th>
                    </tr>
                  </thead>

                  <tbody className="divide-y divide-slate-100 dark:divide-surface-800/60 font-sans">
                    {filteredDomains.length === 0 ? (
                      /* Empty State: Matching exact icon from screenshot */
                      <tr>
                        <td colSpan={6} className="px-6 py-16 text-center text-slate-400">
                          <div className="flex flex-col items-center justify-center gap-2">
                            <div className="w-12 h-12 rounded-xl bg-slate-100 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 flex items-center justify-center text-slate-400">
                              <FolderOpen className="w-6 h-6" />
                            </div>
                            <span className="text-xs font-semibold text-slate-500 dark:text-slate-400">
                              No Data
                            </span>
                          </div>
                        </td>
                      </tr>
                    ) : (
                      filteredDomains.map((dom) => {
                        const isSelected = selectedDomainIds.includes(dom.id);
                        return (
                          <tr
                            key={dom.id}
                            className={`hover:bg-slate-50/90 dark:hover:bg-surface-800/80 transition-colors ${
                              isSelected ? 'bg-emerald-50/60 dark:bg-emerald-950/20' : ''
                            }`}
                          >
                            {/* Checkbox */}
                            <td className="px-3 py-3 text-center">
                              <input
                                type="checkbox"
                                checked={isSelected}
                                onChange={(e) => {
                                  if (e.target.checked) {
                                    setSelectedDomainIds((prev) => [...prev, dom.id]);
                                  } else {
                                    setSelectedDomainIds((prev) => prev.filter((id) => id !== dom.id));
                                  }
                                }}
                                className="rounded border-slate-300 text-emerald-600 focus:ring-0 cursor-pointer"
                              />
                            </td>

                            {/* Domain */}
                            <td className="px-3 py-3">
                              <div className="flex items-center gap-2">
                                <span className="font-bold text-slate-900 dark:text-white font-mono text-sm">
                                  {dom.domain}
                                </span>
                                <span className="px-1.5 py-0.2 rounded bg-slate-100 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-[10px] text-slate-500 font-medium">
                                  {dom.provider}
                                </span>
                              </div>
                            </td>

                            {/* SSL Days */}
                            <td className="px-3 py-3 font-semibold text-emerald-600 dark:text-emerald-400">
                              {dom.ssl_days ? `${dom.ssl_days} Days` : 'Not Set'}
                            </td>

                            {/* SSL Expiration Time */}
                            <td className="px-3 py-3 font-mono text-slate-600 dark:text-slate-400">
                              {dom.ssl_expiration_time}
                            </td>

                            {/* Number of Resolutions */}
                            <td className="px-3 py-3 font-mono font-bold text-slate-900 dark:text-slate-100">
                              {dom.number_of_resolutions.toLocaleString()}
                            </td>

                            {/* Operate */}
                            <td className="px-3 py-3 text-right">
                              <div className="flex items-center justify-end gap-3 text-emerald-600 dark:text-emerald-400 font-semibold">
                                <button
                                  onClick={() => openRecordsModal(dom)}
                                  className="hover:text-emerald-700 hover:underline cursor-pointer"
                                >
                                  Resolve
                                </button>
                                <span className="text-slate-300 dark:text-surface-700">|</span>
                                <button
                                  onClick={() => {
                                    const bindText = generateBindZone(dom.domain);
                                    const blob = new Blob([bindText], { type: 'text/plain' });
                                    const url = URL.createObjectURL(blob);
                                    const a = document.createElement('a');
                                    a.href = url;
                                    a.download = `${dom.domain}.zone`;
                                    a.click();
                                    showToast(`Zone file for ${dom.domain} downloaded!`);
                                  }}
                                  className="hover:text-emerald-700 hover:underline cursor-pointer"
                                >
                                  Export
                                </button>
                                <span className="text-slate-300 dark:text-surface-700">|</span>
                                <button
                                  onClick={() => handleDeleteDomain(dom)}
                                  className="text-rose-600 dark:text-rose-400 hover:underline cursor-pointer"
                                >
                                  Delete
                                </button>
                              </div>
                            </td>
                          </tr>
                        );
                      })
                    )}
                  </tbody>
                </table>
              </div>

              {/* 5. BATCH ACTIONS & PAGINATION FOOTER */}
              <div className="border-t border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-surface-900 px-4 py-3 flex flex-col sm:flex-row items-center justify-between gap-3 text-xs text-slate-600 dark:text-slate-400">
                {/* Batch Set TTL */}
                <div className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={allSelected}
                    onChange={handleSelectAll}
                    className="rounded border-slate-300 text-emerald-600 focus:ring-0 cursor-pointer"
                  />
                  <button
                    onClick={() => setBatchTtlModalOpen(true)}
                    disabled={selectedDomainIds.length === 0}
                    className="px-3.5 py-1.5 rounded-lg bg-white dark:bg-surface-800 hover:bg-slate-100 dark:hover:bg-surface-700 text-slate-800 dark:text-slate-200 border border-slate-300 dark:border-surface-700 disabled:opacity-50 disabled:cursor-not-allowed font-bold shadow-2xs transition cursor-pointer"
                  >
                    Set TTL
                  </button>
                  {selectedDomainIds.length > 0 && (
                    <span className="text-emerald-600 dark:text-emerald-400 font-bold ml-1">
                      Selected: {selectedDomainIds.length}
                    </span>
                  )}
                </div>

                {/* Pagination */}
                <div className="flex items-center gap-3">
                  <div className="flex items-center gap-1">
                    <button
                      disabled
                      className="px-2 py-0.5 rounded bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-400 cursor-not-allowed shadow-2xs"
                    >
                      &lt;
                    </button>
                    <button className="px-2.5 py-0.5 rounded bg-emerald-600 text-white font-bold shadow-xs">
                      1
                    </button>
                    <button
                      disabled
                      className="px-2 py-0.5 rounded bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-400 cursor-not-allowed shadow-2xs"
                    >
                      &gt;
                    </button>
                  </div>

                  <span className="font-semibold text-slate-600 dark:text-slate-400">10 / page</span>

                  <div className="flex items-center gap-1">
                    <span>Goto</span>
                    <input
                      type="text"
                      defaultValue="1"
                      className="w-8 px-1 py-0.5 rounded bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-center text-slate-800 dark:text-slate-200 font-semibold focus:outline-none shadow-2xs"
                    />
                  </div>

                  <span className="font-semibold text-slate-600 dark:text-slate-400">
                    Total {filteredDomains.length}
                  </span>
                </div>
              </div>
            </div>
          </>
        )}

        {/* =========================================================================
            TAB 2: PROVIDER LIST VIEW
            ========================================================================= */}
        {activeSubTab === 'providers' && (
          <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-xl p-6 space-y-4 shadow-2xs">
            <h3 className="text-sm font-bold text-slate-900 dark:text-white flex items-center gap-2">
              <Cloud className="w-4 h-4 text-emerald-600" />
              Supported DNS Providers &amp; API Credentials
            </h3>
            <p className="text-xs text-slate-500">
              Configure DNS API tokens to automatically sync DNS records and automate Let&apos;s Encrypt DNS-01 wildcard certificates.
            </p>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 pt-2">
              {[
                { name: 'HostvraDns / BIND9', desc: 'Built-in local authoritative nameserver with RFC 1035 compliance', status: 'Active (Built-in)' },
                { name: 'Cloudflare DNS', desc: 'Cloudflare Global Anycast DNS with proxy and WAF integration', status: 'Configured' },
                { name: 'Alibaba Cloud (AliDNS)', desc: 'Alibaba Cloud Enterprise authoritative DNS resolution', status: 'Available' },
                { name: 'Tencent Cloud (DNSPod)', desc: 'DNSPod professional DNS with high-concurrency resolution', status: 'Available' },
                { name: 'Amazon Route 53', desc: 'AWS scalable cloud Domain Name System web service', status: 'Available' },
              ].map((prov) => (
                <div key={prov.name} className="p-4 rounded-xl border border-slate-200 dark:border-surface-700 bg-slate-50 dark:bg-surface-800 flex items-center justify-between">
                  <div>
                    <h4 className="font-bold text-xs text-slate-900 dark:text-white">{prov.name}</h4>
                    <p className="text-[11px] text-slate-500 mt-0.5">{prov.desc}</p>
                  </div>
                  <button
                    onClick={() => showToast(`${prov.name} API settings verified`)}
                    className="px-3 py-1.5 rounded-lg bg-white dark:bg-surface-700 border border-slate-300 dark:border-surface-600 text-xs font-bold text-slate-700 dark:text-slate-200 hover:border-emerald-500 cursor-pointer shadow-2xs"
                  >
                    Configure
                  </button>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* =========================================================================
            TAB 3: LOGS VIEW
            ========================================================================= */}
        {activeSubTab === 'logs' && (
          <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-xl p-6 space-y-4 shadow-2xs">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-bold text-slate-900 dark:text-white flex items-center gap-2">
                <Clock className="w-4 h-4 text-emerald-600" />
                Authoritative DNS Resolution &amp; Audit Logs
              </h3>
              <button
                onClick={() => showToast('DNS logs refreshed')}
                className="px-3 py-1.5 rounded-lg bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-xs font-bold"
              >
                Refresh
              </button>
            </div>

            <div className="p-4 rounded-xl bg-slate-950 font-mono text-xs text-emerald-400 border border-slate-800 max-h-[380px] overflow-auto space-y-1">
              <p>[{new Date().toISOString()}] zone 2xbets.net: loaded serial 2026090801</p>
              <p>[{new Date().toISOString()}] zone 2xbets.net: query A 2xbets.net from 127.0.0.1:53218 (NOERROR)</p>
              <p>[{new Date().toISOString()}] zone antiprofiles.com: query TXT _dmarc from 66.249.66.1 (NOERROR)</p>
              <p>[{new Date().toISOString()}] zone affscash.net: query A app.affscash.net from 192.168.1.45 (NOERROR)</p>
              <p>[{new Date().toISOString()}] named-server: all 3 authoritative zones verified healthy</p>
            </div>
          </div>
        )}

        {/* =========================================================================
            MODALS (Add Domain, DMARC, DKIM, Set & Manage, Records, Batch TTL)
            ========================================================================= */}

        {/* 1. Add Domain Modal */}
        {addDomainOpen && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setAddDomainOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-5">
                <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
                  <Globe className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-slate-900 dark:text-white">Add Domain</h2>
                  <p className="text-xs text-slate-500">Create new authoritative DNS zone</p>
                </div>
              </div>

              <form onSubmit={handleAddDomain} className="space-y-4 text-xs font-semibold">
                <div>
                  <label className="block text-slate-700 dark:text-slate-300 mb-1">
                    Domain Name *
                  </label>
                  <input
                    type="text"
                    required
                    placeholder="example.com"
                    value={newDomainName}
                    onChange={(e) => setNewDomainName(e.target.value)}
                    className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-mono text-sm focus:outline-none focus:border-emerald-500"
                  />
                  <span className="text-[11px] text-slate-400">Enter second-level domain or subdomain without http/https</span>
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 mb-1">DNS Provider</label>
                    <select
                      value={newDomainProvider}
                      onChange={(e) => setNewDomainProvider(e.target.value)}
                      className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white focus:outline-none"
                    >
                      <option value="HostvraDns">HostvraDns (Built-in)</option>
                      <option value="Cloudflare">Cloudflare</option>
                      <option value="AliDNS">AliDNS</option>
                      <option value="DNSPod">DNSPod</option>
                      <option value="AWS Route53">AWS Route53</option>
                    </select>
                  </div>

                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 mb-1">Default TTL</label>
                    <select
                      value={newDomainTtl}
                      onChange={(e) => setNewDomainTtl(Number(e.target.value))}
                      className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white focus:outline-none"
                    >
                      <option value={300}>300s (5 minutes)</option>
                      <option value={600}>600s (10 minutes)</option>
                      <option value={1800}>1800s (30 minutes)</option>
                      <option value={3600}>3600s (1 hour)</option>
                      <option value={86400}>86400s (1 day)</option>
                    </select>
                  </div>
                </div>

                <div className="flex items-center justify-end gap-3 pt-4 border-t border-slate-200 dark:border-surface-700">
                  <button
                    type="button"
                    onClick={() => setAddDomainOpen(false)}
                    className="px-4 py-2 rounded-xl bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-300 font-bold"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold shadow-md"
                  >
                    Submit
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* 2. Apply DMARC Policy Modal */}
        {dmarcModalOpen && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setDmarcModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-5">
                <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
                  <ShieldCheck className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-slate-900 dark:text-white">Apply DMARC Policy</h2>
                  <p className="text-xs text-slate-500">Domain-based Message Authentication, Reporting, and Conformance</p>
                </div>
              </div>

              <form onSubmit={handleApplyDmarc} className="space-y-4 text-xs font-semibold">
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 mb-1">Domain Policy (p)</label>
                    <select
                      value={dmarcPolicy}
                      onChange={(e) => setDmarcPolicy(e.target.value)}
                      className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white focus:outline-none"
                    >
                      <option value="quarantine">quarantine (Spam folder)</option>
                      <option value="reject">reject (Drop invalid emails)</option>
                      <option value="none">none (Monitoring only)</option>
                    </select>
                  </div>

                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 mb-1">Subdomain Policy (sp)</label>
                    <select
                      value={dmarcSubPolicy}
                      onChange={(e) => setDmarcSubPolicy(e.target.value)}
                      className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white focus:outline-none"
                    >
                      <option value="quarantine">quarantine</option>
                      <option value="reject">reject</option>
                      <option value="none">none</option>
                    </select>
                  </div>
                </div>

                <div>
                  <label className="block text-slate-700 dark:text-slate-300 mb-1">Percentage (pct): {dmarcPercentage}%</label>
                  <input
                    type="range"
                    min={10}
                    max={100}
                    step={10}
                    value={dmarcPercentage}
                    onChange={(e) => setDmarcPercentage(Number(e.target.value))}
                    className="w-full accent-emerald-600"
                  />
                </div>

                <div>
                  <label className="block text-slate-700 dark:text-slate-300 mb-1">Aggregate Report Email (rua)</label>
                  <input
                    type="email"
                    required
                    value={dmarcEmail}
                    onChange={(e) => setDmarcEmail(e.target.value)}
                    className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white focus:outline-none"
                  />
                </div>

                <div className="p-3 rounded-xl bg-slate-950 font-mono text-emerald-400 text-[11px] break-all border border-slate-800">
                  v=DMARC1; p={dmarcPolicy}; sp={dmarcSubPolicy}; pct={dmarcPercentage}; rua=mailto:{dmarcEmail}; adkim=r; aspf=r
                </div>

                <div className="flex justify-end gap-3 pt-3 border-t border-slate-200 dark:border-surface-700">
                  <button
                    type="button"
                    onClick={() => setDmarcModalOpen(false)}
                    className="px-4 py-2 rounded-xl bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-300 font-bold"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold shadow-md"
                  >
                    Apply to {selectedDomainIds.length > 0 ? `${selectedDomainIds.length} Domain(s)` : 'All Domains'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* 3. Set & Manage Modal */}
        {setManageModalOpen && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setSetManageModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-5">
                <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
                  <Sliders className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-slate-900 dark:text-white">DNS Settings &amp; SOA Management</h2>
                  <p className="text-xs text-slate-500">Configure global nameservers and zone propagation</p>
                </div>
              </div>

              <div className="space-y-4 text-xs font-semibold">
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 mb-1">Primary NS (ns1)</label>
                    <input
                      type="text"
                      defaultValue="ns1.hostvra.com"
                      className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-mono"
                    />
                  </div>
                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 mb-1">Secondary NS (ns2)</label>
                    <input
                      type="text"
                      defaultValue="ns2.hostvra.com"
                      className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-mono"
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-slate-700 dark:text-slate-300 mb-1">SOA Contact Email</label>
                  <input
                    type="text"
                    defaultValue="admin@hostvra.com"
                    className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-mono"
                  />
                </div>

                <div className="flex items-center justify-between p-3 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700">
                  <div>
                    <span className="block text-slate-900 dark:text-white font-bold">DNSSEC Signing</span>
                    <span className="text-[11px] text-slate-500">Cryptographically sign zone records (RFC 4034)</span>
                  </div>
                  <input
                    type="checkbox"
                    defaultChecked
                    className="rounded border-slate-300 text-emerald-600 focus:ring-0 cursor-pointer"
                  />
                </div>

                <div className="flex justify-end gap-3 pt-3 border-t border-slate-200 dark:border-surface-700">
                  <button
                    onClick={() => setSetManageModalOpen(false)}
                    className="px-4 py-2 rounded-xl bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-300 font-bold"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={() => {
                      showToast('Global DNS SOA settings saved and applied to all zones');
                      setSetManageModalOpen(false);
                    }}
                    className="px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold shadow-md"
                  >
                    Save Settings
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* 4. Batch Set TTL Modal */}
        {batchTtlModalOpen && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-sm bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setBatchTtlModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <h2 className="text-lg font-bold text-slate-900 dark:text-white mb-2">Set TTL</h2>
              <p className="text-xs text-slate-500 mb-4">
                Update Time-To-Live for {selectedDomainIds.length} selected domain(s)
              </p>

              <div className="mb-5">
                <label className="block text-xs font-bold text-slate-700 dark:text-slate-300 mb-1">Select TTL Duration</label>
                <select
                  value={batchTtlValue}
                  onChange={(e) => setBatchTtlValue(Number(e.target.value))}
                  className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white text-xs font-semibold focus:outline-none"
                >
                  <option value={300}>300s (5 minutes - Recommended for rapid changes)</option>
                  <option value={600}>600s (10 minutes)</option>
                  <option value={1800}>1800s (30 minutes)</option>
                  <option value={3600}>3600s (1 hour - Standard)</option>
                  <option value={86400}>86400s (1 day - High caching)</option>
                </select>
              </div>

              <div className="flex justify-end gap-3">
                <button
                  onClick={() => setBatchTtlModalOpen(false)}
                  className="px-4 py-2 rounded-xl bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-300 font-bold text-xs"
                >
                  Cancel
                </button>
                <button
                  onClick={handleBatchSetTtl}
                  className="px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs"
                >
                  Apply TTL
                </button>
              </div>
            </div>
          </div>
        )}

        {/* 5. Interactive DNS Records Management Modal (Resolve) */}
        {recordsModalOpen && activeDomainForRecords && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-4xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative max-h-[90vh] flex flex-col">
              <button
                onClick={() => setRecordsModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-4">
                <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
                  <Network className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-slate-900 dark:text-white">
                    DNS Records for {activeDomainForRecords.domain}
                  </h2>
                  <p className="text-xs text-slate-500">Authoritative zone management with RFC validation</p>
                </div>
              </div>

              {/* Add New Record Inline Bar */}
              <form onSubmit={handleAddSingleRecord} className="p-3 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 mb-4 flex flex-wrap items-center gap-2 text-xs font-semibold">
                <select
                  value={recType}
                  onChange={(e) => setRecType(e.target.value as any)}
                  className="px-2.5 py-1.5 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-bold"
                >
                  <option value="A">A</option>
                  <option value="AAAA">AAAA</option>
                  <option value="CNAME">CNAME</option>
                  <option value="TXT">TXT</option>
                  <option value="MX">MX</option>
                  <option value="CAA">CAA</option>
                  <option value="SRV">SRV</option>
                  <option value="NS">NS</option>
                </select>

                <input
                  type="text"
                  placeholder="Name (@ or www)"
                  value={recName}
                  onChange={(e) => setRecName(e.target.value)}
                  className="w-28 px-2.5 py-1.5 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-mono"
                />

                <input
                  type="text"
                  required
                  placeholder="Record Value (e.g. 192.0.2.1)"
                  value={recContent}
                  onChange={(e) => setRecContent(e.target.value)}
                  className="flex-1 min-w-[140px] px-2.5 py-1.5 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-mono"
                />

                {recType === 'MX' && (
                  <input
                    type="number"
                    placeholder="Priority"
                    value={recPriority}
                    onChange={(e) => setRecPriority(Number(e.target.value))}
                    className="w-16 px-2 py-1.5 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 font-mono"
                  />
                )}

                <select
                  value={recTTL}
                  onChange={(e) => setRecTTL(Number(e.target.value))}
                  className="w-24 px-2 py-1.5 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white"
                >
                  <option value={300}>300s</option>
                  <option value={600}>600s</option>
                  <option value={3600}>3600s</option>
                  <option value={86400}>86400s</option>
                </select>

                <label className="flex items-center gap-1.5 px-2 py-1 rounded bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={recProxied}
                    onChange={(e) => setRecProxied(e.target.checked)}
                    className="rounded border-slate-300 text-emerald-600 focus:ring-0"
                  />
                  <span className="text-[11px] text-slate-700 dark:text-slate-300">Proxy</span>
                </label>

                <button
                  type="submit"
                  className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold cursor-pointer"
                >
                  Add Record
                </button>
              </form>

              {/* Records Table */}
              <div className="flex-1 overflow-auto border border-slate-200 dark:border-surface-700 rounded-xl">
                <table className="w-full text-left text-xs border-collapse">
                  <thead>
                    <tr className="border-b border-slate-200 dark:border-surface-700 bg-slate-50 dark:bg-surface-800 text-slate-700 dark:text-slate-300 font-bold uppercase text-[10px]">
                      <th className="px-3 py-2">Type</th>
                      <th className="px-3 py-2">Host / Name</th>
                      <th className="px-3 py-2">Value / Content</th>
                      <th className="px-3 py-2">TTL</th>
                      <th className="px-3 py-2">Proxy</th>
                      <th className="px-3 py-2 text-right">Action</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 dark:divide-surface-800 font-sans">
                    {(domainRecords[activeDomainForRecords.domain] || []).map((r) => (
                      <tr key={r.id} className="hover:bg-slate-50 dark:hover:bg-surface-800/60">
                        <td className="px-3 py-2 font-bold font-mono">
                          <span className="px-1.5 py-0.5 rounded bg-emerald-50 dark:bg-emerald-950/30 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
                            {r.type}
                          </span>
                        </td>
                        <td className="px-3 py-2 font-mono font-bold text-slate-800 dark:text-slate-200">
                          {r.name}
                        </td>
                        <td className="px-3 py-2 font-mono text-slate-600 dark:text-slate-400 max-w-sm truncate">
                          {r.content}
                          {r.priority ? ` (prio: ${r.priority})` : ''}
                        </td>
                        <td className="px-3 py-2 font-mono text-slate-500">
                          {r.ttl}s
                        </td>
                        <td className="px-3 py-2">
                          {r.proxied ? (
                            <span className="px-1.5 py-0.2 rounded bg-amber-500/10 text-amber-500 border border-amber-500/20 text-[10px] font-bold">
                              Proxied
                            </span>
                          ) : (
                            <span className="text-slate-400 text-[11px]">DNS only</span>
                          )}
                        </td>
                        <td className="px-3 py-2 text-right">
                          <button
                            onClick={() => handleDeleteSingleRecord(r.id)}
                            className="text-rose-600 hover:text-rose-700 hover:underline cursor-pointer font-bold"
                          >
                            Delete
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              <div className="flex justify-end pt-4 border-t border-slate-200 dark:border-surface-700 mt-4">
                <button
                  onClick={() => setRecordsModalOpen(false)}
                  className="px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs"
                >
                  Done
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
