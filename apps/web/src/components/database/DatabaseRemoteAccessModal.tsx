'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  X,
  Shield,
  ShieldAlert,
  ShieldCheck,
  Plus,
  Trash2,
  RefreshCw,
  CheckCircle2,
  AlertTriangle,
  Activity,
  Globe,
  Key,
  Clock,
  Lock,
} from 'lucide-react';
import {
  Database,
  DatabaseRemoteAccess,
  listDatabaseRemoteAccess,
  grantDatabaseRemoteAccess,
  revokeDatabaseRemoteAccess,
  testDatabaseRemoteAccess,
} from '@/lib/api';

interface DatabaseRemoteAccessModalProps {
  database: Database | null;
  onClose: () => void;
  showToast: (msg: string) => void;
}

export function DatabaseRemoteAccessModal({
  database,
  onClose,
  showToast,
}: DatabaseRemoteAccessModalProps) {
  const [grants, setGrants] = useState<DatabaseRemoteAccess[]>([]);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [revokingId, setRevokingId] = useState<string | null>(null);

  // Form state
  const [dbUser, setDbUser] = useState('');
  const [authorizedIP, setAuthorizedIP] = useState('');
  const [privilegePreset, setPrivilegePreset] = useState<'ALL' | 'CRUD' | 'READONLY' | 'CUSTOM'>('ALL');
  const [customPrivileges, setCustomPrivileges] = useState('SELECT, INSERT, UPDATE, DELETE');
  const [allowWildcard, setAllowWildcard] = useState(false);
  const [notes, setNotes] = useState('');

  // Test connection state
  const [testingId, setTestingId] = useState<string | null>(null);
  const [testPassword, setTestPassword] = useState('');
  const [testResult, setTestResult] = useState<{
    success: boolean;
    message: string;
  } | null>(null);

  const fetchGrants = useCallback(async () => {
    if (!database) return;
    setLoading(true);
    try {
      const res = await listDatabaseRemoteAccess(database.id);
      if (res.success && res.data) {
        setGrants(res.data);
      }
    } catch {
      showToast('Failed to load remote access grants');
    } finally {
      setLoading(false);
    }
  }, [database, showToast]);

  useEffect(() => {
    if (database) {
      setDbUser(database.username || '');
      fetchGrants();
    }
  }, [database, fetchGrants]);

  const handleGrant = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!database) return;

    const cleanIP = authorizedIP.trim();
    if (!cleanIP) {
      showToast('Please enter an authorized IP address or CIDR');
      return;
    }

    if ((cleanIP === '%' || cleanIP === '0.0.0.0/0') && !allowWildcard) {
      showToast('Please confirm security acknowledgment to allow unrestricted wildcard % access');
      return;
    }

    let privs = 'ALL PRIVILEGES';
    if (privilegePreset === 'CRUD') {
      privs = 'SELECT, INSERT, UPDATE, DELETE';
    } else if (privilegePreset === 'READONLY') {
      privs = 'SELECT';
    } else if (privilegePreset === 'CUSTOM') {
      privs = customPrivileges;
    }

    setSubmitting(true);
    try {
      const res = await grantDatabaseRemoteAccess(database.id, {
        db_user: dbUser || database.username,
        authorized_ip: cleanIP,
        privileges: privs,
        allow_unrestricted_wildcard: allowWildcard,
        notes: notes,
      });

      if (res.success && res.data) {
        showToast('Remote access grant created and firewall updated');
        setAuthorizedIP('');
        setNotes('');
        setAllowWildcard(false);
        fetchGrants();
      } else {
        showToast(res.error?.message || 'Failed to grant remote access');
      }
    } catch {
      showToast('Error communicating with database access engine');
    } finally {
      setSubmitting(false);
    }
  };

  const handleRevoke = async (grantId: string, ip: string) => {
    if (!database) return;
    if (!confirm(`Revoke remote access grant for IP ${ip}? This will drop user privileges and remove the firewall allow rule.`)) {
      return;
    }

    setRevokingId(grantId);
    try {
      const res = await revokeDatabaseRemoteAccess(database.id, grantId);
      if (res.success) {
        showToast(`Remote access for ${ip} revoked`);
        fetchGrants();
      } else {
        showToast(res.error?.message || 'Failed to revoke remote access');
      }
    } catch {
      showToast('Error communicating with database engine');
    } finally {
      setRevokingId(null);
    }
  };

  const handleTestConnection = async (grant: DatabaseRemoteAccess) => {
    if (!database) return;
    setTestingId(grant.id);
    setTestResult(null);
    try {
      const res = await testDatabaseRemoteAccess(database.id, {
        host: grant.authorized_ip === '%' ? '127.0.0.1' : grant.authorized_ip,
        port: 3306,
        db_user: grant.db_user,
        password: testPassword,
      });

      if (res.success && res.data) {
        setTestResult({
          success: res.data.success,
          message: res.data.success
            ? `Connection Verified! Latency: ${res.data.latency_ms}ms`
            : res.data.error || 'Connection probe failed',
        });
      } else {
        setTestResult({
          success: false,
          message: res.error?.message || 'Connection test probe failed',
        });
      }
    } catch {
      setTestResult({ success: false, message: 'Probe execution failed' });
    } finally {
      setTestingId(null);
    }
  };

  if (!database) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-in fade-in duration-200">
      <div className="relative w-full max-w-4xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl shadow-2xl flex flex-col max-h-[90vh] overflow-hidden text-slate-800 dark:text-slate-100">
        {/* Modal Header */}
        <div className="p-5 border-b border-slate-200 dark:border-slate-800 flex items-center justify-between bg-slate-50/50 dark:bg-slate-950/40">
          <div className="flex items-center gap-3">
            <div className="p-2.5 rounded-xl bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
              <Globe className="w-5 h-5" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h3 className="text-base font-bold text-slate-900 dark:text-slate-100">
                  Remote Database Access Control
                </h3>
                <span className="px-2 py-0.5 text-xs font-mono font-bold rounded-md bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300">
                  {database.name}
                </span>
              </div>
              <p className="text-xs text-slate-500 mt-0.5">
                Manage authorized source-IP allowlists, least-privilege grants, and automated firewall rules.
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition cursor-pointer"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Modal Body */}
        <div className="p-6 overflow-y-auto space-y-6 flex-1">
          {/* Grant Creation Form */}
          <form
            onSubmit={handleGrant}
            className="p-4 rounded-xl bg-slate-50 dark:bg-slate-950/60 border border-slate-200 dark:border-slate-800 space-y-4"
          >
            <div className="flex items-center gap-2 border-b border-slate-200 dark:border-slate-800 pb-2">
              <Plus className="w-4 h-4 text-emerald-500" />
              <h4 className="text-xs font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300">
                Grant New Authorized Source IP Access
              </h4>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-3 text-xs">
              {/* Database User */}
              <div className="space-y-1">
                <label className="font-semibold text-slate-700 dark:text-slate-300">Database User</label>
                <input
                  type="text"
                  value={dbUser}
                  onChange={(e) => setDbUser(e.target.value)}
                  placeholder="e.g. app_user"
                  className="w-full p-2 text-xs rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 font-mono"
                  required
                />
              </div>

              {/* Authorized IP or CIDR */}
              <div className="space-y-1 md:col-span-2">
                <label className="font-semibold text-slate-700 dark:text-slate-300">
                  Authorized Source IP / Subnet (CIDR)
                </label>
                <input
                  type="text"
                  value={authorizedIP}
                  onChange={(e) => setAuthorizedIP(e.target.value)}
                  placeholder="e.g. 198.51.100.25 or 10.0.0.0/24 or % for any"
                  className="w-full p-2 text-xs rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 font-mono"
                  required
                />
              </div>

              {/* Privilege Preset */}
              <div className="space-y-1 md:col-span-2">
                <label className="font-semibold text-slate-700 dark:text-slate-300">Privilege Scope</label>
                <div className="grid grid-cols-3 gap-2">
                  <button
                    type="button"
                    onClick={() => setPrivilegePreset('ALL')}
                    className={`p-2 rounded-lg border text-left cursor-pointer transition ${
                      privilegePreset === 'ALL'
                        ? 'border-emerald-500 bg-emerald-50/50 dark:bg-emerald-950/20 text-emerald-800 dark:text-emerald-300 font-bold'
                        : 'border-slate-200 dark:border-slate-700 hover:bg-white dark:hover:bg-slate-800'
                    }`}
                  >
                    <span>ALL PRIVILEGES</span>
                    <span className="block text-[10px] text-slate-400 font-normal">Full schema permissions</span>
                  </button>
                  <button
                    type="button"
                    onClick={() => setPrivilegePreset('CRUD')}
                    className={`p-2 rounded-lg border text-left cursor-pointer transition ${
                      privilegePreset === 'CRUD'
                        ? 'border-emerald-500 bg-emerald-50/50 dark:bg-emerald-950/20 text-emerald-800 dark:text-emerald-300 font-bold'
                        : 'border-slate-200 dark:border-slate-700 hover:bg-white dark:hover:bg-slate-800'
                    }`}
                  >
                    <span>Read / Write (CRUD)</span>
                    <span className="block text-[10px] text-slate-400 font-normal">SELECT, INSERT, UPDATE, DELETE</span>
                  </button>
                  <button
                    type="button"
                    onClick={() => setPrivilegePreset('READONLY')}
                    className={`p-2 rounded-lg border text-left cursor-pointer transition ${
                      privilegePreset === 'READONLY'
                        ? 'border-emerald-500 bg-emerald-50/50 dark:bg-emerald-950/20 text-emerald-800 dark:text-emerald-300 font-bold'
                        : 'border-slate-200 dark:border-slate-700 hover:bg-white dark:hover:bg-slate-800'
                    }`}
                  >
                    <span>Read-Only</span>
                    <span className="block text-[10px] text-slate-400 font-normal">SELECT queries only</span>
                  </button>
                </div>
              </div>

              {/* Notes */}
              <div className="space-y-1">
                <label className="font-semibold text-slate-700 dark:text-slate-300">Reference Notes</label>
                <input
                  type="text"
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  placeholder="e.g. Staging server, BI replica"
                  className="w-full p-2 text-xs rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100"
                />
              </div>
            </div>

            {/* Wildcard Security Warning */}
            {(authorizedIP.trim() === '%' || authorizedIP.trim() === '0.0.0.0/0') && (
              <div className="p-3 rounded-lg bg-rose-500/10 border border-rose-500/30 text-rose-800 dark:text-rose-200 text-xs flex items-start gap-2.5">
                <AlertTriangle className="w-4 h-4 text-rose-500 shrink-0 mt-0.5" />
                <div className="space-y-1">
                  <span className="font-bold">Security Warning: Unrestricted Wildcard Host (%)</span>
                  <p className="text-[11px] opacity-90">
                    Granting access to % exposes your database to any IP on the internet. Ensure your user has a robust 20+ character password.
                  </p>
                  <label className="flex items-center gap-2 pt-1 font-semibold cursor-pointer">
                    <input
                      type="checkbox"
                      checked={allowWildcard}
                      onChange={(e) => setAllowWildcard(e.target.checked)}
                      className="accent-rose-500 rounded"
                    />
                    <span>I understand the security implications and explicitly allow wildcard host (%)</span>
                  </label>
                </div>
              </div>
            )}

            <div className="flex justify-end pt-1">
              <button
                type="submit"
                disabled={submitting}
                className="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 disabled:opacity-50 text-white font-bold text-xs flex items-center gap-1.5 transition shadow-xs cursor-pointer"
              >
                {submitting ? (
                  <>
                    <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                    <span>Granting Access...</span>
                  </>
                ) : (
                  <>
                    <ShieldCheck className="w-3.5 h-3.5" />
                    <span>Authorize Remote IP</span>
                  </>
                )}
              </button>
            </div>
          </form>

          {/* Active Grants List */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <h4 className="text-xs font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300">
                Active Authorized Remote Grants ({grants.length})
              </h4>
              <button
                type="button"
                onClick={fetchGrants}
                disabled={loading}
                className="px-2.5 py-1 text-xs rounded-lg border border-slate-300 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-600 dark:text-slate-300 flex items-center gap-1 transition cursor-pointer"
              >
                <RefreshCw className={`w-3 h-3 ${loading ? 'animate-spin' : ''}`} />
                <span>Reload</span>
              </button>
            </div>

            {testResult && (
              <div
                className={`p-3 rounded-lg text-xs flex items-center justify-between ${
                  testResult.success
                    ? 'bg-emerald-500/10 text-emerald-800 dark:text-emerald-200 border border-emerald-500/30'
                    : 'bg-rose-500/10 text-rose-800 dark:text-rose-200 border border-rose-500/30'
                }`}
              >
                <div className="flex items-center gap-2">
                  {testResult.success ? (
                    <CheckCircle2 className="w-4 h-4 text-emerald-500" />
                  ) : (
                    <AlertTriangle className="w-4 h-4 text-rose-500" />
                  )}
                  <span>{testResult.message}</span>
                </div>
                <button
                  type="button"
                  onClick={() => setTestResult(null)}
                  className="text-xs opacity-70 hover:opacity-100 cursor-pointer"
                >
                  ✕
                </button>
              </div>
            )}

            {grants.length === 0 ? (
              <div className="p-8 text-center border border-dashed border-slate-200 dark:border-slate-800 rounded-xl text-slate-400 text-xs">
                <Shield className="w-8 h-8 mx-auto mb-2 opacity-50" />
                <p>No external source-IP grants configured for this database.</p>
                <p className="text-[11px] text-slate-500 mt-1">
                  Access is strictly confined to localhost/Unix socket. Add an authorized IP above to permit remote access.
                </p>
              </div>
            ) : (
              <div className="border border-slate-200 dark:border-slate-800 rounded-xl overflow-hidden divide-y divide-slate-200 dark:divide-slate-800 text-xs">
                {grants.map((grant) => (
                  <div
                    key={grant.id}
                    className="p-3.5 bg-white dark:bg-slate-900/40 hover:bg-slate-50 dark:hover:bg-slate-800/40 flex flex-col md:flex-row md:items-center justify-between gap-3 transition"
                  >
                    <div className="space-y-1">
                      <div className="flex items-center gap-2">
                        <span className="font-mono font-bold text-slate-900 dark:text-slate-100">
                          {grant.db_user}@{grant.authorized_ip}
                        </span>
                        {grant.firewall_rule_applied ? (
                          <span className="px-2 py-0.5 text-[10px] font-semibold rounded-md bg-emerald-50 dark:bg-emerald-950/40 text-emerald-700 dark:text-emerald-400 border border-emerald-300 dark:border-emerald-800 flex items-center gap-1">
                            <ShieldCheck className="w-3 h-3 text-emerald-500" />
                            UFW Allowed
                          </span>
                        ) : (
                          <span className="px-2 py-0.5 text-[10px] font-semibold rounded-md bg-amber-50 dark:bg-amber-950/40 text-amber-700 dark:text-amber-400 border border-amber-300 dark:border-amber-800 flex items-center gap-1">
                            <ShieldAlert className="w-3 h-3 text-amber-500" />
                            Wildcard Host
                          </span>
                        )}
                        <span className="px-2 py-0.5 text-[10px] rounded-md bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-300 font-mono">
                          {grant.granted_privileges}
                        </span>
                      </div>
                      {grant.notes && (
                        <p className="text-[11px] text-slate-500 dark:text-slate-400 italic">
                          Note: {grant.notes}
                        </p>
                      )}
                    </div>

                    <div className="flex items-center gap-2">
                      <button
                        type="button"
                        onClick={() => handleTestConnection(grant)}
                        disabled={testingId === grant.id}
                        className="px-2.5 py-1 rounded-lg border border-slate-300 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-700 dark:text-slate-300 text-xs font-medium flex items-center gap-1.5 transition cursor-pointer"
                      >
                        <Activity className={`w-3.5 h-3.5 ${testingId === grant.id ? 'animate-spin' : ''}`} />
                        <span>{testingId === grant.id ? 'Probing...' : 'Test Probe'}</span>
                      </button>

                      <button
                        type="button"
                        onClick={() => handleRevoke(grant.id, grant.authorized_ip)}
                        disabled={revokingId === grant.id}
                        className="px-2.5 py-1 rounded-lg bg-rose-50 dark:bg-rose-950/30 border border-rose-300 dark:border-rose-800 text-rose-700 dark:text-rose-400 hover:bg-rose-100 text-xs font-bold flex items-center gap-1.5 transition cursor-pointer"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                        <span>{revokingId === grant.id ? 'Revoking...' : 'Revoke'}</span>
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Modal Footer */}
        <div className="p-4 border-t border-slate-200 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-950/40 flex justify-end">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 rounded-lg border border-slate-300 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800 text-xs font-bold transition cursor-pointer"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
