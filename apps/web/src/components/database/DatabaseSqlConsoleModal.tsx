'use client';

import React, { useState, useEffect } from 'react';
import {
  Terminal,
  Play,
  X,
  RefreshCw,
  Copy,
  Check,
  AlertCircle,
  Clock,
  Database,
  FileCode,
  Download,
  Code2,
  ChevronDown
} from 'lucide-react';
import { apiFetch } from '@/lib/api';

interface QueryResult {
  columns?: string[];
  rows?: Array<Record<string, string | number | null>>;
  execution_time?: string;
  affected_rows?: number;
  error?: string;
}

interface DatabaseSqlConsoleModalProps {
  isOpen: boolean;
  onClose: () => void;
  databaseName: string;
  databasesList: string[];
  onDatabaseChange?: (dbName: string) => void;
}

const SQL_TEMPLATES = [
  { label: 'Show Tables', query: 'SHOW TABLES;' },
  { label: 'Table Status', query: 'SHOW TABLE STATUS;' },
  { label: 'Active Processlist', query: 'SHOW PROCESSLIST;' },
  { label: 'Server Variables', query: 'SHOW VARIABLES LIKE "%version%";' },
  { label: 'Database Size', query: 'SELECT table_schema AS "Database", ROUND(SUM(data_length + index_length) / 1024 / 1024, 2) AS "Size (MB)" FROM information_schema.TABLES WHERE table_schema = DATABASE() GROUP BY table_schema;' },
];

export const DatabaseSqlConsoleModal: React.FC<DatabaseSqlConsoleModalProps> = ({
  isOpen,
  onClose,
  databaseName,
  databasesList,
  onDatabaseChange,
}) => {
  const [activeDb, setActiveDb] = useState(databaseName);
  const [query, setQuery] = useState('SHOW TABLES;');
  const [executing, setExecuting] = useState(false);
  const [result, setResult] = useState<QueryResult | null>(null);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (databaseName) {
      setActiveDb(databaseName);
    }
  }, [databaseName]);

  if (!isOpen) return null;

  const handleRunQuery = async () => {
    if (!query.trim() || !activeDb) return;
    try {
      setExecuting(true);
      setErrorMsg(null);
      setResult(null);

      const res = await apiFetch<any>('/api/v1/databases/query', {
        method: 'POST',
        body: JSON.stringify({
          database: activeDb,
          query: query.trim(),
        }),
      });

      if (res.success && res.data) {
        if (res.data.error) {
          setErrorMsg(res.data.error);
        } else {
          setResult({
            columns: res.data.columns || res.data.Columns || [],
            rows: res.data.rows || res.data.Rows || [],
            execution_time: res.data.execution_time || res.data.ExecutionTime || '0.001s',
            affected_rows: res.data.affected_rows ?? res.data.AffectedRows,
          });
        }
      } else {
        setErrorMsg(res.error?.message || 'Query execution failed');
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Network error running query');
    } finally {
      setExecuting(false);
    }
  };

  const handleExportJson = () => {
    if (!result?.rows || result.rows.length === 0) return;
    const blob = new Blob([JSON.stringify(result.rows, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${activeDb}_query_result_${Date.now()}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const handleExportCsv = () => {
    if (!result?.columns || !result?.rows || result.rows.length === 0) return;
    const headers = result.columns.join(',');
    const rows = result.rows.map((row) =>
      result.columns!.map((col) => {
        const val = row[col] === null || row[col] === undefined ? '' : String(row[col]);
        return `"${val.replace(/"/g, '""')}"`;
      }).join(',')
    );
    const csvContent = [headers, ...rows].join('\n');
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${activeDb}_query_result_${Date.now()}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-xs flex items-center justify-center p-3 sm:p-5 animate-fadeIn">
      <div className="w-full max-w-5xl h-[88vh] bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl flex flex-col overflow-hidden font-sans">
        {/* Header */}
        <div className="px-5 py-3.5 border-b border-slate-200 dark:border-surface-800 bg-slate-50/80 dark:bg-surface-850 flex items-center justify-between gap-3">
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
              <Terminal className="w-5 h-5" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h2 className="text-sm font-bold text-slate-900 dark:text-white">SQL Web Console</h2>
                <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800">
                  Interactive
                </span>
              </div>
              <p className="text-[11px] text-slate-500 dark:text-slate-400">
                Execute queries with live results, formatting, and export
              </p>
            </div>
          </div>

          <div className="flex items-center gap-3">
            {/* Target DB Selector */}
            <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-white dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-xs font-semibold">
              <Database className="w-3.5 h-3.5 text-emerald-600" />
              <span className="text-[11px] text-slate-400 hidden sm:inline">Database:</span>
              <select
                value={activeDb}
                onChange={(e) => {
                  setActiveDb(e.target.value);
                  if (onDatabaseChange) onDatabaseChange(e.target.value);
                }}
                className="bg-transparent text-slate-900 dark:text-white font-mono font-bold focus:outline-none cursor-pointer text-xs"
              >
                {databasesList.map((db) => (
                  <option key={db} value={db} className="bg-white dark:bg-surface-900 text-slate-900 dark:text-white">
                    {db}
                  </option>
                ))}
              </select>
            </div>

            <button
              onClick={onClose}
              className="p-1.5 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-surface-800 transition"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Toolbar & Templates */}
        <div className="px-5 py-2 border-b border-slate-200 dark:border-surface-800 bg-white dark:bg-surface-900 flex flex-wrap items-center justify-between gap-2 text-xs">
          <div className="flex items-center gap-1.5 flex-wrap">
            <span className="text-[11px] font-semibold text-slate-400 mr-1 flex items-center gap-1">
              <Code2 className="w-3 h-3 text-slate-400" /> Quick SQL:
            </span>
            {SQL_TEMPLATES.map((tmpl) => (
              <button
                key={tmpl.label}
                onClick={() => setQuery(tmpl.query)}
                className="px-2.5 py-1 rounded-lg bg-slate-100 dark:bg-surface-800 hover:bg-emerald-50 dark:hover:bg-emerald-950/30 text-slate-700 dark:text-slate-300 hover:text-emerald-600 dark:hover:text-emerald-400 font-medium text-[11px] transition border border-transparent hover:border-emerald-200 dark:hover:border-emerald-800"
              >
                {tmpl.label}
              </button>
            ))}
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => {
                navigator.clipboard.writeText(query);
                setCopied(true);
                setTimeout(() => setCopied(false), 2000);
              }}
              className="px-2.5 py-1 rounded-lg border border-slate-200 dark:border-surface-700 text-slate-600 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-surface-800 text-[11px] font-semibold flex items-center gap-1 transition"
            >
              {copied ? <Check className="w-3 h-3 text-emerald-500" /> : <Copy className="w-3 h-3" />}
              <span>{copied ? 'Copied' : 'Copy SQL'}</span>
            </button>
          </div>
        </div>

        {/* Query Input Editor */}
        <div className="p-4 bg-slate-950 border-b border-slate-800 flex flex-col gap-2">
          <div className="flex items-center justify-between text-[11px] text-slate-400 font-mono">
            <span>Query Console (Ctrl + Enter to Execute)</span>
            <span className="text-emerald-400">Target: {activeDb}</span>
          </div>
          <textarea
            rows={4}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={(e) => {
              if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
                e.preventDefault();
                handleRunQuery();
              }
            }}
            placeholder="SELECT * FROM table_name LIMIT 20;"
            className="w-full bg-slate-900/90 text-emerald-400 font-mono text-xs p-3 rounded-xl border border-slate-800 focus:outline-none focus:border-emerald-500 transition resize-none leading-relaxed"
            spellCheck={false}
          />
          <div className="flex items-center justify-between pt-1">
            <span className="text-[10px] text-slate-500 font-mono">
              Supports SELECT, SHOW, DESCRIBE, EXPLAIN, INSERT, UPDATE, DELETE
            </span>
            <button
              onClick={handleRunQuery}
              disabled={executing || !query.trim()}
              className="px-4 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white font-bold text-xs transition shadow-md flex items-center gap-1.5 cursor-pointer disabled:opacity-50"
            >
              <Play className={`w-3.5 h-3.5 fill-white ${executing ? 'animate-spin' : ''}`} />
              <span>{executing ? 'Executing...' : 'Run Query'}</span>
            </button>
          </div>
        </div>

        {/* Results Area */}
        <div className="flex-1 bg-slate-50/50 dark:bg-surface-950 overflow-auto p-4 flex flex-col">
          {executing ? (
            <div className="m-auto text-center text-slate-400 text-xs py-10">
              <RefreshCw className="w-6 h-6 animate-spin text-emerald-600 mx-auto mb-2" />
              <p className="font-semibold text-slate-700 dark:text-slate-300">Executing SQL query...</p>
              <p className="text-[11px] text-slate-400 font-mono">Querying {activeDb}</p>
            </div>
          ) : errorMsg ? (
            <div className="p-4 rounded-xl bg-rose-50 dark:bg-rose-950/30 border border-rose-200 dark:border-rose-900 text-rose-700 dark:text-rose-300 text-xs flex items-start gap-3">
              <AlertCircle className="w-5 h-5 text-rose-500 shrink-0 mt-0.5" />
              <div className="space-y-1">
                <p className="font-bold">SQL Execution Error</p>
                <pre className="font-mono text-[11px] whitespace-pre-wrap">{errorMsg}</pre>
              </div>
            </div>
          ) : result ? (
            <div className="flex-1 flex flex-col min-h-0 space-y-3">
              {/* Meta Stats bar */}
              <div className="flex items-center justify-between text-xs text-slate-600 dark:text-slate-400 pb-1">
                <div className="flex items-center gap-3">
                  <span className="flex items-center gap-1 font-mono text-[11px]">
                    <Clock className="w-3.5 h-3.5 text-slate-400" />
                    <span>Duration: {result.execution_time}</span>
                  </span>
                  <span className="text-slate-300 dark:text-slate-700">|</span>
                  <span className="font-bold text-emerald-600 dark:text-emerald-400 font-mono text-[11px]">
                    {result.rows ? `${result.rows.length} rows returned` : `${result.affected_rows ?? 0} rows affected`}
                  </span>
                </div>

                {result.rows && result.rows.length > 0 && (
                  <div className="flex items-center gap-2">
                    <button
                      onClick={handleExportCsv}
                      className="px-2.5 py-1 rounded-lg bg-white dark:bg-surface-800 border border-slate-200 dark:border-surface-700 hover:bg-slate-100 dark:hover:bg-surface-700 text-[11px] font-semibold text-slate-700 dark:text-slate-200 flex items-center gap-1 transition"
                    >
                      <Download className="w-3 h-3 text-slate-400" /> CSV
                    </button>
                    <button
                      onClick={handleExportJson}
                      className="px-2.5 py-1 rounded-lg bg-white dark:bg-surface-800 border border-slate-200 dark:border-surface-700 hover:bg-slate-100 dark:hover:bg-surface-700 text-[11px] font-semibold text-slate-700 dark:text-slate-200 flex items-center gap-1 transition"
                    >
                      <Download className="w-3 h-3 text-slate-400" /> JSON
                    </button>
                  </div>
                )}
              </div>

              {/* Data Table */}
              {result.rows && result.rows.length > 0 && result.columns && result.columns.length > 0 ? (
                <div className="flex-1 overflow-auto rounded-xl border border-slate-200 dark:border-surface-800 bg-white dark:bg-surface-900 shadow-2xs">
                  <table className="w-full text-left text-xs border-collapse">
                    <thead className="sticky top-0 bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-300 font-bold uppercase text-[10px] tracking-wider border-b border-slate-200 dark:border-surface-700 select-none">
                      <tr>
                        <th className="py-2 px-3 w-10 text-center font-mono text-slate-400">#</th>
                        {result.columns.map((col) => (
                          <th key={col} className="py-2 px-3 font-mono">
                            {col}
                          </th>
                        ))}
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100 dark:divide-surface-800/60 font-mono text-[11px]">
                      {result.rows.map((row, idx) => (
                        <tr key={idx} className="hover:bg-slate-50 dark:hover:bg-surface-800/50 transition">
                          <td className="py-1.5 px-3 text-center text-slate-400">{idx + 1}</td>
                          {result.columns!.map((col) => {
                            const val = row[col];
                            return (
                              <td key={col} className="py-1.5 px-3 text-slate-800 dark:text-slate-200 max-w-xs truncate">
                                {val === null ? (
                                  <span className="text-slate-400 italic">NULL</span>
                                ) : (
                                  String(val)
                                )}
                              </td>
                            );
                          })}
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <div className="m-auto text-center text-slate-400 text-xs py-8">
                  <Check className="w-6 h-6 text-emerald-500 mx-auto mb-1.5" />
                  <p className="font-bold text-slate-700 dark:text-slate-300">Query executed successfully</p>
                  <p className="text-[11px] text-slate-500 font-mono">
                    {result.affected_rows !== undefined ? `${result.affected_rows} rows affected` : 'No rows returned'}
                  </p>
                </div>
              )}
            </div>
          ) : (
            <div className="m-auto text-center text-slate-400 text-xs py-10">
              <Code2 className="w-8 h-8 text-slate-300 dark:text-slate-700 mx-auto mb-2" />
              <p className="font-semibold text-slate-700 dark:text-slate-300">Ready to execute SQL</p>
              <p className="text-[11px] text-slate-400">
                Write a query above or click a quick template to start.
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
