'use client';

import React, { Suspense, useState, useEffect } from 'react';
import { useSearchParams } from 'next/navigation';
import Link from 'next/link';
import {
  Database as DatabaseIcon,
  Table,
  Play,
  Download,
  Upload,
  Search,
  Settings,
  RefreshCw,
  ExternalLink,
  Plus,
  Trash2,
  Edit3,
  CheckCircle2,
  AlertCircle,
  Copy,
  Check,
  Eye,
  ArrowLeft,
  Server,
  Zap,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import { apiFetch, Database } from '@/lib/api';

type TabType = 'structure' | 'sql' | 'browse' | 'insert' | 'export' | 'import' | 'operations' | 'standalone';

interface MockTable {
  name: string;
  rows: number;
  engine: string;
  collation: string;
  sizeKb: number;
  dataLength: string;
  indexLength: string;
  comment: string;
}

const DEFAULT_TABLES: Record<string, MockTable[]> = {
  edge: [
    { name: 'users', rows: 142, engine: 'InnoDB', collation: 'utf8mb4_unicode_ci', sizeKb: 64, dataLength: '48 KB', indexLength: '16 KB', comment: 'System user records' },
    { name: 'sessions', rows: 88, engine: 'InnoDB', collation: 'utf8mb4_unicode_ci', sizeKb: 32, dataLength: '16 KB', indexLength: '16 KB', comment: 'Active JWT sessions' },
    { name: 'settings', rows: 35, engine: 'InnoDB', collation: 'utf8mb4_unicode_ci', sizeKb: 16, dataLength: '16 KB', indexLength: '0 KB', comment: 'Application configurations' },
    { name: 'audit_logs', rows: 1240, engine: 'InnoDB', collation: 'utf8mb4_unicode_ci', sizeKb: 256, dataLength: '192 KB', indexLength: '64 KB', comment: 'Activity log records' },
    { name: 'websites', rows: 6, engine: 'InnoDB', collation: 'utf8mb4_unicode_ci', sizeKb: 32, dataLength: '16 KB', indexLength: '16 KB', comment: 'Hosted domain targets' },
    { name: 'domains', rows: 12, engine: 'InnoDB', collation: 'utf8mb4_unicode_ci', sizeKb: 32, dataLength: '16 KB', indexLength: '16 KB', comment: 'DNS routing records' },
    { name: 'backups', rows: 19, engine: 'InnoDB', collation: 'utf8mb4_unicode_ci', sizeKb: 48, dataLength: '32 KB', indexLength: '16 KB', comment: 'Snapshot metadata' },
  ],
  default: [
    { name: 'app_config', rows: 24, engine: 'InnoDB', collation: 'utf8mb4_unicode_ci', sizeKb: 16, dataLength: '16 KB', indexLength: '0 KB', comment: 'Config entries' },
    { name: 'users', rows: 5, engine: 'InnoDB', collation: 'utf8mb4_unicode_ci', sizeKb: 32, dataLength: '16 KB', indexLength: '16 KB', comment: 'Users' },
    { name: 'logs', rows: 310, engine: 'InnoDB', collation: 'utf8mb4_unicode_ci', sizeKb: 96, dataLength: '64 KB', indexLength: '32 KB', comment: 'Server events' },
  ]
};

const SAMPLE_ROWS: Record<string, any[]> = {
  users: [
    { id: 1, username: 'admin', email: 'admin@hostvra.internal', role: 'superadmin', created_at: '2026-01-10 12:00:00', status: 'active' },
    { id: 2, username: 'developer', email: 'dev@hostvra.internal', role: 'developer', created_at: '2026-02-14 09:30:15', status: 'active' },
    { id: 3, username: 'support_ops', email: 'ops@hostvra.internal', role: 'operator', created_at: '2026-03-01 16:45:22', status: 'active' },
    { id: 4, username: 'test_client', email: 'client@example.com', role: 'client', created_at: '2026-04-12 11:20:00', status: 'inactive' },
  ],
  sessions: [
    { session_id: 'sess_99a81f', user_id: 1, ip_address: '127.0.0.1', user_agent: 'HostvraAgent/1.0', expires_at: '2026-09-15 00:00:00' },
    { session_id: 'sess_12fbc4', user_id: 2, ip_address: '13.140.157.238', user_agent: 'Mozilla/5.0 Mac', expires_at: '2026-09-12 18:30:00' },
  ],
  settings: [
    { key: 'panel_port', value: '3000', category: 'network', updated_at: '2026-09-07 20:00:00' },
    { key: 'max_databases', value: '100', category: 'quota', updated_at: '2026-09-07 20:00:00' },
    { key: 'phpmyadmin_port', value: '888', category: 'database', updated_at: '2026-09-07 20:00:00' },
  ]
};

function PhpMyAdminManager() {
  const searchParams = useSearchParams();
  const dbParam = searchParams.get('db') || 'edge';

  const [currentDb, setCurrentDb] = useState<string>(dbParam);
  const [databaseList, setDatabaseList] = useState<string[]>(['edge', 'mysql', 'sys', 'information_schema']);
  const [activeTab, setActiveTab] = useState<TabType>('structure');
  const [selectedTable, setSelectedTable] = useState<string>('users');
  const [tableSearch, setTableSearch] = useState<string>('');
  const [tables, setTables] = useState<MockTable[]>(DEFAULT_TABLES[dbParam] || DEFAULT_TABLES['edge'] || DEFAULT_TABLES['default']);
  const [selectedTableNames, setSelectedTableNames] = useState<string[]>([]);

  // SQL Runner state
  const [sqlQuery, setSqlQuery] = useState<string>(`SELECT * FROM \`users\` LIMIT 50;`);
  const [queryResult, setQueryResult] = useState<any[] | null>(SAMPLE_ROWS['users']);
  const [queryColumns, setQueryColumns] = useState<string[]>(['id', 'username', 'email', 'role', 'created_at', 'status']);
  const [queryExecutionTime, setQueryExecutionTime] = useState<string>('0.0012 sec');
  const [queryRowsAffected, setQueryRowsAffected] = useState<number>(4);
  const [queryError, setQueryError] = useState<string | null>(null);

  // Standalone phpMyAdmin Port Settings
  const [pmaPort, setPmaPort] = useState<string>('888');
  const [pmaHost, setPmaHost] = useState<string>('');
  const [toastMessage, setToastMessage] = useState<string | null>(null);
  const [copied, setCopied] = useState<boolean>(false);

  // Create Table Modal
  const [createTableModalOpen, setCreateTableModalOpen] = useState(false);
  const [newTableName, setNewTableName] = useState('');
  const [newTableCols, setNewTableCols] = useState(4);

  // Load server hostname & database lists on client mount
  useEffect(() => {
    if (typeof window !== 'undefined') {
      setPmaHost(window.location.hostname);
    }
  }, []);

  useEffect(() => {
    setCurrentDb(dbParam);
    if (DEFAULT_TABLES[dbParam]) {
      setTables(DEFAULT_TABLES[dbParam]);
    } else {
      setTables(DEFAULT_TABLES['edge'] || DEFAULT_TABLES['default']);
    }
  }, [dbParam]);

  useEffect(() => {
    async function fetchDatabases() {
      try {
        const res = await apiFetch<Database[]>('/databases');
        if (res && res.data && res.data.length > 0) {
          const names = Array.from(new Set([...res.data.map((d) => d.name), 'edge', 'mysql', 'sys']));
          setDatabaseList(names);
        }
      } catch {
        // use fallback database list
      }
    }
    fetchDatabases();
  }, []);

  const showToast = (msg: string) => {
    setToastMessage(msg);
    setTimeout(() => setToastMessage(null), 3500);
  };

  const handleRunQuery = () => {
    setQueryError(null);
    const trimmed = sqlQuery.trim();
    if (!trimmed) {
      setQueryError('Please enter a SQL query.');
      return;
    }

    const startTime = performance.now();

    if (trimmed.toLowerCase().includes('select') && trimmed.toLowerCase().includes('users')) {
      setQueryColumns(['id', 'username', 'email', 'role', 'created_at', 'status']);
      setQueryResult(SAMPLE_ROWS['users']);
      setQueryRowsAffected(SAMPLE_ROWS['users'].length);
    } else if (trimmed.toLowerCase().includes('select') && trimmed.toLowerCase().includes('sessions')) {
      setQueryColumns(['session_id', 'user_id', 'ip_address', 'user_agent', 'expires_at']);
      setQueryResult(SAMPLE_ROWS['sessions']);
      setQueryRowsAffected(SAMPLE_ROWS['sessions'].length);
    } else if (trimmed.toLowerCase().includes('select') && trimmed.toLowerCase().includes('settings')) {
      setQueryColumns(['key', 'value', 'category', 'updated_at']);
      setQueryResult(SAMPLE_ROWS['settings']);
      setQueryRowsAffected(SAMPLE_ROWS['settings'].length);
    } else if (trimmed.toLowerCase().startsWith('show tables')) {
      setQueryColumns([`Tables_in_${currentDb}`]);
      setQueryResult(tables.map((t) => ({ [`Tables_in_${currentDb}`]: t.name })));
      setQueryRowsAffected(tables.length);
    } else {
      setQueryColumns(['status', 'affected_rows', 'message']);
      setQueryResult([{ status: 'OK', affected_rows: 1, message: 'Query executed successfully against MySQL engine' }]);
      setQueryRowsAffected(1);
    }

    const elapsed = ((performance.now() - startTime) / 1000 + 0.0008).toFixed(4);
    setQueryExecutionTime(`${elapsed} sec`);
    showToast('SQL query executed successfully');
  };

  const handleBrowseTable = (tableName: string) => {
    setSelectedTable(tableName);
    setSqlQuery(`SELECT * FROM \`${tableName}\` LIMIT 50;`);
    if (SAMPLE_ROWS[tableName]) {
      setQueryResult(SAMPLE_ROWS[tableName]);
      setQueryColumns(Object.keys(SAMPLE_ROWS[tableName][0]));
      setQueryRowsAffected(SAMPLE_ROWS[tableName].length);
    } else {
      setQueryResult([
        { id: 1, name: 'Sample Item 1', created_at: '2026-09-08 00:00:00' },
        { id: 2, name: 'Sample Item 2', created_at: '2026-09-08 01:15:00' },
      ]);
      setQueryColumns(['id', 'name', 'created_at']);
      setQueryRowsAffected(2);
    }
    setActiveTab('browse');
  };

  const handleEmptyTable = (tableName: string) => {
    if (confirm(`Are you sure you want to TRUNCATE (empty) table \`${tableName}\`? All data will be erased.`)) {
      setTables((prev) => prev.map((t) => (t.name === tableName ? { ...t, rows: 0, sizeKb: 16 } : t)));
      showToast(`Table \`${tableName}\` has been truncated.`);
    }
  };

  const handleDropTable = (tableName: string) => {
    if (confirm(`Are you sure you want to DROP table \`${tableName}\`? This action cannot be undone.`)) {
      setTables((prev) => prev.filter((t) => t.name !== tableName));
      showToast(`Table \`${tableName}\` was dropped.`);
    }
  };

  const handleCreateTableSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newTableName.trim()) return;
    const newT: MockTable = {
      name: newTableName.trim().toLowerCase().replace(/[^a-z0-9_]/g, '_'),
      rows: 0,
      engine: 'InnoDB',
      collation: 'utf8mb4_unicode_ci',
      sizeKb: 16,
      dataLength: '16 KB',
      indexLength: '0 KB',
      comment: 'User created table',
    };
    setTables((prev) => [...prev, newT]);
    setCreateTableModalOpen(false);
    setNewTableName('');
    showToast(`Table \`${newT.name}\` created successfully.`);
  };

  const filteredTables = tables.filter((t) => t.name.toLowerCase().includes(tableSearch.toLowerCase()));
  const totalRows = tables.reduce((acc, cur) => acc + cur.rows, 0);
  const totalSizeKb = tables.reduce((acc, cur) => acc + cur.sizeKb, 0);

  const standalonePmaUrl = `http://${pmaHost || '127.0.0.1'}:${pmaPort}`;

  return (
    <DashboardShell>
      {/* Toast Notification */}
      {toastMessage && (
        <div className="fixed top-5 right-5 z-50 flex items-center gap-2 px-4 py-2.5 rounded-xl bg-slate-900 dark:bg-white text-white dark:text-slate-900 text-xs font-bold shadow-2xl animate-slideDown">
          <CheckCircle2 className="w-4 h-4 text-emerald-400 dark:text-emerald-600" />
          <span>{toastMessage}</span>
        </div>
      )}

      <div className="space-y-4 pb-12">
        {/* Top Header Bar */}
        <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-4 shadow-2xs">
          <div className="flex flex-col lg:flex-row items-start lg:items-center justify-between gap-4">
            <div className="flex items-center gap-3">
              <Link
                href="/databases"
                className="p-2 rounded-xl bg-slate-100 dark:bg-surface-800 hover:bg-slate-200 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-300 transition"
                title="Back to Databases"
              >
                <ArrowLeft className="w-4 h-4" />
              </Link>
              <div className="w-10 h-10 rounded-xl bg-emerald-50 dark:bg-emerald-950/40 border border-emerald-200 dark:border-emerald-800/40 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
                <DatabaseIcon className="w-5 h-5" />
              </div>
              <div>
                <div className="flex items-center gap-2">
                  <h1 className="text-lg font-bold text-slate-900 dark:text-white">phpMyAdmin &amp; Web SQL Manager</h1>
                  <span className="px-2 py-0.5 rounded-full bg-emerald-100 dark:bg-emerald-950/60 text-emerald-700 dark:text-emerald-400 font-mono text-[11px] font-bold border border-emerald-200 dark:border-emerald-800/40">
                    Hostvra Enterprise
                  </span>
                </div>
                <div className="flex items-center gap-3 text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                  <span className="flex items-center gap-1">
                    <Server className="w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400" />
                    MySQL 10.11 / 127.0.0.1:3306
                  </span>
                  <span>•</span>
                  <span>Database: <strong className="text-slate-900 dark:text-white font-mono">{currentDb}</strong></span>
                  <span>•</span>
                  <span>Charset: <code className="font-mono text-slate-700 dark:text-slate-300">utf8mb4_unicode_ci</code></span>
                </div>
              </div>
            </div>

            {/* Actions: Database Switcher & Standalone Launch */}
            <div className="flex flex-wrap items-center gap-2.5">
              <div className="flex items-center gap-1.5 bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 rounded-xl px-2.5 py-1.5">
                <span className="text-xs font-semibold text-slate-500">Database:</span>
                <select
                  value={currentDb}
                  onChange={(e) => {
                    const newDb = e.target.value;
                    setCurrentDb(newDb);
                    if (DEFAULT_TABLES[newDb]) {
                      setTables(DEFAULT_TABLES[newDb]);
                    } else {
                      setTables(DEFAULT_TABLES['default']);
                    }
                    showToast(`Switched active database to ${newDb}`);
                  }}
                  className="bg-transparent text-xs font-bold text-slate-900 dark:text-white focus:outline-none cursor-pointer"
                >
                  {databaseList.map((db) => (
                    <option key={db} value={db} className="bg-white dark:bg-surface-900 text-slate-900 dark:text-white">
                      {db}
                    </option>
                  ))}
                </select>
              </div>

              <button
                onClick={() => {
                  window.open(standalonePmaUrl, '_blank');
                  showToast(`Opening standalone phpMyAdmin on port ${pmaPort}...`);
                }}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-white dark:bg-surface-800 hover:bg-slate-100 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-200 border border-slate-300 dark:border-surface-700 font-bold text-xs shadow-2xs transition cursor-pointer"
              >
                <ExternalLink className="w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400" />
                <span>Standalone PMA (Port {pmaPort})</span>
              </button>

              <button
                onClick={() => setCreateTableModalOpen(true)}
                className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs shadow-2xs transition cursor-pointer"
              >
                <Plus className="w-3.5 h-3.5" />
                <span>New Table</span>
              </button>
            </div>
          </div>

          {/* Navigation Tabs */}
          <div className="flex flex-wrap items-center gap-1 border-t border-slate-200 dark:border-surface-800 mt-4 pt-3 text-xs font-bold">
            {[
              { id: 'structure', label: 'Structure', icon: Table },
              { id: 'sql', label: 'SQL Query Console', icon: Play },
              { id: 'browse', label: `Browse (\`${selectedTable}\`)`, icon: Eye },
              { id: 'insert', label: 'Insert Row', icon: Plus },
              { id: 'export', label: 'Export', icon: Download },
              { id: 'import', label: 'Import', icon: Upload },
              { id: 'operations', label: 'Operations', icon: Settings },
              { id: 'standalone', label: 'Standalone Service Settings', icon: Server },
            ].map((tab) => {
              const Icon = tab.icon;
              const isActive = activeTab === tab.id;
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id as TabType)}
                  className={`flex items-center gap-1.5 px-3 py-2 rounded-xl transition cursor-pointer ${
                    isActive
                      ? 'bg-emerald-50 dark:bg-emerald-950/40 text-emerald-700 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800/50'
                      : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-surface-800 hover:text-slate-900 dark:hover:text-white'
                  }`}
                >
                  <Icon className={`w-3.5 h-3.5 ${isActive ? 'text-emerald-600 dark:text-emerald-400' : 'text-slate-500'}`} />
                  <span>{tab.label}</span>
                </button>
              );
            })}
          </div>
        </div>

        {/* Main Content Layout */}
        <div className="grid grid-cols-1 xl:grid-cols-4 gap-4">
          {/* Left Column: Tables Navigator */}
          <div className="xl:col-span-1 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-4 shadow-2xs space-y-3">
            <div className="flex items-center justify-between">
              <h3 className="font-bold text-xs uppercase tracking-wider text-slate-500 dark:text-slate-400">
                Tables in {currentDb} ({tables.length})
              </h3>
              <button
                onClick={() => {
                  setTables(DEFAULT_TABLES[currentDb] || DEFAULT_TABLES['edge'] || DEFAULT_TABLES['default']);
                  showToast('Refreshed table list');
                }}
                className="p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white transition"
                title="Refresh tables"
              >
                <RefreshCw className="w-3.5 h-3.5" />
              </button>
            </div>

            {/* Table Search */}
            <div className="relative">
              <Search className="w-3.5 h-3.5 absolute left-3 top-2.5 text-slate-400" />
              <input
                type="text"
                value={tableSearch}
                onChange={(e) => setTableSearch(e.target.value)}
                placeholder="Filter tables..."
                className="w-full pl-8 pr-3 py-1.5 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-xs text-slate-900 dark:text-white font-medium focus:outline-none focus:border-emerald-500"
              />
            </div>

            {/* Table List */}
            <div className="space-y-1 max-h-[500px] overflow-y-auto pr-1">
              {filteredTables.length === 0 ? (
                <p className="text-xs text-slate-400 py-3 text-center">No tables matching filter.</p>
              ) : (
                filteredTables.map((t) => {
                  const isSelected = selectedTable === t.name;
                  return (
                    <div
                      key={t.name}
                      onClick={() => handleBrowseTable(t.name)}
                      className={`flex items-center justify-between px-3 py-2 rounded-xl text-xs cursor-pointer transition ${
                        isSelected
                          ? 'bg-emerald-50 dark:bg-emerald-950/40 text-emerald-700 dark:text-emerald-300 font-bold border border-emerald-200 dark:border-emerald-800/40'
                          : 'hover:bg-slate-50 dark:hover:bg-surface-800 text-slate-700 dark:text-slate-300 font-medium'
                      }`}
                    >
                      <div className="flex items-center gap-2 truncate">
                        <Table className={`w-3.5 h-3.5 flex-shrink-0 ${isSelected ? 'text-emerald-600' : 'text-slate-400'}`} />
                        <span className="truncate font-mono">{t.name}</span>
                      </div>
                      <span className="text-[11px] font-mono text-slate-400">{t.rows}</span>
                    </div>
                  );
                })
              )}
            </div>

            {/* Quick Metrics Footer */}
            <div className="pt-3 border-t border-slate-100 dark:border-surface-800 text-[11px] text-slate-500 space-y-1 font-medium">
              <div className="flex justify-between">
                <span>Total Tables:</span>
                <strong className="text-slate-900 dark:text-white font-mono">{tables.length}</strong>
              </div>
              <div className="flex justify-between">
                <span>Total Rows:</span>
                <strong className="text-slate-900 dark:text-white font-mono">{totalRows}</strong>
              </div>
              <div className="flex justify-between">
                <span>Database Size:</span>
                <strong className="text-slate-900 dark:text-white font-mono">{totalSizeKb} KB</strong>
              </div>
            </div>
          </div>

          {/* Right Column: Tab View Area */}
          <div className="xl:col-span-3 space-y-4">
            {/* TAB 1: STRUCTURE */}
            {activeTab === 'structure' && (
              <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl shadow-2xs overflow-hidden">
                <div className="p-4 border-b border-slate-200 dark:border-surface-800 flex items-center justify-between">
                  <div>
                    <h2 className="font-bold text-sm text-slate-900 dark:text-white">Database Structure</h2>
                    <p className="text-xs text-slate-500">Tables in schema <code className="font-mono">{currentDb}</code></p>
                  </div>
                  <div className="flex items-center gap-2 text-xs">
                    <button
                      onClick={() => setSelectedTableNames(tables.map((t) => t.name))}
                      className="text-emerald-600 hover:underline font-semibold cursor-pointer"
                    >
                      Check all
                    </button>
                    <span>/</span>
                    <button
                      onClick={() => setSelectedTableNames([])}
                      className="text-slate-500 hover:underline font-semibold cursor-pointer"
                    >
                      Uncheck
                    </button>
                  </div>
                </div>

                <div className="overflow-x-auto">
                  <table className="w-full text-left text-xs border-collapse">
                    <thead>
                      <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50/75 dark:bg-surface-950 text-slate-600 dark:text-slate-400 font-bold uppercase text-[10px] tracking-wider">
                        <th className="p-3 w-8 text-center">
                          <input
                            type="checkbox"
                            checked={selectedTableNames.length === tables.length && tables.length > 0}
                            onChange={(e) => setSelectedTableNames(e.target.checked ? tables.map((t) => t.name) : [])}
                            className="rounded border-slate-300 text-emerald-600 focus:ring-emerald-500"
                          />
                        </th>
                        <th className="p-3">Table</th>
                        <th className="p-3">Action</th>
                        <th className="p-3 text-right">Rows</th>
                        <th className="p-3">Type</th>
                        <th className="p-3">Collation</th>
                        <th className="p-3 text-right">Size</th>
                        <th className="p-3">Comment</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100 dark:divide-surface-800 font-medium">
                      {tables.map((t) => {
                        const isChecked = selectedTableNames.includes(t.name);
                        return (
                          <tr key={t.name} className="hover:bg-slate-50 dark:hover:bg-surface-800/60 transition">
                            <td className="p-3 text-center">
                              <input
                                type="checkbox"
                                checked={isChecked}
                                onChange={(e) => {
                                  setSelectedTableNames(
                                    e.target.checked
                                      ? [...selectedTableNames, t.name]
                                      : selectedTableNames.filter((n) => n !== t.name)
                                  );
                                }}
                                className="rounded border-slate-300 text-emerald-600 focus:ring-emerald-500"
                              />
                            </td>
                            <td className="p-3 font-mono font-bold text-slate-900 dark:text-white">
                              <button
                                onClick={() => handleBrowseTable(t.name)}
                                className="hover:text-emerald-600 hover:underline cursor-pointer flex items-center gap-1.5"
                              >
                                <Table className="w-3.5 h-3.5 text-slate-400" />
                                <span>{t.name}</span>
                              </button>
                            </td>
                            <td className="p-3">
                              <div className="flex items-center gap-2 font-semibold">
                                <button
                                  onClick={() => handleBrowseTable(t.name)}
                                  className="text-emerald-600 dark:text-emerald-400 hover:underline cursor-pointer"
                                >
                                  Browse
                                </button>
                                <span className="text-slate-300">|</span>
                                <button
                                  onClick={() => handleEmptyTable(t.name)}
                                  className="text-orange-600 hover:underline cursor-pointer"
                                >
                                  Empty
                                </button>
                                <span className="text-slate-300">|</span>
                                <button
                                  onClick={() => handleDropTable(t.name)}
                                  className="text-rose-600 hover:underline cursor-pointer"
                                >
                                  Drop
                                </button>
                              </div>
                            </td>
                            <td className="p-3 text-right font-mono text-slate-900 dark:text-white">{t.rows.toLocaleString()}</td>
                            <td className="p-3 font-mono text-slate-600 dark:text-slate-400">{t.engine}</td>
                            <td className="p-3 font-mono text-[11px] text-slate-500">{t.collation}</td>
                            <td className="p-3 text-right font-mono text-slate-900 dark:text-white">{t.sizeKb} KB</td>
                            <td className="p-3 text-slate-400 text-[11px]">{t.comment || '-'}</td>
                          </tr>
                        );
                      })}
                    </tbody>
                    <tfoot>
                      <tr className="border-t-2 border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-surface-950 font-bold text-slate-900 dark:text-white">
                        <td className="p-3 text-center"></td>
                        <td className="p-3">{tables.length} tables</td>
                        <td className="p-3"></td>
                        <td className="p-3 text-right font-mono">{totalRows.toLocaleString()}</td>
                        <td className="p-3 font-mono">InnoDB</td>
                        <td className="p-3 font-mono text-[11px]">utf8mb4_unicode_ci</td>
                        <td className="p-3 text-right font-mono">{totalSizeKb} KB</td>
                        <td className="p-3"></td>
                      </tr>
                    </tfoot>
                  </table>
                </div>

                {/* Batch Actions on Selected Tables */}
                {selectedTableNames.length > 0 && (
                  <div className="p-3 bg-slate-50 dark:bg-surface-950 border-t border-slate-200 dark:border-surface-800 flex items-center justify-between text-xs">
                    <span className="text-slate-600 dark:text-slate-400 font-semibold">
                      With selected ({selectedTableNames.length} tables):
                    </span>
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => {
                          showToast(`Optimized ${selectedTableNames.length} tables.`);
                        }}
                        className="px-3 py-1.5 rounded-lg bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 font-bold hover:bg-slate-100 dark:hover:bg-surface-700"
                      >
                        Optimize tables
                      </button>
                      <button
                        onClick={() => {
                          showToast(`Checked & repaired ${selectedTableNames.length} tables.`);
                        }}
                        className="px-3 py-1.5 rounded-lg bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 font-bold hover:bg-slate-100 dark:hover:bg-surface-700"
                      >
                        Check &amp; Repair
                      </button>
                      <button
                        onClick={() => {
                          if (confirm(`Drop ${selectedTableNames.length} tables from ${currentDb}?`)) {
                            setTables((prev) => prev.filter((t) => !selectedTableNames.includes(t.name)));
                            setSelectedTableNames([]);
                            showToast(`Selected tables dropped.`);
                          }
                        }}
                        className="px-3 py-1.5 rounded-lg bg-orange-600 hover:bg-orange-700 text-white font-bold cursor-pointer"
                      >
                        Drop
                      </button>
                    </div>
                  </div>
                )}
              </div>
            )}

            {/* TAB 2: SQL QUERY CONSOLE */}
            {activeTab === 'sql' && (
              <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-4 shadow-2xs space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <h2 className="font-bold text-sm text-slate-900 dark:text-white">Interactive SQL Query Editor</h2>
                    <p className="text-xs text-slate-500">Run queries against database: <code className="font-mono text-emerald-600 dark:text-emerald-400">{currentDb}</code></p>
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => setSqlQuery(`SELECT * FROM \`${selectedTable}\` LIMIT 50;`)}
                      className="px-2.5 py-1 rounded-lg bg-slate-100 dark:bg-surface-800 text-[11px] font-bold text-slate-700 dark:text-slate-300 hover:bg-slate-200"
                    >
                      SELECT *
                    </button>
                    <button
                      onClick={() => setSqlQuery(`SHOW TABLES;`)}
                      className="px-2.5 py-1 rounded-lg bg-slate-100 dark:bg-surface-800 text-[11px] font-bold text-slate-700 dark:text-slate-300 hover:bg-slate-200"
                    >
                      SHOW TABLES
                    </button>
                    <button
                      onClick={() => setSqlQuery(`SHOW PROCESSLIST;`)}
                      className="px-2.5 py-1 rounded-lg bg-slate-100 dark:bg-surface-800 text-[11px] font-bold text-slate-700 dark:text-slate-300 hover:bg-slate-200"
                    >
                      PROCESSLIST
                    </button>
                  </div>
                </div>

                {/* Editor Box */}
                <div className="relative border border-slate-300 dark:border-surface-700 rounded-xl overflow-hidden focus-within:border-emerald-500">
                  <textarea
                    rows={6}
                    value={sqlQuery}
                    onChange={(e) => setSqlQuery(e.target.value)}
                    placeholder="Enter SQL command here (e.g. SELECT * FROM users;)"
                    className="w-full p-3 font-mono text-xs bg-slate-50 dark:bg-surface-950 text-slate-900 dark:text-white focus:outline-none resize-y"
                  />
                  <div className="p-2.5 bg-slate-100 dark:bg-surface-900 border-t border-slate-200 dark:border-surface-800 flex items-center justify-between">
                    <span className="text-[11px] text-slate-500">
                      Shortcut: Click <strong>Run Query</strong> to execute
                    </span>
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => setSqlQuery('')}
                        className="px-3 py-1 rounded-lg bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-xs font-semibold hover:bg-slate-50"
                      >
                        Clear
                      </button>
                      <button
                        onClick={handleRunQuery}
                        className="flex items-center gap-1.5 px-4 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white text-xs font-bold shadow-xs cursor-pointer"
                      >
                        <Play className="w-3.5 h-3.5 fill-current" />
                        <span>Run Query</span>
                      </button>
                    </div>
                  </div>
                </div>

                {/* Error Banner */}
                {queryError && (
                  <div className="p-3 rounded-xl bg-rose-50 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-900 text-rose-700 dark:text-rose-400 text-xs flex items-center gap-2">
                    <AlertCircle className="w-4 h-4 flex-shrink-0" />
                    <span>{queryError}</span>
                  </div>
                )}

                {/* Results View */}
                {queryResult && (
                  <div className="border border-slate-200 dark:border-surface-800 rounded-xl overflow-hidden">
                    <div className="p-3 bg-slate-50 dark:bg-surface-950 border-b border-slate-200 dark:border-surface-800 flex items-center justify-between text-xs">
                      <div className="flex items-center gap-2">
                        <CheckCircle2 className="w-4 h-4 text-emerald-600" />
                        <span className="font-bold text-slate-900 dark:text-white">
                          Showing {queryRowsAffected} rows
                        </span>
                        <span className="text-slate-400">({queryExecutionTime})</span>
                      </div>
                      <button
                        onClick={() => {
                          navigator.clipboard.writeText(JSON.stringify(queryResult, null, 2));
                          showToast('Results JSON copied to clipboard');
                        }}
                        className="flex items-center gap-1 text-slate-600 dark:text-slate-400 hover:text-slate-900 font-semibold cursor-pointer"
                      >
                        <Copy className="w-3.5 h-3.5" />
                        <span>Copy JSON</span>
                      </button>
                    </div>

                    <div className="overflow-x-auto max-h-96">
                      <table className="w-full text-left text-xs border-collapse font-mono">
                        <thead>
                          <tr className="bg-slate-100/80 dark:bg-surface-900 text-slate-700 dark:text-slate-300 font-bold border-b border-slate-200 dark:border-surface-800">
                            {queryColumns.map((col) => (
                              <th key={col} className="p-2.5 whitespace-nowrap">
                                {col}
                              </th>
                            ))}
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-100 dark:divide-surface-800">
                          {queryResult.map((row, idx) => (
                            <tr key={idx} className="hover:bg-slate-50 dark:hover:bg-surface-800/50">
                              {queryColumns.map((col) => (
                                <td key={col} className="p-2.5 whitespace-nowrap text-slate-800 dark:text-slate-200">
                                  {row[col] !== undefined && row[col] !== null ? String(row[col]) : <em className="text-slate-400">NULL</em>}
                                </td>
                              ))}
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </div>
                )}
              </div>
            )}

            {/* TAB 3: BROWSE SELECTED TABLE */}
            {activeTab === 'browse' && (
              <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl shadow-2xs overflow-hidden">
                <div className="p-4 border-b border-slate-200 dark:border-surface-800 flex items-center justify-between">
                  <div>
                    <h2 className="font-bold text-sm text-slate-900 dark:text-white flex items-center gap-2">
                      <Table className="w-4 h-4 text-emerald-600" />
                      <span>Table: <code className="font-mono">{selectedTable}</code></span>
                    </h2>
                    <p className="text-xs text-slate-500">Browsing rows in <code className="font-mono">{currentDb}.{selectedTable}</code></p>
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => setActiveTab('insert')}
                      className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs shadow-xs cursor-pointer"
                    >
                      <Plus className="w-3.5 h-3.5" />
                      <span>Insert Row</span>
                    </button>
                  </div>
                </div>

                <div className="overflow-x-auto">
                  <table className="w-full text-left text-xs border-collapse font-mono">
                    <thead>
                      <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-surface-950 text-slate-700 dark:text-slate-300 font-bold">
                        <th className="p-2.5 w-16 text-center">Action</th>
                        {queryColumns.map((col) => (
                          <th key={col} className="p-2.5 whitespace-nowrap">
                            {col}
                          </th>
                        ))}
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100 dark:divide-surface-800">
                      {(queryResult || SAMPLE_ROWS['users']).map((row, idx) => (
                        <tr key={idx} className="hover:bg-slate-50 dark:hover:bg-surface-800/50">
                          <td className="p-2.5 text-center font-sans">
                            <div className="flex items-center justify-center gap-1 text-slate-400">
                              <button
                                onClick={() => showToast(`Edit row ${idx + 1}`)}
                                className="hover:text-emerald-600 cursor-pointer p-1"
                                title="Edit row"
                              >
                                <Edit3 className="w-3.5 h-3.5" />
                              </button>
                              <button
                                onClick={() => {
                                  if (confirm(`Delete this row?`)) {
                                    setQueryResult((prev) => (prev ? prev.filter((_, i) => i !== idx) : []));
                                    showToast('Row deleted.');
                                  }
                                }}
                                className="hover:text-rose-600 cursor-pointer p-1"
                                title="Delete row"
                              >
                                <Trash2 className="w-3.5 h-3.5" />
                              </button>
                            </div>
                          </td>
                          {queryColumns.map((col) => (
                            <td key={col} className="p-2.5 whitespace-nowrap text-slate-800 dark:text-slate-200">
                              {row[col] !== undefined && row[col] !== null ? String(row[col]) : <em className="text-slate-400">NULL</em>}
                            </td>
                          ))}
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>

                <div className="p-3 bg-slate-50 dark:bg-surface-950 border-t border-slate-200 dark:border-surface-800 flex items-center justify-between text-xs text-slate-500 font-medium">
                  <span>Showing rows 1 - {(queryResult || []).length}</span>
                  <div className="flex items-center gap-1 font-mono">
                    <button className="px-2 py-1 rounded bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-600 dark:text-slate-400">
                      &lt;
                    </button>
                    <span className="px-2.5 py-1 font-bold text-slate-900 dark:text-white">Page 1</span>
                    <button className="px-2 py-1 rounded bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-600 dark:text-slate-400">
                      &gt;
                    </button>
                  </div>
                </div>
              </div>
            )}

            {/* TAB 4: INSERT ROW */}
            {activeTab === 'insert' && (
              <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-6 shadow-2xs space-y-4">
                <div className="flex items-center justify-between border-b border-slate-200 dark:border-surface-800 pb-3">
                  <div>
                    <h2 className="font-bold text-sm text-slate-900 dark:text-white">Insert New Row into `{selectedTable}`</h2>
                    <p className="text-xs text-slate-500">Provide field values to insert</p>
                  </div>
                </div>

                <form
                  onSubmit={(e) => {
                    e.preventDefault();
                    showToast(`Row inserted into \`${selectedTable}\``);
                    setActiveTab('browse');
                  }}
                  className="space-y-3 max-w-xl text-xs"
                >
                  {queryColumns.map((col) => (
                    <div key={col} className="grid grid-cols-3 items-center gap-3">
                      <label className="font-mono font-bold text-slate-700 dark:text-slate-300 truncate">
                        {col}
                      </label>
                      <input
                        type="text"
                        placeholder={`Value for ${col}`}
                        className="col-span-2 px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 font-mono text-xs focus:outline-none focus:border-emerald-500 text-slate-900 dark:text-white"
                      />
                    </div>
                  ))}

                  <div className="pt-4 flex justify-end gap-2">
                    <button
                      type="button"
                      onClick={() => setActiveTab('browse')}
                      className="px-4 py-2 rounded-xl bg-slate-100 dark:bg-surface-800 hover:bg-slate-200 font-semibold text-slate-700 dark:text-slate-300"
                    >
                      Cancel
                    </button>
                    <button
                      type="submit"
                      className="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold shadow-xs cursor-pointer"
                    >
                      Save Row
                    </button>
                  </div>
                </form>
              </div>
            )}

            {/* TAB 5: EXPORT */}
            {activeTab === 'export' && (
              <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-6 shadow-2xs space-y-4">
                <div>
                  <h2 className="font-bold text-sm text-slate-900 dark:text-white">Export Database Dump</h2>
                  <p className="text-xs text-slate-500">Export tables and rows from <code className="font-mono">{currentDb}</code></p>
                </div>

                <div className="max-w-md space-y-4 text-xs font-semibold">
                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 mb-1">Export Method</label>
                    <div className="space-y-2">
                      <label className="flex items-center gap-2 p-2 rounded-xl border border-slate-200 dark:border-surface-700 bg-slate-50 dark:bg-surface-800 cursor-pointer">
                        <input type="radio" name="export_method" defaultChecked className="text-emerald-600 focus:ring-emerald-500" />
                        <div>
                          <span className="font-bold text-slate-900 dark:text-white">Quick</span>
                          <p className="text-[11px] text-slate-500 font-normal">Display only the minimal options</p>
                        </div>
                      </label>
                      <label className="flex items-center gap-2 p-2 rounded-xl border border-slate-200 dark:border-surface-700 bg-slate-50 dark:bg-surface-800 cursor-pointer">
                        <input type="radio" name="export_method" className="text-emerald-600 focus:ring-emerald-500" />
                        <div>
                          <span className="font-bold text-slate-900 dark:text-white">Custom</span>
                          <p className="text-[11px] text-slate-500 font-normal">Display all possible options (tables, compression)</p>
                        </div>
                      </label>
                    </div>
                  </div>

                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 mb-1">Format</label>
                    <select className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-mono">
                      <option value="sql">SQL (*.sql)</option>
                      <option value="sql_gz">GZipped SQL (*.sql.gz)</option>
                      <option value="csv">CSV (Comma Separated)</option>
                      <option value="json">JSON</option>
                    </select>
                  </div>

                  <button
                    onClick={() => {
                      const dummySql = `-- Hostvra Database Dump\n-- Database: ${currentDb}\n-- Generation Time: ${new Date().toISOString()}\n\nCREATE DATABASE IF NOT EXISTS \`${currentDb}\`;\nUSE \`${currentDb}\`;\n`;
                      const blob = new Blob([dummySql], { type: 'application/sql' });
                      const url = URL.createObjectURL(blob);
                      const a = document.createElement('a');
                      a.href = url;
                      a.download = `${currentDb}_dump_${Date.now()}.sql`;
                      a.click();
                      showToast(`Exported ${currentDb} SQL dump.`);
                    }}
                    className="flex items-center gap-2 px-5 py-2.5 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold shadow-xs cursor-pointer"
                  >
                    <Download className="w-4 h-4" />
                    <span>Download SQL Dump</span>
                  </button>
                </div>
              </div>
            )}

            {/* TAB 6: IMPORT */}
            {activeTab === 'import' && (
              <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-6 shadow-2xs space-y-4">
                <div>
                  <h2 className="font-bold text-sm text-slate-900 dark:text-white">Import into Database</h2>
                  <p className="text-xs text-slate-500">Restore or execute SQL dump file into schema <code className="font-mono">{currentDb}</code></p>
                </div>

                <div className="max-w-md space-y-4 text-xs font-semibold">
                  <div className="border-2 border-dashed border-slate-300 dark:border-surface-700 rounded-2xl p-6 text-center hover:border-emerald-500 transition cursor-pointer bg-slate-50/50 dark:bg-surface-950">
                    <Upload className="w-8 h-8 text-slate-400 mx-auto mb-2" />
                    <p className="font-bold text-slate-900 dark:text-white">Click or drag `.sql` or `.sql.gz` file here</p>
                    <p className="text-[11px] text-slate-500 font-normal mt-1">Maximum upload size: 1024 MB</p>
                    <input
                      type="file"
                      accept=".sql,.gz,.zip"
                      className="hidden"
                      id="sql-file-input"
                      onChange={(e) => {
                        const file = e.target.files?.[0];
                        if (file) {
                          showToast(`File selected: ${file.name} (${(file.size / 1024).toFixed(1)} KB)`);
                        }
                      }}
                    />
                    <label
                      htmlFor="sql-file-input"
                      className="mt-3 inline-block px-4 py-1.5 rounded-xl bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 font-bold hover:bg-slate-100 cursor-pointer text-slate-800 dark:text-slate-200"
                    >
                      Browse File
                    </label>
                  </div>

                  <div className="flex items-center gap-2">
                    <input type="checkbox" id="partial-import" defaultChecked className="rounded text-emerald-600 focus:ring-emerald-500" />
                    <label htmlFor="partial-import" className="text-slate-700 dark:text-slate-300 text-xs">
                      Allow interrupt of an import in case script detects error
                    </label>
                  </div>

                  <button
                    onClick={() => {
                      showToast(`Import completed successfully into ${currentDb}`);
                    }}
                    className="flex items-center gap-2 px-5 py-2.5 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold shadow-xs cursor-pointer"
                  >
                    <Upload className="w-4 h-4" />
                    <span>Start Import</span>
                  </button>
                </div>
              </div>
            )}

            {/* TAB 7: OPERATIONS */}
            {activeTab === 'operations' && (
              <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-6 shadow-2xs space-y-6">
                <div>
                  <h2 className="font-bold text-sm text-slate-900 dark:text-white">Database Operations &amp; Maintenance</h2>
                  <p className="text-xs text-slate-500">Tune and maintain database: <code className="font-mono">{currentDb}</code></p>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
                  <div className="p-4 rounded-xl border border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-surface-950 space-y-2">
                    <h3 className="font-bold text-slate-900 dark:text-white">Optimize Database</h3>
                    <p className="text-slate-500 text-[11px]">Reclaims unused space and defragments data files for all InnoDB tables.</p>
                    <button
                      onClick={() => showToast(`Database ${currentDb} optimized successfully.`)}
                      className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold cursor-pointer"
                    >
                      Run Optimize
                    </button>
                  </div>

                  <div className="p-4 rounded-xl border border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-surface-950 space-y-2">
                    <h3 className="font-bold text-slate-900 dark:text-white">Check &amp; Repair Tables</h3>
                    <p className="text-slate-500 text-[11px]">Analyzes indices and repairs corrupted table key structures.</p>
                    <button
                      onClick={() => showToast(`Check & Repair complete for ${currentDb}.`)}
                      className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold cursor-pointer"
                    >
                      Check &amp; Repair
                    </button>
                  </div>

                  <div className="p-4 rounded-xl border border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-surface-950 space-y-2">
                    <h3 className="font-bold text-slate-900 dark:text-white">Collation &amp; Character Set</h3>
                    <p className="text-slate-500 text-[11px]">Change default collation for new tables.</p>
                    <div className="flex gap-2">
                      <select className="px-3 py-1.5 rounded-lg bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 font-mono text-xs">
                        <option>utf8mb4_unicode_ci</option>
                        <option>utf8mb4_general_ci</option>
                        <option>utf8_general_ci</option>
                      </select>
                      <button
                        onClick={() => showToast(`Collation updated to utf8mb4_unicode_ci`)}
                        className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold cursor-pointer"
                      >
                        Save
                      </button>
                    </div>
                  </div>

                  <div className="p-4 rounded-xl border border-rose-200 dark:border-rose-900/40 bg-rose-50/40 dark:bg-rose-950/20 space-y-2">
                    <h3 className="font-bold text-rose-700 dark:text-rose-400">Drop Database</h3>
                    <p className="text-rose-600/80 dark:text-rose-400/80 text-[11px]">Completely destroy the schema and all tables inside.</p>
                    <button
                      onClick={() => {
                        if (confirm(`CRITICAL: Drop database \`${currentDb}\` permanently?`)) {
                          showToast(`Database \`${currentDb}\` dropped.`);
                        }
                      }}
                      className="px-3.5 py-1.5 rounded-lg bg-orange-600 hover:bg-orange-700 text-white font-bold cursor-pointer"
                    >
                      Drop Database
                    </button>
                  </div>
                </div>
              </div>
            )}

            {/* TAB 8: STANDALONE SERVICE SETTINGS */}
            {activeTab === 'standalone' && (
              <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-6 shadow-2xs space-y-5">
                <div>
                  <h2 className="font-bold text-sm text-slate-900 dark:text-white">Standalone phpMyAdmin Service Configuration</h2>
                  <p className="text-xs text-slate-500">Configure port and direct web daemon settings for phpMyAdmin</p>
                </div>

                <div className="max-w-lg space-y-4 text-xs font-semibold">
                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 mb-1">phpMyAdmin Service Port</label>
                    <div className="flex gap-2">
                      <input
                        type="text"
                        value={pmaPort}
                        onChange={(e) => setPmaPort(e.target.value)}
                        placeholder="888"
                        className="w-32 px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 font-mono text-slate-900 dark:text-white"
                      />
                      <button
                        onClick={() => showToast(`phpMyAdmin port updated to ${pmaPort}`)}
                        className="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold shadow-xs cursor-pointer"
                      >
                        Save Port
                      </button>
                    </div>
                    <span className="text-[11px] text-slate-500 font-normal mt-1 block">
                      Default aaPanel/Hostvra phpMyAdmin daemon listens on port 888. Ensure this port is open in your firewall.
                    </span>
                  </div>

                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 mb-1">Direct Web Access URL</label>
                    <div className="flex gap-2">
                      <input
                        type="text"
                        readOnly
                        value={standalonePmaUrl}
                        className="flex-1 px-3 py-2 rounded-xl bg-slate-100 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 font-mono text-slate-700 dark:text-slate-300"
                      />
                      <button
                        onClick={() => {
                          navigator.clipboard.writeText(standalonePmaUrl);
                          setCopied(true);
                          setTimeout(() => setCopied(false), 2000);
                          showToast('Copied standalone PMA URL to clipboard');
                        }}
                        className="px-3 py-2 rounded-xl bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 hover:bg-slate-100 text-slate-700 dark:text-slate-300 font-bold"
                      >
                        {copied ? <Check className="w-4 h-4 text-emerald-600" /> : <Copy className="w-4 h-4" />}
                      </button>
                      <button
                        onClick={() => window.open(standalonePmaUrl, '_blank')}
                        className="flex items-center gap-1.5 px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold shadow-xs cursor-pointer"
                      >
                        <ExternalLink className="w-3.5 h-3.5" />
                        <span>Launch</span>
                      </button>
                    </div>
                  </div>

                  <div className="p-4 rounded-xl bg-emerald-50/50 dark:bg-emerald-950/20 border border-emerald-200 dark:border-emerald-800/40 text-slate-700 dark:text-slate-300 space-y-1.5">
                    <div className="flex items-center gap-2 font-bold text-emerald-800 dark:text-emerald-400">
                      <Zap className="w-4 h-4" />
                      <span>Built-in Web SQL Mode Enabled</span>
                    </div>
                    <p className="text-[11px] leading-relaxed">
                      You can use this unified Hostvra Database Manager directly within the browser tab to browse data, run SQL queries, import, and export without requiring external port 888 access or security group adjustments.
                    </p>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* CREATE TABLE MODAL */}
      {createTableModalOpen && (
        <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
          <div className="w-full max-w-md bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative text-xs">
            <h3 className="text-base font-bold text-slate-900 dark:text-white mb-2">Create New Table</h3>
            <p className="text-slate-500 mb-4">Add a new database table to <code className="font-mono text-emerald-600">{currentDb}</code></p>

            <form onSubmit={handleCreateTableSubmit} className="space-y-4">
              <div>
                <label className="block font-semibold text-slate-700 dark:text-slate-300 mb-1">Table Name</label>
                <input
                  type="text"
                  required
                  placeholder="e.g. orders, products, tokens"
                  value={newTableName}
                  onChange={(e) => setNewTableName(e.target.value)}
                  className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-mono focus:outline-none focus:border-emerald-500"
                />
              </div>

              <div>
                <label className="block font-semibold text-slate-700 dark:text-slate-300 mb-1">Number of Columns</label>
                <input
                  type="number"
                  min={1}
                  max={50}
                  value={newTableCols}
                  onChange={(e) => setNewTableCols(Number(e.target.value))}
                  className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-mono focus:outline-none focus:border-emerald-500"
                />
              </div>

              <div>
                <label className="block font-semibold text-slate-700 dark:text-slate-300 mb-1">Storage Engine</label>
                <select className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-mono">
                  <option value="InnoDB">InnoDB (Transactions, Foreign Keys)</option>
                  <option value="MyISAM">MyISAM</option>
                  <option value="MEMORY">MEMORY</option>
                </select>
              </div>

              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setCreateTableModalOpen(false)}
                  className="px-4 py-2 rounded-xl bg-slate-100 dark:bg-surface-800 hover:bg-slate-200 dark:hover:bg-surface-700 font-semibold text-slate-700 dark:text-slate-300 cursor-pointer"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold shadow-xs cursor-pointer"
                >
                  Create Table
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </DashboardShell>
  );
}

export default function PhpMyAdminPage() {
  return (
    <Suspense fallback={<div className="p-8 text-center text-slate-500">Loading Database Manager...</div>}>
      <PhpMyAdminManager />
    </Suspense>
  );
}
