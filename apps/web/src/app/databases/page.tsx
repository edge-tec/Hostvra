'use client';

import React, { useState, useEffect } from 'react';
import {
  Database as DatabaseIcon,
  Plus,
  Trash2,
  RefreshCw,
  Search,
  Server as ServerIcon,
  Users,
  Key,
  X,
  Check,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import { apiFetch, Database, DatabaseUser, Server } from '@/lib/api';

export default function DatabasesPage() {
  const [servers, setServers] = useState<Server[]>([]);
  const [selectedServer, setSelectedServer] = useState<string>('');
  const [databases, setDatabases] = useState<Database[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');

  // Modals
  const [createDbOpen, setCreateDbOpen] = useState(false);
  const [dbName, setDbName] = useState('');
  const [dbType, setDbType] = useState<'mysql' | 'mariadb' | 'postgresql'>('mysql');
  const [creating, setCreating] = useState(false);

  const fetchServers = async () => {
    const res = await apiFetch<Server[]>('/api/v1/servers');
    if (res.success && res.data) {
      setServers(res.data);
      if (res.data.length > 0) {
        setSelectedServer(res.data[0].id);
      }
    }
  };

  const fetchDatabases = async (serverId: string) => {
    if (!serverId) return;
    setLoading(true);
    const res = await apiFetch<Database[]>(`/api/v1/databases?server_id=${serverId}`);
    if (res.success && res.data) {
      setDatabases(res.data);
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchServers();
  }, []);

  useEffect(() => {
    if (selectedServer) {
      fetchDatabases(selectedServer);
    } else {
      setLoading(false);
    }
  }, [selectedServer]);

  const handleCreateDatabase = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreating(true);

    const res = await apiFetch<Database>('/api/v1/databases', {
      method: 'POST',
      body: JSON.stringify({
        server_id: selectedServer,
        db_type: dbType,
        name: dbName,
        character_set: 'utf8mb4',
        collation: 'utf8mb4_unicode_ci',
      }),
    });

    setCreating(false);
    if (res.success) {
      setCreateDbOpen(false);
      setDbName('');
      fetchDatabases(selectedServer);
    }
  };

  const handleDeleteDatabase = async (dbId: string, name: string) => {
    if (!confirm(`Are you sure you want to drop database '${name}'? This cannot be undone.`)) {
      return;
    }

    const res = await apiFetch(`/api/v1/databases/${dbId}`, {
      method: 'DELETE',
    });

    if (res.success) {
      fetchDatabases(selectedServer);
    }
  };

  const filteredDbs = databases.filter((d) =>
    d.name.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <DashboardShell>
      <div className="space-y-8">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-white">Database Management</h1>
            <p className="text-sm text-slate-400 mt-1">
              Create and configure MySQL, MariaDB, and PostgreSQL databases and users.
            </p>
          </div>
          <div className="flex items-center gap-3">
            {servers.length > 0 && (
              <select
                value={selectedServer}
                onChange={(e) => setSelectedServer(e.target.value)}
                className="px-3.5 py-2 rounded-xl bg-surface-900 border border-surface-800 text-slate-200 text-sm focus:outline-none"
              >
                {servers.map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.name} ({s.ip_address})
                  </option>
                ))}
              </select>
            )}
            <button
              onClick={() => fetchDatabases(selectedServer)}
              className="p-2.5 rounded-xl bg-surface-900 border border-surface-800 text-slate-300 hover:text-white hover:bg-surface-800 transition-colors"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            </button>
            <button
              onClick={() => setCreateDbOpen(true)}
              disabled={!selectedServer}
              className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 disabled:bg-indigo-600/50 text-white text-sm font-semibold shadow-lg shadow-indigo-600/25 transition-all"
            >
              <Plus className="w-4 h-4" />
              Create Database
            </button>
          </div>
        </div>

        {/* Search & Stats Bar */}
        <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3">
          <div className="relative flex-1 max-w-md">
            <div className="flex items-center gap-3 bg-white dark:bg-[#121824] border border-slate-300 dark:border-surface-700 rounded-xl px-4 py-2.5 shadow-xs focus-within:border-[#20a53a] focus-within:ring-2 focus-within:ring-[#20a53a]/20 transition-all">
              <Search className="w-4 h-4 text-slate-400 dark:text-slate-500 flex-shrink-0" />
              <input
                type="text"
                placeholder="Search databases..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full bg-transparent text-sm font-medium text-slate-950 dark:text-white placeholder:text-slate-400 focus:outline-none"
              />
              {search && (
                <button
                  type="button"
                  onClick={() => setSearch('')}
                  className="text-xs font-semibold px-1.5 py-0.5 rounded bg-slate-100 dark:bg-surface-800 text-slate-500 hover:text-black dark:hover:text-white"
                >
                  Clear
                </button>
              )}
            </div>
          </div>
          <div className="flex items-center gap-2 text-xs font-semibold text-slate-600 dark:text-slate-400">
            <span className="px-2.5 py-1 rounded-lg bg-slate-100 dark:bg-surface-800 border border-slate-200 dark:border-surface-700">
              Total: <strong className="text-slate-900 dark:text-white">{filteredDbs.length}</strong>
            </span>
          </div>
        </div>

        {/* Database List */}
        {loading ? (
          <div className="py-20 text-center text-slate-500">
            <div className="w-8 h-8 border-2 border-emerald-500 border-t-transparent rounded-full animate-spin mx-auto mb-3" />
            Loading databases...
          </div>
        ) : servers.length === 0 ? (
          <div className="py-16 text-center border border-dashed border-slate-300 dark:border-surface-800 rounded-2xl bg-white dark:bg-surface-900/50 p-6 shadow-xs">
            <ServerIcon className="w-12 h-12 mx-auto text-slate-400 dark:text-slate-600 mb-3" />
            <h3 className="text-base font-bold text-slate-900 dark:text-white">No Servers Available</h3>
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-1 max-w-sm mx-auto font-medium">
              You must connect at least one server before managing databases.
            </p>
          </div>
        ) : filteredDbs.length === 0 ? (
          <div className="py-16 text-center border border-dashed border-slate-300 dark:border-surface-800 rounded-2xl bg-white dark:bg-surface-900/50 p-6 shadow-xs">
            <DatabaseIcon className="w-12 h-12 mx-auto text-slate-400 dark:text-slate-600 mb-3" />
            <h3 className="text-base font-bold text-slate-900 dark:text-white">No databases found on this server</h3>
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-1 max-w-sm mx-auto font-medium">
              Create a MySQL or PostgreSQL database to connect to your websites or applications.
            </p>
            <button
              onClick={() => setCreateDbOpen(true)}
              className="mt-4 inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-[#20a53a] hover:bg-[#1b8c31] text-white text-xs font-bold shadow-md transition-all cursor-pointer"
            >
              <Plus className="w-3.5 h-3.5" />
              Create Database
            </button>
          </div>
        ) : (
          <div className="bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl overflow-hidden shadow-xs dark:shadow-xl">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm border-collapse">
                <thead>
                  <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-[#151b28]">
                    <th className="px-6 py-3.5 text-[11px] font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300">Database Name</th>
                    <th className="px-6 py-3.5 text-[11px] font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300">Engine</th>
                    <th className="px-6 py-3.5 text-[11px] font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300">Character Set & Collation</th>
                    <th className="px-6 py-3.5 text-[11px] font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300">Estimated Size</th>
                    <th className="px-6 py-3.5 text-[11px] font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200/80 dark:divide-surface-800/80">
                {filteredDbs.map((db) => (
                  <tr key={db.id} className="hover:bg-slate-50/90 dark:hover:bg-[#182030] transition-colors">
                    <td className="px-6 py-4 font-mono font-bold text-slate-900 dark:text-white flex items-center gap-2.5">
                      <DatabaseIcon className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
                      <span>{db.name}</span>
                    </td>
                    <td className="px-6 py-4">
                      <span className="inline-flex items-center uppercase text-[11px] font-bold px-2.5 py-0.5 rounded-md bg-slate-100 dark:bg-surface-800 text-slate-800 dark:text-slate-300 border border-slate-200 dark:border-surface-700">
                        {db.db_type}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-xs font-mono text-slate-600 dark:text-slate-400">
                      {db.character_set || 'utf8mb4'} / {db.collation || 'utf8mb4_unicode_ci'}
                    </td>
                    <td className="px-6 py-4 text-xs font-medium text-slate-600 dark:text-slate-400">
                      {db.size_bytes ? `${(db.size_bytes / 1024 / 1024).toFixed(2)} MB` : '0 MB'}
                    </td>
                    <td className="px-6 py-4 text-right">
                      <button
                        onClick={() => handleDeleteDatabase(db.id, db.name)}
                        title="Drop Database"
                        className="p-1.5 rounded-lg border border-slate-300 dark:border-surface-700 text-slate-500 hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/20 transition-colors"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
        )}

        {/* Create Database Modal */}
        {createDbOpen && (
          <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="w-full max-w-md bg-surface-900 border border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setCreateDbOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-white hover:bg-surface-800 transition-colors"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-5">
                <div className="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-400">
                  <DatabaseIcon className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-white">Create Database</h2>
                  <p className="text-xs text-slate-400">Provision database schema on selected server</p>
                </div>
              </div>

              <form onSubmit={handleCreateDatabase} className="space-y-4">
                <div>
                  <label className="block text-xs font-semibold uppercase tracking-wider text-slate-300 mb-1.5">
                    Database Engine
                  </label>
                  <select
                    value={dbType}
                    onChange={(e) => setDbType(e.target.value as any)}
                    className="w-full px-3.5 py-2.5 rounded-xl bg-surface-950 border border-surface-700 text-slate-100 text-sm focus:outline-none focus:border-indigo-500"
                  >
                    <option value="mysql">MySQL 8.0</option>
                    <option value="mariadb">MariaDB 10.11</option>
                    <option value="postgresql">PostgreSQL 16</option>
                  </select>
                </div>

                <div>
                  <label className="block text-xs font-semibold uppercase tracking-wider text-slate-300 mb-1.5">
                    Database Name
                  </label>
                  <input
                    type="text"
                    required
                    value={dbName}
                    onChange={(e) => setDbName(e.target.value.toLowerCase().replace(/[^a-z0-9_]/g, ''))}
                    placeholder="app_production"
                    className="w-full px-3.5 py-2.5 rounded-xl bg-surface-950 border border-surface-700 text-slate-100 font-mono text-sm placeholder-slate-500 focus:outline-none focus:border-indigo-500"
                  />
                </div>

                <div className="flex justify-end gap-3 pt-3">
                  <button
                    type="button"
                    onClick={() => setCreateDbOpen(false)}
                    className="px-4 py-2 rounded-xl text-sm text-slate-400 hover:text-white transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={creating || !dbName}
                    className="px-5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 disabled:bg-indigo-600/50 text-white text-sm font-semibold shadow-md transition-all flex items-center gap-2"
                  >
                    {creating ? (
                      <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                    ) : (
                      'Create Database'
                    )}
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
