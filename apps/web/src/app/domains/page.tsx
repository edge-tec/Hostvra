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
  Unlock,
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
  ChevronDown,
  Eye,
  EyeOff,
  Send,
  HelpCircle,
} from 'lucide-react';
import {
  apiFetch,
  Website,
  TLDPricing,
  WhoisRecord,
  DomainRegistrarConfig,
  DomainSearchResultItem,
  RegisteredDomain,
  DomainOrder,
  DomainContact,
  DomainDnsRecordItem,
  fetchRegisteredDomains,
  searchDomainAvailability,
  fetchTLDPricings,
  orderDomainRegistration,
  initiateDomainTransfer,
  renewDomainSubscription,
  fetchDomainNameservers,
  updateDomainNameservers,
  fetchDomainDnsRecords,
  addDomainDnsRecord,
  deleteDomainDnsRecord,
  fetchDomainRegistrarLock,
  setDomainRegistrarLock,
  fetchDomainEPPCode,
  fetchDomainContacts,
} from '@/lib/api';

const USD_TO_BDT_RATE = 122;

interface DNSRecord {
  id: string;
  type: string;
  name: string;
  content: string;
  ttl: number;
  priority?: number;
}

export default function DomainsPage() {
  const [activeTab, setActiveTab] = useState<'registered' | 'hosted' | 'search' | 'transfer' | 'whois'>('registered');
  const [currency, setCurrency] = useState<'USD' | 'BDT'>('USD');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Server Info & Telemetry
  const [serverIP, setServerIP] = useState('185.193.17.42');
  const [defaultNameservers] = useState(['ns1.hostvra.com', 'ns2.hostvra.com']);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);

  // --------------------------------------------------------------------------
  // 1. Registered Customer Domains (ResellerClub)
  // --------------------------------------------------------------------------
  const [registeredDomains, setRegisteredDomains] = useState<RegisteredDomain[]>([]);
  const [registeredSearchQuery, setRegisteredSearchQuery] = useState('');
  const [registeredStatusFilter, setRegisteredStatusFilter] = useState<string>('all');
  const [manageDomain, setManageDomain] = useState<RegisteredDomain | null>(null);
  const [manageTab, setManageTab] = useState<'nameservers' | 'dns' | 'security' | 'renew' | 'contacts'>('nameservers');

  // Manage Domain State
  const [domainNsList, setDomainNsList] = useState<string[]>([]);
  const [customNsInputs, setCustomNsInputs] = useState<string[]>(['', '']);
  const [savingNs, setSavingNs] = useState(false);

  const [domainDnsList, setDomainDnsList] = useState<DomainDnsRecordItem[]>([]);
  const [loadingDomainDns, setLoadingDomainDns] = useState(false);
  const [newDomainDns, setNewDomainDns] = useState({
    type: 'A',
    name: '@',
    content: '',
    ttl: 3600,
    priority: 10,
  });
  const [addingDomainDns, setAddingDomainDns] = useState(false);

  const [domainLockStatus, setDomainLockStatus] = useState<boolean>(true);
  const [togglingLock, setTogglingLock] = useState(false);
  const [eppCode, setEppCode] = useState<string | null>(null);
  const [fetchingEpp, setFetchingEpp] = useState(false);

  const [domainContacts, setDomainContacts] = useState<{ registrant?: DomainContact; admin?: DomainContact } | null>(null);
  const [loadingContacts, setLoadingContacts] = useState(false);

  const [renewYears, setRenewYears] = useState(1);
  const [renewPaymentMethod, setRenewPaymentMethod] = useState('stripe');
  const [renewingDomain, setRenewingDomain] = useState(false);

  // --------------------------------------------------------------------------
  // 2. Hosted Domains / Websites State (Local Server)
  // --------------------------------------------------------------------------
  const [hostedDomains, setHostedDomains] = useState<Website[]>([]);
  const [hostedSearchQuery, setHostedSearchQuery] = useState('');
  const [hostedStatusFilter, setHostedStatusFilter] = useState<'all' | 'active' | 'suspended'>('all');
  const [selectedHostedIds, setSelectedHostedIds] = useState<string[]>([]);
  const [addHostedModalOpen, setAddHostedModalOpen] = useState(false);
  const [editHostedModalOpen, setEditHostedModalOpen] = useState(false);
  const [dnsHostedModalOpen, setDnsHostedModalOpen] = useState(false);
  const [deleteHostedModalOpen, setDeleteHostedModalOpen] = useState(false);
  const [activeHostedDomain, setActiveHostedDomain] = useState<Website | null>(null);
  const [hostedFormData, setHostedFormData] = useState({
    domain: '',
    documentRoot: '',
    port: '80',
    phpVersion: '8.3',
    enableSSL: true,
  });
  const [hostedFormSubmitting, setHostedFormSubmitting] = useState(false);
  const [hostedDnsRecords, setHostedDnsRecords] = useState<DNSRecord[]>([]);
  const [loadingHostedDns, setLoadingHostedDns] = useState(false);
  const [newHostedDnsRecord, setNewHostedDnsRecord] = useState({
    type: 'A',
    name: '@',
    content: '',
    ttl: 3600,
    priority: 10,
  });

  // --------------------------------------------------------------------------
  // 3. Domain Search & Registration
  // --------------------------------------------------------------------------
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState<DomainSearchResultItem[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [tldList, setTldList] = useState<TLDPricing[]>([]);
  const [orderModalOpen, setOrderModalOpen] = useState(false);
  const [selectedDomainItem, setSelectedDomainItem] = useState<DomainSearchResultItem | null>(null);
  const [orderYears, setOrderYears] = useState(1);
  const [orderPaymentMethod, setOrderPaymentMethod] = useState('stripe');
  const [orderAutoRenew, setOrderAutoRenew] = useState(true);
  const [useDefaultNs, setUseDefaultNs] = useState(true);
  const [orderCustomNs, setOrderCustomNs] = useState(['ns1.hostvra.com', 'ns2.hostvra.com']);
  const [orderRegistrant, setOrderRegistrant] = useState<DomainContact>({
    first_name: 'Account',
    last_name: 'Owner',
    email: 'billing@hostvra.com',
    phone: '15551234567',
    company_name: 'Hostvra Cloud',
    address1: '100 Hostvra Way',
    city: 'Wilmington',
    state: 'DE',
    postal_code: '19801',
    country: 'US',
  });
  const [isPlacingOrder, setIsPlacingOrder] = useState(false);
  const [orderSuccessData, setOrderSuccessData] = useState<{ domain: string; invoice_id?: string; amount: number } | null>(null);

  // --------------------------------------------------------------------------
  // 4. Inbound Transfer
  // --------------------------------------------------------------------------
  const [transferForm, setTransferForm] = useState({
    domain: '',
    auth_code: '',
    years: 1,
    payment_method: 'stripe',
  });
  const [isSubmittingTransfer, setIsSubmittingTransfer] = useState(false);

  // --------------------------------------------------------------------------
  // 5. WHOIS Lookup
  // --------------------------------------------------------------------------
  const [whoisQuery, setWhoisQuery] = useState('');
  const [whoisData, setWhoisData] = useState<WhoisRecord | null>(null);
  const [isWhoisLoading, setIsWhoisLoading] = useState(false);

  // --------------------------------------------------------------------------
  // Data Fetching
  // --------------------------------------------------------------------------
  const fetchAllData = async () => {
    try {
      setLoading(true);
      const [regRes, webRes, tldRes, telRes] = await Promise.all([
        fetchRegisteredDomains().catch(() => null),
        apiFetch<Website[]>('/api/v1/websites').catch(() => null),
        fetchTLDPricings().catch(() => null),
        apiFetch<{ telemetry: { public_ip?: string; ipv4?: string } }>('/api/v1/system/telemetry').catch(() => null),
      ]);

      if (regRes?.success && regRes.data) {
        setRegisteredDomains(regRes.data);
      }
      if (webRes?.success && webRes.data) {
        setHostedDomains(webRes.data);
      }
      if (tldRes?.success && tldRes.data) {
        setTldList(tldRes.data);
      }
      if (telRes?.success && telRes.data?.telemetry) {
        const ip = telRes.data.telemetry.public_ip || telRes.data.telemetry.ipv4;
        if (ip) setServerIP(ip);
      }
    } catch (err: any) {
      console.error('Failed loading domain data:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAllData();
  }, []);

  const copyToClipboard = (text: string, key: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(key);
    setTimeout(() => setCopiedKey(null), 2000);
  };

  const formatPrice = (priceUSD: number) => {
    if (currency === 'BDT') {
      return `৳ ${(priceUSD * USD_TO_BDT_RATE).toLocaleString()}`;
    }
    return `$${priceUSD.toFixed(2)}`;
  };

  // --------------------------------------------------------------------------
  // Manage Domain Actions
  // --------------------------------------------------------------------------
  const openManageModal = async (domain: RegisteredDomain) => {
    setManageDomain(domain);
    setManageTab('nameservers');
    setEppCode(null);
    setDomainLockStatus(domain.is_locked);
    setDomainNsList(domain.nameservers || []);
    setCustomNsInputs(domain.nameservers && domain.nameservers.length > 0 ? domain.nameservers : ['ns1.hostvra.com', 'ns2.hostvra.com']);

    // Fetch nameservers and lock in background
    fetchDomainNameservers(domain.id).then((res) => {
      if (res.success && res.data) {
        setDomainNsList(res.data);
        setCustomNsInputs(res.data);
      }
    });

    fetchDomainRegistrarLock(domain.id).then((res) => {
      if (res.success && res.data) {
        setDomainLockStatus(res.data.locked);
      }
    });
  };

  const handleSaveNameservers = async () => {
    if (!manageDomain) return;
    const cleanNs = customNsInputs.map((s) => s.trim().toLowerCase()).filter(Boolean);
    if (cleanNs.length < 2) {
      setError('Please provide at least 2 valid nameservers');
      return;
    }
    try {
      setSavingNs(true);
      setError(null);
      const res = await updateDomainNameservers(manageDomain.id, cleanNs);
      if (res.success) {
        setDomainNsList(cleanNs);
        setSuccessMessage(`Nameservers updated for ${manageDomain.domain_name}`);
      } else {
        setError(res.error?.message || 'Failed to update nameservers');
      }
    } catch (err: any) {
      setError(err.message || 'Error updating nameservers');
    } finally {
      setSavingNs(false);
    }
  };

  const handleLoadDomainDns = async () => {
    if (!manageDomain) return;
    try {
      setLoadingDomainDns(true);
      setError(null);
      const res = await fetchDomainDnsRecords(manageDomain.id);
      if (res.success && res.data) {
        setDomainDnsList(res.data);
      } else {
        setDomainDnsList([]);
      }
    } catch (err: any) {
      setError(err.message || 'Failed to load DNS records');
    } finally {
      setLoadingDomainDns(false);
    }
  };

  const handleAddDomainDns = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!manageDomain || !newDomainDns.content) return;
    try {
      setAddingDomainDns(true);
      setError(null);
      const res = await addDomainDnsRecord(manageDomain.id, {
        type: newDomainDns.type,
        name: newDomainDns.name,
        content: newDomainDns.content,
        ttl: Number(newDomainDns.ttl),
        priority: newDomainDns.type === 'MX' ? Number(newDomainDns.priority) : undefined,
      });
      if (res.success && res.data) {
        setDomainDnsList((prev) => [...prev, res.data!]);
        setNewDomainDns({ type: 'A', name: '@', content: '', ttl: 3600, priority: 10 });
        setSuccessMessage('DNS record created successfully');
      } else {
        setError(res.error?.message || 'Failed to add DNS record');
      }
    } catch (err: any) {
      setError(err.message || 'Error adding DNS record');
    } finally {
      setAddingDomainDns(false);
    }
  };

  const handleDeleteDomainDns = async (recordId: string) => {
    if (!manageDomain) return;
    try {
      setError(null);
      const res = await deleteDomainDnsRecord(manageDomain.id, recordId);
      if (res.success) {
        setDomainDnsList((prev) => prev.filter((r) => r.id !== recordId));
        setSuccessMessage('DNS record removed');
      } else {
        setError(res.error?.message || 'Failed to delete DNS record');
      }
    } catch (err: any) {
      setError(err.message || 'Error deleting DNS record');
    }
  };

  const handleToggleLock = async () => {
    if (!manageDomain) return;
    try {
      setTogglingLock(true);
      setError(null);
      const nextState = !domainLockStatus;
      const res = await setDomainRegistrarLock(manageDomain.id, nextState);
      if (res.success) {
        setDomainLockStatus(nextState);
        setSuccessMessage(`Registrar Lock ${nextState ? 'enabled' : 'disabled'} for ${manageDomain.domain_name}`);
      } else {
        setError(res.error?.message || 'Failed to toggle registrar lock');
      }
    } catch (err: any) {
      setError(err.message || 'Error setting registrar lock');
    } finally {
      setTogglingLock(false);
    }
  };

  const handleFetchEppCode = async () => {
    if (!manageDomain) return;
    try {
      setFetchingEpp(true);
      setError(null);
      const res = await fetchDomainEPPCode(manageDomain.id);
      if (res.success && res.data) {
        setEppCode(res.data.epp_code);
      } else {
        setError(res.error?.message || 'Failed to retrieve EPP code');
      }
    } catch (err: any) {
      setError(err.message || 'Error retrieving EPP code');
    } finally {
      setFetchingEpp(false);
    }
  };

  const handleLoadContacts = async () => {
    if (!manageDomain) return;
    try {
      setLoadingContacts(true);
      setError(null);
      const res = await fetchDomainContacts(manageDomain.id);
      if (res.success && res.data) {
        setDomainContacts(res.data);
      }
    } catch (err: any) {
      setError(err.message || 'Failed to load contact information');
    } finally {
      setLoadingContacts(false);
    }
  };

  const handleRenewDomain = async () => {
    if (!manageDomain) return;
    try {
      setRenewingDomain(true);
      setError(null);
      const res = await renewDomainSubscription(manageDomain.id, {
        years: renewYears,
        payment_method: renewPaymentMethod,
      });
      if (res.success) {
        setSuccessMessage(`Domain renewal order placed for ${manageDomain.domain_name} (${renewYears} year)`);
        setManageDomain(null);
        fetchAllData();
      } else {
        setError(res.error?.message || 'Failed to place renewal order');
      }
    } catch (err: any) {
      setError(err.message || 'Error processing renewal');
    } finally {
      setRenewingDomain(false);
    }
  };

  // --------------------------------------------------------------------------
  // Domain Search & Registration
  // --------------------------------------------------------------------------
  const handleDomainSearch = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!searchQuery.trim()) return;
    setIsSearching(true);
    setError(null);
    try {
      const res = await searchDomainAvailability(searchQuery.trim());
      if (res.success && res.data && res.data.length > 0) {
        setSearchResults(res.data);
      } else {
        setSearchResults([]);
        if (res.error?.message) setError(res.error.message);
      }
    } catch (err: any) {
      setSearchResults([]);
      setError('Unable to query domain registrar API. Please verify configuration.');
    } finally {
      setIsSearching(false);
    }
  };

  const handleOrderRegistrationSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedDomainItem) return;
    try {
      setIsPlacingOrder(true);
      setError(null);
      const ns = useDefaultNs ? defaultNameservers : orderCustomNs.filter(Boolean);
      const res = await orderDomainRegistration({
        domain: selectedDomainItem.domain,
        years: orderYears,
        nameservers: ns,
        payment_method: orderPaymentMethod,
        auto_renew: orderAutoRenew,
        registrant: orderRegistrant,
      });

      if (res.success && res.data) {
        setOrderSuccessData({
          domain: selectedDomainItem.domain,
          invoice_id: res.data.invoice?.id,
          amount: res.data.amount,
        });
        setSuccessMessage(`Order created for ${selectedDomainItem.domain}. Payment invoice generated.`);
        setOrderModalOpen(false);
        fetchAllData();
      } else {
        setError(res.error?.message || 'Failed to place domain order');
      }
    } catch (err: any) {
      setError(err.message || 'Error submitting registration order');
    } finally {
      setIsPlacingOrder(false);
    }
  };

  // --------------------------------------------------------------------------
  // Inbound Transfer
  // --------------------------------------------------------------------------
  const handleTransferSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!transferForm.domain || !transferForm.auth_code) {
      setError('Please provide both domain name and authorization EPP code');
      return;
    }
    try {
      setIsSubmittingTransfer(true);
      setError(null);
      const res = await initiateDomainTransfer({
        domain: transferForm.domain.trim().toLowerCase(),
        auth_code: transferForm.auth_code.trim(),
        years: transferForm.years,
        payment_method: transferForm.payment_method,
      });
      if (res.success) {
        setSuccessMessage(`Inbound transfer initiated for ${transferForm.domain}. Check billing for invoice.`);
        setTransferForm({ domain: '', auth_code: '', years: 1, payment_method: 'stripe' });
        fetchAllData();
      } else {
        setError(res.error?.message || 'Failed to initiate domain transfer');
      }
    } catch (err: any) {
      setError(err.message || 'Error processing transfer');
    } finally {
      setIsSubmittingTransfer(false);
    }
  };

  // --------------------------------------------------------------------------
  // WHOIS Lookup
  // --------------------------------------------------------------------------
  const handleWhoisLookup = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!whoisQuery.trim()) return;
    setIsWhoisLoading(true);
    setWhoisData(null);
    setError(null);
    try {
      const res = await apiFetch<WhoisRecord>(`/api/v1/domains/whois?domain=${encodeURIComponent(whoisQuery.trim())}`);
      if (res.success && res.data) {
        setWhoisData(res.data);
      } else {
        setError(res.error?.message || 'Domain WHOIS record not found or lookup failed');
      }
    } catch (err: any) {
      setError(err.message || 'WHOIS query failed');
    } finally {
      setIsWhoisLoading(false);
    }
  };

  // Filtered lists
  const filteredRegistered = registeredDomains.filter((d) => {
    const match = d.domain_name.toLowerCase().includes(registeredSearchQuery.toLowerCase());
    const matchStatus = registeredStatusFilter === 'all' || d.status === registeredStatusFilter;
    return match && matchStatus;
  });

  const filteredHosted = hostedDomains.filter((d) => {
    const match = d.primary_domain.toLowerCase().includes(hostedSearchQuery.toLowerCase());
    const matchStatus = hostedStatusFilter === 'all' || d.status === hostedStatusFilter;
    return match && matchStatus;
  });

  return (
    <DashboardShell>
      <div className="space-y-6 max-w-7xl mx-auto">
        {/* Breadcrumb & Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <nav className="flex items-center gap-1.5 text-xs text-slate-500 dark:text-slate-400 mb-1.5 font-medium">
              <Link href="/dashboard" className="hover:text-slate-900 dark:hover:text-white transition">
                Hostvra
              </Link>
              <ChevronRight className="w-3.5 h-3.5 text-slate-400" />
              <span className="text-slate-900 dark:text-white font-semibold">Domains</span>
            </nav>
            <h1 className="text-2xl font-bold text-slate-900 dark:text-white tracking-tight flex items-center gap-2.5">
              <Globe className="w-6 h-6 text-emerald-600" />
              Domain Manager
            </h1>
            <p className="text-xs sm:text-sm text-slate-500 dark:text-slate-400 mt-0.5">
              Register new domains, manage DNS, nameservers, theft protection, and server hosting.
            </p>
          </div>

          <div className="flex items-center gap-2.5 flex-wrap">
            <Link
              href="/admin/domains"
              className="inline-flex items-center justify-center gap-1.5 px-3.5 py-2 rounded-lg bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 text-slate-700 dark:text-slate-200 text-xs font-semibold hover:border-slate-300 dark:hover:border-slate-700 shadow-2xs transition active:scale-98 cursor-pointer"
            >
              <Settings className="w-3.5 h-3.5 text-purple-500" />
              <span>Reseller Admin</span>
            </Link>

            <button
              onClick={() => setActiveTab('search')}
              className="inline-flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white text-xs font-bold shadow-xs transition active:scale-98 cursor-pointer"
            >
              <Plus className="w-4 h-4 stroke-[2.5]" />
              <span>Register Domain</span>
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
            <button onClick={() => setSuccessMessage(null)} className="text-emerald-700 dark:text-emerald-400 hover:opacity-75">
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
            <button onClick={() => setError(null)} className="text-rose-700 dark:text-rose-400 hover:opacity-75">
              <X className="w-4 h-4" />
            </button>
          </div>
        )}

        {/* Info Banner */}
        <div className="p-4 sm:p-5 rounded-xl bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 flex flex-col md:flex-row md:items-center justify-between gap-4 shadow-2xs">
          <div className="flex items-start gap-3.5">
            <div className="w-9 h-9 rounded-lg bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400 flex-shrink-0 mt-0.5">
              <Info className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
            </div>
            <div className="space-y-1">
              <h3 className="text-sm font-bold text-slate-900 dark:text-white">
                Hostvra Default Nameservers
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400 leading-relaxed max-w-2xl">
                Point any domain registered with Hostvra or external registrars to our high-performance anycast nameservers for automated DNS management and SSL issuance.
              </p>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-2 self-start md:self-center flex-shrink-0">
            <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-800 text-xs shadow-2xs">
              <span className="text-slate-500 dark:text-slate-400 font-medium">NS1:</span>
              <span className="font-mono font-medium text-slate-900 dark:text-white">{defaultNameservers[0]}</span>
              <button
                onClick={() => copyToClipboard(defaultNameservers[0], 'ns1')}
                className="text-slate-400 hover:text-emerald-600 dark:hover:text-emerald-400 transition"
              >
                {copiedKey === 'ns1' ? <Check className="w-3.5 h-3.5 text-emerald-600" /> : <Copy className="w-3.5 h-3.5" />}
              </button>
            </div>

            <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-800 text-xs shadow-2xs">
              <span className="text-slate-500 dark:text-slate-400 font-medium">NS2:</span>
              <span className="font-mono font-medium text-slate-900 dark:text-white">{defaultNameservers[1]}</span>
              <button
                onClick={() => copyToClipboard(defaultNameservers[1], 'ns2')}
                className="text-slate-400 hover:text-emerald-600 dark:hover:text-emerald-400 transition"
              >
                {copiedKey === 'ns2' ? <Check className="w-3.5 h-3.5 text-emerald-600" /> : <Copy className="w-3.5 h-3.5" />}
              </button>
            </div>
          </div>
        </div>

        {/* Navigation Tabs */}
        <div className="flex items-center gap-2 border-b border-slate-200 dark:border-slate-800 pb-px overflow-x-auto">
          <button
            onClick={() => setActiveTab('registered')}
            className={`flex items-center gap-2 px-4 py-2.5 text-xs sm:text-sm font-semibold transition border-b-2 whitespace-nowrap cursor-pointer ${
              activeTab === 'registered'
                ? 'border-emerald-600 text-emerald-600 dark:border-emerald-400 dark:text-emerald-400'
                : 'border-transparent text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white'
            }`}
          >
            <ShieldCheck className="w-4 h-4" />
            <span>My Registered Domains</span>
            <span className="px-1.5 py-0.5 rounded-full text-[10px] font-bold bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800">
              {registeredDomains.length}
            </span>
          </button>

          <button
            onClick={() => setActiveTab('search')}
            className={`flex items-center gap-2 px-4 py-2.5 text-xs sm:text-sm font-semibold transition border-b-2 whitespace-nowrap cursor-pointer ${
              activeTab === 'search'
                ? 'border-emerald-600 text-emerald-600 dark:border-emerald-400 dark:text-emerald-400'
                : 'border-transparent text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white'
            }`}
          >
            <Search className="w-4 h-4" />
            <span>Register Domain</span>
          </button>

          <button
            onClick={() => setActiveTab('transfer')}
            className={`flex items-center gap-2 px-4 py-2.5 text-xs sm:text-sm font-semibold transition border-b-2 whitespace-nowrap cursor-pointer ${
              activeTab === 'transfer'
                ? 'border-emerald-600 text-emerald-600 dark:border-emerald-400 dark:text-emerald-400'
                : 'border-transparent text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white'
            }`}
          >
            <Layers className="w-4 h-4" />
            <span>Transfer In</span>
          </button>

          <button
            onClick={() => setActiveTab('hosted')}
            className={`flex items-center gap-2 px-4 py-2.5 text-xs sm:text-sm font-semibold transition border-b-2 whitespace-nowrap cursor-pointer ${
              activeTab === 'hosted'
                ? 'border-emerald-600 text-emerald-600 dark:border-emerald-400 dark:text-emerald-400'
                : 'border-transparent text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white'
            }`}
          >
            <Server className="w-4 h-4" />
            <span>Hosted Websites</span>
            <span className="px-1.5 py-0.5 rounded-full text-[10px] font-bold bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-300">
              {hostedDomains.length}
            </span>
          </button>

          <button
            onClick={() => setActiveTab('whois')}
            className={`flex items-center gap-2 px-4 py-2.5 text-xs sm:text-sm font-semibold transition border-b-2 whitespace-nowrap cursor-pointer ${
              activeTab === 'whois'
                ? 'border-emerald-600 text-emerald-600 dark:border-emerald-400 dark:text-emerald-400'
                : 'border-transparent text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white'
            }`}
          >
            <Info className="w-4 h-4" />
            <span>WHOIS Lookup</span>
          </button>
        </div>

        {/* ==================================================== */}
        {/* TAB 1: MY REGISTERED DOMAINS (RESELLERCLUB) */}
        {/* ==================================================== */}
        {activeTab === 'registered' && (
          <div className="bg-white dark:bg-[#131B2E] rounded-xl border border-slate-200 dark:border-slate-800 shadow-xs overflow-hidden">
            <div className="p-4 sm:p-5 border-b border-slate-200 dark:border-slate-800 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
              <div className="flex items-center gap-2.5">
                <h2 className="text-base font-bold text-slate-900 dark:text-white">
                  Domains Under Management
                </h2>
                <span className="px-2 py-0.5 rounded-full text-xs font-semibold bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-300">
                  {filteredRegistered.length}
                </span>
              </div>

              <div className="flex flex-wrap items-center gap-2.5">
                <div className="relative">
                  <Search className="w-3.5 h-3.5 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none" />
                  <input
                    type="text"
                    value={registeredSearchQuery}
                    onChange={(e) => setRegisteredSearchQuery(e.target.value)}
                    placeholder="Search your domains..."
                    className="pl-8 pr-3 py-1.5 text-xs text-slate-900 dark:text-white bg-slate-50 dark:bg-[#0B1120] placeholder-slate-400 border border-slate-200 dark:border-slate-700 rounded-lg focus:outline-none focus:border-emerald-600 w-48 sm:w-60"
                  />
                </div>

                <select
                  value={registeredStatusFilter}
                  onChange={(e) => setRegisteredStatusFilter(e.target.value)}
                  className="px-2.5 py-1.5 text-xs text-slate-900 dark:text-white bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-700 rounded-lg focus:outline-none focus:border-emerald-600"
                >
                  <option value="all">All Statuses</option>
                  <option value="active">Active</option>
                  <option value="pending">Pending</option>
                  <option value="expired">Expired</option>
                </select>

                <button
                  onClick={fetchAllData}
                  disabled={loading}
                  className="p-1.5 rounded-lg text-slate-500 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800 border border-slate-200 dark:border-slate-700 transition"
                  title="Refresh"
                >
                  <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin text-emerald-600' : ''}`} />
                </button>
              </div>
            </div>

            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs border-collapse">
                <thead>
                  <tr className="border-b border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-[#0B1120] text-[11px] font-bold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                    <th className="px-4 py-3">Domain Name</th>
                    <th className="px-4 py-3">Status</th>
                    <th className="px-4 py-3">Registration Date</th>
                    <th className="px-4 py-3">Expires On</th>
                    <th className="px-4 py-3">Auto Renew</th>
                    <th className="px-4 py-3">Theft Lock</th>
                    <th className="px-4 py-3 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                  {filteredRegistered.length === 0 ? (
                    <tr>
                      <td colSpan={7} className="py-12 text-center text-slate-400">
                        <Globe className="w-8 h-8 text-slate-300 dark:text-slate-600 mx-auto mb-2" />
                        <p className="font-medium text-slate-600 dark:text-slate-400">No registered domains found</p>
                        <p className="text-xs text-slate-400 mt-0.5">
                          Search and register a new domain or transfer your existing domain in.
                        </p>
                      </td>
                    </tr>
                  ) : (
                    filteredRegistered.map((d) => (
                      <tr key={d.id} className="hover:bg-slate-50/50 dark:hover:bg-slate-800/50 transition-colors">
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-2">
                            <span className="font-bold text-slate-900 dark:text-white text-sm">
                              {d.domain_name}
                            </span>
                          </div>
                        </td>
                        <td className="px-4 py-3">
                          <span
                            className={`inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-wider ${
                              d.status === 'active'
                                ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
                                : d.status === 'pending'
                                ? 'bg-amber-500/10 text-amber-600 dark:text-amber-400'
                                : 'bg-slate-100 dark:bg-slate-800 text-slate-500'
                            }`}
                          >
                            {d.status}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-slate-600 dark:text-slate-400">
                          {d.registration_date ? new Date(d.registration_date).toLocaleDateString() : '—'}
                        </td>
                        <td className="px-4 py-3 font-medium text-slate-900 dark:text-white">
                          {d.expiry_date ? new Date(d.expiry_date).toLocaleDateString() : '—'}
                        </td>
                        <td className="px-4 py-3">
                          <span className={d.auto_renew ? 'text-emerald-600 font-semibold' : 'text-slate-400'}>
                            {d.auto_renew ? 'Enabled' : 'Disabled'}
                          </span>
                        </td>
                        <td className="px-4 py-3">
                          {d.is_locked ? (
                            <span className="inline-flex items-center gap-1 text-emerald-600 font-semibold text-[11px]">
                              <Lock className="w-3 h-3" /> Locked
                            </span>
                          ) : (
                            <span className="inline-flex items-center gap-1 text-amber-600 font-semibold text-[11px]">
                              <Unlock className="w-3 h-3" /> Unlocked
                            </span>
                          )}
                        </td>
                        <td className="px-4 py-3 text-right">
                          <button
                            onClick={() => openManageModal(d)}
                            className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs shadow-xs transition cursor-pointer"
                          >
                            <Settings className="w-3 h-3" />
                            <span>Manage</span>
                          </button>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* ==================================================== */}
        {/* TAB 2: REGISTER DOMAIN (SEARCH & CART) */}
        {/* ==================================================== */}
        {activeTab === 'search' && (
          <div className="space-y-6">
            <div className="p-6 sm:p-8 rounded-xl bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 shadow-xs text-center space-y-4">
              <div className="max-w-xl mx-auto space-y-2">
                <h2 className="text-xl font-bold text-slate-900 dark:text-white">
                  Find and Register Your Perfect Domain
                </h2>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                  Direct ResellerClub API integration with live availability checks and instant provisioning.
                </p>
              </div>

              <form onSubmit={handleDomainSearch} className="max-w-2xl mx-auto flex gap-2">
                <div className="relative flex-1">
                  <Search className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2 pointer-events-none" />
                  <input
                    type="text"
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    placeholder="Search domain (e.g. hostvrademo.com)..."
                    className="w-full pl-10 pr-4 py-2.5 text-xs sm:text-sm text-slate-900 dark:text-white bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-700 rounded-lg focus:outline-none focus:border-emerald-600"
                  />
                </div>
                <button
                  type="submit"
                  disabled={isSearching}
                  className="px-5 py-2.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white text-xs sm:text-sm font-bold transition flex items-center gap-1.5 shadow-xs disabled:opacity-50 cursor-pointer"
                >
                  {isSearching ? <RefreshCw className="w-4 h-4 animate-spin" /> : <Search className="w-4 h-4" />}
                  <span>Check Availability</span>
                </button>
              </form>

              <div className="flex justify-center items-center gap-1 text-xs pt-1">
                <span className="text-slate-500">Currency:</span>
                <button
                  onClick={() => setCurrency('USD')}
                  className={`px-2 py-0.5 rounded font-semibold cursor-pointer ${currency === 'USD' ? 'bg-emerald-600 text-white' : 'text-slate-500'}`}
                >
                  USD ($)
                </button>
                <button
                  onClick={() => setCurrency('BDT')}
                  className={`px-2 py-0.5 rounded font-semibold cursor-pointer ${currency === 'BDT' ? 'bg-emerald-600 text-white' : 'text-slate-500'}`}
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
                    Registrar Availability Results
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
                              ? 'bg-emerald-500/10 text-emerald-600 border border-emerald-500/20'
                              : 'bg-slate-100 dark:bg-slate-800 text-slate-400'
                          }`}
                        >
                          {item.available ? <CheckCircle2 className="w-4 h-4" /> : <XCircle className="w-4 h-4" />}
                        </div>
                        <div>
                          <div className="flex items-center gap-2">
                            <span className="text-sm font-bold text-slate-900 dark:text-white">{item.domain}</span>
                            {item.is_popular && (
                              <span className="px-1.5 py-0.5 rounded text-[9px] font-bold bg-amber-500/10 text-amber-600 border border-amber-500/20">
                                Popular
                              </span>
                            )}
                          </div>
                          <span className={`text-xs font-medium ${item.available ? 'text-emerald-600' : 'text-slate-400'}`}>
                            {item.available ? 'Available for instant registration' : 'Taken / Unavailable'}
                          </span>
                        </div>
                      </div>

                      <div className="flex items-center gap-3 self-end sm:self-center">
                        <div className="text-right">
                          <span className="text-sm font-bold text-slate-900 dark:text-white">
                            {formatPrice(item.register_price)}
                          </span>
                          <div className="text-[11px] text-slate-500">
                            renews at {formatPrice(item.renew_price || item.register_price)}/yr
                          </div>
                        </div>

                        {item.available && (
                          <button
                            onClick={() => {
                              setSelectedDomainItem(item);
                              setOrderModalOpen(true);
                            }}
                            className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white text-xs font-bold shadow-xs transition cursor-pointer"
                          >
                            Register Now
                          </button>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Popular TLDs Pricing Table */}
            <div className="bg-white dark:bg-[#131B2E] rounded-xl border border-slate-200 dark:border-slate-800 overflow-hidden shadow-xs">
              <div className="p-4 border-b border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-[#0B1120]">
                <h3 className="text-xs font-bold text-slate-900 dark:text-white uppercase tracking-wider">
                  Supported TLD Pricing ({tldList.length})
                </h3>
              </div>
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead className="border-b border-slate-200 dark:border-slate-800 text-slate-500 font-semibold uppercase text-[10px]">
                    <tr>
                      <th className="px-4 py-2.5">TLD Extension</th>
                      <th className="px-4 py-2.5">Registration (1 yr)</th>
                      <th className="px-4 py-2.5">Renewal (1 yr)</th>
                      <th className="px-4 py-2.5">Transfer In</th>
                      <th className="px-4 py-2.5">Min Years</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                    {tldList.map((tld) => (
                      <tr key={tld.id} className="hover:bg-slate-50/50 dark:hover:bg-slate-800/50">
                        <td className="px-4 py-2.5 font-bold font-mono text-slate-900 dark:text-white">{tld.tld}</td>
                        <td className="px-4 py-2.5 font-bold text-emerald-600">{formatPrice(tld.register_price)}</td>
                        <td className="px-4 py-2.5 text-slate-600 dark:text-slate-300">{formatPrice(tld.renew_price)}</td>
                        <td className="px-4 py-2.5 text-slate-600 dark:text-slate-300">{formatPrice(tld.transfer_price)}</td>
                        <td className="px-4 py-2.5 text-slate-500">{tld.min_years} yr</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}

        {/* ==================================================== */}
        {/* TAB 3: TRANSFER IN DOMAIN */}
        {/* ==================================================== */}
        {activeTab === 'transfer' && (
          <div className="max-w-xl mx-auto p-6 sm:p-8 rounded-xl bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 shadow-xs space-y-5">
            <div className="space-y-1 text-center">
              <h2 className="text-lg font-bold text-slate-900 dark:text-white flex items-center justify-center gap-2">
                <Layers className="w-5 h-5 text-emerald-600" />
                Transfer Your Domain to Hostvra
              </h2>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                Seamlessly migrate your domains from Namecheap, GoDaddy, or Cloudflare with 1 free year renewal included.
              </p>
            </div>

            <form onSubmit={handleTransferSubmit} className="space-y-4 text-xs">
              <div className="space-y-1">
                <label className="font-semibold text-slate-700 dark:text-slate-300">Domain Name *</label>
                <input
                  type="text"
                  required
                  value={transferForm.domain}
                  onChange={(e) => setTransferForm({ ...transferForm, domain: e.target.value })}
                  placeholder="yourdomain.com"
                  className="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-700 text-slate-900 dark:text-white focus:outline-none focus:border-emerald-600"
                />
              </div>

              <div className="space-y-1">
                <label className="font-semibold text-slate-700 dark:text-slate-300">
                  EPP / Authorization Code *
                </label>
                <input
                  type="password"
                  required
                  value={transferForm.auth_code}
                  onChange={(e) => setTransferForm({ ...transferForm, auth_code: e.target.value })}
                  placeholder="Obtain from your current registrar"
                  className="w-full px-3 py-2 font-mono rounded-lg bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-700 text-slate-900 dark:text-white focus:outline-none focus:border-emerald-600"
                />
                <p className="text-[11px] text-slate-400">
                  Ensure the domain is unlocked and WHOIS privacy is disabled at your current registrar before initiating.
                </p>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <label className="font-semibold text-slate-700 dark:text-slate-300">Extension Years</label>
                  <select
                    value={transferForm.years}
                    onChange={(e) => setTransferForm({ ...transferForm, years: parseInt(e.target.value) || 1 })}
                    className="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-700 text-slate-900 dark:text-white focus:outline-none focus:border-emerald-600"
                  >
                    {[1, 2, 3, 5].map((y) => (
                      <option key={y} value={y}>{y} Year (+{y} yr extension)</option>
                    ))}
                  </select>
                </div>

                <div className="space-y-1">
                  <label className="font-semibold text-slate-700 dark:text-slate-300">Payment Gateway</label>
                  <select
                    value={transferForm.payment_method}
                    onChange={(e) => setTransferForm({ ...transferForm, payment_method: e.target.value })}
                    className="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-700 text-slate-900 dark:text-white focus:outline-none focus:border-emerald-600"
                  >
                    <option value="bkash">bKash (Bangladesh)</option>
                    <option value="stripe">Stripe (Card / Apple Pay)</option>
                    <option value="paypal">PayPal</option>
                    <option value="balance">Account Balance</option>
                  </select>
                </div>
              </div>

              <button
                type="submit"
                disabled={isSubmittingTransfer}
                className="w-full py-2.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs shadow-xs transition flex items-center justify-center gap-2 disabled:opacity-50 cursor-pointer"
              >
                {isSubmittingTransfer ? <RefreshCw className="w-4 h-4 animate-spin" /> : <Send className="w-4 h-4" />}
                <span>Initiate Transfer Checkout</span>
              </button>
            </form>
          </div>
        )}

        {/* ==================================================== */}
        {/* TAB 4: HOSTED WEBSITES (LOCAL SERVER) */}
        {/* ==================================================== */}
        {activeTab === 'hosted' && (
          <div className="bg-white dark:bg-[#131B2E] rounded-xl border border-slate-200 dark:border-slate-800 shadow-xs overflow-hidden">
            <div className="p-4 sm:p-5 border-b border-slate-200 dark:border-slate-800 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
              <div className="flex items-center gap-2.5">
                <h2 className="text-base font-bold text-slate-900 dark:text-white">
                  Local Virtual Hosts
                </h2>
                <span className="px-2 py-0.5 rounded-full text-xs font-semibold bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-300">
                  {filteredHosted.length}
                </span>
              </div>

              <div className="flex items-center gap-2.5">
                <div className="relative">
                  <Search className="w-3.5 h-3.5 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none" />
                  <input
                    type="text"
                    value={hostedSearchQuery}
                    onChange={(e) => setHostedSearchQuery(e.target.value)}
                    placeholder="Search hosted sites..."
                    className="pl-8 pr-3 py-1.5 text-xs text-slate-900 dark:text-white bg-slate-50 dark:bg-[#0B1120] placeholder-slate-400 border border-slate-200 dark:border-slate-700 rounded-lg focus:outline-none focus:border-emerald-600 w-48"
                  />
                </div>

                <button
                  onClick={fetchAllData}
                  disabled={loading}
                  className="p-1.5 rounded-lg text-slate-500 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800 border border-slate-200 dark:border-slate-700 transition"
                  title="Refresh"
                >
                  <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin text-emerald-600' : ''}`} />
                </button>
              </div>
            </div>

            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs border-collapse">
                <thead>
                  <tr className="border-b border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-[#0B1120] text-[11px] font-bold text-slate-500 uppercase tracking-wider">
                    <th className="px-4 py-3">Website Domain</th>
                    <th className="px-4 py-3">Document Root</th>
                    <th className="px-4 py-3">PHP / Runtime</th>
                    <th className="px-4 py-3">SSL</th>
                    <th className="px-4 py-3">Status</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                  {filteredHosted.length === 0 ? (
                    <tr>
                      <td colSpan={5} className="py-8 text-center text-slate-400">
                        No virtual hosts found on this server.
                      </td>
                    </tr>
                  ) : (
                    filteredHosted.map((w) => (
                      <tr key={w.id} className="hover:bg-slate-50/50 dark:hover:bg-slate-800/50">
                        <td className="px-4 py-3 font-bold text-slate-900 dark:text-white">
                          <Link href={`https://${w.primary_domain}`} target="_blank" className="hover:text-emerald-600 inline-flex items-center gap-1.5">
                            {w.primary_domain}
                            <ExternalLink className="w-3 h-3 text-slate-400" />
                          </Link>
                        </td>
                        <td className="px-4 py-3 font-mono text-slate-500">{w.document_root}</td>
                        <td className="px-4 py-3">{w.php_version ? `PHP ${w.php_version}` : w.app_type}</td>
                        <td className="px-4 py-3">
                          {w.ssl_enabled ? (
                            <span className="text-emerald-600 font-semibold flex items-center gap-1">
                              <ShieldCheck className="w-3.5 h-3.5" /> Active
                            </span>
                          ) : (
                            <span className="text-slate-400">Disabled</span>
                          )}
                        </td>
                        <td className="px-4 py-3">
                          <span className="px-2 py-0.5 rounded-full text-[10px] font-bold uppercase bg-emerald-500/10 text-emerald-600">
                            {w.status}
                          </span>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* ==================================================== */}
        {/* TAB 5: REAL WHOIS LOOKUP (ZERO MOCKS) */}
        {/* ==================================================== */}
        {activeTab === 'whois' && (
          <div className="space-y-6">
            <div className="p-6 sm:p-8 rounded-xl bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 shadow-xs text-center space-y-4">
              <div className="max-w-xl mx-auto space-y-2">
                <h2 className="text-xl font-bold text-slate-900 dark:text-white">
                  Live Public WHOIS Lookup
                </h2>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                  Real-time lookup through official ICANN / registry WHOIS servers.
                </p>
              </div>

              <form onSubmit={handleWhoisLookup} className="max-w-xl mx-auto flex gap-2">
                <input
                  type="text"
                  value={whoisQuery}
                  onChange={(e) => setWhoisQuery(e.target.value)}
                  placeholder="Enter domain (e.g. google.com)..."
                  className="flex-1 px-4 py-2.5 text-xs text-slate-900 dark:text-white bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-700 rounded-lg focus:outline-none focus:border-emerald-600"
                />
                <button
                  type="submit"
                  disabled={isWhoisLoading}
                  className="px-5 py-2.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white text-xs font-bold transition flex items-center gap-1.5 shadow-xs disabled:opacity-50 cursor-pointer"
                >
                  {isWhoisLoading ? <RefreshCw className="w-4 h-4 animate-spin" /> : <Search className="w-4 h-4" />}
                  <span>Lookup</span>
                </button>
              </form>
            </div>

            {whoisData && (
              <div className="bg-white dark:bg-[#131B2E] rounded-xl border border-slate-200 dark:border-slate-800 p-6 shadow-xs space-y-4">
                <div className="border-b border-slate-200 dark:border-slate-800 pb-3 flex items-center justify-between">
                  <h3 className="text-sm font-bold text-slate-900 dark:text-white">
                    WHOIS Record: <span className="font-mono text-emerald-600">{whoisData.domain}</span>
                  </h3>
                  <span className="text-xs text-slate-400">
                    Queried: {new Date(whoisData.queried_at).toLocaleString()}
                  </span>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
                  <div>
                    <span className="text-slate-500">Registrar:</span>
                    <p className="font-semibold text-slate-900 dark:text-white">{whoisData.registrar || 'Unknown'}</p>
                  </div>
                  <div>
                    <span className="text-slate-500">Creation Date:</span>
                    <p className="font-semibold text-slate-900 dark:text-white">
                      {whoisData.created_date ? new Date(whoisData.created_date).toLocaleDateString() : '—'}
                    </p>
                  </div>
                  <div>
                    <span className="text-slate-500">Expiry Date:</span>
                    <p className="font-semibold text-slate-900 dark:text-white">
                      {whoisData.expiry_date ? new Date(whoisData.expiry_date).toLocaleDateString() : '—'}
                    </p>
                  </div>
                  <div>
                    <span className="text-slate-500">Nameservers:</span>
                    <p className="font-mono font-medium text-slate-900 dark:text-white">
                      {whoisData.nameservers?.join(', ') || '—'}
                    </p>
                  </div>
                </div>

                {whoisData.raw_whois && (
                  <div className="pt-2">
                    <span className="text-xs font-semibold text-slate-700 dark:text-slate-300 block mb-1">
                      Raw WHOIS Output:
                    </span>
                    <pre className="p-3 rounded-lg bg-slate-900 text-slate-200 font-mono text-[11px] overflow-x-auto max-h-60 leading-tight">
                      {whoisData.raw_whois}
                    </pre>
                  </div>
                )}
              </div>
            )}
          </div>
        )}

        {/* ==================================================== */}
        {/* MODAL: MANAGE REGISTERED DOMAIN */}
        {/* ==================================================== */}
        {manageDomain && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs animate-fadeIn">
            <div className="bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 rounded-2xl w-full max-w-3xl shadow-2xl overflow-hidden p-6 space-y-5">
              <div className="flex items-center justify-between border-b border-slate-100 dark:border-slate-800 pb-3">
                <div>
                  <h3 className="text-lg font-bold text-slate-900 dark:text-white flex items-center gap-2">
                    <Globe className="w-5 h-5 text-emerald-600" />
                    <span>Manage: {manageDomain.domain_name}</span>
                  </h3>
                  <div className="flex items-center gap-3 text-xs text-slate-500 mt-0.5">
                    <span>Status: <strong className="uppercase text-emerald-600">{manageDomain.status}</strong></span>
                    <span>•</span>
                    <span>Expires: {manageDomain.expiry_date ? new Date(manageDomain.expiry_date).toLocaleDateString() : '—'}</span>
                  </div>
                </div>
                <button
                  onClick={() => setManageDomain(null)}
                  className="w-8 h-8 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800 flex items-center justify-center cursor-pointer"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              {/* Sub-Tabs */}
              <div className="flex border-b border-slate-200 dark:border-slate-800 gap-4 text-xs font-semibold">
                <button
                  onClick={() => setManageTab('nameservers')}
                  className={`pb-2 border-b-2 transition cursor-pointer ${manageTab === 'nameservers' ? 'border-emerald-600 text-emerald-600' : 'border-transparent text-slate-500'}`}
                >
                  Nameservers
                </button>
                <button
                  onClick={() => {
                    setManageTab('dns');
                    handleLoadDomainDns();
                  }}
                  className={`pb-2 border-b-2 transition cursor-pointer ${manageTab === 'dns' ? 'border-emerald-600 text-emerald-600' : 'border-transparent text-slate-500'}`}
                >
                  DNS Records
                </button>
                <button
                  onClick={() => setManageTab('security')}
                  className={`pb-2 border-b-2 transition cursor-pointer ${manageTab === 'security' ? 'border-emerald-600 text-emerald-600' : 'border-transparent text-slate-500'}`}
                >
                  Lock &amp; EPP Code
                </button>
                <button
                  onClick={() => setManageTab('renew')}
                  className={`pb-2 border-b-2 transition cursor-pointer ${manageTab === 'renew' ? 'border-emerald-600 text-emerald-600' : 'border-transparent text-slate-500'}`}
                >
                  Renew Domain
                </button>
                <button
                  onClick={() => {
                    setManageTab('contacts');
                    handleLoadContacts();
                  }}
                  className={`pb-2 border-b-2 transition cursor-pointer ${manageTab === 'contacts' ? 'border-emerald-600 text-emerald-600' : 'border-transparent text-slate-500'}`}
                >
                  Contact Info
                </button>
              </div>

              {/* TAB CONTENT */}
              {manageTab === 'nameservers' && (
                <div className="space-y-4 text-xs">
                  <div className="flex items-center justify-between">
                    <p className="text-slate-600 dark:text-slate-400">
                      Configure authoritative nameservers for this domain with ResellerClub registry.
                    </p>
                    <button
                      type="button"
                      onClick={() => setCustomNsInputs(['ns1.hostvra.com', 'ns2.hostvra.com'])}
                      className="text-emerald-600 hover:underline font-semibold cursor-pointer"
                    >
                      Use Hostvra Defaults
                    </button>
                  </div>

                  <div className="space-y-2">
                    {customNsInputs.map((ns, idx) => (
                      <div key={idx} className="flex items-center gap-2">
                        <span className="w-12 text-slate-400 font-mono">NS {idx + 1}:</span>
                        <input
                          type="text"
                          value={ns}
                          onChange={(e) => {
                            const copy = [...customNsInputs];
                            copy[idx] = e.target.value;
                            setCustomNsInputs(copy);
                          }}
                          placeholder={`ns${idx + 1}.example.com`}
                          className="flex-1 px-3 py-1.5 font-mono text-xs rounded-lg bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-700 text-slate-900 dark:text-white focus:outline-none focus:border-emerald-600"
                        />
                        {customNsInputs.length > 2 && (
                          <button
                            type="button"
                            onClick={() => setCustomNsInputs(customNsInputs.filter((_, i) => i !== idx))}
                            className="text-rose-500 hover:text-rose-700 p-1"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        )}
                      </div>
                    ))}
                  </div>

                  {customNsInputs.length < 5 && (
                    <button
                      type="button"
                      onClick={() => setCustomNsInputs([...customNsInputs, ''])}
                      className="text-emerald-600 hover:underline font-semibold cursor-pointer"
                    >
                      + Add Another Nameserver
                    </button>
                  )}

                  <div className="pt-3 border-t border-slate-100 dark:border-slate-800 flex justify-end">
                    <button
                      onClick={handleSaveNameservers}
                      disabled={savingNs}
                      className="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold transition disabled:opacity-50 cursor-pointer"
                    >
                      {savingNs ? 'Updating...' : 'Save Nameservers'}
                    </button>
                  </div>
                </div>
              )}

              {manageTab === 'dns' && (
                <div className="space-y-4 text-xs">
                  {/* Add record form */}
                  <form onSubmit={handleAddDomainDns} className="p-3 rounded-lg bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-800 grid grid-cols-1 sm:grid-cols-5 gap-2 items-end">
                    <div>
                      <label className="block text-[10px] uppercase font-bold text-slate-400 mb-1">Type</label>
                      <select
                        value={newDomainDns.type}
                        onChange={(e) => setNewDomainDns({ ...newDomainDns, type: e.target.value })}
                        className="w-full px-2 py-1.5 rounded bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-700 text-slate-900 dark:text-white font-semibold"
                      >
                        <option value="A">A</option>
                        <option value="AAAA">AAAA</option>
                        <option value="CNAME">CNAME</option>
                        <option value="MX">MX</option>
                        <option value="TXT">TXT</option>
                        <option value="SRV">SRV</option>
                      </select>
                    </div>

                    <div>
                      <label className="block text-[10px] uppercase font-bold text-slate-400 mb-1">Host (@/sub)</label>
                      <input
                        type="text"
                        value={newDomainDns.name}
                        onChange={(e) => setNewDomainDns({ ...newDomainDns, name: e.target.value })}
                        className="w-full px-2 py-1.5 rounded bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-700 text-slate-900 dark:text-white"
                        placeholder="@"
                        required
                      />
                    </div>

                    <div className="sm:col-span-2">
                      <label className="block text-[10px] uppercase font-bold text-slate-400 mb-1">Value / Target</label>
                      <input
                        type="text"
                        value={newDomainDns.content}
                        onChange={(e) => setNewDomainDns({ ...newDomainDns, content: e.target.value })}
                        className="w-full px-2 py-1.5 rounded bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-700 text-slate-900 dark:text-white"
                        placeholder="IP or domain target"
                        required
                      />
                    </div>

                    <button
                      type="submit"
                      disabled={addingDomainDns}
                      className="w-full py-1.5 rounded bg-emerald-600 hover:bg-emerald-700 text-white font-bold transition disabled:opacity-50 cursor-pointer"
                    >
                      {addingDomainDns ? 'Adding...' : 'Add Record'}
                    </button>
                  </form>

                  {/* List of DNS records */}
                  <div className="border border-slate-200 dark:border-slate-800 rounded-lg overflow-hidden">
                    <table className="w-full text-left text-xs">
                      <thead className="bg-slate-50 dark:bg-[#0B1120] text-slate-500 uppercase text-[10px]">
                        <tr>
                          <th className="p-2.5">Type</th>
                          <th className="p-2.5">Name</th>
                          <th className="p-2.5">Value</th>
                          <th className="p-2.5">TTL</th>
                          <th className="p-2.5 text-right">Action</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                        {loadingDomainDns ? (
                          <tr><td colSpan={5} className="p-4 text-center text-slate-400">Loading DNS records...</td></tr>
                        ) : domainDnsList.length === 0 ? (
                          <tr><td colSpan={5} className="p-4 text-center text-slate-400">No DNS records found or external nameservers active.</td></tr>
                        ) : (
                          domainDnsList.map((r) => (
                            <tr key={r.id}>
                              <td className="p-2.5 font-bold font-mono text-emerald-600">{r.type}</td>
                              <td className="p-2.5 font-mono">{r.name}</td>
                              <td className="p-2.5 font-mono truncate max-w-xs">{r.content}</td>
                              <td className="p-2.5 text-slate-400">{r.ttl}s</td>
                              <td className="p-2.5 text-right">
                                <button
                                  onClick={() => handleDeleteDomainDns(r.id)}
                                  className="text-rose-500 hover:text-rose-700 p-1 cursor-pointer"
                                >
                                  <Trash2 className="w-3.5 h-3.5" />
                                </button>
                              </td>
                            </tr>
                          ))
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}

              {manageTab === 'security' && (
                <div className="space-y-6 text-xs">
                  <div className="p-4 rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-[#0B1120] flex items-center justify-between">
                    <div>
                      <h4 className="font-bold text-slate-900 dark:text-white flex items-center gap-1.5">
                        <Lock className="w-4 h-4 text-emerald-600" />
                        Registrar Transfer Lock (Theft Protection)
                      </h4>
                      <p className="text-slate-500 text-[11px] mt-0.5">
                        Prevents unauthorized domain transfer attempts to another registrar.
                      </p>
                    </div>

                    <button
                      onClick={handleToggleLock}
                      disabled={togglingLock}
                      className={`px-3 py-1.5 rounded-lg text-xs font-bold transition cursor-pointer ${
                        domainLockStatus
                          ? 'bg-emerald-600 text-white hover:bg-emerald-700'
                          : 'bg-rose-600 text-white hover:bg-rose-700'
                      }`}
                    >
                      {togglingLock ? 'Updating...' : domainLockStatus ? 'Locked (Protected)' : 'Unlocked (Exposed)'}
                    </button>
                  </div>

                  <div className="p-4 rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-[#0B1120] space-y-3">
                    <div>
                      <h4 className="font-bold text-slate-900 dark:text-white flex items-center gap-1.5">
                        <Key className="w-4 h-4 text-purple-500" />
                        EPP / Auth Code
                      </h4>
                      <p className="text-slate-500 text-[11px] mt-0.5">
                        Required if you ever decide to transfer this domain away to another registrar.
                      </p>
                    </div>

                    {eppCode ? (
                      <div className="flex items-center gap-2 p-2.5 rounded bg-slate-900 text-emerald-400 font-mono text-xs">
                        <span className="flex-1">{eppCode}</span>
                        <button
                          onClick={() => copyToClipboard(eppCode, 'epp')}
                          className="px-2 py-1 rounded bg-slate-800 text-white hover:bg-slate-700 transition"
                        >
                          {copiedKey === 'epp' ? 'Copied' : 'Copy'}
                        </button>
                      </div>
                    ) : (
                      <button
                        onClick={handleFetchEppCode}
                        disabled={fetchingEpp}
                        className="px-3 py-1.5 rounded-lg bg-slate-800 text-white hover:bg-slate-700 font-bold transition cursor-pointer"
                      >
                        {fetchingEpp ? 'Retrieving EPP Code...' : 'Reveal Authorization Code'}
                      </button>
                    )}
                  </div>
                </div>
              )}

              {manageTab === 'renew' && (
                <div className="space-y-4 text-xs max-w-md">
                  <div className="space-y-1">
                    <label className="font-semibold text-slate-700 dark:text-slate-300">Renew For</label>
                    <select
                      value={renewYears}
                      onChange={(e) => setRenewYears(parseInt(e.target.value) || 1)}
                      className="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-700 text-slate-900 dark:text-white font-semibold"
                    >
                      {[1, 2, 3, 5, 10].map((y) => (
                        <option key={y} value={y}>{y} Year{y > 1 ? 's' : ''}</option>
                      ))}
                    </select>
                  </div>

                  <div className="space-y-1">
                    <label className="font-semibold text-slate-700 dark:text-slate-300">Payment Method</label>
                    <select
                      value={renewPaymentMethod}
                      onChange={(e) => setRenewPaymentMethod(e.target.value)}
                      className="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-700 text-slate-900 dark:text-white"
                    >
                      <option value="bkash">bKash</option>
                      <option value="stripe">Stripe (Card)</option>
                      <option value="paypal">PayPal</option>
                      <option value="balance">Account Balance</option>
                    </select>
                  </div>

                  <button
                    onClick={handleRenewDomain}
                    disabled={renewingDomain}
                    className="w-full py-2.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold transition disabled:opacity-50 cursor-pointer"
                  >
                    {renewingDomain ? 'Processing Renewal...' : 'Confirm & Renew Domain'}
                  </button>
                </div>
              )}

              {manageTab === 'contacts' && (
                <div className="space-y-3 text-xs">
                  {loadingContacts ? (
                    <p className="text-slate-400 py-4 text-center">Loading WHOIS contact information...</p>
                  ) : domainContacts?.registrant ? (
                    <div className="grid grid-cols-2 gap-4 p-4 rounded-xl bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-800">
                      <div>
                        <span className="text-slate-400">Name:</span>
                        <p className="font-bold text-slate-900 dark:text-white">{domainContacts.registrant.first_name} {domainContacts.registrant.last_name}</p>
                      </div>
                      <div>
                        <span className="text-slate-400">Email:</span>
                        <p className="font-mono text-slate-900 dark:text-white">{domainContacts.registrant.email}</p>
                      </div>
                      <div>
                        <span className="text-slate-400">Phone:</span>
                        <p className="font-mono text-slate-900 dark:text-white">{domainContacts.registrant.phone}</p>
                      </div>
                      <div>
                        <span className="text-slate-400">Address:</span>
                        <p className="text-slate-900 dark:text-white">{domainContacts.registrant.address1}, {domainContacts.registrant.city}, {domainContacts.registrant.country}</p>
                      </div>
                    </div>
                  ) : (
                    <p className="text-slate-400 py-4 text-center">No contact information available.</p>
                  )}
                </div>
              )}
            </div>
          </div>
        )}

        {/* ==================================================== */}
        {/* MODAL: ORDER REGISTRATION CHECKOUT */}
        {/* ==================================================== */}
        {orderModalOpen && selectedDomainItem && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs animate-fadeIn">
            <div className="bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 rounded-2xl w-full max-w-lg shadow-2xl overflow-hidden p-6 space-y-5">
              <div className="flex items-center justify-between border-b border-slate-100 dark:border-slate-800 pb-3">
                <h3 className="text-base font-bold text-slate-900 dark:text-white flex items-center gap-2">
                  <ShoppingBag className="w-5 h-5 text-emerald-600" />
                  <span>Register: {selectedDomainItem.domain}</span>
                </h3>
                <button
                  onClick={() => setOrderModalOpen(false)}
                  className="text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 cursor-pointer"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              <form onSubmit={handleOrderRegistrationSubmit} className="space-y-4 text-xs">
                <div className="p-3.5 rounded-xl bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-800 flex items-center justify-between">
                  <div>
                    <span className="text-slate-500">Total Price:</span>
                    <div className="text-xl font-bold font-mono text-emerald-600">
                      {formatPrice(selectedDomainItem.register_price * orderYears)}
                    </div>
                  </div>
                  <div className="text-right">
                    <span className="text-slate-500">Duration:</span>
                    <select
                      value={orderYears}
                      onChange={(e) => setOrderYears(parseInt(e.target.value) || 1)}
                      className="block mt-1 px-2.5 py-1 rounded bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-700 font-bold"
                    >
                      {[1, 2, 3, 5, 10].map((y) => (
                        <option key={y} value={y}>{y} Year{y > 1 ? 's' : ''}</option>
                      ))}
                    </select>
                  </div>
                </div>

                {/* Nameservers */}
                <div className="space-y-2">
                  <label className="font-semibold text-slate-700 dark:text-slate-300 block">Nameserver Configuration</label>
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={useDefaultNs}
                      onChange={(e) => setUseDefaultNs(e.target.checked)}
                      className="rounded text-emerald-600"
                    />
                    <span>Use Hostvra Default Anycast Nameservers ({defaultNameservers.join(', ')})</span>
                  </label>
                </div>

                {/* Registrant Contact */}
                <div className="space-y-2 pt-2 border-t border-slate-100 dark:border-slate-800">
                  <label className="font-semibold text-slate-700 dark:text-slate-300 block">Registrant Information</label>
                  <div className="grid grid-cols-2 gap-2">
                    <input
                      type="text"
                      placeholder="First Name"
                      value={orderRegistrant.first_name}
                      onChange={(e) => setOrderRegistrant({ ...orderRegistrant, first_name: e.target.value })}
                      className="px-2.5 py-1.5 rounded bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-700"
                      required
                    />
                    <input
                      type="text"
                      placeholder="Last Name"
                      value={orderRegistrant.last_name}
                      onChange={(e) => setOrderRegistrant({ ...orderRegistrant, last_name: e.target.value })}
                      className="px-2.5 py-1.5 rounded bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-700"
                      required
                    />
                  </div>
                  <div className="grid grid-cols-2 gap-2">
                    <input
                      type="email"
                      placeholder="Email"
                      value={orderRegistrant.email}
                      onChange={(e) => setOrderRegistrant({ ...orderRegistrant, email: e.target.value })}
                      className="px-2.5 py-1.5 rounded bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-700"
                      required
                    />
                    <input
                      type="text"
                      placeholder="Phone (e.g. 15551234567)"
                      value={orderRegistrant.phone}
                      onChange={(e) => setOrderRegistrant({ ...orderRegistrant, phone: e.target.value })}
                      className="px-2.5 py-1.5 rounded bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-700"
                      required
                    />
                  </div>
                </div>

                {/* Payment Gateway */}
                <div className="space-y-1 pt-2 border-t border-slate-100 dark:border-slate-800">
                  <label className="font-semibold text-slate-700 dark:text-slate-300">Payment Gateway</label>
                  <select
                    value={orderPaymentMethod}
                    onChange={(e) => setOrderPaymentMethod(e.target.value)}
                    className="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-700 font-semibold"
                  >
                    <option value="bkash">bKash (Instant Mobile Wallet)</option>
                    <option value="stripe">Stripe (Credit / Debit Card)</option>
                    <option value="paypal">PayPal</option>
                    <option value="balance">Account Credits</option>
                  </select>
                </div>

                <div className="flex items-center justify-end gap-2.5 pt-3 border-t border-slate-100 dark:border-slate-800">
                  <button
                    type="button"
                    onClick={() => setOrderModalOpen(false)}
                    className="px-4 py-2 rounded-lg border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-300 cursor-pointer"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={isPlacingOrder}
                    className="px-5 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold transition disabled:opacity-50 cursor-pointer"
                  >
                    {isPlacingOrder ? 'Creating Order...' : 'Confirm Order & Pay'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
