'use client';

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import { DashboardShell } from '@/components/DashboardShell';
import {
  Globe,
  DollarSign,
  TrendingUp,
  RefreshCw,
  CheckCircle2,
  AlertTriangle,
  AlertCircle,
  ShieldCheck,
  ChevronRight,
  Server,
  Layers,
  Settings,
  Edit2,
  Play,
  Clock,
  Search,
  Check,
  X,
  ExternalLink,
  Shield,
  Activity,
  Zap,
} from 'lucide-react';
import {
  DomainAdminMetrics,
  DomainAdminPrice,
  DomainOrder,
  ResellerClubTestResult,
  fetchDomainAdminMetrics,
  fetchDomainAdminPrices,
  fetchDomainAdminOrders,
  updateDomainAdminPrice,
  retryDomainAdminOrder,
  testResellerClubConnection,
  triggerDomainReconciliation,
} from '@/lib/api';

export default function AdminDomainsPage() {
  const [activeTab, setActiveTab] = useState<'prices' | 'orders' | 'connection'>('prices');
  const [loading, setLoading] = useState(true);
  const [metrics, setMetrics] = useState<DomainAdminMetrics | null>(null);
  const [prices, setPrices] = useState<DomainAdminPrice[]>([]);
  const [orders, setOrders] = useState<DomainOrder[]>([]);
  const [orderFilter, setOrderFilter] = useState<string>('all');
  const [tldSearchQuery, setTldSearchQuery] = useState('');

  // Notifications
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  // ResellerClub Connection Test State
  const [testingConnection, setTestingConnection] = useState(false);
  const [connectionResult, setConnectionResult] = useState<ResellerClubTestResult | null>(null);

  // Reconciliation State
  const [reconciling, setReconciling] = useState(false);

  // Edit Price Modal
  const [editingPrice, setEditingPrice] = useState<DomainAdminPrice | null>(null);
  const [savingPrice, setSavingPrice] = useState(false);
  const [priceForm, setPriceForm] = useState({
    register_price: 0,
    renew_price: 0,
    transfer_price: 0,
    cost_price: 0,
    enabled: true,
    is_popular: false,
  });

  // Retrying Order State
  const [retryingOrderId, setRetryingOrderId] = useState<string | null>(null);

  const loadData = async () => {
    try {
      setLoading(true);
      const [mRes, pRes, oRes] = await Promise.all([
        fetchDomainAdminMetrics().catch(() => null),
        fetchDomainAdminPrices().catch(() => null),
        fetchDomainAdminOrders().catch(() => null),
      ]);

      if (mRes?.success && mRes.data) {
        setMetrics(mRes.data);
      }
      if (pRes?.success && pRes.data) {
        setPrices(pRes.data);
      }
      if (oRes?.success && oRes.data) {
        setOrders(oRes.data);
      }
    } catch (err: any) {
      setErrorMessage(err.message || 'Failed to load domain administration data');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const handleTestConnection = async () => {
    try {
      setTestingConnection(true);
      setConnectionResult(null);
      setErrorMessage(null);
      const res = await testResellerClubConnection();
      if (res.success && res.data) {
        setConnectionResult(res.data);
        setSuccessMessage(`ResellerClub API connection verified (${res.data.latency_ms}ms)`);
      } else {
        setErrorMessage(res.error?.message || 'Connection test failed');
      }
    } catch (err: any) {
      setErrorMessage(err.message || 'Error testing connection');
    } finally {
      setTestingConnection(false);
    }
  };

  const handleReconciliation = async () => {
    try {
      setReconciling(true);
      setErrorMessage(null);
      const res = await triggerDomainReconciliation();
      if (res.success) {
        setSuccessMessage(res.data?.message || 'Domain reconciliation completed successfully');
        loadData();
      } else {
        setErrorMessage(res.error?.message || 'Reconciliation failed');
      }
    } catch (err: any) {
      setErrorMessage(err.message || 'Reconciliation error');
    } finally {
      setReconciling(false);
    }
  };

  const handleOpenEditPrice = (price: DomainAdminPrice) => {
    setEditingPrice(price);
    setPriceForm({
      register_price: price.register_price,
      renew_price: price.renew_price,
      transfer_price: price.transfer_price,
      cost_price: price.cost_price,
      enabled: price.enabled,
      is_popular: price.is_popular,
    });
  };

  const handleSavePrice = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingPrice) return;
    try {
      setSavingPrice(true);
      setErrorMessage(null);
      const res = await updateDomainAdminPrice(editingPrice.id, priceForm);
      if (res.success && res.data) {
        setPrices((prev) => prev.map((p) => (p.id === editingPrice.id ? res.data! : p)));
        setSuccessMessage(`Pricing for ${editingPrice.tld} updated successfully`);
        setEditingPrice(null);
      } else {
        setErrorMessage(res.error?.message || 'Failed to update pricing');
      }
    } catch (err: any) {
      setErrorMessage(err.message || 'Error updating price');
    } finally {
      setSavingPrice(false);
    }
  };

  const handleRetryOrder = async (orderId: string) => {
    try {
      setRetryingOrderId(orderId);
      setErrorMessage(null);
      const res = await retryDomainAdminOrder(orderId);
      if (res.success) {
        setSuccessMessage('Registration retry queued and dispatched');
        loadData();
      } else {
        setErrorMessage(res.error?.message || 'Retry failed');
      }
    } catch (err: any) {
      setErrorMessage(err.message || 'Error retrying order');
    } finally {
      setRetryingOrderId(null);
    }
  };

  const filteredPrices = prices.filter((p) =>
    p.tld.toLowerCase().includes(tldSearchQuery.toLowerCase())
  );

  const filteredOrders = orders.filter((o) => {
    if (orderFilter === 'all') return true;
    return o.status === orderFilter;
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
              <Link href="/domains" className="hover:text-slate-900 dark:hover:text-white transition">
                Domains
              </Link>
              <ChevronRight className="w-3.5 h-3.5 text-slate-400" />
              <span className="text-slate-900 dark:text-white font-semibold">Reseller Admin</span>
            </nav>
            <h1 className="text-2xl font-bold text-slate-900 dark:text-white tracking-tight flex items-center gap-2.5">
              <Globe className="w-6 h-6 text-emerald-600" />
              Domain Reseller Administration
            </h1>
            <p className="text-xs sm:text-sm text-slate-500 dark:text-slate-400 mt-0.5">
              Manage ResellerClub upstream integration, wholesale profit margins, TLD pricing, and provisioning orders.
            </p>
          </div>

          <div className="flex items-center gap-2.5 flex-wrap">
            <button
              onClick={handleTestConnection}
              disabled={testingConnection}
              className="inline-flex items-center justify-center gap-2 px-3.5 py-2 rounded-lg bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 hover:border-slate-300 dark:hover:border-slate-700 text-slate-700 dark:text-slate-200 text-xs font-semibold shadow-2xs transition active:scale-98 disabled:opacity-60 cursor-pointer"
            >
              <Activity className={`w-3.5 h-3.5 ${testingConnection ? 'animate-spin text-emerald-500' : 'text-emerald-500'}`} />
              <span>{testingConnection ? 'Testing...' : 'Test Connection'}</span>
            </button>

            <button
              onClick={handleReconciliation}
              disabled={reconciling}
              className="inline-flex items-center justify-center gap-2 px-3.5 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white text-xs font-semibold shadow-xs transition active:scale-98 disabled:opacity-60 cursor-pointer"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${reconciling ? 'animate-spin' : ''}`} />
              <span>{reconciling ? 'Syncing...' : 'Sync & Reconcile'}</span>
            </button>
          </div>
        </div>

        {/* Notifications */}
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

        {errorMessage && (
          <div className="p-3.5 rounded-lg bg-rose-50 dark:bg-rose-950/30 border border-rose-200 dark:border-rose-800 text-xs font-medium text-rose-700 dark:text-rose-400 flex items-center justify-between animate-fadeIn">
            <div className="flex items-center gap-2">
              <AlertCircle className="w-4 h-4 flex-shrink-0" />
              <span>{errorMessage}</span>
            </div>
            <button onClick={() => setErrorMessage(null)} className="text-rose-700 dark:text-rose-400 hover:opacity-75">
              <X className="w-4 h-4" />
            </button>
          </div>
        )}

        {/* ResellerClub Connection Status Box */}
        {connectionResult && (
          <div className="p-4 rounded-xl bg-slate-900 text-white border border-slate-800 flex flex-col sm:flex-row sm:items-center justify-between gap-4 shadow-sm">
            <div className="flex items-center gap-3">
              <div className="w-9 h-9 rounded-lg bg-emerald-500/20 border border-emerald-500/30 flex items-center justify-center text-emerald-400">
                <ShieldCheck className="w-5 h-5" />
              </div>
              <div>
                <div className="flex items-center gap-2">
                  <span className="text-sm font-bold">{connectionResult.message}</span>
                  <span className="text-[10px] uppercase font-bold tracking-wider px-2 py-0.5 rounded-full bg-emerald-500/20 text-emerald-300 border border-emerald-500/30">
                    {connectionResult.mode}
                  </span>
                </div>
                <div className="text-xs text-slate-400 mt-0.5 flex items-center gap-3">
                  <span>Reseller ID: <strong className="font-mono text-slate-200">{connectionResult.reseller_id}</strong></span>
                  <span>•</span>
                  <span>Endpoint: <strong className="font-mono text-slate-200">{connectionResult.base_url}</strong></span>
                </div>
              </div>
            </div>
            <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-slate-800 text-xs font-mono text-emerald-400 border border-slate-700">
              <Zap className="w-3.5 h-3.5 text-amber-400" />
              <span>{connectionResult.latency_ms} ms</span>
            </div>
          </div>
        )}

        {/* Metrics Grid */}
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3">
          <div className="p-4 rounded-xl bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 shadow-2xs">
            <div className="text-[11px] font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">Total Domains</div>
            <div className="text-xl font-bold text-slate-900 dark:text-white mt-1">
              {metrics?.total_domains ?? 0}
            </div>
            <div className="text-[10px] text-slate-400 mt-0.5">Under management</div>
          </div>

          <div className="p-4 rounded-xl bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 shadow-2xs">
            <div className="text-[11px] font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">Active Domains</div>
            <div className="text-xl font-bold text-emerald-600 dark:text-emerald-400 mt-1">
              {metrics?.active_domains ?? 0}
            </div>
            <div className="text-[10px] text-slate-400 mt-0.5">Live & resolving</div>
          </div>

          <div className="p-4 rounded-xl bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 shadow-2xs">
            <div className="text-[11px] font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">Expiring in 30d</div>
            <div className={`text-xl font-bold mt-1 ${(metrics?.expiring_30_days ?? 0) > 0 ? 'text-amber-500' : 'text-slate-900 dark:text-white'}`}>
              {metrics?.expiring_30_days ?? 0}
            </div>
            <div className="text-[10px] text-slate-400 mt-0.5">Renewal notices sent</div>
          </div>

          <div className="p-4 rounded-xl bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 shadow-2xs">
            <div className="text-[11px] font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">Total Revenue</div>
            <div className="text-xl font-bold text-slate-900 dark:text-white mt-1">
              ${(metrics?.total_revenue ?? 0).toFixed(2)}
            </div>
            <div className="text-[10px] text-slate-400 mt-0.5">Customer payments</div>
          </div>

          <div className="p-4 rounded-xl bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 shadow-2xs">
            <div className="text-[11px] font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">Wholesale Cost</div>
            <div className="text-xl font-bold text-slate-600 dark:text-slate-300 mt-1">
              ${(metrics?.total_cost ?? 0).toFixed(2)}
            </div>
            <div className="text-[10px] text-slate-400 mt-0.5">ResellerClub charges</div>
          </div>

          <div className="p-4 rounded-xl bg-emerald-50/50 dark:bg-emerald-950/20 border border-emerald-200 dark:border-emerald-800/60 shadow-2xs">
            <div className="text-[11px] font-semibold text-emerald-700 dark:text-emerald-400 uppercase tracking-wider">Gross Profit</div>
            <div className="text-xl font-bold text-emerald-600 dark:text-emerald-400 mt-1">
              ${(metrics?.gross_profit ?? 0).toFixed(2)}
            </div>
            <div className="text-[10px] text-emerald-600/80 dark:text-emerald-400/80 mt-0.5">
              Net retail margin
            </div>
          </div>
        </div>

        {/* Tab Navigation */}
        <div className="flex border-b border-slate-200 dark:border-slate-800 gap-6 text-sm font-semibold">
          <button
            onClick={() => setActiveTab('prices')}
            className={`pb-3 border-b-2 transition-colors cursor-pointer ${
              activeTab === 'prices'
                ? 'border-emerald-600 text-emerald-600 dark:text-emerald-400'
                : 'border-transparent text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white'
            }`}
          >
            TLD Pricing & Margins ({prices.length})
          </button>
          <button
            onClick={() => setActiveTab('orders')}
            className={`pb-3 border-b-2 transition-colors cursor-pointer ${
              activeTab === 'orders'
                ? 'border-emerald-600 text-emerald-600 dark:text-emerald-400'
                : 'border-transparent text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white'
            }`}
          >
            Orders & Provisioning Log ({orders.length})
          </button>
          <button
            onClick={() => setActiveTab('connection')}
            className={`pb-3 border-b-2 transition-colors cursor-pointer ${
              activeTab === 'connection'
                ? 'border-emerald-600 text-emerald-600 dark:text-emerald-400'
                : 'border-transparent text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white'
            }`}
          >
            Registrar Integration Guide
          </button>
        </div>

        {/* ==================================================== */}
        {/* TAB 1: TLD PRICING & MARGINS */}
        {/* ==================================================== */}
        {activeTab === 'prices' && (
          <div className="space-y-4">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
              <div className="relative flex-1 max-w-sm">
                <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
                <input
                  type="text"
                  placeholder="Filter TLDs (e.g. .com, .net, .io)..."
                  value={tldSearchQuery}
                  onChange={(e) => setTldSearchQuery(e.target.value)}
                  className="w-full pl-9 pr-3 py-1.5 rounded-lg bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 text-xs text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:border-emerald-500"
                />
              </div>
              <div className="text-xs text-slate-500 dark:text-slate-400">
                Showing {filteredPrices.length} of {prices.length} TLDs. Wholesale costs are hidden from customers.
              </div>
            </div>

            <div className="bg-white dark:bg-[#131B2E] rounded-xl border border-slate-200 dark:border-slate-800 overflow-hidden shadow-2xs">
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead className="bg-slate-50 dark:bg-[#0B1120] border-b border-slate-200 dark:border-slate-800 text-slate-500 dark:text-slate-400 uppercase tracking-wider font-semibold">
                    <tr>
                      <th className="py-3 px-4">TLD</th>
                      <th className="py-3 px-4">Status</th>
                      <th className="py-3 px-4">Wholesale Cost (USD)</th>
                      <th className="py-3 px-4">Retail Price (USD)</th>
                      <th className="py-3 px-4">Renewal Price (USD)</th>
                      <th className="py-3 px-4">Transfer Price (USD)</th>
                      <th className="py-3 px-4">Gross Margin</th>
                      <th className="py-3 px-4 text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                    {filteredPrices.map((price) => {
                      const margin = price.register_price - price.cost_price;
                      const marginPct = price.cost_price > 0 ? (margin / price.cost_price) * 100 : 0;
                      return (
                        <tr key={price.id} className="hover:bg-slate-50/50 dark:hover:bg-slate-800/50 transition-colors">
                          <td className="py-3 px-4 font-mono font-bold text-slate-900 dark:text-white">
                            <span className="inline-flex items-center gap-1.5">
                              {price.tld}
                              {price.is_popular && (
                                <span className="text-[10px] px-1.5 py-0.5 rounded bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 font-sans font-semibold">
                                  Popular
                                </span>
                              )}
                            </span>
                          </td>
                          <td className="py-3 px-4">
                            <span
                              className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-bold ${
                                price.enabled
                                  ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
                                  : 'bg-slate-100 dark:bg-slate-800 text-slate-500'
                              }`}
                            >
                              {price.enabled ? 'Enabled' : 'Disabled'}
                            </span>
                          </td>
                          <td className="py-3 px-4 font-mono font-medium text-slate-500 dark:text-slate-400">
                            ${price.cost_price.toFixed(2)}
                          </td>
                          <td className="py-3 px-4 font-mono font-bold text-slate-900 dark:text-white">
                            ${price.register_price.toFixed(2)}
                          </td>
                          <td className="py-3 px-4 font-mono text-slate-700 dark:text-slate-300">
                            ${price.renew_price.toFixed(2)}
                          </td>
                          <td className="py-3 px-4 font-mono text-slate-700 dark:text-slate-300">
                            ${price.transfer_price.toFixed(2)}
                          </td>
                          <td className="py-3 px-4">
                            <span className={`font-mono font-bold ${margin >= 0 ? 'text-emerald-600 dark:text-emerald-400' : 'text-rose-600'}`}>
                              +${margin.toFixed(2)} ({marginPct.toFixed(0)}%)
                            </span>
                          </td>
                          <td className="py-3 px-4 text-right">
                            <button
                              onClick={() => handleOpenEditPrice(price)}
                              className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded bg-slate-100 hover:bg-slate-200 dark:bg-slate-800 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-300 font-medium transition cursor-pointer"
                            >
                              <Edit2 className="w-3 h-3" />
                              <span>Edit</span>
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

        {/* ==================================================== */}
        {/* TAB 2: ORDERS & PROVISIONING LOG */}
        {/* ==================================================== */}
        {activeTab === 'orders' && (
          <div className="space-y-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div className="flex items-center gap-2">
                {['all', 'completed', 'pending_payment', 'processing', 'failed'].map((st) => (
                  <button
                    key={st}
                    onClick={() => setOrderFilter(st)}
                    className={`px-3 py-1 rounded-lg text-xs font-semibold capitalize transition cursor-pointer ${
                      orderFilter === st
                        ? 'bg-emerald-600 text-white'
                        : 'bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 text-slate-600 dark:text-slate-300 hover:bg-slate-50'
                    }`}
                  >
                    {st.replace('_', ' ')}
                  </button>
                ))}
              </div>
              <div className="text-xs text-slate-500">
                Showing {filteredOrders.length} orders
              </div>
            </div>

            <div className="bg-white dark:bg-[#131B2E] rounded-xl border border-slate-200 dark:border-slate-800 overflow-hidden shadow-2xs">
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead className="bg-slate-50 dark:bg-[#0B1120] border-b border-slate-200 dark:border-slate-800 text-slate-500 dark:text-slate-400 uppercase tracking-wider font-semibold">
                    <tr>
                      <th className="py-3 px-4">Order ID</th>
                      <th className="py-3 px-4">Domain Name</th>
                      <th className="py-3 px-4">Action</th>
                      <th className="py-3 px-4">Years</th>
                      <th className="py-3 px-4">Amount</th>
                      <th className="py-3 px-4">Payment</th>
                      <th className="py-3 px-4">Provisioning Status</th>
                      <th className="py-3 px-4">Reseller Order ID</th>
                      <th className="py-3 px-4 text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                    {filteredOrders.length === 0 ? (
                      <tr>
                        <td colSpan={9} className="py-8 text-center text-slate-400">
                          No domain orders found for this status.
                        </td>
                      </tr>
                    ) : (
                      filteredOrders.map((order) => (
                        <tr key={order.id} className="hover:bg-slate-50/50 dark:hover:bg-slate-800/50 transition-colors">
                          <td className="py-3 px-4 font-mono text-slate-500">
                            {order.id.substring(0, 8)}...
                          </td>
                          <td className="py-3 px-4 font-bold text-slate-900 dark:text-white">
                            {order.domain_name}
                          </td>
                          <td className="py-3 px-4 uppercase text-[10px] font-bold text-slate-500">
                            {order.action}
                          </td>
                          <td className="py-3 px-4">{order.years} yr</td>
                          <td className="py-3 px-4 font-mono font-bold text-slate-900 dark:text-white">
                            ${order.amount.toFixed(2)}
                          </td>
                          <td className="py-3 px-4">
                            <span
                              className={`px-2 py-0.5 rounded text-[10px] font-bold uppercase ${
                                order.payment_status === 'paid'
                                  ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
                                  : 'bg-amber-500/10 text-amber-600 dark:text-amber-400'
                              }`}
                            >
                              {order.payment_status}
                            </span>
                          </td>
                          <td className="py-3 px-4">
                            <span
                              className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[10px] font-bold uppercase ${
                                order.status === 'completed'
                                  ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
                                  : order.status === 'failed'
                                  ? 'bg-rose-500/10 text-rose-600 dark:text-rose-400'
                                  : order.status === 'processing'
                                  ? 'bg-blue-500/10 text-blue-600 dark:text-blue-400'
                                  : 'bg-slate-100 dark:bg-slate-800 text-slate-500'
                              }`}
                            >
                              {order.status}
                            </span>
                            {order.error_message && (
                              <div className="text-[10px] text-rose-500 mt-1 max-w-xs truncate" title={order.error_message}>
                                {order.error_message}
                              </div>
                            )}
                          </td>
                          <td className="py-3 px-4 font-mono text-slate-500">
                            {order.reseller_order_id || '—'}
                          </td>
                          <td className="py-3 px-4 text-right">
                            {order.status === 'failed' && (
                              <button
                                onClick={() => handleRetryOrder(order.id)}
                                disabled={retryingOrderId === order.id}
                                className="inline-flex items-center gap-1 px-2.5 py-1 rounded bg-amber-50 hover:bg-amber-100 text-amber-700 border border-amber-200 text-[11px] font-bold transition cursor-pointer disabled:opacity-50"
                              >
                                <Play className="w-3 h-3" />
                                <span>{retryingOrderId === order.id ? 'Retrying...' : 'Retry'}</span>
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
          </div>
        )}

        {/* ==================================================== */}
        {/* TAB 3: REGISTRAR INTEGRATION GUIDE */}
        {/* ==================================================== */}
        {activeTab === 'connection' && (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="p-6 rounded-xl bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 space-y-4 shadow-2xs">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-lg bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
                  <Shield className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="text-base font-bold text-slate-900 dark:text-white">
                    ResellerClub Sandbox vs Production
                  </h3>
                  <p className="text-xs text-slate-500 dark:text-slate-400">
                    Hostvra strict zero-mock deployment architecture
                  </p>
                </div>
              </div>

              <div className="space-y-3 text-xs text-slate-600 dark:text-slate-300 leading-relaxed">
                <p>
                  Hostvra integrates with the real <strong>ResellerClub HTTP API</strong> (LogicBoxes platform). The system supports seamless switching between Sandbox and Production without code changes:
                </p>

                <div className="p-3.5 rounded-lg bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-800 space-y-2 font-mono text-[11px]">
                  <div><strong className="text-emerald-600">Sandbox API:</strong> https://test.httpapi.com/api/</div>
                  <div><strong className="text-blue-600">Production API:</strong> https://httpapi.com/api/</div>
                </div>

                <div className="p-3.5 rounded-lg bg-amber-50 dark:bg-amber-950/20 border border-amber-200 dark:border-amber-800 text-amber-800 dark:text-amber-300 text-xs">
                  <strong className="font-bold flex items-center gap-1.5 mb-1">
                    <AlertTriangle className="w-4 h-4 text-amber-600" />
                    IP Whitelist Requirement:
                  </strong>
                  ResellerClub requires you to whitelist your server IP in your ResellerClub Control Panel under <em>Settings &gt; API &gt; Authorized IP Addresses</em>. Requests from unlisted IPs will receive <code className="font-mono">IP_NOT_AUTHORIZED</code>.
                </div>
              </div>
            </div>

            <div className="p-6 rounded-xl bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 space-y-4 shadow-2xs">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-lg bg-blue-500/10 border border-blue-500/20 flex items-center justify-center text-blue-600 dark:text-blue-400">
                  <Settings className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="text-base font-bold text-slate-900 dark:text-white">
                    Environment Variables
                  </h3>
                  <p className="text-xs text-slate-500 dark:text-slate-400">
                    Configured in server .env
                  </p>
                </div>
              </div>

              <div className="p-4 rounded-lg bg-slate-900 text-slate-200 font-mono text-[11px] space-y-1.5 overflow-x-auto">
                <div className="text-slate-500"># ResellerClub Provider Config</div>
                <div>DOMAIN_PROVIDER=resellerclub</div>
                <div>RESELLERCLUB_MODE=sandbox <span className="text-slate-500"># or production</span></div>
                <div>RESELLERCLUB_RESELLER_ID=your_reseller_id</div>
                <div>RESELLERCLUB_API_KEY=your_api_key</div>
                <div>RESELLERCLUB_TIMEOUT_SECONDS=20</div>
                <div className="text-slate-500 mt-2"># Automated Sync & Reconciliation</div>
                <div>DOMAIN_RECONCILE_INTERVAL_HOURS=6</div>
              </div>

              <p className="text-xs text-slate-500 dark:text-slate-400">
                In production mode, Hostvra strictly enforces fail-fast startup validation. If Reseller ID or API key is missing, the API refuses to launch to protect transaction integrity.
              </p>
            </div>
          </div>
        )}

        {/* ==================================================== */}
        {/* EDIT TLD PRICING MODAL */}
        {/* ==================================================== */}
        {editingPrice && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-xs p-4 animate-fadeIn">
            <div className="bg-white dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 rounded-2xl max-w-md w-full p-6 shadow-2xl space-y-5">
              <div className="flex items-center justify-between border-b border-slate-100 dark:border-slate-800 pb-3">
                <h3 className="text-lg font-bold text-slate-900 dark:text-white flex items-center gap-2">
                  <Globe className="w-5 h-5 text-emerald-600" />
                  Edit Pricing: <span className="font-mono text-emerald-600">{editingPrice.tld}</span>
                </h3>
                <button
                  onClick={() => setEditingPrice(null)}
                  className="text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 cursor-pointer"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              <form onSubmit={handleSavePrice} className="space-y-4 text-xs">
                <div>
                  <label className="block font-semibold text-slate-700 dark:text-slate-300 mb-1">
                    Wholesale Cost (USD) — Internal Only
                  </label>
                  <input
                    type="number"
                    step="0.01"
                    value={priceForm.cost_price}
                    onChange={(e) => setPriceForm({ ...priceForm, cost_price: parseFloat(e.target.value) || 0 })}
                    className="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-800 text-slate-900 dark:text-white focus:outline-none focus:border-emerald-500 font-mono"
                    required
                  />
                  <span className="text-[10px] text-slate-400">Actual amount ResellerClub debits from your wallet</span>
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block font-semibold text-slate-700 dark:text-slate-300 mb-1">
                      Customer Retail (USD)
                    </label>
                    <input
                      type="number"
                      step="0.01"
                      value={priceForm.register_price}
                      onChange={(e) => setPriceForm({ ...priceForm, register_price: parseFloat(e.target.value) || 0 })}
                      className="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-800 text-slate-900 dark:text-white focus:outline-none focus:border-emerald-500 font-mono font-bold"
                      required
                    />
                  </div>

                  <div>
                    <label className="block font-semibold text-slate-700 dark:text-slate-300 mb-1">
                      Renewal Price (USD)
                    </label>
                    <input
                      type="number"
                      step="0.01"
                      value={priceForm.renew_price}
                      onChange={(e) => setPriceForm({ ...priceForm, renew_price: parseFloat(e.target.value) || 0 })}
                      className="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-800 text-slate-900 dark:text-white focus:outline-none focus:border-emerald-500 font-mono"
                      required
                    />
                  </div>
                </div>

                <div>
                  <label className="block font-semibold text-slate-700 dark:text-slate-300 mb-1">
                    Transfer In Price (USD)
                  </label>
                  <input
                    type="number"
                    step="0.01"
                    value={priceForm.transfer_price}
                    onChange={(e) => setPriceForm({ ...priceForm, transfer_price: parseFloat(e.target.value) || 0 })}
                    className="w-full px-3 py-2 rounded-lg bg-slate-50 dark:bg-[#0B1120] border border-slate-200 dark:border-slate-800 text-slate-900 dark:text-white focus:outline-none focus:border-emerald-500 font-mono"
                    required
                  />
                </div>

                <div className="pt-2 flex items-center justify-between border-t border-slate-100 dark:border-slate-800">
                  <label className="flex items-center gap-2 font-semibold text-slate-700 dark:text-slate-300 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={priceForm.enabled}
                      onChange={(e) => setPriceForm({ ...priceForm, enabled: e.target.checked })}
                      className="rounded border-slate-300 text-emerald-600 focus:ring-emerald-500"
                    />
                    <span>TLD Enabled for Public Registration</span>
                  </label>
                </div>

                <div className="flex items-center justify-end gap-3 pt-3">
                  <button
                    type="button"
                    onClick={() => setEditingPrice(null)}
                    className="px-4 py-2 rounded-lg border border-slate-200 dark:border-slate-800 text-slate-600 dark:text-slate-300 hover:bg-slate-50 text-xs font-semibold cursor-pointer"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={savingPrice}
                    className="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white text-xs font-bold transition disabled:opacity-50 cursor-pointer"
                  >
                    {savingPrice ? 'Saving...' : 'Update Pricing'}
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
