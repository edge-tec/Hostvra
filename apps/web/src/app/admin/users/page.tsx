'use client';

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import {
  Users,
  Search,
  RefreshCw,
  Edit2,
  Trash2,
  Sliders,
  Shield,
  ShieldAlert,
  CheckCircle2,
  XCircle,
  AlertCircle,
  Package,
  Layers,
  ArrowRight,
  Terminal,
  HardDrive,
  Globe,
  Database,
  Mail,
  Zap,
  Lock,
  Unlock,
  X,
  Save,
  RotateCcw,
  Sparkles,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import {
  AdminUserListItem,
  EffectiveUserPlan,
  HostingPlan,
  UserPlanOverride,
  fetchAdminUsers,
  apiFetch,
  updateAdminUserPlan,
  updateAdminUserOverrides,
  deleteAdminUserOverrides,
  updateAdminUserStatus,
  getStoredUserRole,
} from '@/lib/api';

export default function AdminUsersPage() {
  const [users, setUsers] = useState<AdminUserListItem[]>([]);
  const [plans, setPlans] = useState<HostingPlan[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [filterRole, setFilterRole] = useState('all');
  const [selectedUser, setSelectedUser] = useState<AdminUserListItem | null>(null);

  // Modals
  const [editPlanModal, setEditPlanModal] = useState(false);
  const [editOverrideModal, setEditOverrideModal] = useState(false);

  // Form states
  const [selectedPlanId, setSelectedPlanId] = useState('');
  const [selectedCycle, setSelectedCycle] = useState<'monthly' | 'yearly'>('monthly');

  // Override Form
  const [overrideWebsites, setOverrideWebsites] = useState<string>('');
  const [overrideDatabases, setOverrideDatabases] = useState<string>('');
  const [overrideMailboxes, setOverrideMailboxes] = useState<string>('');
  const [overrideDiskMB, setOverrideDiskMB] = useState<string>('');
  const [overrideBandwidthMB, setOverrideBandwidthMB] = useState<string>('');
  const [permTerminal, setPermTerminal] = useState<boolean | null>(null);
  const [permBackups, setPermBackups] = useState<boolean | null>(null);
  const [permFileManager, setPermFileManager] = useState<boolean | null>(null);
  const [overrideNotes, setOverrideNotes] = useState('');

  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const loadData = async () => {
    setLoading(true);
    try {
      const [usersRes, plansRes] = await Promise.all([
        fetchAdminUsers(),
        apiFetch<HostingPlan[]>('/api/v1/billing/plans'),
      ]);

      if (usersRes.success && usersRes.data) {
        setUsers(usersRes.data);
      }
      if (plansRes.success && plansRes.data) {
        setPlans(plansRes.data);
      }
    } catch (err: any) {
      setMessage({ type: 'error', text: err?.message || 'Failed to fetch admin users' });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const openPlanModal = (u: AdminUserListItem) => {
    setSelectedUser(u);
    if (u.effective_plan) {
      setSelectedPlanId(u.effective_plan.plan_id);
    } else if (plans.length > 0) {
      setSelectedPlanId(plans[0].id);
    }
    setEditPlanModal(true);
  };

  const handleSavePlan = async () => {
    if (!selectedUser || !selectedPlanId) return;
    setSaving(true);
    try {
      const res = await updateAdminUserPlan(selectedUser.id, selectedPlanId, selectedCycle);
      if (res.success) {
        setMessage({ type: 'success', text: `Package updated for ${selectedUser.email}` });
        setEditPlanModal(false);
        loadData();
      } else {
        setMessage({ type: 'error', text: res.error?.message || 'Failed to update plan' });
      }
    } catch (err: any) {
      setMessage({ type: 'error', text: err?.message || 'Request failed' });
    } finally {
      setSaving(false);
    }
  };

  const openOverrideModal = (u: AdminUserListItem) => {
    setSelectedUser(u);
    const ov = u.effective_plan?.overrides;
    setOverrideWebsites(ov?.max_websites !== undefined && ov?.max_websites !== null ? String(ov.max_websites) : '');
    setOverrideDatabases(ov?.max_databases !== undefined && ov?.max_databases !== null ? String(ov.max_databases) : '');
    setOverrideMailboxes(ov?.max_mailboxes !== undefined && ov?.max_mailboxes !== null ? String(ov.max_mailboxes) : '');
    setOverrideDiskMB(ov?.disk_space_mb !== undefined && ov?.disk_space_mb !== null ? String(ov.disk_space_mb) : '');
    setOverrideBandwidthMB(ov?.bandwidth_mb !== undefined && ov?.bandwidth_mb !== null ? String(ov.bandwidth_mb) : '');
    setPermTerminal(ov?.permission_terminal !== undefined ? ov.permission_terminal : null);
    setPermBackups(ov?.permission_backups !== undefined ? ov.permission_backups : null);
    setPermFileManager(ov?.permission_file_manager !== undefined ? ov.permission_file_manager : null);
    setOverrideNotes(ov?.notes || '');
    setEditOverrideModal(true);
  };

  const handleSaveOverrides = async () => {
    if (!selectedUser) return;
    setSaving(true);

    const payload: Partial<UserPlanOverride> = {
      notes: overrideNotes,
    };
    if (overrideWebsites !== '') payload.max_websites = parseInt(overrideWebsites, 10);
    if (overrideDatabases !== '') payload.max_databases = parseInt(overrideDatabases, 10);
    if (overrideMailboxes !== '') payload.max_mailboxes = parseInt(overrideMailboxes, 10);
    if (overrideDiskMB !== '') payload.disk_space_mb = parseInt(overrideDiskMB, 10);
    if (overrideBandwidthMB !== '') payload.bandwidth_mb = parseInt(overrideBandwidthMB, 10);
    if (permTerminal !== null) payload.permission_terminal = permTerminal;
    if (permBackups !== null) payload.permission_backups = permBackups;
    if (permFileManager !== null) payload.permission_file_manager = permFileManager;

    try {
      const res = await updateAdminUserOverrides(selectedUser.id, payload);
      if (res.success) {
        setMessage({ type: 'success', text: `Custom overrides saved for ${selectedUser.email}` });
        setEditOverrideModal(false);
        loadData();
      } else {
        setMessage({ type: 'error', text: res.error?.message || 'Failed to save overrides' });
      }
    } catch (err: any) {
      setMessage({ type: 'error', text: err?.message || 'Request failed' });
    } finally {
      setSaving(false);
    }
  };

  const handleClearOverrides = async (u: AdminUserListItem) => {
    if (!confirm(`Revert all custom limits and permissions for ${u.email} back to package defaults?`)) return;
    try {
      const res = await deleteAdminUserOverrides(u.id);
      if (res.success) {
        setMessage({ type: 'success', text: `Overrides removed. Reverted to package defaults for ${u.email}` });
        loadData();
      } else {
        setMessage({ type: 'error', text: res.error?.message || 'Failed to remove overrides' });
      }
    } catch (err: any) {
      setMessage({ type: 'error', text: err?.message || 'Request failed' });
    }
  };

  const handleToggleStatus = async (u: AdminUserListItem) => {
    const nextStatus = !u.is_active;
    const action = nextStatus ? 'activate' : 'suspend';
    if (!confirm(`Are you sure you want to ${action} ${u.email}?`)) return;

    try {
      const res = await updateAdminUserStatus(u.id, nextStatus);
      if (res.success) {
        setMessage({ type: 'success', text: `User ${u.email} is now ${nextStatus ? 'active' : 'suspended'}` });
        loadData();
      } else {
        setMessage({ type: 'error', text: res.error?.message || 'Failed to update status' });
      }
    } catch (err: any) {
      setMessage({ type: 'error', text: err?.message || 'Request failed' });
    }
  };

  const filteredUsers = users.filter((u) => {
    const matchesSearch =
      u.email.toLowerCase().includes(search.toLowerCase()) ||
      u.full_name.toLowerCase().includes(search.toLowerCase());
    const matchesRole = filterRole === 'all' || u.role === filterRole;
    return matchesSearch && matchesRole;
  });

  return (
    <DashboardShell>
      <div className="space-y-6 pb-12">
        {/* Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-black text-slate-900 dark:text-white flex items-center gap-2.5">
              <Users className="w-6 h-6 text-indigo-600 dark:text-indigo-400" />
              Tenant User Management
            </h1>
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
              Audit customer identities, assign hosting packages, modify limits and apply custom permissions.
            </p>
          </div>

          <button
            onClick={loadData}
            disabled={loading}
            className="inline-flex items-center gap-2 px-3.5 py-2 rounded-xl text-xs font-semibold bg-white dark:bg-surface-800 border border-slate-200 dark:border-surface-700 hover:bg-slate-50 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-200 transition-colors"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
            Refresh Users
          </button>
        </div>

        {/* Message Banner */}
        {message && (
          <div
            className={`p-4 rounded-xl text-xs flex items-center justify-between ${
              message.type === 'success'
                ? 'bg-emerald-50 dark:bg-emerald-500/10 border border-emerald-200 dark:border-emerald-500/20 text-emerald-800 dark:text-emerald-400'
                : 'bg-rose-50 dark:bg-rose-500/10 border border-rose-200 dark:border-rose-500/20 text-rose-800 dark:text-rose-400'
            }`}
          >
            <div className="flex items-center gap-2">
              {message.type === 'success' ? (
                <CheckCircle2 className="w-4 h-4 shrink-0" />
              ) : (
                <AlertCircle className="w-4 h-4 shrink-0" />
              )}
              <span className="font-semibold">{message.text}</span>
            </div>
            <button onClick={() => setMessage(null)} className="p-1 hover:opacity-75">
              <X className="w-3.5 h-3.5" />
            </button>
          </div>
        )}

        {/* Filter and Search Bar */}
        <div className="flex flex-col sm:flex-row items-center gap-3">
          <div className="relative flex-1 w-full">
            <Search className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2 pointer-events-none" />
            <input
              type="text"
              placeholder="Search users by name, email or ID..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full pl-9 pr-4 py-2 text-xs rounded-xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 text-slate-900 dark:text-white"
            />
          </div>

          <div className="flex items-center gap-2 w-full sm:w-auto">
            <select
              value={filterRole}
              onChange={(e) => setFilterRole(e.target.value)}
              className="px-3 py-2 text-xs rounded-xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 text-slate-700 dark:text-slate-300 focus:outline-none"
            >
              <option value="all">All Roles</option>
              <option value="customer">Customers</option>
              <option value="user">Users</option>
              <option value="admin">Administrators</option>
            </select>
          </div>
        </div>

        {/* Users Table */}
        <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl overflow-hidden shadow-sm">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="bg-slate-50 dark:bg-surface-800/60 border-b border-slate-200 dark:border-surface-800 text-slate-500 font-bold uppercase tracking-wider text-[10px]">
                <tr>
                  <th className="py-3 px-4">User / Email</th>
                  <th className="py-3 px-4">Role</th>
                  <th className="py-3 px-4">Status</th>
                  <th className="py-3 px-4">Package Plan</th>
                  <th className="py-3 px-4">Effective Quotas</th>
                  <th className="py-3 px-4">Live Usage</th>
                  <th className="py-3 px-4 text-right">Admin Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 dark:divide-surface-800">
                {loading ? (
                  <tr>
                    <td colSpan={7} className="py-8 text-center text-slate-400">
                      Loading users list...
                    </td>
                  </tr>
                ) : filteredUsers.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="py-8 text-center text-slate-400">
                      No matching users found.
                    </td>
                  </tr>
                ) : (
                  filteredUsers.map((u) => {
                    const plan = u.effective_plan;
                    const isCust = u.role === 'customer' || u.role === 'user';
                    return (
                      <tr
                        key={u.id}
                        className="hover:bg-slate-50/50 dark:hover:bg-surface-800/40 transition-colors"
                      >
                        {/* User Identity */}
                        <td className="py-3.5 px-4">
                          <div className="font-bold text-slate-900 dark:text-white">
                            {u.full_name || 'Anonymous User'}
                          </div>
                          <div className="text-[11px] text-slate-500 dark:text-slate-400 font-mono">
                            {u.email}
                          </div>
                        </td>

                        {/* Role */}
                        <td className="py-3.5 px-4">
                          <span
                            className={`px-2 py-0.5 rounded-full text-[10px] font-bold uppercase ${
                              u.role === 'admin' || u.is_superadmin
                                ? 'bg-purple-100 dark:bg-purple-500/20 text-purple-700 dark:text-purple-300 border border-purple-200 dark:border-purple-500/30'
                                : 'bg-emerald-100 dark:bg-emerald-500/20 text-emerald-700 dark:text-emerald-300 border border-emerald-200 dark:border-emerald-500/30'
                            }`}
                          >
                            {u.is_superadmin ? 'SuperAdmin' : u.role}
                          </span>
                        </td>

                        {/* Status */}
                        <td className="py-3.5 px-4">
                          <button
                            onClick={() => handleToggleStatus(u)}
                            className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-bold ${
                              u.is_active
                                ? 'bg-emerald-50 dark:bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 hover:bg-emerald-100'
                                : 'bg-rose-50 dark:bg-rose-500/10 text-rose-600 dark:text-rose-400 hover:bg-rose-100'
                            }`}
                            title="Click to toggle status"
                          >
                            {u.is_active ? (
                              <>
                                <CheckCircle2 className="w-3 h-3" /> Active
                              </>
                            ) : (
                              <>
                                <XCircle className="w-3 h-3" /> Suspended
                              </>
                            )}
                          </button>
                        </td>

                        {/* Package Plan */}
                        <td className="py-3.5 px-4">
                          <div className="flex items-center gap-1.5">
                            <span className="font-semibold text-slate-800 dark:text-slate-200">
                              {plan?.plan_name || 'Starter Cloud'}
                            </span>
                            {u.has_custom_overrides && (
                              <span
                                className="px-1.5 py-0.2 rounded bg-amber-100 dark:bg-amber-500/20 text-amber-700 dark:text-amber-400 text-[9px] font-bold"
                                title="Custom Overrides Active"
                              >
                                Overridden
                              </span>
                            )}
                          </div>
                          <div className="text-[10px] text-slate-400 capitalize">
                            Status: {plan?.subscription_status || 'Active'}
                          </div>
                        </td>

                        {/* Effective Quotas */}
                        <td className="py-3.5 px-4 text-slate-600 dark:text-slate-300">
                          {plan ? (
                            <div className="space-y-0.5 text-[11px]">
                              <div>Sites: <strong className="text-slate-900 dark:text-white">{plan.max_websites}</strong> | DBs: <strong className="text-slate-900 dark:text-white">{plan.max_databases}</strong></div>
                              <div>Disk: <strong className="text-slate-900 dark:text-white">{(plan.disk_space_mb / 1024).toFixed(0)} GB</strong> | Term: <strong className={plan.permissions?.terminal ? 'text-emerald-500' : 'text-slate-400'}>{plan.permissions?.terminal ? 'Yes' : 'No'}</strong></div>
                            </div>
                          ) : (
                            <span className="text-slate-400">Default</span>
                          )}
                        </td>

                        {/* Live Usage */}
                        <td className="py-3.5 px-4">
                          {plan?.usage ? (
                            <div className="text-[11px] space-y-0.5 text-slate-600 dark:text-slate-400">
                              <div>Sites: <strong className="text-slate-900 dark:text-white">{plan.usage.websites_count}</strong> used</div>
                              <div>DBs: <strong className="text-slate-900 dark:text-white">{plan.usage.databases_count}</strong> used</div>
                            </div>
                          ) : (
                            <span className="text-slate-400">—</span>
                          )}
                        </td>

                        {/* Admin Controls */}
                        <td className="py-3.5 px-4 text-right">
                          <div className="inline-flex items-center gap-1.5">
                            <button
                              onClick={() => openPlanModal(u)}
                              className="px-2.5 py-1 rounded-lg text-xs font-semibold bg-indigo-50 dark:bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 hover:bg-indigo-100 transition-colors"
                              title="Change Hosting Package"
                            >
                              Plan
                            </button>
                            <button
                              onClick={() => openOverrideModal(u)}
                              className="px-2.5 py-1 rounded-lg text-xs font-semibold bg-amber-50 dark:bg-amber-500/10 text-amber-600 dark:text-amber-400 hover:bg-amber-100 transition-colors flex items-center gap-1"
                              title="Set Custom Quotas and Permissions"
                            >
                              <Sliders className="w-3 h-3" /> Overrides
                            </button>
                            {u.has_custom_overrides && (
                              <button
                                onClick={() => handleClearOverrides(u)}
                                className="p-1 rounded-lg text-rose-500 hover:bg-rose-50 dark:hover:bg-rose-500/10 transition-colors"
                                title="Revert to Package Defaults"
                              >
                                <RotateCcw className="w-3.5 h-3.5" />
                              </button>
                            )}
                          </div>
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>
        </div>

        {/* Change Plan Modal */}
        {editPlanModal && selectedUser && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl w-full max-w-md p-6 space-y-5 shadow-2xl">
              <div className="flex items-center justify-between border-b border-slate-100 dark:border-surface-800 pb-3">
                <h3 className="text-base font-bold text-slate-900 dark:text-white flex items-center gap-2">
                  <Package className="w-4 h-4 text-indigo-500" />
                  Assign Package: {selectedUser.email}
                </h3>
                <button
                  onClick={() => setEditPlanModal(false)}
                  className="p-1 text-slate-400 hover:text-slate-600"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>

              <div className="space-y-4">
                <div>
                  <label className="block text-xs font-bold text-slate-700 dark:text-slate-300 mb-1">
                    Select Hosting Package
                  </label>
                  <select
                    value={selectedPlanId}
                    onChange={(e) => setSelectedPlanId(e.target.value)}
                    className="w-full px-3 py-2 text-xs rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-slate-900 dark:text-white focus:outline-none"
                  >
                    {plans.map((p) => (
                      <option key={p.id} value={p.id}>
                        {p.name} ({p.tier.toUpperCase()}) — ${p.price_monthly}/mo
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="block text-xs font-bold text-slate-700 dark:text-slate-300 mb-1">
                    Billing Cycle
                  </label>
                  <div className="grid grid-cols-2 gap-2">
                    <button
                      type="button"
                      onClick={() => setSelectedCycle('monthly')}
                      className={`py-2 text-xs font-bold rounded-xl border ${
                        selectedCycle === 'monthly'
                          ? 'bg-indigo-600 text-white border-indigo-600'
                          : 'border-slate-200 dark:border-surface-700 text-slate-700 dark:text-slate-300'
                      }`}
                    >
                      Monthly
                    </button>
                    <button
                      type="button"
                      onClick={() => setSelectedCycle('yearly')}
                      className={`py-2 text-xs font-bold rounded-xl border ${
                        selectedCycle === 'yearly'
                          ? 'bg-indigo-600 text-white border-indigo-600'
                          : 'border-slate-200 dark:border-surface-700 text-slate-700 dark:text-slate-300'
                      }`}
                    >
                      Yearly
                    </button>
                  </div>
                </div>
              </div>

              <div className="flex items-center justify-end gap-2 pt-2 border-t border-slate-100 dark:border-surface-800">
                <button
                  type="button"
                  onClick={() => setEditPlanModal(false)}
                  className="px-3.5 py-2 text-xs font-semibold text-slate-600 hover:text-slate-800"
                >
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={handleSavePlan}
                  disabled={saving}
                  className="px-4 py-2 text-xs font-bold bg-indigo-600 text-white rounded-xl hover:bg-indigo-700 disabled:opacity-50"
                >
                  {saving ? 'Updating...' : 'Save Plan'}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Custom Overrides Modal */}
        {editOverrideModal && selectedUser && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl w-full max-w-lg p-6 space-y-5 shadow-2xl max-h-[90vh] overflow-y-auto">
              <div className="flex items-center justify-between border-b border-slate-100 dark:border-surface-800 pb-3">
                <div>
                  <h3 className="text-base font-bold text-slate-900 dark:text-white flex items-center gap-2">
                    <Sliders className="w-4 h-4 text-amber-500" />
                    Admin Custom Overrides: {selectedUser.email}
                  </h3>
                  <p className="text-[11px] text-slate-400 mt-0.5">
                    User overrides take strict priority over package defaults. Leave blank to inherit package defaults.
                  </p>
                </div>
                <button
                  onClick={() => setEditOverrideModal(false)}
                  className="p-1 text-slate-400 hover:text-slate-600"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>

              <div className="space-y-4 text-xs">
                {/* Resource Limits Section */}
                <div className="space-y-3">
                  <h4 className="font-bold text-slate-900 dark:text-white flex items-center gap-1.5 text-xs">
                    <HardDrive className="w-3.5 h-3.5 text-indigo-500" /> Resource Quota Overrides
                  </h4>

                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="block text-[11px] text-slate-500 mb-1">
                        Max Websites (Default: {selectedUser.effective_plan?.max_websites})
                      </label>
                      <input
                        type="number"
                        placeholder="Inherit package"
                        value={overrideWebsites}
                        onChange={(e) => setOverrideWebsites(e.target.value)}
                        className="w-full px-3 py-1.5 text-xs rounded-lg bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700"
                      />
                    </div>

                    <div>
                      <label className="block text-[11px] text-slate-500 mb-1">
                        Max Databases (Default: {selectedUser.effective_plan?.max_databases})
                      </label>
                      <input
                        type="number"
                        placeholder="Inherit package"
                        value={overrideDatabases}
                        onChange={(e) => setOverrideDatabases(e.target.value)}
                        className="w-full px-3 py-1.5 text-xs rounded-lg bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700"
                      />
                    </div>

                    <div>
                      <label className="block text-[11px] text-slate-500 mb-1">
                        Max Mailboxes (Default: {selectedUser.effective_plan?.max_mailboxes})
                      </label>
                      <input
                        type="number"
                        placeholder="Inherit package"
                        value={overrideMailboxes}
                        onChange={(e) => setOverrideMailboxes(e.target.value)}
                        className="w-full px-3 py-1.5 text-xs rounded-lg bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700"
                      />
                    </div>

                    <div>
                      <label className="block text-[11px] text-slate-500 mb-1">
                        Disk Space MB (Default: {selectedUser.effective_plan?.disk_space_mb})
                      </label>
                      <input
                        type="number"
                        placeholder="Inherit package"
                        value={overrideDiskMB}
                        onChange={(e) => setOverrideDiskMB(e.target.value)}
                        className="w-full px-3 py-1.5 text-xs rounded-lg bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700"
                      />
                    </div>
                  </div>
                </div>

                {/* Feature Permissions Toggles */}
                <div className="space-y-3 pt-3 border-t border-slate-100 dark:border-surface-800">
                  <h4 className="font-bold text-slate-900 dark:text-white flex items-center gap-1.5 text-xs">
                    <Shield className="w-3.5 h-3.5 text-emerald-500" /> Feature Entitlement Overrides
                  </h4>

                  <div className="space-y-2">
                    {/* Terminal */}
                    <div className="flex items-center justify-between p-2.5 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200/60 dark:border-surface-700">
                      <div>
                        <span className="font-bold text-slate-800 dark:text-slate-200">
                          SSH / Web Terminal Access
                        </span>
                        <p className="text-[10px] text-slate-400">
                          Default: {selectedUser.effective_plan?.permissions?.terminal ? 'Enabled' : 'Disabled'}
                        </p>
                      </div>
                      <div className="flex items-center gap-1">
                        <button
                          type="button"
                          onClick={() => setPermTerminal(permTerminal === true ? null : true)}
                          className={`px-2 py-1 text-[11px] font-bold rounded ${
                            permTerminal === true ? 'bg-emerald-600 text-white' : 'bg-slate-200 dark:bg-surface-700 text-slate-600 dark:text-slate-300'
                          }`}
                        >
                          Enable
                        </button>
                        <button
                          type="button"
                          onClick={() => setPermTerminal(permTerminal === false ? null : false)}
                          className={`px-2 py-1 text-[11px] font-bold rounded ${
                            permTerminal === false ? 'bg-rose-600 text-white' : 'bg-slate-200 dark:bg-surface-700 text-slate-600 dark:text-slate-300'
                          }`}
                        >
                          Disable
                        </button>
                      </div>
                    </div>

                    {/* Backups */}
                    <div className="flex items-center justify-between p-2.5 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200/60 dark:border-surface-700">
                      <div>
                        <span className="font-bold text-slate-800 dark:text-slate-200">
                          Snapshots & Backups
                        </span>
                        <p className="text-[10px] text-slate-400">
                          Default: {selectedUser.effective_plan?.permissions?.backups ? 'Enabled' : 'Disabled'}
                        </p>
                      </div>
                      <div className="flex items-center gap-1">
                        <button
                          type="button"
                          onClick={() => setPermBackups(permBackups === true ? null : true)}
                          className={`px-2 py-1 text-[11px] font-bold rounded ${
                            permBackups === true ? 'bg-emerald-600 text-white' : 'bg-slate-200 dark:bg-surface-700 text-slate-600 dark:text-slate-300'
                          }`}
                        >
                          Enable
                        </button>
                        <button
                          type="button"
                          onClick={() => setPermBackups(permBackups === false ? null : false)}
                          className={`px-2 py-1 text-[11px] font-bold rounded ${
                            permBackups === false ? 'bg-rose-600 text-white' : 'bg-slate-200 dark:bg-surface-700 text-slate-600 dark:text-slate-300'
                          }`}
                        >
                          Disable
                        </button>
                      </div>
                    </div>
                  </div>
                </div>

                {/* Audit Notes */}
                <div className="pt-2">
                  <label className="block text-[11px] text-slate-500 mb-1 font-bold">
                    Admin Audit Notes
                  </label>
                  <input
                    type="text"
                    placeholder="e.g. VIP client upgrade, enterprise custom trial"
                    value={overrideNotes}
                    onChange={(e) => setOverrideNotes(e.target.value)}
                    className="w-full px-3 py-1.5 text-xs rounded-lg bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700"
                  />
                </div>
              </div>

              <div className="flex items-center justify-end gap-2 pt-2 border-t border-slate-100 dark:border-surface-800">
                <button
                  type="button"
                  onClick={() => setEditOverrideModal(false)}
                  className="px-3.5 py-2 text-xs font-semibold text-slate-600 hover:text-slate-800"
                >
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={handleSaveOverrides}
                  disabled={saving}
                  className="px-4 py-2 text-xs font-bold bg-amber-600 text-white rounded-xl hover:bg-amber-700 disabled:opacity-50"
                >
                  {saving ? 'Saving...' : 'Apply Overrides'}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
