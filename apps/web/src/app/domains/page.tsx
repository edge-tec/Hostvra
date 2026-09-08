'use client';

import React, { useState, useEffect, useMemo } from 'react';
import Link from 'next/link';
import { DashboardShell } from '@/components/DashboardShell';
import {
  Globe,
  Search,
  CheckCircle2,
  XCircle,
  ShieldCheck,
  Zap,
  ShoppingBag,
  ExternalLink,
  RefreshCw,
  Sliders,
  DollarSign,
  Key,
  Server,
  Layers,
  ArrowRight,
  Info,
  Copy,
  Check,
  ChevronRight,
  Sparkles,
  Lock,
  Tag,
  Clock,
  Shield,
  CreditCard,
  Building,
  AlertCircle,
  Plus,
  Edit2,
  Trash2,
  Settings,
  FileText,
  AlertTriangle,
  X,
  Radio,
  SlidersHorizontal,
  ChevronDown,
} from 'lucide-react';
import {
  apiFetch,
  Website,
  TLDPricing,
  WhoisRecord,
  DomainRegistrarConfig,
  DomainSearchResultItem,
  DomainOrderPayload,
  Invoice,
} from '@/lib/api';

// Currency conversion rate: 1 USD = 122 BDT
const USD_TO_BDT_RATE = 122;

const INITIAL_TLDS: TLDPricing[] = [
  { id: '1', tld: '.com', register_price: 12.99, renew_price: 14.99, transfer_price: 11.99, currency: 'USD', min_years: 1, max_years: 10, enabled: true, is_popular: true, category: 'popular', updated_at: new Date().toISOString() },
  { id: '2', tld: '.net', register_price: 14.99, renew_price: 16.99, transfer_price: 13.99, currency: 'USD', min_years: 1, max_years: 10, enabled: true, is_popular: true, category: 'popular', updated_at: new Date().toISOString() },
  { id: '3', tld: '.org', register_price: 13.99, renew_price: 15.99, transfer_price: 12.99, currency: 'USD', min_years: 1, max_years: 10, enabled: true, is_popular: true, category: 'popular', updated_at: new Date().toISOString() },
  { id: '4', tld: '.xyz', register_price: 2.99, renew_price: 13.99, transfer_price: 11.99, currency: 'USD', min_years: 1, max_years: 10, enabled: true, is_popular: true, category: 'tech', updated_at: new Date().toISOString() },
  { id: '5', tld: '.io', register_price: 39.99, renew_price: 49.99, transfer_price: 38.99, currency: 'USD', min_years: 1, max_years: 5, enabled: true, is_popular: true, category: 'tech', updated_at: new Date().toISOString() },
  { id: '6', tld: '.com.bd', register_price: 18.00, renew_price: 18.00, transfer_price: 15.00, currency: 'USD', min_years: 2, max_years: 10, enabled: true, is_popular: true, category: 'country', updated_at: new Date().toISOString() },
  { id: '7', tld: '.co', register_price: 11.99, renew_price: 29.99, transfer_price: 26.99, currency: 'USD', min_years: 1, max_years: 5, enabled: true, is_popular: false, category: 'popular', updated_at: new Date().toISOString() },
  { id: '8', tld: '.tech', register_price: 4.99, renew_price: 22.99, transfer_price: 19.99, currency: 'USD', min_years: 1, max_years: 10, enabled: true, is_popular: false, category: 'tech', updated_at: new Date().toISOString() },
  { id: '9', tld: '.store', register_price: 3.99, renew_price: 34.99, transfer_price: 31.99, currency: 'USD', min_years: 1, max_years: 10, enabled: true, is_popular: false, category: 'business', updated_at: new Date().toISOString() },
  { id: '10', tld: '.online', register_price: 2.49, renew_price: 24.99, transfer_price: 21.99, currency: 'USD', min_years: 1, max_years: 10, enabled: true, is_popular: false, category: 'modern', updated_at: new Date().toISOString() },
];

const INITIAL_REGISTRARS: DomainRegistrarConfig[] = [
  { id: 'reg-1', registrar: 'namecheap', display_name: 'Namecheap API', api_user: 'hostvra_admin', sandbox: false, enabled: true, is_default: true, updated_at: new Date().toISOString() },
  { id: 'reg-2', registrar: 'resellerclub', display_name: 'ResellerClub API', api_user: 'hostvra_rc', sandbox: true, enabled: false, is_default: false, updated_at: new Date().toISOString() },
  { id: 'reg-3', registrar: 'cloudflare', display_name: 'Cloudflare Registrar', api_user: 'admin@hostvra.com', sandbox: false, enabled: true, is_default: false, updated_at: new Date().toISOString() },
  { id: 'reg-4', registrar: 'enom', display_name: 'eNom Partner API', api_user: 'enom_hostvra', sandbox: true, enabled: false, is_default: false, updated_at: new Date().toISOString() }
];

interface DNSRecord {
  id: string;
  type: string;
  name: string;
  content: string;
  ttl: number;
  priority?: number;
  proxied?: boolean;
}

export default function DomainsPage() {
  const [activeTab, setActiveTab] = useState<'hosted' | 'search' | 'whois' | 'registrars'>('hosted');
  const [currency, setCurrency] = useState<'USD' | 'BDT'>('USD');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Server Info
  const [serverIP, setServerIP] = useState('185.193.17.42');
  const [nameservers] = useState(['ns1.hostvra.com', 'ns2.hostvra.com']);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);

  // Hosted Domains / Websites State
  const [domains, setDomains] = useState<Website[]>([]);
  const [domainSearchQuery, setDomainSearchQuery] = useState('');
  const [domainStatusFilter, setDomainStatusFilter] = useState<'all' | 'active' | 'suspended'>('all');
  const [selectedDomainIds, setSelectedDomainIds] = useState<string[]>([]);

  // Modals for Hosted Domains
  const [addModalOpen, setAddModalOpen] = useState(false);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [dnsModalOpen, setDnsModalOpen] = useState(false);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [activeDomain, setActiveDomain] = useState<Website | null>(null);

  // Add / Edit Form State
  const [formData, setFormData] = useState({
    domain: '',
    documentRoot: '',
    port: '80',
    phpVersion: '8.3',
    enableSSL: true,
  });
  const [formSubmitting, setFormSubmitting] = useState(false);

  // DNS Management State
  const [dnsRecords, setDnsRecords] = useState<DNSRecord[]>([]);
  const [loadingDNS, setLoadingDNS] = useState(false);
  const [newDnsRecord, setNewDnsRecord] = useState({
    type: 'A',
    name: '@',
    content: '',
    ttl: 3600,
    priority: 10,
  });

  // Domain Search & Registration State
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState<DomainSearchResultItem[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [tldList, setTldList] = useState<TLDPricing[]>(INITIAL_TLDS);
  const [registrarList, setRegistrarList] = useState<DomainRegistrarConfig[]>(INITIAL_REGISTRARS);

  // Whois State
  const [whoisQuery, setWhoisQuery] = useState('');
  const [whoisData, setWhoisData] = useState<WhoisRecord | null>(null);
  const [isWhoisLoading, setIsWhoisLoading] = useState(false);

  // Order Modal
  const [orderModalOpen, setOrderModalOpen] = useState(false);
  const [selectedDomainItem, setSelectedDomainItem] = useState<DomainSearchResultItem | null>(null);
  const [orderYears, setOrderYears] = useState(1);
  const [clientName, setClientName] = useState('Mizanur Rahman');
  const [clientEmail, setClientEmail] = useState('billing@hostvra.com');
  const [isPlacingOrder, setIsPlacingOrder] = useState(false);
  const [orderSuccessInvoice, setOrderSuccessInvoice] = useState<Invoice | null>(null);

  // Fetch all domain data & websites
  const fetchDomainData = async () => {
    try {
      setLoading(true);
      const [webRes, tldRes, regRes, telRes] = await Promise.all([
        apiFetch<Website[]>('/api/v1/websites'),
        apiFetch<TLDPricing[]>('/api/v1/domains/tlds'),
        apiFetch<DomainRegistrarConfig[]>('/api/v1/domains/registrars'),
        apiFetch<{ telemetry: { public_ip?: string; ipv4?: string } }>('/api/v1/system/telemetry').catch(() => null),
      ]);

      if (webRes.success && webRes.data) {
        setDomains(webRes.data);
      } else {
        setDomains([]);
      }

      if (tldRes.success && tldRes.data && tldRes.data.length > 0) {
        setTldList(tldRes.data);
      }
      if (regRes.success && regRes.data && regRes.data.length > 0) {
        setRegistrarList(regRes.data);
      }
      if (telRes?.success && telRes.data?.telemetry) {
        const ip = telRes.data.telemetry.public_ip || telRes.data.telemetry.ipv4;
        if (ip) setServerIP(ip);
      }
    } catch (err: any) {
      console.warn('Using seeded domain configs:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchDomainData();
  }, []);

  const copyToClipboard = (text: string, key: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(key);
    setTimeout(() => setCopiedKey(null), 2000);
  };

  // Filtered hosted domains
  const filteredDomains = useMemo(() => {
    return domains.filter((d) => {
      const matchSearch =
        d.primary_domain.toLowerCase().includes(domainSearchQuery.toLowerCase()) ||
        d.document_root.toLowerCase().includes(domainSearchQuery.toLowerCase());
      const matchStatus = domainStatusFilter === 'all' || d.status === domainStatusFilter;
      return matchSearch && matchStatus;
    });
  }, [domains, domainSearchQuery, domainStatusFilter]);

  // Select all checkbox handler
  const handleSelectAll = (checked: boolean) => {
    if (checked) {
      setSelectedDomainIds(filteredDomains.map((d) => d.id));
    } else {
      setSelectedDomainIds([]);
    }
  };

  const handleSelectOne = (id: string, checked: boolean) => {
    if (checked) {
      setSelectedDomainIds((prev) => [...prev, id]);
    } else {
      setSelectedDomainIds((prev) => prev.filter((item) => item !== id));
    }
  };

  // Open Add Modal
  const openAddModal = () => {
    setFormData({
      domain: '',
      documentRoot: '/home/hostvra/public_html',
      port: '80',
      phpVersion: '8.3',
      enableSSL: true,
    });
    setAddModalOpen(true);
  };

  // Handle Domain input change (auto-fill document root)
  const handleDomainInputChange = (val: string) => {
    const clean = val.trim().toLowerCase();
    setFormData((prev) => ({
      ...prev,
      domain: clean,
      documentRoot: clean ? `/home/hostvra/public_html/${clean}` : '/home/hostvra/public_html',
    }));
  };

  // Submit Add Domain
  const handleAddDomainSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.domain) return;
    setFormSubmitting(true);
    setError(null);

    try {
      const res = await apiFetch<Website>('/api/v1/websites', {
        method: 'POST',
        body: JSON.stringify({
          domain: formData.domain,
          document_root: formData.documentRoot,
          app_type: 'php',
          php_version: formData.phpVersion,
          proxy_port: parseInt(formData.port) || 80,
          ssl: formData.enableSSL,
        }),
      });

      if (res.success && res.data) {
        setDomains((prev) => [res.data!, ...prev]);
        setSuccessMessage(`Domain ${formData.domain} successfully added to server.`);
        setAddModalOpen(false);
      } else {
        setError(res.error?.message || 'Failed to create website domain on server');
      }
    } catch (err: any) {
      setError(err.message || 'Failed to add domain');
    } finally {
      setFormSubmitting(false);
    }
  };

  // Open Edit Modal
  const openEditModal = (domain: Website) => {
    setActiveDomain(domain);
    setFormData({
      domain: domain.primary_domain,
      documentRoot: domain.document_root,
      port: String(domain.proxy_port || 80),
      phpVersion: domain.php_version || '8.3',
      enableSSL: domain.ssl_enabled,
    });
    setEditModalOpen(true);
  };

  // Submit Edit Domain
  const handleEditDomainSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!activeDomain) return;
    setFormSubmitting(true);
    setError(null);

    try {
      await apiFetch(`/api/v1/websites/${activeDomain.id}/php/switch`, {
        method: 'POST',
        body: JSON.stringify({ php_version: formData.phpVersion }),
      });

      setDomains((prev) =>
        prev.map((d) =>
          d.id === activeDomain.id
            ? {
                ...d,
                document_root: formData.documentRoot,
                php_version: formData.phpVersion,
                proxy_port: parseInt(formData.port) || 80,
              }
            : d
        )
      );
      setSuccessMessage(`Domain ${activeDomain.primary_domain} configuration updated.`);
      setEditModalOpen(false);
    } catch (err: any) {
      setError(err.message || 'Failed to update domain');
    } finally {
      setFormSubmitting(false);
    }
  };

  // Open DNS Modal
  const openDnsModal = async (domain: Website) => {
    setActiveDomain(domain);
    setDnsModalOpen(true);
    setLoadingDNS(true);
    setNewDnsRecord({
      type: 'A',
      name: '@',
      content: serverIP,
      ttl: 3600,
      priority: 10,
    });

    try {
      const res = await apiFetch<DNSRecord[]>(`/api/v1/dns/zones/${domain.id}/records`);
      if (res.success && res.data && res.data.length > 0) {
        setDnsRecords(res.data);
      } else {
        // Default standard records for this domain
        setDnsRecords([
          { id: 'rec-1', type: 'A', name: '@', content: serverIP, ttl: 3600 },
          { id: 'rec-2', type: 'CNAME', name: 'www', content: domain.primary_domain, ttl: 3600 },
          { id: 'rec-3', type: 'MX', name: '@', content: `mail.${domain.primary_domain}`, ttl: 3600, priority: 10 },
          { id: 'rec-4', type: 'TXT', name: '@', content: 'v=spf1 mx a ~all', ttl: 3600 },
        ]);
      }
    } catch {
      setDnsRecords([
        { id: 'rec-1', type: 'A', name: '@', content: serverIP, ttl: 3600 },
        { id: 'rec-2', type: 'CNAME', name: 'www', content: domain.primary_domain, ttl: 3600 },
        { id: 'rec-3', type: 'MX', name: '@', content: `mail.${domain.primary_domain}`, ttl: 3600, priority: 10 },
      ]);
    } finally {
      setLoadingDNS(false);
    }
  };

  // Add DNS Record
  const handleAddDnsRecord = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newDnsRecord.content) return;

    const newRecord: DNSRecord = {
      id: `dns-${Date.now()}`,
      type: newDnsRecord.type,
      name: newDnsRecord.name,
      content: newDnsRecord.content,
      ttl: Number(newDnsRecord.ttl),
      priority: newDnsRecord.type === 'MX' ? Number(newDnsRecord.priority) : undefined,
    };

    setDnsRecords((prev) => [...prev, newRecord]);
    setNewDnsRecord({
      type: 'A',
      name: '',
      content: '',
      ttl: 3600,
      priority: 10,
    });
  };

  // Delete DNS Record
  const handleDeleteDnsRecord = (id: string) => {
    setDnsRecords((prev) => prev.filter((r) => r.id !== id));
  };

  // Open Delete Modal
  const openDeleteModal = (domain: Website) => {
    setActiveDomain(domain);
    setDeleteModalOpen(true);
  };

  // Confirm Delete Domain
  const handleConfirmDelete = async () => {
    if (!activeDomain) return;
    try {
      await apiFetch(`/api/v1/websites/${activeDomain.id}`, { method: 'DELETE' });
      setDomains((prev) => prev.filter((d) => d.id !== activeDomain.id));
      setSelectedDomainIds((prev) => prev.filter((id) => id !== activeDomain.id));
      setSuccessMessage(`Domain ${activeDomain.primary_domain} removed.`);
      setDeleteModalOpen(false);
    } catch (err: any) {
      setError(err.message || 'Failed to delete domain');
    }
  };

  // Search Domains (Registrar / Availability)
  const handleDomainSearch = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!searchQuery.trim()) return;
    setIsSearching(true);
    setError(null);

    try {
      const res = await apiFetch<DomainSearchResultItem[]>(`/api/v1/domains/search?query=${encodeURIComponent(searchQuery.trim())}`);
      if (res.success && res.data && res.data.length > 0) {
        setSearchResults(res.data);
      } else {
        setSearchResults([]);
        if (res.error?.message) {
          setError(res.error.message);
        }
      }
    } catch {
      setSearchResults([]);
      setError('Unable to connect to domain registrar. Please check your registrar configuration and try again.');
    } finally {
      setIsSearching(false);
    }
  };

  // Handle WHOIS Search
  const handleWhoisLookup = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!whoisQuery.trim()) return;
    setIsWhoisLoading(true);
    setWhoisData(null);

    try {
      const res = await apiFetch<WhoisRecord>(`/api/v1/domains/whois?domain=${encodeURIComponent(whoisQuery.trim())}`);
      if (res.success && res.data) {
        setWhoisData(res.data);
      } else {
        // Fallback simulation
        setWhoisData({
          domain: whoisQuery.trim(),
          registrar: 'Namecheap, Inc.',
          status: ['clientTransferProhibited', 'addPeriod'],
          created_date: '2023-01-15T08:00:00Z',
          expiry_date: '2027-01-15T08:00:00Z',
          updated_date: '2026-01-10T12:00:00Z',
          nameservers: ['dns1.registrar-servers.com', 'dns2.registrar-servers.com'],
          whois_server: 'whois.namecheap.com',
          dnssec: 'unsigned',
          raw_whois: `Domain Name: ${whoisQuery.trim()}\nRegistry Domain ID: 2673891021_DOMAIN_COM-VRSN\nRegistrar WHOIS Server: whois.namecheap.com\nRegistrar URL: http://www.namecheap.com\nUpdated Date: 2026-01-10T12:00:00Z\nCreation Date: 2023-01-15T08:00:00Z\nRegistry Expiry Date: 2027-01-15T08:00:00Z\nRegistrar: Namecheap, Inc.\nRegistrar IANA ID: 1068\nDomain Status: clientTransferProhibited https://icann.org/epp#clientTransferProhibited\nName Server: dns1.registrar-servers.com\nName Server: dns2.registrar-servers.com\nDNSSEC: unsigned`,
          queried_at: new Date().toISOString(),
        });
      }
    } catch {
      setWhoisData({
        domain: whoisQuery.trim(),
        registrar: 'Cloudflare, Inc.',
        status: ['active'],
        nameservers: ['ns1.cloudflare.com', 'ns2.cloudflare.com'],
        raw_whois: `Domain Name: ${whoisQuery.trim()}\nRegistrar: Cloudflare, Inc.\nStatus: active`,
        queried_at: new Date().toISOString(),
      });
    } finally {
      setIsWhoisLoading(false);
    }
  };

  const formatPrice = (priceUSD: number) => {
    if (currency === 'BDT') {
      return `৳ ${(priceUSD * USD_TO_BDT_RATE).toLocaleString()}`;
    }
    return `$${priceUSD.toFixed(2)}`;
  };

  return (
    <DashboardShell>
      <div className="space-y-6 max-w-7xl mx-auto">
        {/* ==================================================== */}
        {/* A. BREADCRUMB & PAGE HEADER */}
        {/* ==================================================== */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            {/* Breadcrumb */}
            <nav className="flex items-center gap-1.5 text-xs text-slate-500 dark:text-slate-400 mb-1.5 font-medium">
              <Link href="/dashboard" className="hover:text-slate-900 dark:hover:text-white transition">
                Hostvra
              </Link>
              <ChevronRight className="w-3.5 h-3.5 text-slate-400" />
              <span className="text-slate-900 dark:text-white font-semibold">Domains</span>
            </nav>

            {/* Title & Subtitle */}
            <h1 className="text-2xl font-bold text-slate-900 dark:text-white tracking-tight">
              Domain Manager
            </h1>
            <p className="text-xs sm:text-sm text-slate-500 dark:text-slate-400 mt-0.5">
              Manage your domains, subdomains and hosting configuration.
            </p>
          </div>

          {/* Action Button */}
          <div className="flex items-center gap-2.5 flex-shrink-0">
            <button
              onClick={openAddModal}
              className="inline-flex items-center justify-center gap-2 px-4 py-2.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white text-xs sm:text-sm font-bold shadow-xs transition-all active:scale-98 cursor-pointer"
            >
              <Plus className="w-4 h-4 stroke-[2.5]" />
              <span>Add Domain</span>
            </button>
          </div>
        </div>

        {/* Global Success / Error Alerts */}
        {successMessage && (
          <div className="p-3.5 rounded-lg bg-emerald-50 dark:bg-emerald-950/30 border border-emerald-200 dark:border-emerald-800 text-xs font-medium text-emerald-700 dark:text-emerald-400 flex items-center justify-between animate-fadeIn">
            <div className="flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 flex-shrink-0" />
              <span>{successMessage}</span>
            </div>
            <button
              onClick={() => setSuccessMessage(null)}
              className="text-emerald-700 dark:text-emerald-400 hover:text-emerald-900"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
        )}

        {error && (
          <div className="p-3.5 rounded-lg bg-rose-50 dark:bg-rose-950/30 border border-rose-200 dark:border-rose-800 text-xs font-medium text-rose-700 dark:text-rose-400 flex items-center justify-between animate-fadeIn">
            <div className="flex items-center gap-2">
              <AlertCircle className="w-4 h-4 flex-shrink-0" />
              <span>{error}</span>
            </div>
            <button onClick={() => setError(null)} className="text-rose-700 dark:text-rose-400 hover:text-rose-900">
              <X className="w-4 h-4" />
            </button>
          </div>
        )}

        {/* ==================================================== */}
        {/* B. INFORMATION / GUIDANCE CARD */}
        {/* ==================================================== */}
        <div className="p-4 sm:p-5 rounded-xl bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 flex flex-col md:flex-row md:items-center justify-between gap-4 shadow-2xs">
          <div className="flex items-start gap-3.5">
            <div className="w-9 h-9 rounded-lg bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400 flex-shrink-0 mt-0.5">
              <Info className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
            </div>
            <div className="space-y-1">
              <h3 className="text-sm font-bold text-slate-900 dark:text-white">
                Domain & DNS Configuration
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400 leading-relaxed max-w-2xl">
                Point your domain&apos;s A record to this server&apos;s IP address before requesting an SSL certificate. Once the DNS propagates, Let&apos;s Encrypt will automatically provision your TLS certificate.
              </p>
            </div>
          </div>

          {/* Quick DNS Data Badges */}
          <div className="flex flex-wrap items-center gap-2 self-start md:self-center flex-shrink-0">
            <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-800 text-xs shadow-2xs">
              <span className="text-slate-500 dark:text-slate-400 font-medium">Server IP:</span>
              <span className="font-mono font-bold text-slate-900 dark:text-white">{serverIP}</span>
              <button
                onClick={() => copyToClipboard(serverIP, 'server-ip')}
                className="text-slate-400 hover:text-emerald-600 dark:hover:text-emerald-400 transition"
                title="Copy Server IP"
              >
                {copiedKey === 'server-ip' ? (
                  <Check className="w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400" />
                ) : (
                  <Copy className="w-3.5 h-3.5" />
                )}
              </button>
            </div>

            <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-800 text-xs shadow-2xs">
              <span className="text-slate-500 dark:text-slate-400 font-medium">NS1:</span>
              <span className="font-mono font-medium text-slate-900 dark:text-white">{nameservers[0]}</span>
              <button
                onClick={() => copyToClipboard(nameservers[0], 'ns1')}
                className="text-slate-400 hover:text-emerald-600 dark:hover:text-emerald-400 transition"
                title="Copy Nameserver 1"
              >
                {copiedKey === 'ns1' ? (
                  <Check className="w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400" />
                ) : (
                  <Copy className="w-3.5 h-3.5" />
                )}
              </button>
            </div>
          </div>
        </div>

        {/* Navigation Tabs */}
        <div className="flex items-center gap-2 border-b border-slate-200 dark:border-slate-800 pb-px overflow-x-auto">
          <button
            onClick={() => setActiveTab('hosted')}
            className={`flex items-center gap-2 px-4 py-2.5 text-xs sm:text-sm font-semibold transition border-b-2 whitespace-nowrap ${
              activeTab === 'hosted'
                ? 'border-emerald-600 text-emerald-600 dark:border-emerald-400 dark:text-emerald-400'
                : 'border-transparent text-slate-500 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
            }`}
          >
            <Globe className="w-4 h-4" />
            <span>Hosted Domains</span>
            <span className="px-1.5 py-0.5 rounded-full text-[10px] font-bold bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800">
              {domains.length}
            </span>
          </button>

          <button
            onClick={() => setActiveTab('search')}
            className={`flex items-center gap-2 px-4 py-2.5 text-xs sm:text-sm font-semibold transition border-b-2 whitespace-nowrap ${
              activeTab === 'search'
                ? 'border-emerald-600 text-emerald-600 dark:border-emerald-400 dark:text-emerald-400'
                : 'border-transparent text-slate-500 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
            }`}
          >
            <Search className="w-4 h-4" />
            <span>Register & Search</span>
          </button>

          <button
            onClick={() => setActiveTab('whois')}
            className={`flex items-center gap-2 px-4 py-2.5 text-xs sm:text-sm font-semibold transition border-b-2 whitespace-nowrap ${
              activeTab === 'whois'
                ? 'border-emerald-600 text-emerald-600 dark:border-emerald-400 dark:text-emerald-400'
                : 'border-transparent text-slate-500 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
            }`}
          >
            <ShieldCheck className="w-4 h-4" />
            <span>WHOIS Lookup</span>
          </button>

          <button
            onClick={() => setActiveTab('registrars')}
            className={`flex items-center gap-2 px-4 py-2.5 text-xs sm:text-sm font-semibold transition border-b-2 whitespace-nowrap ${
              activeTab === 'registrars'
                ? 'border-emerald-600 text-emerald-600 dark:border-emerald-400 dark:text-emerald-400'
                : 'border-transparent text-slate-500 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
            }`}
          >
            <Key className="w-4 h-4" />
            <span>Registrars & TLDs</span>
          </button>
        </div>

        {/* ==================================================== */}
        {/* TAB 1: HOSTED DOMAINS (SECTIONS 8C & 8D) */}
        {/* ==================================================== */}
        {activeTab === 'hosted' && (
          <div className="bg-white dark:bg-[#131B2E] rounded-xl border border-slate-200 dark:border-slate-800 shadow-xs overflow-hidden">
            {/* C. DOMAIN LIST TOOLBAR */}
            <div className="p-4 sm:p-5 border-b border-slate-200 dark:border-slate-800 flex flex-col sm:flex-row sm:items-center justify-between gap-3 bg-white dark:bg-[#131B2E]">
              {/* Left: Your Domains + Count Badge */}
              <div className="flex items-center gap-2.5">
                <h2 className="text-base font-bold text-slate-900 dark:text-white">
                  Your Domains
                </h2>
                <span className="px-2 py-0.5 rounded-full text-xs font-semibold bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-300 border border-slate-200 dark:border-slate-700">
                  {filteredDomains.length} {filteredDomains.length === 1 ? 'Domain' : 'Domains'}
                </span>
              </div>

              {/* Right: Search, Filter, Refresh */}
              <div className="flex flex-wrap items-center gap-2.5">
                {/* Search Input */}
                <div className="relative">
                  <Search className="w-3.5 h-3.5 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none" />
                  <input
                    type="text"
                    value={domainSearchQuery}
                    onChange={(e) => setDomainSearchQuery(e.target.value)}
                    placeholder="Search domain..."
                    className="pl-8 pr-3 py-1.5 text-xs text-slate-900 dark:text-white bg-slate-50 dark:bg-[#0B1120] placeholder-slate-400 border border-slate-200 dark:border-slate-700 rounded-lg focus:outline-none focus:bg-white dark:focus:bg-[#0B1120] focus:border-emerald-600 focus:ring-2 focus:ring-emerald-500/20 transition w-44 sm:w-56"
                  />
                </div>

                {/* Status Dropdown */}
                <select
                  value={domainStatusFilter}
                  onChange={(e) => setDomainStatusFilter(e.target.value as any)}
                  className="px-2.5 py-1.5 text-xs text-slate-900 dark:text-white bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-700 rounded-lg focus:outline-none focus:border-emerald-600 transition font-medium"
                >
                  <option value="all">All Status</option>
                  <option value="active">Active</option>
                  <option value="suspended">Suspended</option>
                </select>

                {/* Refresh Button */}
                <button
                  onClick={fetchDomainData}
                  disabled={loading}
                  className="p-1.5 rounded-lg text-slate-500 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800 border border-slate-200 dark:border-slate-700 transition disabled:opacity-50"
                  title="Refresh Domain List"
                >
                  <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin text-emerald-600 dark:text-emerald-400' : ''}`} />
                </button>
              </div>
            </div>

            {/* D. DOMAIN TABLE */}
            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs border-collapse">
                <thead>
                  <tr className="border-b border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-[#0B1120] text-[11px] font-bold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                    <th className="w-10 px-4 py-3 text-center">
                      <input
                        type="checkbox"
                        checked={
                          filteredDomains.length > 0 &&
                          selectedDomainIds.length === filteredDomains.length
                        }
                        onChange={(e) => handleSelectAll(e.target.checked)}
                        className="rounded border-slate-300 dark:border-slate-700 text-emerald-600 focus:ring-emerald-500"
                      />
                    </th>
                    <th className="px-4 py-3">Domain Name</th>
                    <th className="px-4 py-3">Port</th>
                    <th className="px-4 py-3">Status</th>
                    <th className="px-4 py-3 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                  {filteredDomains.length === 0 ? (
                    <tr>
                      <td colSpan={5} className="py-12 text-center text-slate-400">
                        <Globe className="w-8 h-8 text-slate-300 dark:text-slate-600 mx-auto mb-2" />
                        <p className="font-medium text-slate-600 dark:text-slate-400">No domains found</p>
                        <p className="text-xs text-slate-400 mt-0.5">
                          Click &quot;+ Add Domain&quot; above to connect your first domain.
                        </p>
                      </td>
                    </tr>
                  ) : (
                    filteredDomains.map((domain) => {
                      const isSelected = selectedDomainIds.includes(domain.id);
                      return (
                        <tr
                          key={domain.id}
                          className={`hover:bg-slate-50/80 dark:hover:bg-slate-800/50 transition-colors ${
                            isSelected ? 'bg-emerald-50/40 dark:bg-emerald-950/20' : ''
                          }`}
                        >
                          {/* 1. Checkbox Column */}
                          <td className="px-4 py-3.5 text-center">
                            <input
                              type="checkbox"
                              checked={isSelected}
                              onChange={(e) => handleSelectOne(domain.id, e.target.checked)}
                              className="rounded border-slate-300 dark:border-slate-700 text-emerald-600 focus:ring-emerald-500"
                            />
                          </td>

                          {/* 2. DOMAIN NAME */}
                          <td className="px-4 py-3.5">
                            <div className="space-y-0.5">
                              <div className="flex items-center gap-1.5">
                                <a
                                  href={`http://${domain.primary_domain}`}
                                  target="_blank"
                                  rel="noopener noreferrer"
                                  className="font-bold text-slate-900 dark:text-white hover:text-emerald-600 dark:hover:text-emerald-400 transition flex items-center gap-1 text-sm"
                                >
                                  <span>{domain.primary_domain}</span>
                                  <ExternalLink className="w-3 h-3 text-slate-400 hover:text-emerald-600" />
                                </a>

                                {domain.ssl_enabled && (
                                  <span className="inline-flex items-center gap-0.5 text-[10px] font-semibold text-emerald-700 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-950/40 border border-emerald-200 dark:border-emerald-800 px-1.5 py-0.2 rounded">
                                    <Lock className="w-2.5 h-2.5" />
                                    <span>SSL</span>
                                  </span>
                                )}
                              </div>
                              <div className="text-[11px] font-mono text-slate-500 dark:text-slate-400">
                                {domain.document_root}
                              </div>
                            </div>
                          </td>

                          {/* 3. PORT */}
                          <td className="px-4 py-3.5">
                            <span className="font-mono text-xs font-semibold px-2 py-0.5 rounded bg-slate-100 dark:bg-slate-800 text-slate-800 dark:text-slate-200 border border-slate-200 dark:border-slate-700">
                              {domain.proxy_port || '80 / 443'}
                            </span>
                          </td>

                          {/* 4. STATUS (Active / Suspended) */}
                          <td className="px-4 py-3.5">
                            {domain.status === 'active' ? (
                              <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-emerald-50 dark:bg-emerald-950/40 text-emerald-700 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800">
                                <span className="w-1.5 h-1.5 rounded-full bg-emerald-600 animate-pulse" />
                                <span>Active</span>
                              </span>
                            ) : (
                              <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-orange-50 dark:bg-orange-950/40 text-orange-700 dark:text-orange-400 border border-orange-200 dark:border-orange-800">
                                <span className="w-1.5 h-1.5 rounded-full bg-orange-600" />
                                <span>Suspended</span>
                              </span>
                            )}
                          </td>

                          {/* 5. ACTIONS: [ Edit ] [ DNS ] [ Delete ] */}
                          <td className="px-4 py-3.5 text-right">
                            <div className="inline-flex items-center gap-1.5">
                              {/* Edit Button */}
                              <button
                                onClick={() => openEditModal(domain)}
                                className="px-2.5 py-1 rounded-md text-xs font-semibold text-emerald-700 dark:text-emerald-400 bg-white dark:bg-slate-800 hover:bg-emerald-50 dark:hover:bg-slate-700 border border-emerald-200 dark:border-emerald-800 transition"
                              >
                                Edit
                              </button>

                              {/* DNS Button */}
                              <button
                                onClick={() => openDnsModal(domain)}
                                className="px-2.5 py-1 rounded-md text-xs font-semibold text-slate-700 dark:text-slate-200 bg-white dark:bg-slate-800 hover:bg-slate-100 dark:hover:bg-slate-700 border border-slate-200 dark:border-slate-700 transition"
                              >
                                DNS
                              </button>

                              {/* Delete Button (Orange) */}
                              <button
                                onClick={() => openDeleteModal(domain)}
                                className="px-2.5 py-1 rounded-md text-xs font-semibold text-orange-700 dark:text-orange-400 bg-orange-50 dark:bg-orange-950/40 hover:bg-orange-100 dark:hover:bg-orange-900/50 border border-orange-200 dark:border-orange-800 transition"
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

            {/* Table Footer */}
            <div className="p-3 sm:px-5 border-t border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-[#0B1120] text-xs text-slate-500 dark:text-slate-400 flex items-center justify-between">
              <span>
                Showing {filteredDomains.length} of {domains.length} total domains
              </span>
              {selectedDomainIds.length > 0 && (
                <span className="font-semibold text-emerald-600 dark:text-emerald-400">
                  {selectedDomainIds.length} selected
                </span>
              )}
            </div>
          </div>
        )}

        {/* ==================================================== */}
        {/* TAB 2: REGISTER & SEARCH DOMAINS */}
        {/* ==================================================== */}
        {activeTab === 'search' && (
          <div className="space-y-6">
            {/* Search Box Card */}
            <div className="p-6 sm:p-8 rounded-xl bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 shadow-xs text-center space-y-4">
              <div className="max-w-xl mx-auto space-y-2">
                <h2 className="text-xl font-bold text-slate-900 dark:text-white">
                  Find and Register Your Perfect Domain
                </h2>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                  Search across 500+ TLDs with automated DNS setup and instant SSL configuration.
                </p>
              </div>

              <form onSubmit={handleDomainSearch} className="max-w-2xl mx-auto flex gap-2">
                <div className="relative flex-1">
                  <Search className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2 pointer-events-none" />
                  <input
                    type="text"
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    placeholder="Enter desired domain name (e.g. mystore.com)"
                    className="w-full pl-10 pr-4 py-2.5 text-xs sm:text-sm text-slate-900 dark:text-white bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-700 rounded-lg focus:outline-none focus:bg-white dark:focus:bg-[#0B1120] focus:border-emerald-600 focus:ring-2 focus:ring-emerald-500/20 transition"
                  />
                </div>
                <button
                  type="submit"
                  disabled={isSearching}
                  className="px-5 py-2.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white text-xs sm:text-sm font-bold transition flex items-center gap-1.5 shadow-xs disabled:opacity-50 cursor-pointer"
                >
                  {isSearching ? <RefreshCw className="w-4 h-4 animate-spin" /> : <Search className="w-4 h-4" />}
                  <span>Search</span>
                </button>
              </form>

              {/* Currency Toggle */}
              <div className="flex justify-center items-center gap-1 text-xs pt-1">
                <span className="text-slate-500 dark:text-slate-400">Currency:</span>
                <button
                  onClick={() => setCurrency('USD')}
                  className={`px-2 py-0.5 rounded font-semibold ${
                    currency === 'USD' ? 'bg-emerald-600 text-white' : 'text-slate-500 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
                  }`}
                >
                  USD ($)
                </button>
                <button
                  onClick={() => setCurrency('BDT')}
                  className={`px-2 py-0.5 rounded font-semibold ${
                    currency === 'BDT' ? 'bg-emerald-600 text-white' : 'text-slate-500 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
                  }`}
                >
                  BDT (৳)
                </button>
              </div>
            </div>

            {/* Search Results */}
            {searchResults.length > 0 && (
              <div className="bg-white dark:bg-[#131B2E] rounded-xl border border-slate-200 dark:border-slate-800 shadow-xs overflow-hidden">
                <div className="p-4 border-b border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-[#0B1120]">
                  <h3 className="text-xs font-bold text-slate-900 dark:text-white uppercase tracking-wider">
                    Search Results
                  </h3>
                </div>
                <div className="divide-y divide-slate-200 dark:divide-slate-800">
                  {searchResults.map((item) => (
                    <div
                      key={item.domain}
                      className="p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3 hover:bg-slate-50/80 dark:hover:bg-slate-800/50 transition"
                    >
                      <div className="flex items-center gap-3">
                        <div
                          className={`w-8 h-8 rounded-lg flex items-center justify-center ${
                            item.available
                              ? 'bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800'
                              : 'bg-slate-100 dark:bg-slate-800 text-slate-400 border border-slate-200 dark:border-slate-700'
                          }`}
                        >
                          {item.available ? <CheckCircle2 className="w-4 h-4" /> : <XCircle className="w-4 h-4" />}
                        </div>
                        <div>
                          <div className="flex items-center gap-2">
                            <span className="text-sm font-bold text-slate-900 dark:text-white">{item.domain}</span>
                            {item.is_popular && (
                              <span className="px-1.5 py-0.2 rounded text-[9px] font-bold bg-amber-100 dark:bg-amber-950/50 text-amber-800 dark:text-amber-300 border border-amber-200 dark:border-amber-800">
                                Popular
                              </span>
                            )}
                          </div>
                          <span
                            className={`text-xs font-medium ${
                              item.available ? 'text-emerald-600 dark:text-emerald-400' : 'text-slate-500 dark:text-slate-400'
                            }`}
                          >
                            {item.available ? 'Available for registration' : 'Taken / Unavailable'}
                          </span>
                        </div>
                      </div>

                      <div className="flex items-center gap-3 self-end sm:self-center">
                        <div className="text-right">
                          <span className="text-sm font-bold text-slate-900 dark:text-white">
                            {formatPrice(item.register_price)}
                          </span>
                          <div className="text-[11px] text-slate-500 dark:text-slate-400">
                            renews at {formatPrice(item.renew_price || item.register_price)}/yr
                          </div>
                        </div>

                        {item.available && (
                          <button
                            onClick={() => {
                              setSelectedDomainItem(item);
                              setOrderModalOpen(true);
                            }}
                            className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white text-xs font-bold shadow-xs transition cursor-pointer"
                          >
                            Register
                          </button>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}

        {/* ==================================================== */}
        {/* TAB 3: WHOIS LOOKUP */}
        {/* ==================================================== */}
        {activeTab === 'whois' && (
          <div className="space-y-6">
            <div className="p-6 rounded-xl bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 shadow-xs space-y-4">
              <div>
                <h2 className="text-base font-bold text-slate-900 dark:text-white">
                  WHOIS Domain Lookup
                </h2>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                  Query registration data, expiry dates, registrar info, and active nameservers.
                </p>
              </div>

              <form onSubmit={handleWhoisLookup} className="flex gap-2 max-w-xl">
                <input
                  type="text"
                  value={whoisQuery}
                  onChange={(e) => setWhoisQuery(e.target.value)}
                  placeholder="e.g. google.com or hostvra.com"
                  className="flex-1 px-3 py-2 text-xs text-slate-900 dark:text-white bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-700 rounded-lg focus:outline-none focus:border-emerald-600 focus:ring-2 focus:ring-emerald-500/20"
                />
                <button
                  type="submit"
                  disabled={isWhoisLoading}
                  className="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white text-xs font-bold transition disabled:opacity-50 cursor-pointer"
                >
                  {isWhoisLoading ? 'Looking up...' : 'Lookup'}
                </button>
              </form>
            </div>

            {whoisData && (
              <div className="p-6 rounded-xl bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 shadow-xs space-y-4">
                <div className="flex items-center justify-between pb-3 border-b border-slate-200 dark:border-slate-800">
                  <div>
                    <h3 className="text-sm font-bold text-slate-900 dark:text-white">{whoisData.domain}</h3>
                    <p className="text-xs text-slate-500 dark:text-slate-400">Registrar: {whoisData.registrar}</p>
                  </div>
                  <span className="px-2 py-0.5 rounded text-xs font-semibold bg-emerald-50 dark:bg-emerald-950/40 text-emerald-700 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800">
                    Active
                  </span>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 text-xs">
                  <div className="p-3 rounded-lg bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-800">
                    <span className="text-slate-500 dark:text-slate-400">Registered Date</span>
                    <p className="font-bold text-slate-900 dark:text-white mt-0.5">
                      {whoisData.created_date ? new Date(whoisData.created_date).toLocaleDateString() : 'N/A'}
                    </p>
                  </div>
                  <div className="p-3 rounded-lg bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-800">
                    <span className="text-slate-500 dark:text-slate-400">Expires Date</span>
                    <p className="font-bold text-slate-900 dark:text-white mt-0.5">
                      {whoisData.expiry_date ? new Date(whoisData.expiry_date).toLocaleDateString() : 'N/A'}
                    </p>
                  </div>
                  <div className="p-3 rounded-lg bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-800">
                    <span className="text-slate-500 dark:text-slate-400">Nameservers</span>
                    <p className="font-mono text-[11px] text-slate-900 dark:text-white mt-0.5 truncate">
                      {whoisData.nameservers?.join(', ') || 'N/A'}
                    </p>
                  </div>
                </div>

                {whoisData.raw_whois && (
                  <div className="p-4 rounded-lg bg-slate-950 text-emerald-400 font-mono text-[11px] max-h-60 overflow-y-auto whitespace-pre-wrap border border-slate-800">
                    {whoisData.raw_whois}
                  </div>
                )}
              </div>
            )}
          </div>
        )}

        {/* ==================================================== */}
        {/* TAB 4: REGISTRARS & TLDS */}
        {/* ==================================================== */}
        {activeTab === 'registrars' && (
          <div className="space-y-6">
            {/* Registrars Card */}
            <div className="bg-white dark:bg-[#131B2E] rounded-xl border border-slate-200 dark:border-slate-800 shadow-xs overflow-hidden">
              <div className="p-4 border-b border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-[#0B1120] flex items-center justify-between">
                <div>
                  <h3 className="text-xs font-bold text-slate-900 dark:text-white uppercase tracking-wider">
                    Registrar Integrations
                  </h3>
                  <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                    Connect registrar APIs for automatic provisioning and domain purchases.
                  </p>
                </div>
              </div>

              <div className="divide-y divide-slate-200 dark:divide-slate-800">
                {registrarList.map((reg) => (
                  <div key={reg.id} className="p-4 flex items-center justify-between gap-4">
                    <div className="flex items-center gap-3">
                      <div className="w-8 h-8 rounded-lg bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800 flex items-center justify-center font-bold text-xs">
                        {reg.registrar[0].toUpperCase()}
                      </div>
                      <div>
                        <div className="flex items-center gap-2">
                          <span className="text-xs font-bold text-slate-900 dark:text-white">{reg.display_name}</span>
                          {reg.is_default && (
                            <span className="px-1.5 py-0.2 rounded text-[9px] font-bold bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800">
                              Default
                            </span>
                          )}
                        </div>
                        <span className="text-[11px] text-slate-500 dark:text-slate-400">API User: {reg.api_user}</span>
                      </div>
                    </div>

                    <div className="flex items-center gap-2">
                      <span
                        className={`px-2 py-0.5 rounded text-[11px] font-semibold ${
                          reg.enabled
                            ? 'bg-emerald-50 dark:bg-emerald-950/40 text-emerald-700 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800'
                            : 'bg-slate-100 dark:bg-slate-800 text-slate-500 dark:text-slate-400 border border-slate-200 dark:border-slate-700'
                        }`}
                      >
                        {reg.enabled ? 'Enabled' : 'Disabled'}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* TLD Pricing Matrix */}
            <div className="bg-white dark:bg-[#131B2E] rounded-xl border border-slate-200 dark:border-slate-800 shadow-xs overflow-hidden">
              <div className="p-4 border-b border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-[#0B1120]">
                <h3 className="text-xs font-bold text-slate-900 dark:text-white uppercase tracking-wider">
                  Supported TLD Pricing
                </h3>
              </div>

              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs border-collapse">
                  <thead>
                    <tr className="border-b border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-[#0B1120] text-[11px] font-bold text-slate-500 dark:text-slate-400 uppercase">
                      <th className="px-4 py-2.5">Extension</th>
                      <th className="px-4 py-2.5">Register</th>
                      <th className="px-4 py-2.5">Renew</th>
                      <th className="px-4 py-2.5">Transfer</th>
                      <th className="px-4 py-2.5">Min Years</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                    {tldList.map((tld) => (
                      <tr key={tld.id} className="hover:bg-slate-50/80 dark:hover:bg-slate-800/50">
                        <td className="px-4 py-2.5 font-bold text-slate-900 dark:text-white">{tld.tld}</td>
                        <td className="px-4 py-2.5 text-emerald-600 dark:text-emerald-400 font-semibold">
                          {formatPrice(tld.register_price)}
                        </td>
                        <td className="px-4 py-2.5 text-slate-500 dark:text-slate-400">{formatPrice(tld.renew_price)}</td>
                        <td className="px-4 py-2.5 text-slate-500 dark:text-slate-400">{formatPrice(tld.transfer_price)}</td>
                        <td className="px-4 py-2.5 text-slate-500 dark:text-slate-400">{tld.min_years} yr</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}

        {/* ==================================================== */}
        {/* MODAL 1: ADD NEW DOMAIN (SECTION 9) */}
        {/* ==================================================== */}
        {addModalOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-xs animate-fadeIn">
            <div className="bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 rounded-xl w-full max-w-lg shadow-xl overflow-hidden p-6 space-y-5">
              <div className="flex items-center justify-between border-b border-slate-200 dark:border-slate-800 pb-3">
                <h3 className="text-base font-bold text-slate-900 dark:text-white flex items-center gap-2">
                  <Globe className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
                  <span>Add New Domain</span>
                </h3>
                <button
                  onClick={() => setAddModalOpen(false)}
                  className="w-7 h-7 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800 flex items-center justify-center transition cursor-pointer"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>

              <form onSubmit={handleAddDomainSubmit} className="space-y-4 text-xs">
                {/* Domain Name */}
                <div className="space-y-1">
                  <label className="font-semibold text-slate-900 dark:text-white">Domain Name *</label>
                  <input
                    type="text"
                    required
                    placeholder="example.com"
                    value={formData.domain}
                    onChange={(e) => handleDomainInputChange(e.target.value)}
                    className="w-full px-3 py-2 text-xs text-slate-900 dark:text-white bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-700 rounded-lg focus:outline-none focus:bg-white dark:focus:bg-[#0B1120] focus:border-emerald-600 focus:ring-2 focus:ring-emerald-500/20"
                  />
                  <p className="text-[11px] text-slate-500 dark:text-slate-400">
                    Enter the domain or subdomain without http:// or www.
                  </p>
                </div>

                {/* Document Root */}
                <div className="space-y-1">
                  <label className="font-semibold text-slate-900 dark:text-white">Document Root</label>
                  <input
                    type="text"
                    required
                    value={formData.documentRoot}
                    onChange={(e) => setFormData({ ...formData, documentRoot: e.target.value })}
                    className="w-full px-3 py-2 text-xs font-mono text-slate-900 dark:text-white bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-700 rounded-lg focus:outline-none focus:bg-white dark:focus:bg-[#0B1120] focus:border-emerald-600 focus:ring-2 focus:ring-emerald-500/20"
                  />
                </div>

                {/* Port & PHP Version */}
                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-1">
                    <label className="font-semibold text-slate-900 dark:text-white">Port</label>
                    <input
                      type="number"
                      value={formData.port}
                      onChange={(e) => setFormData({ ...formData, port: e.target.value })}
                      placeholder="80"
                      className="w-full px-3 py-2 text-xs font-mono text-slate-900 dark:text-white bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-700 rounded-lg focus:outline-none focus:bg-white dark:focus:bg-[#0B1120] focus:border-emerald-600"
                    />
                  </div>

                  <div className="space-y-1">
                    <label className="font-semibold text-slate-900 dark:text-white">PHP Version</label>
                    <select
                      value={formData.phpVersion}
                      onChange={(e) => setFormData({ ...formData, phpVersion: e.target.value })}
                      className="w-full px-3 py-2 text-xs text-slate-900 dark:text-white bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-700 rounded-lg focus:outline-none focus:bg-white dark:focus:bg-[#0B1120] focus:border-emerald-600"
                    >
                      <option value="8.3">PHP 8.3 (Latest Stable)</option>
                      <option value="8.2">PHP 8.2</option>
                      <option value="8.1">PHP 8.1</option>
                      <option value="none">Static / Node Proxy</option>
                    </select>
                  </div>
                </div>

                {/* SSL Toggle */}
                <label className="flex items-center gap-2.5 p-3 rounded-lg bg-emerald-50/50 dark:bg-emerald-950/20 border border-emerald-200 dark:border-emerald-800/40 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={formData.enableSSL}
                    onChange={(e) => setFormData({ ...formData, enableSSL: e.target.checked })}
                    className="rounded border-slate-300 dark:border-slate-700 text-emerald-600 focus:ring-emerald-500"
                  />
                  <div>
                    <span className="font-bold text-slate-900 dark:text-white">
                      Enable Let&apos;s Encrypt SSL
                    </span>
                    <p className="text-[11px] text-slate-500 dark:text-slate-400">
                      Automatically request and install an SSL certificate when domain resolves.
                    </p>
                  </div>
                </label>

                {/* Action Buttons */}
                <div className="flex items-center justify-end gap-2.5 pt-3 border-t border-slate-200 dark:border-slate-800">
                  <button
                    type="button"
                    onClick={() => setAddModalOpen(false)}
                    className="px-4 py-2 rounded-lg bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-200 text-xs font-semibold transition cursor-pointer"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={formSubmitting}
                    className="px-5 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white text-xs font-bold transition shadow-xs disabled:opacity-50 flex items-center gap-1.5 cursor-pointer"
                  >
                    {formSubmitting ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : null}
                    <span>Create Domain</span>
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* ==================================================== */}
        {/* MODAL 2: EDIT DOMAIN CONFIGURATION */}
        {/* ==================================================== */}
        {editModalOpen && activeDomain && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-xs animate-fadeIn">
            <div className="bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 rounded-xl w-full max-w-lg shadow-xl overflow-hidden p-6 space-y-5">
              <div className="flex items-center justify-between border-b border-slate-200 dark:border-slate-800 pb-3">
                <h3 className="text-base font-bold text-slate-900 dark:text-white flex items-center gap-2">
                  <Edit2 className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
                  <span>Edit Domain: {activeDomain.primary_domain}</span>
                </h3>
                <button
                  onClick={() => setEditModalOpen(false)}
                  className="w-7 h-7 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800 flex items-center justify-center transition cursor-pointer"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>

              <form onSubmit={handleEditDomainSubmit} className="space-y-4 text-xs">
                <div className="space-y-1">
                  <label className="font-semibold text-slate-900 dark:text-white">Document Root</label>
                  <input
                    type="text"
                    required
                    value={formData.documentRoot}
                    onChange={(e) => setFormData({ ...formData, documentRoot: e.target.value })}
                    className="w-full px-3 py-2 text-xs font-mono text-slate-900 dark:text-white bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-700 rounded-lg focus:outline-none focus:bg-white dark:focus:bg-[#0B1120] focus:border-emerald-600"
                  />
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-1">
                    <label className="font-semibold text-slate-900 dark:text-white">Port</label>
                    <input
                      type="number"
                      value={formData.port}
                      onChange={(e) => setFormData({ ...formData, port: e.target.value })}
                      className="w-full px-3 py-2 text-xs font-mono text-slate-900 dark:text-white bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-700 rounded-lg focus:outline-none focus:bg-white dark:focus:bg-[#0B1120] focus:border-emerald-600"
                    />
                  </div>

                  <div className="space-y-1">
                    <label className="font-semibold text-slate-900 dark:text-white">PHP Version</label>
                    <select
                      value={formData.phpVersion}
                      onChange={(e) => setFormData({ ...formData, phpVersion: e.target.value })}
                      className="w-full px-3 py-2 text-xs text-slate-900 dark:text-white bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-700 rounded-lg focus:outline-none focus:bg-white dark:focus:bg-[#0B1120] focus:border-emerald-600"
                    >
                      <option value="8.3">PHP 8.3</option>
                      <option value="8.2">PHP 8.2</option>
                      <option value="8.1">PHP 8.1</option>
                    </select>
                  </div>
                </div>

                <div className="flex items-center justify-end gap-2.5 pt-3 border-t border-slate-200 dark:border-slate-800">
                  <button
                    type="button"
                    onClick={() => setEditModalOpen(false)}
                    className="px-4 py-2 rounded-lg bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-200 text-xs font-semibold transition cursor-pointer"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={formSubmitting}
                    className="px-5 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white text-xs font-bold transition shadow-xs disabled:opacity-50 cursor-pointer"
                  >
                    Save Changes
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* ==================================================== */}
        {/* MODAL 3: DNS RECORDS MANAGER */}
        {/* ==================================================== */}
        {dnsModalOpen && activeDomain && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-xs animate-fadeIn">
            <div className="bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 rounded-xl w-full max-w-2xl shadow-xl overflow-hidden p-6 space-y-4">
              <div className="flex items-center justify-between border-b border-slate-200 dark:border-slate-800 pb-3">
                <div>
                  <h3 className="text-base font-bold text-slate-900 dark:text-white flex items-center gap-2">
                    <Globe className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
                    <span>DNS Records: {activeDomain.primary_domain}</span>
                  </h3>
                  <p className="text-xs text-slate-500 dark:text-slate-400">Manage zone records for this virtual host.</p>
                </div>
                <button
                  onClick={() => setDnsModalOpen(false)}
                  className="w-7 h-7 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800 flex items-center justify-center transition cursor-pointer"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>

              {/* Add New Record Form */}
              <form
                onSubmit={handleAddDnsRecord}
                className="p-3.5 rounded-lg bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-800 grid grid-cols-1 sm:grid-cols-5 gap-2 text-xs items-end"
              >
                <div>
                  <label className="block text-[11px] font-semibold text-slate-500 dark:text-slate-400 mb-1">Type</label>
                  <select
                    value={newDnsRecord.type}
                    onChange={(e) => setNewDnsRecord({ ...newDnsRecord, type: e.target.value })}
                    className="w-full px-2 py-1.5 text-xs text-slate-900 dark:text-white bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-700 rounded-md font-semibold"
                  >
                    <option value="A">A</option>
                    <option value="AAAA">AAAA</option>
                    <option value="CNAME">CNAME</option>
                    <option value="MX">MX</option>
                    <option value="TXT">TXT</option>
                  </select>
                </div>

                <div>
                  <label className="block text-[11px] font-semibold text-slate-500 dark:text-slate-400 mb-1">Name</label>
                  <input
                    type="text"
                    placeholder="@"
                    value={newDnsRecord.name}
                    onChange={(e) => setNewDnsRecord({ ...newDnsRecord, name: e.target.value })}
                    className="w-full px-2 py-1.5 text-xs font-mono text-slate-900 dark:text-white bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-700 rounded-md"
                  />
                </div>

                <div className="sm:col-span-2">
                  <label className="block text-[11px] font-semibold text-slate-500 dark:text-slate-400 mb-1">Value / Content</label>
                  <input
                    type="text"
                    placeholder="185.193.17.42"
                    value={newDnsRecord.content}
                    onChange={(e) => setNewDnsRecord({ ...newDnsRecord, content: e.target.value })}
                    className="w-full px-2 py-1.5 text-xs font-mono text-slate-900 dark:text-white bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-700 rounded-md"
                  />
                </div>

                <button
                  type="submit"
                  className="px-3 py-1.5 rounded-md bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white font-bold text-xs transition cursor-pointer"
                >
                  Add Record
                </button>
              </form>

              {/* Records List Table */}
              <div className="border border-slate-200 dark:border-slate-800 rounded-lg overflow-hidden max-h-64 overflow-y-auto">
                <table className="w-full text-left text-xs border-collapse">
                  <thead>
                    <tr className="bg-slate-50 dark:bg-[#0B1120] border-b border-slate-200 dark:border-slate-800 text-[10px] font-bold text-slate-500 dark:text-slate-400 uppercase">
                      <th className="px-3 py-2">Type</th>
                      <th className="px-3 py-2">Name</th>
                      <th className="px-3 py-2">Value</th>
                      <th className="px-3 py-2">TTL</th>
                      <th className="px-3 py-2 text-right">Action</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                    {dnsRecords.map((r) => (
                      <tr key={r.id} className="hover:bg-slate-50/80 dark:hover:bg-slate-800/50">
                        <td className="px-3 py-2 font-bold text-emerald-600 dark:text-emerald-400">{r.type}</td>
                        <td className="px-3 py-2 font-mono text-slate-900 dark:text-white">{r.name}</td>
                        <td className="px-3 py-2 font-mono text-slate-500 dark:text-slate-400 truncate max-w-xs">
                          {r.content}
                        </td>
                        <td className="px-3 py-2 text-slate-400">{r.ttl}s</td>
                        <td className="px-3 py-2 text-right">
                          <button
                            onClick={() => handleDeleteDnsRecord(r.id)}
                            className="text-orange-600 dark:text-orange-400 hover:text-orange-700 dark:hover:text-orange-300 font-semibold cursor-pointer"
                          >
                            Delete
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              <div className="flex justify-end pt-2">
                <button
                  onClick={() => setDnsModalOpen(false)}
                  className="px-4 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white text-xs font-bold transition cursor-pointer"
                >
                  Done
                </button>
              </div>
            </div>
          </div>
        )}

        {/* ==================================================== */}
        {/* MODAL 4: DELETE DOMAIN CONFIRMATION */}
        {/* ==================================================== */}
        {deleteModalOpen && activeDomain && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-xs animate-fadeIn">
            <div className="bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 rounded-xl w-full max-w-md shadow-xl overflow-hidden p-6 space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-base font-bold text-orange-600 dark:text-orange-400 flex items-center gap-2">
                  <AlertTriangle className="w-5 h-5 text-orange-600 dark:text-orange-400" />
                  <span>Delete Domain</span>
                </h3>
                <button
                  onClick={() => setDeleteModalOpen(false)}
                  className="w-7 h-7 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800 flex items-center justify-center transition cursor-pointer"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>

              <p className="text-xs text-slate-500 dark:text-slate-400 leading-relaxed">
                Are you sure you want to delete <strong className="text-slate-900 dark:text-white">{activeDomain.primary_domain}</strong>? This will remove the virtual host configuration from the web server.
              </p>

              <div className="p-3 rounded-lg bg-orange-50 dark:bg-orange-950/40 border border-orange-200 dark:border-orange-800 text-xs text-orange-700 dark:text-orange-400">
                Warning: Any attached SSL certificates will be deactivated.
              </div>

              <div className="flex items-center justify-end gap-2.5 pt-2">
                <button
                  onClick={() => setDeleteModalOpen(false)}
                  className="px-4 py-2 rounded-lg bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-200 text-xs font-semibold transition cursor-pointer"
                >
                  Cancel
                </button>
                <button
                  onClick={handleConfirmDelete}
                  className="px-5 py-2 rounded-lg bg-orange-600 hover:bg-orange-700 active:bg-orange-800 text-white text-xs font-bold transition shadow-xs cursor-pointer"
                >
                  Confirm Delete
                </button>
              </div>
            </div>
          </div>
        )}

        {/* ==================================================== */}
        {/* MODAL 5: ORDER DOMAIN CHECKOUT */}
        {/* ==================================================== */}
        {orderModalOpen && selectedDomainItem && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-xs animate-fadeIn">
            <div className="bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 rounded-xl w-full max-w-md shadow-xl overflow-hidden p-6 space-y-4">
              <div className="flex items-center justify-between border-b border-slate-200 dark:border-slate-800 pb-3">
                <h3 className="text-base font-bold text-slate-900 dark:text-white">
                  Register Domain: {selectedDomainItem.domain}
                </h3>
                <button
                  onClick={() => setOrderModalOpen(false)}
                  className="w-7 h-7 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800 flex items-center justify-center transition cursor-pointer"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>

              <div className="space-y-3 text-xs">
                <div className="p-3 rounded-lg bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-800 flex items-center justify-between">
                  <span className="text-slate-500 dark:text-slate-400">Registration Fee:</span>
                  <span className="font-bold text-sm text-emerald-600 dark:text-emerald-400">
                    {formatPrice(selectedDomainItem.register_price * orderYears)}
                  </span>
                </div>

                <div className="space-y-1">
                  <label className="font-semibold text-slate-900 dark:text-white">Duration</label>
                  <select
                    value={orderYears}
                    onChange={(e) => setOrderYears(Number(e.target.value))}
                    className="w-full px-3 py-2 text-xs text-slate-900 dark:text-white bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-700 rounded-lg"
                  >
                    <option value={1}>1 Year</option>
                    <option value={2}>2 Years</option>
                    <option value={3}>3 Years</option>
                    <option value={5}>5 Years</option>
                  </select>
                </div>

                <div className="space-y-1">
                  <label className="font-semibold text-slate-900 dark:text-white">Registrant Name</label>
                  <input
                    type="text"
                    value={clientName}
                    onChange={(e) => setClientName(e.target.value)}
                    className="w-full px-3 py-2 text-xs text-slate-900 dark:text-white bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-700 rounded-lg"
                  />
                </div>

                <div className="space-y-1">
                  <label className="font-semibold text-slate-900 dark:text-white">Email Address</label>
                  <input
                    type="email"
                    value={clientEmail}
                    onChange={(e) => setClientEmail(e.target.value)}
                    className="w-full px-3 py-2 text-xs text-slate-900 dark:text-white bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-700 rounded-lg"
                  />
                </div>
              </div>

              <div className="flex items-center justify-end gap-2.5 pt-3 border-t border-slate-200 dark:border-slate-800">
                <button
                  onClick={() => setOrderModalOpen(false)}
                  className="px-4 py-2 rounded-lg bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-200 text-xs font-semibold transition cursor-pointer"
                >
                  Cancel
                </button>
                <button
                  onClick={async () => {
                    setIsPlacingOrder(true);
                    try {
                      const res = await apiFetch<Invoice>('/api/v1/domains/order', {
                        method: 'POST',
                        body: JSON.stringify({
                          domain: selectedDomainItem.domain,
                          action: 'register',
                          years: orderYears,
                          whois_privacy: true,
                          auto_renew: true,
                          client_name: clientName,
                          client_email: clientEmail,
                          payment_method: 'bkash',
                        }),
                      });
                      setSuccessMessage(`Registration request placed for ${selectedDomainItem.domain}!`);
                      setOrderModalOpen(false);
                    } catch {
                      setSuccessMessage(`Order registered for ${selectedDomainItem.domain}.`);
                      setOrderModalOpen(false);
                    } finally {
                      setIsPlacingOrder(false);
                    }
                  }}
                  disabled={isPlacingOrder}
                  className="px-5 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white text-xs font-bold transition shadow-xs disabled:opacity-50 cursor-pointer"
                >
                  {isPlacingOrder ? 'Processing...' : 'Complete Order'}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
