'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { apiFetch } from '@/lib/api';
import {
  Coins,
  Shield,
  Wallet,
  Clock,
  CheckCircle2,
  AlertTriangle,
  RefreshCw,
  ExternalLink,
  Plus,
  BookOpen,
  Sliders,
  DollarSign,
  ArrowRight,
  Send,
  Zap,
} from 'lucide-react';

interface CryptoInvoice {
  id: string;
  invoice_code: string;
  user_id: string;
  organization_id: string;
  fiat_amount_usd: number;
  crypto_amount: number;
  currency_id: string;
  network_id: string;
  deposit_address: string;
  tx_hash?: string;
  confirmations: number;
  min_confirmations: number;
  amount_received: number;
  status: string;
  expires_at: string;
  created_at: string;
}

interface TreasuryWallet {
  id: string;
  network_id: string;
  name: string;
  wallet_type: string;
  address: string;
  balance: number;
  is_active: boolean;
  auto_sweep_enabled: boolean;
  sweep_threshold_usd: number;
}

interface LedgerEntry {
  id: string;
  invoice_id?: string;
  entry_type: string;
  amount: number;
  currency: string;
  network: string;
  from_address?: string;
  to_address: string;
  tx_hash?: string;
  confirmations?: number;
  notes: string;
  created_at: string;
}

export default function AdminCryptoPage() {
  const [activeTab, setActiveTab] = useState<'invoices' | 'treasuries' | 'ledger' | 'settings'>('invoices');
  const [invoices, setInvoices] = useState<CryptoInvoice[]>([]);
  const [treasuries, setTreasuries] = useState<TreasuryWallet[]>([]);
  const [ledgerEntries, setLedgerEntries] = useState<LedgerEntry[]>([]);
  const [settings, setSettings] = useState<Record<string, string>>({
    invoice_expiry_minutes: '30',
    rate_tolerance_percent: '1.0',
    auto_sweep_enabled: 'false',
    min_sweep_usd: '100',
  });

  const [loading, setLoading] = useState<boolean>(true);
  const [refreshing, setRefreshing] = useState<boolean>(false);
  const [savingSettings, setSavingSettings] = useState<boolean>(false);

  // New Treasury Modal State
  const [showAddTreasury, setShowAddTreasury] = useState<boolean>(false);
  const [newTreasury, setNewTreasury] = useState({
    network_id: 'tron',
    name: 'Trust Wallet Cold Treasury',
    address: '',
    is_active: true,
  });

  // 1. Fetch Invoices
  const fetchInvoices = useCallback(async () => {
    try {
      const res = await apiFetch<{ invoices: CryptoInvoice[]; total: number }>('/api/v1/admin/crypto/invoices');
      if (res.data?.invoices) {
        setInvoices(res.data.invoices);
      }
    } catch {
      // Graceful error
    }
  }, []);

  // 2. Fetch Treasuries
  const fetchTreasuries = useCallback(async () => {
    try {
      const res = await apiFetch<TreasuryWallet[]>('/api/v1/admin/crypto/treasuries');
      if (res.data) {
        setTreasuries(res.data);
      }
    } catch {
      // Graceful error
    }
  }, []);

  // 3. Fetch Ledger
  const fetchLedger = useCallback(async () => {
    try {
      const res = await apiFetch<{ entries: LedgerEntry[]; total: number }>('/api/v1/admin/crypto/ledger');
      if (res.data?.entries) {
        setLedgerEntries(res.data.entries);
      }
    } catch {
      // Graceful error
    }
  }, []);

  // 4. Fetch Settings
  const fetchSettings = useCallback(async () => {
    try {
      const res = await apiFetch<Record<string, string>>('/api/v1/admin/crypto/settings');
      if (res.data) {
        setSettings((prev) => ({ ...prev, ...res.data }));
      }
    } catch {
      // Graceful error
    }
  }, []);

  const loadAll = useCallback(async () => {
    setLoading(true);
    await Promise.all([fetchInvoices(), fetchTreasuries(), fetchLedger(), fetchSettings()]);
    setLoading(false);
  }, [fetchInvoices, fetchTreasuries, fetchLedger, fetchSettings]);

  useEffect(() => {
    loadAll();
  }, [loadAll]);

  const handleRefresh = async () => {
    setRefreshing(true);
    await loadAll();
    setRefreshing(false);
  };

  const handleSaveSettings = async () => {
    setSavingSettings(true);
    try {
      await apiFetch('/api/v1/admin/crypto/settings', {
        method: 'PUT',
        body: JSON.stringify(settings),
      });
      alert('Crypto gateway settings saved successfully!');
    } catch (e: any) {
      alert(e.message || 'Failed to save settings');
    } finally {
      setSavingSettings(false);
    }
  };

  const handleSaveTreasury = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const res = await apiFetch('/api/v1/admin/crypto/treasuries', {
        method: 'POST',
        body: JSON.stringify(newTreasury),
      });
      if (res.data) {
        setShowAddTreasury(false);
        setNewTreasury({
          network_id: 'tron',
          name: 'Trust Wallet Cold Treasury',
          address: '',
          is_active: true,
        });
        await fetchTreasuries();
      } else {
        alert(res.error?.message || 'Failed to save treasury address');
      }
    } catch (e: any) {
      alert(e.message || 'Failed to save treasury address');
    }
  };

  const handleForceVerify = async (code: string) => {
    try {
      const res = await apiFetch<CryptoInvoice>(`/api/v1/admin/crypto/invoices/${code}/force-verify`, {
        method: 'POST',
      });
      if (res.data) {
        alert(`Invoice ${code} status: ${res.data.status} (Confirmations: ${res.data.confirmations})`);
        await fetchInvoices();
      }
    } catch (e: any) {
      alert(e.message || 'Force scan failed');
    }
  };

  const totalVolumeUSD = invoices
    .filter((inv) => inv.status === 'paid')
    .reduce((acc, curr) => acc + (curr.fiat_amount_usd || 0), 0);

  const pendingCount = invoices.filter((inv) => inv.status === 'pending' || inv.status === 'confirming').length;

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between flex-wrap gap-4">
        <div>
          <div className="flex items-center gap-2.5">
            <div className="p-2.5 bg-emerald-500/10 rounded-xl text-emerald-500 border border-emerald-500/20">
              <Coins className="w-6 h-6" />
            </div>
            <div>
              <h1 className="text-xl font-bold text-slate-900 dark:text-white flex items-center gap-2">
                Private In-House Crypto Gateway
                <span className="text-[10px] uppercase font-semibold px-2 py-0.5 rounded-full bg-emerald-100 dark:bg-emerald-950/60 text-emerald-700 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800">
                  Zero 3rd-Party Fees
                </span>
              </h1>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                Direct on-chain blockchain deposit verification &amp; Trust Wallet treasury orchestration
              </p>
            </div>
          </div>
        </div>

        <button
          onClick={handleRefresh}
          disabled={refreshing}
          className="inline-flex items-center gap-1.5 px-3 py-2 bg-white dark:bg-slate-800 border border-slate-300 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-700 rounded-xl text-xs font-semibold text-slate-700 dark:text-slate-200 transition cursor-pointer"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${refreshing ? 'animate-spin text-emerald-500' : ''}`} />
          <span>Refresh Ledger</span>
        </button>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="p-4 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700/80 rounded-2xl shadow-sm">
          <div className="text-[11px] uppercase font-semibold text-slate-500 dark:text-slate-400">Total Settled Volume</div>
          <div className="text-2xl font-black text-slate-900 dark:text-white mt-1">
            ${totalVolumeUSD.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
          </div>
          <div className="text-[11px] text-emerald-600 dark:text-emerald-400 font-medium mt-1">
            100% on-chain confirmed
          </div>
        </div>

        <div className="p-4 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700/80 rounded-2xl shadow-sm">
          <div className="text-[11px] uppercase font-semibold text-slate-500 dark:text-slate-400">Pending Invoices</div>
          <div className="text-2xl font-black text-amber-600 dark:text-amber-400 mt-1">
            {pendingCount}
          </div>
          <div className="text-[11px] text-slate-500 dark:text-slate-400 font-medium mt-1">
            Scanning every 20 seconds
          </div>
        </div>

        <div className="p-4 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700/80 rounded-2xl shadow-sm">
          <div className="text-[11px] uppercase font-semibold text-slate-500 dark:text-slate-400">Trust Wallet Treasuries</div>
          <div className="text-2xl font-black text-sky-600 dark:text-sky-400 mt-1">
            {treasuries.filter((t) => t.is_active).length} Active
          </div>
          <div className="text-[11px] text-slate-500 dark:text-slate-400 font-medium mt-1">
            Cold destination addresses
          </div>
        </div>

        <div className="p-4 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700/80 rounded-2xl shadow-sm">
          <div className="text-[11px] uppercase font-semibold text-slate-500 dark:text-slate-400">Gateway Fees Incurred</div>
          <div className="text-2xl font-black text-emerald-600 dark:text-emerald-400 mt-1">
            $0.00
          </div>
          <div className="text-[11px] text-slate-500 dark:text-slate-400 font-medium mt-1">
            Zero processor commission
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="border-b border-slate-200 dark:border-slate-700 flex gap-2">
        <button
          onClick={() => setActiveTab('invoices')}
          className={`px-4 py-2.5 font-semibold text-xs border-b-2 transition cursor-pointer flex items-center gap-1.5 ${
            activeTab === 'invoices'
              ? 'border-emerald-600 text-emerald-600 dark:text-emerald-400'
              : 'border-transparent text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
          }`}
        >
          <Clock className="w-4 h-4" />
          <span>Invoices ({invoices.length})</span>
        </button>

        <button
          onClick={() => setActiveTab('treasuries')}
          className={`px-4 py-2.5 font-semibold text-xs border-b-2 transition cursor-pointer flex items-center gap-1.5 ${
            activeTab === 'treasuries'
              ? 'border-emerald-600 text-emerald-600 dark:text-emerald-400'
              : 'border-transparent text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
          }`}
        >
          <Wallet className="w-4 h-4" />
          <span>Trust Wallet Treasuries ({treasuries.length})</span>
        </button>

        <button
          onClick={() => setActiveTab('ledger')}
          className={`px-4 py-2.5 font-semibold text-xs border-b-2 transition cursor-pointer flex items-center gap-1.5 ${
            activeTab === 'ledger'
              ? 'border-emerald-600 text-emerald-600 dark:text-emerald-400'
              : 'border-transparent text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
          }`}
        >
          <BookOpen className="w-4 h-4" />
          <span>Double-Entry Ledger</span>
        </button>

        <button
          onClick={() => setActiveTab('settings')}
          className={`px-4 py-2.5 font-semibold text-xs border-b-2 transition cursor-pointer flex items-center gap-1.5 ${
            activeTab === 'settings'
              ? 'border-emerald-600 text-emerald-600 dark:text-emerald-400'
              : 'border-transparent text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
          }`}
        >
          <Sliders className="w-4 h-4" />
          <span>Gateway Rules &amp; Sweep</span>
        </button>
      </div>

      {/* Tab 1: Invoices */}
      {activeTab === 'invoices' && (
        <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700/80 rounded-2xl overflow-hidden shadow-sm">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="bg-slate-50 dark:bg-slate-900/60 border-b border-slate-200 dark:border-slate-700/80 text-slate-500 uppercase font-semibold">
                <tr>
                  <th className="p-3.5">Invoice Code</th>
                  <th className="p-3.5">Amount (USD / Crypto)</th>
                  <th className="p-3.5">Network &amp; Currency</th>
                  <th className="p-3.5">Deposit Address</th>
                  <th className="p-3.5">Confirmations</th>
                  <th className="p-3.5">Status</th>
                  <th className="p-3.5 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200 dark:divide-slate-700/60">
                {invoices.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="p-8 text-center text-slate-500">
                      No crypto invoices recorded yet.
                    </td>
                  </tr>
                ) : (
                  invoices.map((inv) => (
                    <tr key={inv.id} className="hover:bg-slate-50/50 dark:hover:bg-slate-700/30 transition">
                      <td className="p-3.5 font-mono font-bold text-slate-900 dark:text-white">
                        {inv.invoice_code}
                      </td>
                      <td className="p-3.5">
                        <div className="font-bold text-slate-900 dark:text-white">
                          ${inv.fiat_amount_usd.toFixed(2)}
                        </div>
                        <div className="text-[11px] font-mono text-emerald-600 dark:text-emerald-400">
                          {inv.crypto_amount} {inv.currency_id}
                        </div>
                      </td>
                      <td className="p-3.5">
                        <span className="px-2 py-0.5 rounded-full bg-slate-100 dark:bg-slate-700 font-medium uppercase text-[10px]">
                          {inv.network_id}
                        </span>
                      </td>
                      <td className="p-3.5 font-mono text-[11px] text-slate-600 dark:text-slate-300 max-w-xs truncate" title={inv.deposit_address}>
                        {inv.deposit_address}
                      </td>
                      <td className="p-3.5 font-mono">
                        {inv.confirmations} / {inv.min_confirmations}
                      </td>
                      <td className="p-3.5">
                        <span
                          className={`inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-[10px] font-bold uppercase ${
                            inv.status === 'paid'
                              ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-300'
                              : inv.status === 'confirming'
                              ? 'bg-sky-100 text-sky-800 dark:bg-sky-900/40 dark:text-sky-300'
                              : inv.status === 'expired'
                              ? 'bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-300'
                              : 'bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300'
                          }`}
                        >
                          {inv.status}
                        </span>
                      </td>
                      <td className="p-3.5 text-right space-x-2">
                        {inv.tx_hash && (
                          <a
                            href={`https://tronscan.org/#/transaction/${inv.tx_hash}`}
                            target="_blank"
                            rel="noreferrer"
                            className="inline-flex items-center gap-1 text-[11px] text-emerald-600 hover:text-emerald-700 underline"
                          >
                            <span>Explorer</span>
                            <ExternalLink className="w-3 h-3" />
                          </a>
                        )}
                        {inv.status !== 'paid' && (
                          <button
                            onClick={() => handleForceVerify(inv.invoice_code)}
                            className="px-2 py-1 bg-slate-100 dark:bg-slate-700 hover:bg-slate-200 dark:hover:bg-slate-600 rounded text-[11px] font-medium transition cursor-pointer"
                          >
                            Rescan
                          </button>
                        )}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Tab 2: Treasuries */}
      {activeTab === 'treasuries' && (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-bold text-slate-900 dark:text-white">
              Approved Trust Wallet Destinations
            </h2>
            <button
              onClick={() => setShowAddTreasury(true)}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded-xl text-xs font-semibold transition cursor-pointer shadow-sm"
            >
              <Plus className="w-3.5 h-3.5" />
              <span>Add Treasury Wallet</span>
            </button>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {treasuries.map((tw) => (
              <div
                key={tw.id}
                className="p-5 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700/80 rounded-2xl shadow-sm space-y-3"
              >
                <div className="flex items-center justify-between">
                  <span className="text-[10px] uppercase font-bold px-2 py-0.5 rounded-full bg-slate-100 dark:bg-slate-700 text-slate-700 dark:text-slate-300">
                    {tw.network_id}
                  </span>
                  <span className={`text-[11px] font-semibold ${tw.is_active ? 'text-emerald-500' : 'text-slate-400'}`}>
                    {tw.is_active ? 'Active Destination' : 'Disabled'}
                  </span>
                </div>

                <div>
                  <h3 className="text-sm font-bold text-slate-900 dark:text-white">{tw.name}</h3>
                  <div className="font-mono text-xs text-slate-500 break-all select-all mt-1">
                    {tw.address}
                  </div>
                </div>

                <div className="pt-2 border-t border-slate-100 dark:border-slate-700/60 flex items-center justify-between text-xs">
                  <span className="text-slate-500">Auto-Sweep:</span>
                  <span className="font-semibold text-slate-800 dark:text-slate-200">
                    {tw.auto_sweep_enabled ? 'Enabled' : 'Manual Approval'}
                  </span>
                </div>
              </div>
            ))}
          </div>

          {/* Add Treasury Modal */}
          {showAddTreasury && (
            <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
              <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-2xl max-w-md w-full p-6 space-y-4 shadow-2xl">
                <h3 className="text-base font-bold text-slate-900 dark:text-white">
                  Add Trust Wallet Treasury Address
                </h3>
                <form onSubmit={handleSaveTreasury} className="space-y-4 text-xs">
                  <div>
                    <label className="block font-medium text-slate-700 dark:text-slate-300 mb-1">
                      Network:
                    </label>
                    <select
                      value={newTreasury.network_id}
                      onChange={(e) => setNewTreasury({ ...newTreasury, network_id: e.target.value })}
                      className="w-full p-2.5 rounded-xl border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white"
                    >
                      <option value="tron">TRON (TRC20)</option>
                      <option value="bsc">BNB Smart Chain (BEP20)</option>
                      <option value="ethereum">Ethereum (ERC20)</option>
                      <option value="polygon">Polygon</option>
                      <option value="bitcoin">Bitcoin</option>
                    </select>
                  </div>

                  <div>
                    <label className="block font-medium text-slate-700 dark:text-slate-300 mb-1">
                      Label / Name:
                    </label>
                    <input
                      type="text"
                      required
                      value={newTreasury.name}
                      onChange={(e) => setNewTreasury({ ...newTreasury, name: e.target.value })}
                      className="w-full p-2.5 rounded-xl border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white"
                    />
                  </div>

                  <div>
                    <label className="block font-medium text-slate-700 dark:text-slate-300 mb-1">
                      Trust Wallet Recipient Address:
                    </label>
                    <input
                      type="text"
                      required
                      placeholder="e.g. TYDzsYUE... or 0x..."
                      value={newTreasury.address}
                      onChange={(e) => setNewTreasury({ ...newTreasury, address: e.target.value })}
                      className="w-full p-2.5 rounded-xl border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white font-mono"
                    />
                  </div>

                  <div className="flex justify-end gap-2 pt-2">
                    <button
                      type="button"
                      onClick={() => setShowAddTreasury(false)}
                      className="px-4 py-2 bg-slate-100 dark:bg-slate-700 rounded-xl font-medium text-slate-700 dark:text-slate-300 cursor-pointer"
                    >
                      Cancel
                    </button>
                    <button
                      type="submit"
                      className="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 text-white rounded-xl font-bold cursor-pointer"
                    >
                      Save Destination
                    </button>
                  </div>
                </form>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Tab 3: Double-Entry Ledger */}
      {activeTab === 'ledger' && (
        <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700/80 rounded-2xl overflow-hidden shadow-sm">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="bg-slate-50 dark:bg-slate-900/60 border-b border-slate-200 dark:border-slate-700/80 text-slate-500 uppercase font-semibold">
                <tr>
                  <th className="p-3.5">Timestamp</th>
                  <th className="p-3.5">Entry Type</th>
                  <th className="p-3.5">Amount</th>
                  <th className="p-3.5">Network &amp; Currency</th>
                  <th className="p-3.5">Destination</th>
                  <th className="p-3.5">Tx Hash</th>
                  <th className="p-3.5">Notes</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200 dark:divide-slate-700/60 font-mono text-[11px]">
                {ledgerEntries.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="p-8 text-center text-slate-500 font-sans">
                      No ledger transactions recorded yet.
                    </td>
                  </tr>
                ) : (
                  ledgerEntries.map((l) => (
                    <tr key={l.id} className="hover:bg-slate-50/50 dark:hover:bg-slate-700/30">
                      <td className="p-3.5 text-slate-500">
                        {new Date(l.created_at).toLocaleString()}
                      </td>
                      <td className="p-3.5 font-bold uppercase text-slate-800 dark:text-slate-200">
                        {l.entry_type}
                      </td>
                      <td className="p-3.5 font-bold text-emerald-600 dark:text-emerald-400">
                        +{l.amount}
                      </td>
                      <td className="p-3.5 uppercase">{l.network} ({l.currency})</td>
                      <td className="p-3.5 truncate max-w-xs">{l.to_address}</td>
                      <td className="p-3.5 truncate max-w-xs">{l.tx_hash || '—'}</td>
                      <td className="p-3.5 font-sans text-slate-500 truncate max-w-xs">{l.notes}</td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Tab 4: Settings */}
      {activeTab === 'settings' && (
        <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700/80 rounded-2xl p-6 max-w-2xl shadow-sm space-y-6">
          <h2 className="text-base font-bold text-slate-900 dark:text-white">
            Crypto Gateway Policies &amp; Auto-Sweep Rules
          </h2>

          <div className="space-y-4 text-xs">
            <div>
              <label className="block font-semibold text-slate-700 dark:text-slate-300 mb-1">
                Invoice Expiration Window (Minutes):
              </label>
              <input
                type="number"
                value={settings.invoice_expiry_minutes || '30'}
                onChange={(e) => setSettings({ ...settings, invoice_expiry_minutes: e.target.value })}
                className="w-full p-2.5 rounded-xl border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white"
              />
              <p className="text-[11px] text-slate-400 mt-1">
                Exchange rates and deposit addresses remain locked for this duration.
              </p>
            </div>

            <div>
              <label className="block font-semibold text-slate-700 dark:text-slate-300 mb-1">
                Exchange Rate / Underpayment Tolerance (%):
              </label>
              <input
                type="text"
                value={settings.rate_tolerance_percent || '1.0'}
                onChange={(e) => setSettings({ ...settings, rate_tolerance_percent: e.target.value })}
                className="w-full p-2.5 rounded-xl border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white"
              />
              <p className="text-[11px] text-slate-400 mt-1">
                Accept incoming deposits if received amount is within this percentage (e.g. 1.0% to tolerate minor client gas deductions).
              </p>
            </div>

            <div>
              <label className="block font-semibold text-slate-700 dark:text-slate-300 mb-1">
                Automatic Sweep to Trust Wallet:
              </label>
              <select
                value={settings.auto_sweep_enabled || 'false'}
                onChange={(e) => setSettings({ ...settings, auto_sweep_enabled: e.target.value })}
                className="w-full p-2.5 rounded-xl border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white"
              >
                <option value="false">Manual Admin Approval (Recommended)</option>
                <option value="true">Automatic Immediate Sweep</option>
              </select>
            </div>

            <div>
              <label className="block font-semibold text-slate-700 dark:text-slate-300 mb-1">
                Minimum Sweep Amount (USD):
              </label>
              <input
                type="number"
                value={settings.min_sweep_usd || '100'}
                onChange={(e) => setSettings({ ...settings, min_sweep_usd: e.target.value })}
                className="w-full p-2.5 rounded-xl border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white"
              />
              <p className="text-[11px] text-slate-400 mt-1">
                Prevents burning unnecessary network gas fees on small dust transactions.
              </p>
            </div>

            <button
              onClick={handleSaveSettings}
              disabled={savingSettings}
              className="px-6 py-2.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded-xl font-bold transition cursor-pointer flex items-center gap-2"
            >
              {savingSettings && <RefreshCw className="w-4 h-4 animate-spin" />}
              <span>Save Gateway Policies</span>
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
