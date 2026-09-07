'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  FolderSync,
  Plus,
  Trash2,
  Power,
  RefreshCw,
  Search,
  CheckCircle2,
  AlertCircle,
  X,
  Key,
  Folder,
  Copy,
  Check,
  Server,
  HardDrive,
  Activity,
  Edit2,
  ShieldCheck,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import {
  apiFetch,
  FTPDaemonStatus,
  FTPUser,
} from '@/lib/api';

export default function FTPPage() {
  const [status, setStatus] = useState<FTPDaemonStatus | null>(null);
  const [users, setUsers] = useState<FTPUser[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [search, setSearch] = useState('');

  // Alerts
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  // Add User Modal
  const [addModalOpen, setAddModalOpen] = useState(false);
  const [newUsername, setNewUsername] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [newHomeDir, setNewHomeDir] = useState('/var/www');
  const [newQuota, setNewQuota] = useState('2048');
  const [newUploadSpeed, setNewUploadSpeed] = useState('0');
  const [newDownloadSpeed, setNewDownloadSpeed] = useState('0');
  const [submittingAdd, setSubmittingAdd] = useState(false);

  // Change Password Modal
  const [passwordModalOpen, setPasswordModalOpen] = useState(false);
  const [targetUsername, setTargetUsername] = useState('');
  const [changedPassword, setChangedPassword] = useState('');
  const [submittingPassword, setSubmittingPassword] = useState(false);

  // Edit User Modal
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [editUsername, setEditUsername] = useState('');
  const [editHomeDir, setEditHomeDir] = useState('');
  const [editQuota, setEditQuota] = useState('');
  const [editUploadSpeed, setEditUploadSpeed] = useState('0');
  const [editDownloadSpeed, setEditDownloadSpeed] = useState('0');
  const [submittingEdit, setSubmittingEdit] = useState(false);

  // Action Loading states
  const [actionLoadingId, setActionLoadingId] = useState<string | null>(null);
  const [copiedItem, setCopiedItem] = useState<string | null>(null);

  const showNotification = (msg: string, isError = false) => {
    if (isError) {
      setErrorMsg(msg);
      setTimeout(() => setErrorMsg(null), 7000);
    } else {
      setSuccessMsg(msg);
      setTimeout(() => setSuccessMsg(null), 5000);
    }
  };

  const loadData = useCallback(async (showIndicator = true) => {
    if (showIndicator) setRefreshing(true);
    setErrorMsg(null);

    try {
      // 1. Status
      const statusRes = await apiFetch<FTPDaemonStatus>('/api/v1/ftp/status');
      if (statusRes.success && statusRes.data) {
        setStatus(statusRes.data);
      }

      // 2. Users
      const usersRes = await apiFetch<{ users: FTPUser[]; count: number }>('/api/v1/ftp/users');
      if (usersRes.success && usersRes.data) {
        setUsers(usersRes.data.users || []);
      }
    } catch (err: any) {
      showNotification(err.message || 'Failed to fetch FTP accounts', true);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    loadData(false);
  }, [loadData]);

  // Generate strong random password
  const generatePassword = () => {
    const chars = 'abcdefghjkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789!@#$%&*';
    let pass = '';
    for (let i = 0; i < 16; i++) {
      pass += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    return pass;
  };

  // Open Add Modal
  const handleOpenAdd = () => {
    setNewUsername('');
    setNewPassword(generatePassword());
    setNewHomeDir('/var/www');
    setNewQuota('2048');
    setNewUploadSpeed('0');
    setNewDownloadSpeed('0');
    setAddModalOpen(true);
  };

  // Create User
  const handleCreateUser = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newUsername.trim() || !newPassword.trim() || !newHomeDir.trim()) {
      showNotification('Username, password, and home directory are required', true);
      return;
    }

    setSubmittingAdd(true);
    const res = await apiFetch<FTPUser>('/api/v1/ftp/users', {
      method: 'POST',
      body: JSON.stringify({
        username: newUsername.trim(),
        password: newPassword.trim(),
        home_dir: newHomeDir.trim(),
        quota_mb: parseInt(newQuota, 10) || 0,
        upload_bandwidth_kbps: parseInt(newUploadSpeed, 10) || 0,
        download_bandwidth_kbps: parseInt(newDownloadSpeed, 10) || 0,
      }),
    });
    setSubmittingAdd(false);

    if (res.success) {
      showNotification(`FTP account [${newUsername}] created successfully`);
      setAddModalOpen(false);
      loadData(false);
    } else {
      showNotification(res.error?.message || 'Failed to create FTP account', true);
    }
  };

  // Change Password
  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!changedPassword.trim() || changedPassword.length < 6) {
      showNotification('Password must be at least 6 characters', true);
      return;
    }

    setSubmittingPassword(true);
    const res = await apiFetch<any>(`/api/v1/ftp/users/${targetUsername}/password`, {
      method: 'PUT',
      body: JSON.stringify({ password: changedPassword.trim() }),
    });
    setSubmittingPassword(false);

    if (res.success) {
      showNotification(`Password for [${targetUsername}] updated successfully`);
      setPasswordModalOpen(false);
    } else {
      showNotification(res.error?.message || 'Failed to update password', true);
    }
  };

  // Open Edit Modal
  const handleOpenEdit = (user: FTPUser) => {
    setEditUsername(user.username);
    setEditHomeDir(user.home_dir);
    setEditQuota((user.quota_mb || 0).toString());
    setEditUploadSpeed((user.upload_bandwidth_kbps || 0).toString());
    setEditDownloadSpeed((user.download_bandwidth_kbps || 0).toString());
    setEditModalOpen(true);
  };

  // Update User
  const handleUpdateUser = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editHomeDir.trim()) return;

    setSubmittingEdit(true);
    const res = await apiFetch<any>(`/api/v1/ftp/users/${editUsername}`, {
      method: 'PUT',
      body: JSON.stringify({
        home_dir: editHomeDir.trim(),
        quota_mb: parseInt(editQuota, 10) || 0,
        upload_bandwidth_kbps: parseInt(editUploadSpeed, 10) || 0,
        download_bandwidth_kbps: parseInt(editDownloadSpeed, 10) || 0,
      }),
    });
    setSubmittingEdit(false);

    if (res.success) {
      showNotification(`FTP user [${editUsername}] updated successfully`);
      setEditModalOpen(false);
      loadData(false);
    } else {
      showNotification(res.error?.message || 'Failed to update FTP account', true);
    }
  };

  // Toggle User
  const handleToggleUser = async (username: string) => {
    setActionLoadingId(`toggle-${username}`);
    const res = await apiFetch<FTPUser>(`/api/v1/ftp/users/${username}/toggle`, {
      method: 'POST',
    });
    setActionLoadingId(null);

    if (res.success && res.data) {
      setUsers((prev) =>
        prev.map((u) => (u.username === username ? { ...u, is_enabled: res.data!.is_enabled } : u))
      );
      showNotification(res.data.is_enabled ? `Account [${username}] enabled` : `Account [${username}] disabled`);
    } else {
      showNotification(res.error?.message || 'Failed to toggle account state', true);
    }
  };

  // Delete User
  const handleDeleteUser = async (user: FTPUser) => {
    if (!window.confirm(`Are you sure you want to permanently delete FTP user "${user.username}"?`)) {
      return;
    }

    setActionLoadingId(`del-${user.username}`);
    const res = await apiFetch<any>(`/api/v1/ftp/users/${user.username}`, {
      method: 'DELETE',
    });
    setActionLoadingId(null);

    if (res.success) {
      showNotification(`FTP user [${user.username}] removed`);
      loadData(false);
    } else {
      showNotification(res.error?.message || 'Failed to delete FTP user', true);
    }
  };

  const copyToClipboard = (text: string, id: string) => {
    navigator.clipboard.writeText(text);
    setCopiedItem(id);
    setTimeout(() => setCopiedItem(null), 2000);
  };

  const filteredUsers = users.filter(
    (u) =>
      u.username.toLowerCase().includes(search.toLowerCase()) ||
      u.home_dir.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <DashboardShell>
      <div className="space-y-6 animate-fadeIn max-w-7xl mx-auto pb-16">
        {/* Alerts */}
        {errorMsg && (
          <div className="p-4 rounded-xl bg-rose-50 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-900/50 flex items-start gap-3 text-rose-800 dark:text-rose-200 text-sm shadow-xs">
            <AlertCircle className="w-5 h-5 flex-shrink-0 text-rose-600 dark:text-rose-400 mt-0.5" />
            <div className="flex-1 font-medium">{errorMsg}</div>
            <button onClick={() => setErrorMsg(null)} className="text-rose-600 dark:text-rose-400 hover:opacity-75">
              <X className="w-4 h-4" />
            </button>
          </div>
        )}

        {successMsg && (
          <div className="p-4 rounded-xl bg-emerald-50 dark:bg-emerald-950/40 border border-emerald-200 dark:border-emerald-900/50 flex items-start gap-3 text-emerald-800 dark:text-emerald-200 text-sm shadow-xs">
            <CheckCircle2 className="w-5 h-5 flex-shrink-0 text-emerald-600 dark:text-emerald-400 mt-0.5" />
            <div className="flex-1 font-medium">{successMsg}</div>
            <button onClick={() => setSuccessMsg(null)} className="text-emerald-600 dark:text-emerald-400 hover:opacity-75">
              <X className="w-4 h-4" />
            </button>
          </div>
        )}

        {/* Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold tracking-tight text-slate-950 dark:text-white">
                FTP User Management
              </h1>
              <span className="px-2.5 py-0.5 rounded-full text-xs font-mono font-semibold uppercase bg-indigo-50 dark:bg-indigo-950/60 text-indigo-700 dark:text-indigo-400 border border-indigo-200 dark:border-indigo-800">
                {status?.daemon_name || 'Pure-FTPd'}
              </span>
            </div>
            <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
              Virtual user accounts with chroot jail directory isolation, storage quotas, and bandwidth limits.
            </p>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => loadData(true)}
              disabled={refreshing}
              className="p-2.5 rounded-xl border border-slate-300 dark:border-surface-700 bg-white dark:bg-surface-900 text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-surface-800 transition-colors shadow-xs"
              title="Refresh FTP Users"
            >
              <RefreshCw className={`w-4 h-4 ${refreshing ? 'animate-spin text-indigo-500' : ''}`} />
            </button>

            <button
              onClick={handleOpenAdd}
              className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-semibold shadow-md shadow-indigo-600/20 transition-all"
            >
              <Plus className="w-4 h-4" />
              Add FTP Account
            </button>
          </div>
        </div>

        {/* Connection Parameters & Status Banner */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {/* Card 1: Daemon Status */}
          <div className="p-5 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 shadow-xs flex items-center gap-3.5">
            <div
              className={`w-11 h-11 rounded-xl flex items-center justify-center ${
                status?.is_active
                  ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20'
                  : 'bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20'
              }`}
            >
              <Server className="w-6 h-6" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="font-bold text-slate-950 dark:text-white text-sm">
                  {status?.is_active ? 'FTP Service Active' : 'FTP Service Offline'}
                </span>
                <span
                  className={`w-2 h-2 rounded-full ${
                    status?.is_active ? 'bg-emerald-500 animate-pulse' : 'bg-amber-500'
                  }`}
                />
              </div>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5 font-medium">
                Standard Port 21 (FTP/FTPS TLS Enabled)
              </p>
            </div>
          </div>

          {/* Card 2: Total Virtual Accounts */}
          <div className="p-5 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 shadow-xs flex items-center gap-3.5">
            <div className="w-11 h-11 rounded-xl bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 border border-indigo-500/20 flex items-center justify-center">
              <FolderSync className="w-6 h-6" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="font-bold text-slate-950 dark:text-white text-sm">FTP Accounts</span>
                <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 border border-indigo-500/20">
                  {users.length} Virtual Users
                </span>
              </div>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5 font-medium">
                {users.filter((u) => u.is_enabled).length} Active • {users.filter((u) => !u.is_enabled).length} Suspended
              </p>
            </div>
          </div>

          {/* Card 3: Security & Chroot */}
          <div className="p-5 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 shadow-xs flex items-center gap-3.5">
            <div className="w-11 h-11 rounded-xl bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20 flex items-center justify-center">
              <ShieldCheck className="w-6 h-6" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="font-bold text-slate-950 dark:text-white text-sm">Chroot Isolation</span>
                <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
                  Enforced
                </span>
              </div>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5 font-medium">
                Users jailed to designated directory tree
              </p>
            </div>
          </div>
        </div>

        {/* Search Bar */}
        <div className="flex flex-col sm:flex-row items-center justify-between gap-4 bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl p-3 shadow-xs">
          <div className="flex items-center gap-3 w-full sm:w-80 bg-slate-50 dark:bg-[#121824] border border-slate-300 dark:border-surface-700 rounded-xl px-3.5 py-2 shadow-xs focus-within:border-indigo-500 focus-within:ring-2 focus-within:ring-indigo-500/20 transition-all">
            <Search className="w-4 h-4 text-slate-400 flex-shrink-0" />
            <input
              type="text"
              placeholder="Search by username or home path..."
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
          <div className="text-xs font-semibold text-slate-600 dark:text-slate-400 px-2 self-end sm:self-center">
            Showing {filteredUsers.length} of {users.length} FTP accounts
          </div>
        </div>

        {/* Users Table */}
        <div className="bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl overflow-hidden shadow-xs">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm border-collapse">
              <thead>
                <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-[#121824] text-slate-700 dark:text-slate-300 text-[11px] font-bold uppercase tracking-wider">
                  <th className="px-6 py-3.5">FTP Username</th>
                  <th className="px-6 py-3.5">Chroot Home Directory</th>
                  <th className="px-6 py-3.5">Quota Limit</th>
                  <th className="px-6 py-3.5">Bandwidth (Up/Down)</th>
                  <th className="px-6 py-3.5">Account Status</th>
                  <th className="px-6 py-3.5 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200 dark:divide-surface-800/80">
                {loading ? (
                  <tr>
                    <td colSpan={6} className="px-6 py-12 text-center text-slate-500 font-medium text-sm">
                      <RefreshCw className="w-6 h-6 animate-spin mx-auto mb-2 text-indigo-500" />
                      Loading virtual FTP accounts...
                    </td>
                  </tr>
                ) : filteredUsers.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="px-6 py-12 text-center text-slate-500 font-medium text-sm">
                      No FTP accounts found. Click "Add FTP Account" to configure one.
                    </td>
                  </tr>
                ) : (
                  filteredUsers.map((u) => (
                    <tr key={u.username} className="hover:bg-slate-50/80 dark:hover:bg-[#151d2d] transition-colors">
                      <td className="px-6 py-4 font-bold text-slate-950 dark:text-white flex items-center gap-2">
                        <FolderSync className="w-4 h-4 text-indigo-500 flex-shrink-0" />
                        <span>{u.username}</span>
                      </td>
                      <td className="px-6 py-4 font-mono text-xs text-slate-700 dark:text-slate-300">
                        <div className="flex items-center gap-2">
                          <span className="truncate max-w-sm">{u.home_dir}</span>
                          <button
                            onClick={() => copyToClipboard(u.home_dir, u.username)}
                            className="p-1 rounded text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
                            title="Copy Path"
                          >
                            {copiedItem === u.username ? (
                              <Check className="w-3 h-3 text-emerald-500" />
                            ) : (
                              <Copy className="w-3 h-3" />
                            )}
                          </button>
                        </div>
                      </td>
                      <td className="px-6 py-4 font-mono text-xs text-slate-700 dark:text-slate-300">
                        {u.quota_mb ? `${u.quota_mb} MB` : 'Unlimited'}
                      </td>
                      <td className="px-6 py-4 font-mono text-xs text-slate-700 dark:text-slate-300">
                        {u.upload_bandwidth_kbps || u.download_bandwidth_kbps
                          ? `${u.upload_bandwidth_kbps || '∞'} / ${u.download_bandwidth_kbps || '∞'} kB/s`
                          : 'Unlimited'}
                      </td>
                      <td className="px-6 py-4">
                        <span
                          className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-[11px] font-bold uppercase ${
                            u.is_enabled
                              ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20'
                              : 'bg-slate-100 dark:bg-surface-800 text-slate-500 border border-slate-300 dark:border-surface-700'
                          }`}
                        >
                          {u.is_enabled ? 'Active' : 'Suspended'}
                        </span>
                      </td>
                      <td className="px-6 py-4 text-right">
                        <div className="flex items-center justify-end gap-1.5">
                          <button
                            onClick={() => {
                              setTargetUsername(u.username);
                              setChangedPassword(generatePassword());
                              setPasswordModalOpen(true);
                            }}
                            title="Change Password"
                            className="p-1.5 rounded-lg border border-slate-300 dark:border-surface-700 text-slate-600 dark:text-slate-400 hover:text-indigo-600 dark:hover:text-indigo-400 hover:bg-indigo-50 dark:hover:bg-indigo-950/30 transition-colors"
                          >
                            <Key className="w-3.5 h-3.5" />
                          </button>

                          <button
                            onClick={() => handleOpenEdit(u)}
                            title="Edit Account Quota & Dir"
                            className="p-1.5 rounded-lg border border-slate-300 dark:border-surface-700 text-slate-600 dark:text-slate-400 hover:text-indigo-600 dark:hover:text-indigo-400 hover:bg-indigo-50 dark:hover:bg-indigo-950/30 transition-colors"
                          >
                            <Edit2 className="w-3.5 h-3.5" />
                          </button>

                          <button
                            onClick={() => handleToggleUser(u.username)}
                            disabled={actionLoadingId === `toggle-${u.username}`}
                            title={u.is_enabled ? 'Suspend Account' : 'Activate Account'}
                            className="p-1.5 rounded-lg border border-slate-300 dark:border-surface-700 text-slate-600 dark:text-slate-400 hover:text-amber-600 dark:hover:text-amber-400 hover:bg-amber-50 dark:hover:bg-amber-950/30 transition-colors"
                          >
                            <Power className="w-3.5 h-3.5" />
                          </button>

                          <button
                            onClick={() => handleDeleteUser(u)}
                            disabled={actionLoadingId === `del-${u.username}`}
                            title="Delete FTP User"
                            className="p-1.5 rounded-lg border border-slate-300 dark:border-surface-700 text-slate-600 dark:text-slate-400 hover:text-rose-600 dark:hover:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-950/30 transition-colors"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>

        {/* Modal: Add FTP Account */}
        {addModalOpen && (
          <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="w-full max-w-lg bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setAddModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white transition-colors"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-5">
                <div className="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-600 dark:text-indigo-400">
                  <FolderSync className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-slate-950 dark:text-white">Add FTP Account</h2>
                  <p className="text-xs text-slate-500 dark:text-slate-400">
                    Create a virtual user jailed to a directory path
                  </p>
                </div>
              </div>

              <form onSubmit={handleCreateUser} className="space-y-4">
                <div>
                  <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300 mb-1.5">
                    FTP Username *
                  </label>
                  <input
                    type="text"
                    required
                    placeholder="e.g. site_ftp"
                    value={newUsername}
                    onChange={(e) => setNewUsername(e.target.value)}
                    className="w-full px-3.5 py-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 font-mono text-sm focus:outline-none focus:border-indigo-500"
                  />
                </div>

                <div>
                  <div className="flex justify-between items-center mb-1.5">
                    <label className="text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300">
                      Password *
                    </label>
                    <button
                      type="button"
                      onClick={() => setNewPassword(generatePassword())}
                      className="text-xs text-indigo-600 dark:text-indigo-400 font-semibold hover:underline"
                    >
                      Generate New
                    </button>
                  </div>
                  <input
                    type="text"
                    required
                    value={newPassword}
                    onChange={(e) => setNewPassword(e.target.value)}
                    className="w-full px-3.5 py-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 font-mono text-sm focus:outline-none focus:border-indigo-500"
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300 mb-1.5">
                    Chroot Home Directory *
                  </label>
                  <input
                    type="text"
                    required
                    placeholder="/var/www/mywebsite.com"
                    value={newHomeDir}
                    onChange={(e) => setNewHomeDir(e.target.value)}
                    className="w-full px-3.5 py-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 font-mono text-xs focus:outline-none focus:border-indigo-500"
                  />
                </div>

                <div className="grid grid-cols-3 gap-3">
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300 mb-1.5">
                      Quota (MB)
                    </label>
                    <input
                      type="number"
                      placeholder="0 for unltd"
                      value={newQuota}
                      onChange={(e) => setNewQuota(e.target.value)}
                      className="w-full px-3.5 py-2 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 font-mono text-xs focus:outline-none focus:border-indigo-500"
                    />
                  </div>

                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300 mb-1.5">
                      Upload (kB/s)
                    </label>
                    <input
                      type="number"
                      placeholder="0 for unltd"
                      value={newUploadSpeed}
                      onChange={(e) => setNewUploadSpeed(e.target.value)}
                      className="w-full px-3.5 py-2 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 font-mono text-xs focus:outline-none focus:border-indigo-500"
                    />
                  </div>

                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300 mb-1.5">
                      Down (kB/s)
                    </label>
                    <input
                      type="number"
                      placeholder="0 for unltd"
                      value={newDownloadSpeed}
                      onChange={(e) => setNewDownloadSpeed(e.target.value)}
                      className="w-full px-3.5 py-2 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 font-mono text-xs focus:outline-none focus:border-indigo-500"
                    />
                  </div>
                </div>

                <div className="flex justify-end gap-2 pt-3">
                  <button
                    type="button"
                    onClick={() => setAddModalOpen(false)}
                    className="px-4 py-2.5 rounded-xl text-xs font-medium text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-surface-800 transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={submittingAdd}
                    className="px-5 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-md shadow-indigo-600/20 transition-all flex items-center gap-1.5"
                  >
                    {submittingAdd ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : null}
                    Create Account
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* Modal: Change Password */}
        {passwordModalOpen && (
          <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="w-full max-w-md bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setPasswordModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white transition-colors"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-5">
                <div className="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-600 dark:text-indigo-400">
                  <Key className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-slate-950 dark:text-white">Change Password</h2>
                  <p className="text-xs text-slate-500 dark:text-slate-400">
                    Update password for user <span className="font-mono font-bold text-indigo-600">{targetUsername}</span>
                  </p>
                </div>
              </div>

              <form onSubmit={handleChangePassword} className="space-y-4">
                <div>
                  <div className="flex justify-between items-center mb-1.5">
                    <label className="text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300">
                      New Password *
                    </label>
                    <button
                      type="button"
                      onClick={() => setChangedPassword(generatePassword())}
                      className="text-xs text-indigo-600 dark:text-indigo-400 font-semibold hover:underline"
                    >
                      Generate New
                    </button>
                  </div>
                  <input
                    type="text"
                    required
                    value={changedPassword}
                    onChange={(e) => setChangedPassword(e.target.value)}
                    className="w-full px-3.5 py-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 font-mono text-sm focus:outline-none focus:border-indigo-500"
                  />
                </div>

                <div className="flex justify-end gap-2 pt-3">
                  <button
                    type="button"
                    onClick={() => setPasswordModalOpen(false)}
                    className="px-4 py-2.5 rounded-xl text-xs font-medium text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-surface-800 transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={submittingPassword}
                    className="px-5 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-md shadow-indigo-600/20 transition-all flex items-center gap-1.5"
                  >
                    {submittingPassword ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : null}
                    Save Password
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* Modal: Edit User (Quota, HomeDir) */}
        {editModalOpen && (
          <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="w-full max-w-lg bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setEditModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white transition-colors"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-5">
                <div className="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-600 dark:text-indigo-400">
                  <Edit2 className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-slate-950 dark:text-white">Edit FTP Account</h2>
                  <p className="text-xs text-slate-500 dark:text-slate-400">
                    Modifying settings for <span className="font-mono font-bold text-indigo-600">{editUsername}</span>
                  </p>
                </div>
              </div>

              <form onSubmit={handleUpdateUser} className="space-y-4">
                <div>
                  <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300 mb-1.5">
                    Chroot Home Directory *
                  </label>
                  <input
                    type="text"
                    required
                    value={editHomeDir}
                    onChange={(e) => setEditHomeDir(e.target.value)}
                    className="w-full px-3.5 py-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 font-mono text-xs focus:outline-none focus:border-indigo-500"
                  />
                </div>

                <div className="grid grid-cols-3 gap-3">
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300 mb-1.5">
                      Quota (MB)
                    </label>
                    <input
                      type="number"
                      value={editQuota}
                      onChange={(e) => setEditQuota(e.target.value)}
                      className="w-full px-3.5 py-2 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 font-mono text-xs focus:outline-none focus:border-indigo-500"
                    />
                  </div>

                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300 mb-1.5">
                      Upload (kB/s)
                    </label>
                    <input
                      type="number"
                      value={editUploadSpeed}
                      onChange={(e) => setEditUploadSpeed(e.target.value)}
                      className="w-full px-3.5 py-2 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 font-mono text-xs focus:outline-none focus:border-indigo-500"
                    />
                  </div>

                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300 mb-1.5">
                      Down (kB/s)
                    </label>
                    <input
                      type="number"
                      value={editDownloadSpeed}
                      onChange={(e) => setEditDownloadSpeed(e.target.value)}
                      className="w-full px-3.5 py-2 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 font-mono text-xs focus:outline-none focus:border-indigo-500"
                    />
                  </div>
                </div>

                <div className="flex justify-end gap-2 pt-3">
                  <button
                    type="button"
                    onClick={() => setEditModalOpen(false)}
                    className="px-4 py-2.5 rounded-xl text-xs font-medium text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-surface-800 transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={submittingEdit}
                    className="px-5 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-md shadow-indigo-600/20 transition-all flex items-center gap-1.5"
                  >
                    {submittingEdit ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : null}
                    Save Changes
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
