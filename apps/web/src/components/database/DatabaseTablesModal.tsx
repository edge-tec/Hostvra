'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Table,
  X,
  RefreshCw,
  Search,
  ExternalLink,
  Wrench,
  Layers,
  Database,
  CheckCircle2,
  AlertCircle
} from 'lucide-react';
import { apiFetch } from '@/lib/api';

interface TableInfo {
  name: string;
  engine?: string;
  rows?: number;
  data_length?: number;
  index_length?: number;
  collation?: string;
  comment?: string;
}

interface DatabaseTablesModalProps {
  isOpen: boolean;
  onClose: () => void;
  databaseName: string;
  onOpenQueryConsole?: (query: string) => void;
}

export const DatabaseTablesModal: React.FC<DatabaseTablesModalProps> = ({
  isOpen,
  onClose,
  databaseName,
  onOpenQueryConsole,
}) => {
  const [tables, setTables] = useState<TableInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [search, setSearch] = useState('');

  const fetchTables = useCallback(async () => {
    if (!databaseName) return;
    try {
      setLoading(true);
      setErrorMsg(null);
      // Query information_schema / SHOW TABLE STATUS
      const res = await apiFetch<any>('/api/v1/databases/query', {
        method: 'POST',
        body: JSON.stringify({
          database: databaseName,
          query: 'SHOW TABLE STATUS;',
        }),
      });

      if (res.success && res.data) {
        const rows = res.data.rows || res.data.Rows || [];
        const parsed: TableInfo[] = rows.map((r: any) => ({
          name: r.Name || r.name || 'unknown',
          engine: r.Engine || r.engine || 'InnoDB',
          rows: Number(r.Rows ?? r.rows ?? 0),
          data_length: Number(r.Data_length ?? r.data_length ?? 0),
          index_length: Number(r.Index_length ?? r.index_length ?? 0),
          collation: r.Collation || r.collation || 'utf8mb4_unicode_ci',
          comment: r.Comment || r.comment || '',
        }));
        setTables(parsed);
      } else {
        // Fallback to /api/v1/databases/tables
        const fallback = await apiFetch<any[]>(`/api/v1/databases/tables?database=${encodeURIComponent(databaseName)}`);
        if (fallback.success && fallback.data) {
          setTables(fallback.data.map((t: any) => ({
            name: typeof t === 'string' ? t : t.name || 'table',
            engine: t.engine || 'InnoDB',
            rows: t.rows || 0,
          })));
        } else {
          setErrorMsg(res.error?.message || 'Could not load tables');
        }
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Error loading database tables');
    } finally {
      setLoading(false);
    }
  }, [databaseName]);

  useEffect(() => {
    if (isOpen) {
      fetchTables();
    }
  }, [isOpen, fetchTables]);

  if (!isOpen) return null;

  const formatSize = (bytes: number) => {
    if (!bytes || bytes === 0) return '0 KB';
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
  };

  const filtered = tables.filter((t) =>
    t.name.toLowerCase().includes(search.toLowerCase())
  );

  const totalRows = tables.reduce((acc, t) => acc + (t.rows || 0), 0);
  const totalDataSize = tables.reduce((acc, t) => acc + (t.data_length || 0), 0);

  return (
    <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-xs flex items-center justify-center p-3 sm:p-5 animate-fadeIn">
      <div className="w-full max-w-4xl h-[82vh] bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl flex flex-col overflow-hidden font-sans">
        {/* Header */}
        <div className="px-5 py-4 border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-surface-850 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
              <Table className="w-5 h-5" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h2 className="text-sm font-bold text-slate-900 dark:text-white">Database Tables & Schema</h2>
                <span className="px-2 py-0.5 rounded-full text-[10px] font-mono bg-emerald-100 dark:bg-emerald-950/40 text-emerald-700 dark:text-emerald-300">
                  {databaseName}
                </span>
              </div>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                {tables.length} tables • {totalRows.toLocaleString()} total rows • {formatSize(totalDataSize)} total size
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={fetchTables}
              className="p-1.5 rounded-lg border border-slate-200 dark:border-surface-700 text-slate-600 hover:bg-slate-100 dark:hover:bg-surface-800 transition"
              title="Refresh tables"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin text-emerald-600' : ''}`} />
            </button>
            <button
              onClick={onClose}
              className="p-1.5 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-surface-800 transition"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Toolbar */}
        <div className="px-5 py-2.5 border-b border-slate-200 dark:border-surface-800 bg-white dark:bg-surface-900 flex items-center justify-between gap-3 text-xs">
          <div className="relative w-64">
            <Search className="w-3.5 h-3.5 text-slate-400 absolute left-3 top-2.5 pointer-events-none" />
            <input
              type="text"
              placeholder="Search tables..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full pl-8 pr-3 py-1.5 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-slate-900 dark:text-white text-xs focus:outline-none focus:border-emerald-500"
            />
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => {
                if (onOpenQueryConsole) {
                  onClose();
                  onOpenQueryConsole(`SHOW TABLES;`);
                }
              }}
              className="px-3 py-1.5 rounded-lg bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800 font-bold hover:bg-emerald-100 transition flex items-center gap-1.5 text-xs"
            >
              <ExternalLink className="w-3.5 h-3.5" />
              <span>Open in SQL Console</span>
            </button>
          </div>
        </div>

        {/* Tables Content */}
        <div className="flex-1 overflow-auto p-4 bg-slate-50/50 dark:bg-surface-950">
          {loading ? (
            <div className="py-20 text-center text-slate-400 text-xs">
              <RefreshCw className="w-6 h-6 animate-spin text-emerald-600 mx-auto mb-2" />
              <span>Inspecting database tables...</span>
            </div>
          ) : errorMsg ? (
            <div className="p-4 rounded-xl bg-rose-50 dark:bg-rose-950/30 border border-rose-200 dark:border-rose-900 text-rose-700 dark:text-rose-300 text-xs flex items-center gap-2">
              <AlertCircle className="w-4 h-4 text-rose-500" />
              <span>{errorMsg}</span>
            </div>
          ) : filtered.length === 0 ? (
            <div className="py-20 text-center text-slate-400 text-xs">
              <Table className="w-8 h-8 text-slate-300 dark:text-slate-700 mx-auto mb-2" />
              <p className="font-semibold text-slate-700 dark:text-slate-300">No tables found</p>
              <p className="text-[11px] text-slate-400">Database is empty or no tables matched your search.</p>
            </div>
          ) : (
            <div className="rounded-xl border border-slate-200 dark:border-surface-800 bg-white dark:bg-surface-900 overflow-hidden shadow-2xs">
              <table className="w-full text-left text-xs border-collapse">
                <thead className="bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-300 font-bold uppercase text-[10px] tracking-wider border-b border-slate-200 dark:border-surface-700">
                  <tr>
                    <th className="py-2.5 px-3">Table Name</th>
                    <th className="py-2.5 px-3">Engine</th>
                    <th className="py-2.5 px-3 text-right">Rows</th>
                    <th className="py-2.5 px-3 text-right">Data Size</th>
                    <th className="py-2.5 px-3 text-right">Index Size</th>
                    <th className="py-2.5 px-3">Collation</th>
                    <th className="py-2.5 px-3 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100 dark:divide-surface-800/60 font-mono text-[11px]">
                  {filtered.map((t) => (
                    <tr key={t.name} className="hover:bg-slate-50 dark:hover:bg-surface-800/50 transition">
                      <td className="py-2 px-3 font-bold text-slate-900 dark:text-white">
                        <div className="flex items-center gap-1.5">
                          <Table className="w-3.5 h-3.5 text-emerald-600" />
                          <span>{t.name}</span>
                        </div>
                      </td>
                      <td className="py-2 px-3 text-slate-600 dark:text-slate-400">{t.engine}</td>
                      <td className="py-2 px-3 text-right font-bold text-emerald-600 dark:text-emerald-400">
                        {t.rows?.toLocaleString() ?? 0}
                      </td>
                      <td className="py-2 px-3 text-right text-slate-700 dark:text-slate-300">
                        {formatSize(t.data_length || 0)}
                      </td>
                      <td className="py-2 px-3 text-right text-slate-500">
                        {formatSize(t.index_length || 0)}
                      </td>
                      <td className="py-2 px-3 text-slate-500 text-[10px] truncate max-w-[130px]">
                        {t.collation}
                      </td>
                      <td className="py-2 px-3 text-right font-sans">
                        <button
                          onClick={() => {
                            if (onOpenQueryConsole) {
                              onClose();
                              onOpenQueryConsole(`SELECT * FROM \`${t.name}\` LIMIT 25;`);
                            }
                          }}
                          className="px-2.5 py-1 rounded-lg bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600 dark:text-emerald-400 hover:bg-emerald-100 text-[11px] font-bold transition cursor-pointer"
                        >
                          Browse Rows
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
