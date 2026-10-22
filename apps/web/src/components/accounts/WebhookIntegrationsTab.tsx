'use client';

import React, { useState, useEffect } from 'react';
import {
  Webhook,
  Plus,
  RefreshCw,
  Copy,
  Check,
  Play,
  Trash2,
  CheckCircle2,
  AlertCircle,
  Eye,
  Key,
  Globe,
  Radio,
  Server,
  Layers,
  Code2,
  X,
  FileText,
} from 'lucide-react';
import {
  ProvisioningIntegration,
  ProvisioningWebhookLog,
  listProvisioningIntegrations,
  createProvisioningIntegration,
  deleteProvisioningIntegration,
  listProvisioningWebhookLogs,
  replayWebhookLog,
  Server as ServerType,
  HostingPlan,
} from '@/lib/api';

interface WebhookIntegrationsTabProps {
  servers: ServerType[];
  plans: HostingPlan[];
  showNotify: (type: 'success' | 'error', message: string) => void;
}

export function WebhookIntegrationsTab({ servers, plans, showNotify }: WebhookIntegrationsTabProps) {
  const [integrations, setIntegrations] = useState<ProvisioningIntegration[]>([]);
  const [logs, setLogs] = useState<ProvisioningWebhookLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);

  // Modals
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [inspectModalOpen, setInspectModalOpen] = useState(false);
  const [selectedLog, setSelectedLog] = useState<ProvisioningWebhookLog | null>(null);

  // Form State
  const [provider, setProvider] = useState<'whmcs' | 'blesta' | 'custom'>('whmcs');
  const [name, setName] = useState('');
  const [defaultServerId, setDefaultServerId] = useState('');
  const [defaultPlanId, setDefaultPlanId] = useState('');

  // Newly created integration key notice
  const [newlyCreatedSecret, setNewlyCreatedSecret] = useState<{ id: string; secret: string } | null>(null);

  // Filter
  const [statusFilter, setStatusFilter] = useState<'all' | 'success' | 'failed' | 'replayed'>('all');
  const [providerFilter, setProviderFilter] = useState<string>('all');

  const loadData = async () => {
    setLoading(true);
    try {
      const [intRes, logsRes] = await Promise.all([
        listProvisioningIntegrations(),
        listProvisioningWebhookLogs(),
      ]);

      if (intRes.data) {
        setIntegrations(intRes.data);
      }
      if (logsRes.data) {
        setLogs(logsRes.data);
      }
    } catch (err: any) {
      console.warn('Failed to load webhook provisioning data:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const handleCopy = (text: string, id: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(id);
    setTimeout(() => setCopiedKey(null), 2000);
  };

  const handleCreateIntegration = async (e: React.FormEvent) => {
    e.preventDefault();
    setActionLoading('create');
    try {
      const res = await createProvisioningIntegration({
        provider,
        name: name || `${provider.toUpperCase()} Integration`,
        default_server_id: defaultServerId || undefined,
        default_plan_id: defaultPlanId || undefined,
        is_active: true,
      });

      if (res.data) {
        showNotify('success', `Integration "${res.data.name}" created successfully!`);
        if (res.data.raw_secret_key) {
          setNewlyCreatedSecret({ id: res.data.id, secret: res.data.raw_secret_key });
        }
        setCreateModalOpen(false);
        setName('');
        loadData();
      } else {
        showNotify('error', res.error?.message || 'Failed to create integration');
      }
    } catch (err: any) {
      showNotify('error', err.message || 'Server error creating integration');
    } finally {
      setActionLoading(null);
    }
  };

  const handleDeleteIntegration = async (id: string, intName: string) => {
    if (!confirm(`Are you sure you want to delete integration "${intName}"? Incoming webhooks will be rejected.`)) {
      return;
    }
    setActionLoading(`del-${id}`);
    try {
      const res = await deleteProvisioningIntegration(id);
      if (res.data) {
        showNotify('success', `Integration "${intName}" deleted.`);
        loadData();
      }
    } catch (err: any) {
      showNotify('error', err.message || 'Failed to delete integration');
    } finally {
      setActionLoading(null);
    }
  };

  const handleReplayLog = async (logItem: ProvisioningWebhookLog) => {
    setActionLoading(`replay-${logItem.id}`);
    try {
      const res = await replayWebhookLog(logItem.id);
      if (res.data) {
        showNotify('success', `Webhook action "${logItem.action}" replayed successfully!`);
        loadData();
      } else {
        showNotify('error', res.error?.message || 'Failed to replay webhook');
      }
    } catch (err: any) {
      showNotify('error', err.message || 'Error executing webhook replay');
    } finally {
      setActionLoading(null);
    }
  };

  const filteredLogs = logs.filter(l => {
    if (statusFilter !== 'all' && l.status !== statusFilter) return false;
    if (providerFilter !== 'all' && l.provider.toLowerCase() !== providerFilter.toLowerCase()) return false;
    return true;
  });

  const getWebhookUrl = (p: string) => {
    const origin = typeof window !== 'undefined' ? window.location.origin : 'https://panel.hostvra.com';
    return `${origin}/api/v1/billing/provisioning/webhook/${p}`;
  };

  return (
    <div className="space-y-8 animate-in fade-in duration-200">
      {/* Newly Created Secret Notice Banner */}
      {newlyCreatedSecret && (
        <div className="p-5 rounded-2xl bg-emerald-50 border border-emerald-200 text-emerald-950 flex flex-col md:flex-row md:items-center justify-between gap-4 shadow-sm">
          <div className="space-y-1">
            <div className="flex items-center gap-2 font-bold text-emerald-800 text-sm">
              <CheckCircle2 className="w-5 h-5 text-emerald-600" />
              <span>Integration Secret Key Generated — Save It Now!</span>
            </div>
            <p className="text-xs text-emerald-700 max-w-2xl">
              This secret key is used to sign HMAC-SHA256 webhooks from WHMCS or Blesta. For security, this raw key will not be shown again.
            </p>
            <div className="font-mono text-xs bg-white px-3 py-1.5 rounded-lg border border-emerald-300 select-all inline-block mt-1 text-slate-800">
              {newlyCreatedSecret.secret}
            </div>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => handleCopy(newlyCreatedSecret.secret, 'new-secret')}
              className="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white text-xs font-semibold flex items-center gap-2 shadow-sm transition-all"
            >
              {copiedKey === 'new-secret' ? <Check className="w-4 h-4" /> : <Copy className="w-4 h-4" />}
              {copiedKey === 'new-secret' ? 'Copied' : 'Copy Secret'}
            </button>
            <button
              onClick={() => setNewlyCreatedSecret(null)}
              className="px-3 py-2 rounded-xl text-slate-500 hover:text-slate-800 text-xs font-medium"
            >
              Dismiss
            </button>
          </div>
        </div>
      )}

      {/* Header Info Banner */}
      <div className="rounded-3xl bg-white p-6 sm:p-8 border border-slate-200 shadow-sm flex flex-col md:flex-row md:items-center justify-between gap-6">
        <div className="space-y-2">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-blue-50 border border-blue-100 text-xs font-semibold text-blue-700">
            <Webhook className="w-3.5 h-3.5 text-blue-600" />
            <span>Automated Billing & Provisioning Gateway</span>
          </div>
          <h2 className="text-2xl font-bold text-slate-900">WHMCS & Blesta Webhook Provisioning</h2>
          <p className="text-sm text-slate-600 max-w-2xl leading-relaxed">
            Connect your WHMCS or Blesta billing system for zero-touch account creation, suspensions, upgrades, and terminations with cryptographic HMAC-SHA256 signature verification.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={loadData}
            disabled={loading}
            className="p-2.5 rounded-xl border border-slate-200 bg-white text-slate-600 hover:text-slate-900 shadow-sm transition-all"
            title="Refresh logs"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin text-blue-600' : ''}`} />
          </button>
          <button
            onClick={() => setCreateModalOpen(true)}
            className="px-4 py-2.5 rounded-xl bg-[#16A34A] hover:bg-emerald-700 text-white text-xs font-bold flex items-center gap-2 shadow-sm transition-all"
          >
            <Plus className="w-4 h-4" />
            <span>Add Integration</span>
          </button>
        </div>
      </div>

      {/* Active Integrations Grid */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-base font-bold text-slate-900 flex items-center gap-2">
            <Radio className="w-4 h-4 text-emerald-600" />
            <span>Configured Endpoints ({integrations.length})</span>
          </h3>
        </div>

        {integrations.length === 0 ? (
          <div className="p-8 text-center bg-white rounded-2xl border border-slate-200 space-y-3">
            <Webhook className="w-10 h-10 text-slate-300 mx-auto" />
            <h4 className="text-sm font-bold text-slate-800">No Webhook Integrations Configured</h4>
            <p className="text-xs text-slate-500 max-w-md mx-auto">
              Create a WHMCS or Blesta integration to generate a secure endpoint URL and HMAC signing key.
            </p>
            <button
              onClick={() => setCreateModalOpen(true)}
              className="px-4 py-2 rounded-xl bg-[#16A34A] text-white text-xs font-semibold inline-flex items-center gap-1.5 shadow-sm"
            >
              <Plus className="w-4 h-4" />
              <span>Create First Integration</span>
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
            {integrations.map(item => {
              const webhookUrl = getWebhookUrl(item.provider);
              return (
                <div key={item.id} className="p-5 rounded-2xl bg-white border border-slate-200 shadow-sm space-y-4 hover:border-slate-300 transition-all">
                  <div className="flex items-start justify-between">
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-bold text-slate-900">{item.name}</span>
                        <span className={`px-2 py-0.5 rounded-full text-[10px] font-bold uppercase ${
                          item.provider === 'whmcs'
                            ? 'bg-blue-100 text-blue-700'
                            : item.provider === 'blesta'
                            ? 'bg-purple-100 text-purple-700'
                            : 'bg-slate-100 text-slate-700'
                        }`}>
                          {item.provider}
                        </span>
                      </div>
                      <div className="text-[11px] text-slate-500 mt-0.5">ID: {item.id.slice(0, 8)}...</div>
                    </div>
                    <button
                      onClick={() => handleDeleteIntegration(item.id, item.name)}
                      disabled={actionLoading === `del-${item.id}`}
                      className="p-1.5 text-slate-400 hover:text-rose-600 rounded-lg hover:bg-rose-50 transition-all"
                      title="Delete Integration"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>

                  {/* Webhook URL Box */}
                  <div className="space-y-1">
                    <label className="text-[11px] font-semibold text-slate-500 uppercase tracking-wider">Webhook URL</label>
                    <div className="flex items-center gap-2 bg-slate-50 p-2 rounded-xl border border-slate-200 text-xs font-mono text-slate-800 overflow-hidden">
                      <span className="truncate flex-1">{webhookUrl}</span>
                      <button
                        onClick={() => handleCopy(webhookUrl, `url-${item.id}`)}
                        className="text-slate-500 hover:text-slate-800 p-1 rounded hover:bg-slate-200"
                        title="Copy URL"
                      >
                        {copiedKey === `url-${item.id}` ? <Check className="w-3.5 h-3.5 text-emerald-600" /> : <Copy className="w-3.5 h-3.5" />}
                      </button>
                    </div>
                  </div>

                  {/* Capabilities Pill */}
                  <div className="pt-2 border-t border-slate-100 flex items-center justify-between text-[11px] text-slate-500">
                    <span className="flex items-center gap-1.5 text-emerald-700 font-medium">
                      <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
                      Active (HMAC-SHA256)
                    </span>
                    <span>Created: {new Date(item.created_at).toLocaleDateString()}</span>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Live Webhook Logs Table */}
      <div className="space-y-4">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h3 className="text-base font-bold text-slate-900 flex items-center gap-2">
              <FileText className="w-4 h-4 text-blue-600" />
              <span>Real-Time Webhook Activity Logs</span>
            </h3>
            <p className="text-xs text-slate-500">
              Audit trail of all incoming WHMCS / Blesta calls with request replay capability.
            </p>
          </div>

          <div className="flex items-center gap-3">
            {/* Provider Filter */}
            <select
              value={providerFilter}
              onChange={e => setProviderFilter(e.target.value)}
              className="text-xs bg-white border border-slate-200 rounded-xl px-3 py-2 text-slate-700 font-medium focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="all">All Providers</option>
              <option value="whmcs">WHMCS</option>
              <option value="blesta">Blesta</option>
              <option value="custom">Custom</option>
            </select>

            {/* Status Filter */}
            <div className="flex items-center bg-slate-100 p-1 rounded-xl border border-slate-200 text-xs font-semibold">
              <button
                onClick={() => setStatusFilter('all')}
                className={`px-3 py-1 rounded-lg transition-all ${statusFilter === 'all' ? 'bg-white text-blue-600 shadow-sm' : 'text-slate-500'}`}
              >
                All
              </button>
              <button
                onClick={() => setStatusFilter('success')}
                className={`px-3 py-1 rounded-lg transition-all ${statusFilter === 'success' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500'}`}
              >
                Success
              </button>
              <button
                onClick={() => setStatusFilter('failed')}
                className={`px-3 py-1 rounded-lg transition-all ${statusFilter === 'failed' ? 'bg-white text-rose-600 shadow-sm' : 'text-slate-500'}`}
              >
                Failed
              </button>
              <button
                onClick={() => setStatusFilter('replayed')}
                className={`px-3 py-1 rounded-lg transition-all ${statusFilter === 'replayed' ? 'bg-white text-amber-600 shadow-sm' : 'text-slate-500'}`}
              >
                Replayed
              </button>
            </div>
          </div>
        </div>

        <div className="overflow-x-auto rounded-2xl border border-slate-200 bg-white shadow-sm">
          <table className="w-full text-left text-xs text-slate-600">
            <thead className="bg-slate-50 text-[11px] font-bold uppercase tracking-wider text-slate-500 border-b border-slate-200">
              <tr>
                <th className="px-5 py-3.5">Timestamp</th>
                <th className="px-4 py-3.5">Provider</th>
                <th className="px-4 py-3.5">Action</th>
                <th className="px-4 py-3.5">Target Account</th>
                <th className="px-4 py-3.5">HTTP Status</th>
                <th className="px-4 py-3.5">Client IP</th>
                <th className="px-5 py-3.5 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {filteredLogs.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-5 py-8 text-center text-slate-400">
                    No webhook logs recorded yet. Send a test webhook from WHMCS to see live activity.
                  </td>
                </tr>
              ) : (
                filteredLogs.map(log => {
                  const isSuccess = log.status === 'success' || log.status === 'replayed';
                  return (
                    <tr key={log.id} className="hover:bg-slate-50/70 transition-colors">
                      <td className="px-5 py-3.5 font-mono text-[11px] text-slate-500">
                        {new Date(log.created_at).toLocaleString()}
                      </td>
                      <td className="px-4 py-3.5">
                        <span className="px-2 py-0.5 rounded-full text-[10px] font-bold uppercase bg-slate-100 text-slate-700">
                          {log.provider}
                        </span>
                      </td>
                      <td className="px-4 py-3.5 font-semibold text-slate-900">
                        <span className={`px-2 py-0.5 rounded-lg text-[11px] font-mono ${
                          log.action === 'create_account'
                            ? 'bg-emerald-50 text-emerald-700 border border-emerald-200'
                            : log.action === 'suspend'
                            ? 'bg-rose-50 text-rose-700 border border-rose-200'
                            : log.action === 'unsuspend'
                            ? 'bg-blue-50 text-blue-700 border border-blue-200'
                            : log.action === 'change_package'
                            ? 'bg-purple-50 text-purple-700 border border-purple-200'
                            : 'bg-slate-100 text-slate-700'
                        }`}>
                          {log.action}
                        </span>
                      </td>
                      <td className="px-4 py-3.5">
                        <div className="font-medium text-slate-900">{log.domain || '—'}</div>
                        {log.username && <div className="text-[11px] text-slate-400">@{log.username}</div>}
                      </td>
                      <td className="px-4 py-3.5">
                        <span className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md text-[11px] font-bold ${
                          isSuccess
                            ? 'bg-emerald-50 text-emerald-700 border border-emerald-200'
                            : 'bg-rose-50 text-rose-700 border border-rose-200'
                        }`}>
                          {isSuccess ? <CheckCircle2 className="w-3 h-3 text-emerald-600" /> : <AlertCircle className="w-3 h-3 text-rose-600" />}
                          {log.http_status || (isSuccess ? 200 : 400)}
                        </span>
                      </td>
                      <td className="px-4 py-3.5 font-mono text-[11px] text-slate-500">
                        {log.ip_address || '127.0.0.1'}
                      </td>
                      <td className="px-5 py-3.5 text-right space-x-2">
                        <button
                          onClick={() => {
                            setSelectedLog(log);
                            setInspectModalOpen(true);
                          }}
                          className="px-2.5 py-1.5 rounded-lg text-slate-600 hover:text-slate-900 hover:bg-slate-100 text-xs font-semibold inline-flex items-center gap-1 border border-slate-200"
                        >
                          <Eye className="w-3.5 h-3.5" />
                          <span>Inspect</span>
                        </button>
                        <button
                          onClick={() => handleReplayLog(log)}
                          disabled={actionLoading === `replay-${log.id}`}
                          className="px-2.5 py-1.5 rounded-lg text-blue-600 hover:text-white hover:bg-blue-600 text-xs font-semibold inline-flex items-center gap-1 border border-blue-200 transition-all"
                          title="Replay this exact payload"
                        >
                          <Play className={`w-3.5 h-3.5 ${actionLoading === `replay-${log.id}` ? 'animate-spin' : ''}`} />
                          <span>Replay</span>
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

      {/* Add Integration Modal */}
      {createModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-sm animate-in fade-in duration-200">
          <div className="relative w-full max-w-lg rounded-3xl bg-white p-7 shadow-2xl border border-slate-200 space-y-6">
            <div className="flex items-center justify-between pb-4 border-b border-slate-100">
              <div className="flex items-center gap-3">
                <div className="p-2.5 rounded-2xl bg-emerald-50 text-[#16A34A] border border-emerald-100">
                  <Webhook className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="text-lg font-bold text-slate-900">Add Provisioning Integration</h3>
                  <p className="text-xs text-slate-500">Configure WHMCS, Blesta, or Custom Webhook</p>
                </div>
              </div>
              <button
                onClick={() => setCreateModalOpen(false)}
                className="p-1 rounded-xl text-slate-400 hover:text-slate-700 hover:bg-slate-100"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <form onSubmit={handleCreateIntegration} className="space-y-4">
              <div>
                <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-1.5">
                  Billing Provider
                </label>
                <div className="grid grid-cols-3 gap-3">
                  {(['whmcs', 'blesta', 'custom'] as const).map(p => (
                    <button
                      type="button"
                      key={p}
                      onClick={() => setProvider(p)}
                      className={`p-3 rounded-2xl border text-xs font-bold capitalize transition-all ${
                        provider === p
                          ? 'border-emerald-500 bg-emerald-50 text-emerald-800 shadow-sm'
                          : 'border-slate-200 bg-white text-slate-600 hover:bg-slate-50'
                      }`}
                    >
                      {p}
                    </button>
                  ))}
                </div>
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-1.5">
                  Integration Label / Name
                </label>
                <input
                  type="text"
                  value={name}
                  onChange={e => setName(e.target.value)}
                  placeholder={`e.g. ${provider.toUpperCase()} Production Store`}
                  className="w-full px-4 py-2.5 rounded-xl border border-slate-200 text-xs text-slate-900 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-1.5">
                  Default Target Server (Optional)
                </label>
                <select
                  value={defaultServerId}
                  onChange={e => setDefaultServerId(e.target.value)}
                  className="w-full px-4 py-2.5 rounded-xl border border-slate-200 text-xs text-slate-900 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                >
                  <option value="">Auto-Assign (First Online Node)</option>
                  {servers.map(s => (
                    <option key={s.id} value={s.id}>
                      {s.name} ({s.ip_address})
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-1.5">
                  Default Hosting Plan (Optional)
                </label>
                <select
                  value={defaultPlanId}
                  onChange={e => setDefaultPlanId(e.target.value)}
                  className="w-full px-4 py-2.5 rounded-xl border border-slate-200 text-xs text-slate-900 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                >
                  <option value="">Map dynamically from incoming payload</option>
                  {plans.map(p => (
                    <option key={p.id} value={p.id}>
                      {p.name} ({p.disk_space_mb} MB Disk)
                    </option>
                  ))}
                </select>
              </div>

              <div className="pt-4 flex items-center justify-end gap-3 border-t border-slate-100">
                <button
                  type="button"
                  onClick={() => setCreateModalOpen(false)}
                  className="px-4 py-2.5 rounded-xl border border-slate-200 text-xs font-bold text-slate-600 hover:bg-slate-50"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={actionLoading === 'create'}
                  className="px-5 py-2.5 rounded-xl bg-[#16A34A] hover:bg-emerald-700 text-white text-xs font-bold shadow-sm transition-all flex items-center gap-2"
                >
                  {actionLoading === 'create' ? <RefreshCw className="w-4 h-4 animate-spin" /> : <Plus className="w-4 h-4" />}
                  <span>Generate Key & Create</span>
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Inspect Log Modal */}
      {inspectModalOpen && selectedLog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-sm animate-in fade-in duration-200">
          <div className="relative w-full max-w-2xl rounded-3xl bg-white p-7 shadow-2xl border border-slate-200 space-y-5 max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between pb-4 border-b border-slate-100">
              <div className="flex items-center gap-3">
                <div className="p-2.5 rounded-2xl bg-blue-50 text-blue-600 border border-blue-100">
                  <Code2 className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="text-lg font-bold text-slate-900">Webhook Payload Inspector</h3>
                  <p className="text-xs text-slate-500">
                    {selectedLog.provider.toUpperCase()} • Action: {selectedLog.action} • {new Date(selectedLog.created_at).toLocaleString()}
                  </p>
                </div>
              </div>
              <button
                onClick={() => setInspectModalOpen(false)}
                className="p-1 rounded-xl text-slate-400 hover:text-slate-700 hover:bg-slate-100"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="space-y-4 text-xs">
              <div>
                <label className="font-bold text-slate-700 uppercase tracking-wider block mb-1">
                  Incoming Payload (Decoded)
                </label>
                <pre className="p-4 rounded-2xl bg-slate-900 text-slate-100 font-mono text-[11px] overflow-x-auto max-h-60 leading-relaxed">
                  {JSON.stringify(selectedLog.payload, null, 2)}
                </pre>
              </div>

              {selectedLog.response_body && (
                <div>
                  <label className="font-bold text-slate-700 uppercase tracking-wider block mb-1">
                    API Response Body
                  </label>
                  <pre className="p-4 rounded-2xl bg-slate-900 text-emerald-400 font-mono text-[11px] overflow-x-auto max-h-48 leading-relaxed">
                    {selectedLog.response_body}
                  </pre>
                </div>
              )}

              {selectedLog.error_message && (
                <div className="p-4 rounded-2xl bg-rose-50 border border-rose-200 text-rose-800 space-y-1">
                  <div className="font-bold flex items-center gap-1.5">
                    <AlertCircle className="w-4 h-4 text-rose-600" />
                    <span>Error Detail</span>
                  </div>
                  <div className="font-mono text-[11px]">{selectedLog.error_message}</div>
                </div>
              )}
            </div>

            <div className="pt-4 flex items-center justify-between border-t border-slate-100">
              <span className="text-[11px] text-slate-400">Client IP: {selectedLog.ip_address || '127.0.0.1'}</span>
              <div className="flex items-center gap-3">
                <button
                  type="button"
                  onClick={() => {
                    setInspectModalOpen(false);
                    handleReplayLog(selectedLog);
                  }}
                  className="px-4 py-2 rounded-xl bg-blue-600 hover:bg-blue-700 text-white text-xs font-bold flex items-center gap-1.5 transition-all"
                >
                  <Play className="w-3.5 h-3.5" />
                  <span>Replay This Webhook</span>
                </button>
                <button
                  type="button"
                  onClick={() => setInspectModalOpen(false)}
                  className="px-4 py-2 rounded-xl border border-slate-200 text-xs font-bold text-slate-600 hover:bg-slate-50"
                >
                  Close
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
