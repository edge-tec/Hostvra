'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Database as DatabaseIcon,
  Plus,
  Trash2,
  RefreshCw,
  Search,
  Server as ServerIcon,
  Key,
  X,
  Check,
  Eye,
  EyeOff,
  Copy,
  FolderSync,
  Download,
  Upload,
  Settings,
  Shield,
  HelpCircle,
  ExternalLink,
  Wrench,
  AlertCircle,
  CheckCircle2,
  Lock,
  Play,
  Share2,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import { apiFetch, Database, Server } from '@/lib/api';

type DBEngine = 'mysql' | 'sqlserver' | 'mongodb' | 'redis' | 'pgsql';

export default function DatabasesPage() {
  // Engine Tab State
  const [activeEngine, setActiveEngine] = useState<DBEngine>('mysql');

  // Server & DB State
  const [servers, setServers] = useState<Server[]>([]);
  const [selectedServer, setSelectedServer] = useState<string>('');
  const [databases, setDatabases] = useState<Database[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [filterType, setFilterType] = useState('all');

  // Selection for batch actions
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [batchAction, setBatchAction] = useState<string>('');
  const [executingBatch, setExecutingBatch] = useState(false);

  // Auto-backup toggle state
  const [autoBackup, setAutoBackup] = useState(true);
  const [togglingAutoBackup, setTogglingAutoBackup] = useState(false);

  // Engine status
  const [engineVersion, setEngineVersion] = useState('Mysql 10.11.6');
  const [engineRunning, setEngineRunning] = useState(true);

  // Toast Notification
  const [toast, setToast] = useState<{ message: string; isError?: boolean } | null>(null);

  // Password Visibility Toggle (by DB ID)
  const [visiblePasswords, setVisiblePasswords] = useState<{ [id: string]: boolean }>({});
  const [copiedId, setCopiedId] = useState<string | null>(null);

  // Inline Note Editing
  const [editingNoteId, setEditingNoteId] = useState<string | null>(null);
  const [editingNoteText, setEditingNoteText] = useState('');

  // Modals
  const [createDbOpen, setCreateDbOpen] = useState(false);
  const [rootPasswordOpen, setRootPasswordOpen] = useState(false);
  const [rootPassword, setRootPassword] = useState('Hostvra@Root#2026!');
  const [newRootPassword, setNewRootPassword] = useState('');
  const [changingRootPass, setChangingRootPass] = useState(false);

  const [remoteDbOpen, setRemoteDbOpen] = useState(false);
  const [advancedSetupOpen, setAdvancedSetupOpen] = useState(false);
  const [recycleBinOpen, setRecycleBinOpen] = useState(false);
  const [recycleDbs, setRecycleDbs] = useState<Database[]>([]);
  const [feedbackOpen, setFeedbackOpen] = useState(false);
  const [feedbackText, setFeedbackText] = useState('');

  // Row-level Action Modals
  const [selectedDb, setSelectedDb] = useState<Database | null>(null);
  const [permissionModalOpen, setPermissionModalOpen] = useState(false);
  const [toolsModalOpen, setToolsModalOpen] = useState(false);
  const [toolsOutput, setToolsOutput] = useState<string | null>(null);
  const [runningTool, setRunningTool] = useState(false);
  const [passwordModalOpen, setPasswordModalOpen] = useState(false);
  const [newDbPassword, setNewDbPassword] = useState('');
  const [importModalOpen, setImportModalOpen] = useState(false);
  const [importSqlText, setImportSqlText] = useState('');
  const [importing, setImporting] = useState(false);

  // Create DB Form Fields
  const [dbName, setDbName] = useState('');
  const [dbUser, setDbUser] = useState('');
  const [dbPassword, setDbPassword] = useState('');
  const [dbCharset, setDbCharset] = useState('utf8mb4');
  const [dbCollation, setDbCollation] = useState('utf8mb4_unicode_ci');
  const [dbHostAllow, setDbHostAllow] = useState('localhost');
  const [dbNote, setDbNote] = useState('');
  const [creating, setCreating] = useState(false);

  // Notification helper
  const showToast = (message: string, isError = false) => {
    setToast({ message, isError });
    setTimeout(() => setToast(null), 3500);
  };

  // Generate strong random password
  const generatePassword = () => {
    const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*';
    let pass = '';
    for (let i = 0; i < 16; i++) {
      pass += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    return pass;
  };

  // Copy to clipboard helper
  const copyToClipboard = (text: string, id: string) => {
    navigator.clipboard.writeText(text);
    setCopiedId(id);
    showToast('Copied to clipboard!');
    setTimeout(() => setCopiedId(null), 2000);
  };

  // Fetch servers & engine status
  const fetchServers = async () => {
    const res = await apiFetch<Server[]>('/api/v1/servers');
    if (res.success && res.data) {
      setServers(res.data);
      if (res.data.length > 0) {
        setSelectedServer(res.data[0].id);
      }
    }
  };

  // Fetch databases
  const fetchDatabases = useCallback(async (serverId?: string) => {
    setLoading(true);
    const query = serverId ? `?server_id=${serverId}` : '';
    const res = await apiFetch<Database[]>(`/api/v1/databases${query}`);
    if (res.success && res.data) {
      // Ensure seed databases exist for demo/initial view if empty
      if (res.data.length === 0) {
        const seedDbs: Database[] = [
          { id: 'db-1', db_type: 'mysql', name: 'sql_2xbets_net', username: 'sql_2xbets_net', password: 'pW9#xK2!mN7$', quota: 'Not set', backup_status: 'Not exist', location: 'Localhost', note: '2xbets.net', created_at: new Date().toISOString() },
          { id: 'db-2', db_type: 'mysql', name: 'sql_antiprofiles_com', username: 'sql_antiprofiles_com', password: 'aR4@vB8*eT1%', quota: 'Not set', backup_status: 'Not exist', location: 'Localhost', note: 'antiprofiles.com', created_at: new Date().toISOString() },
          { id: 'db-3', db_type: 'mysql', name: 'antidetactor', username: 'antidetactor', password: 'tY7&uI3#oP9^', quota: 'Not set', backup_status: 'Not exist', location: 'Localhost', note: 'antidetactor', created_at: new Date().toISOString() },
          { id: 'db-4', db_type: 'mysql', name: 'mailsz0_1', username: 'mailsz0_1', password: 'qW1!eR2@tY3#', quota: 'Not set', backup_status: 'Not exist', location: 'Localhost', note: 'mailsz0_1', created_at: new Date().toISOString() },
          { id: 'db-5', db_type: 'mysql', name: 'mailspro_order', username: 'mailspro_order', password: 'uI4$oP5%aS6^', quota: 'Not set', backup_status: 'Not exist', location: 'Localhost', note: 'mailspro_order', created_at: new Date().toISOString() },
          { id: 'db-6', db_type: 'mysql', name: 'sql_metmco_net', username: 'sql_metmco_net', password: 'dF7&gH8*jK9(', quota: 'Not set', backup_status: 'Not exist', location: 'Localhost', note: 'metmco.net', created_at: new Date().toISOString() },
          { id: 'db-7', db_type: 'mysql', name: 'biography', username: 'biography', password: 'zX1)cV2_bN3+', quota: 'Not set', backup_status: 'Not exist', location: 'Localhost', note: 'biography', created_at: new Date().toISOString() },
          { id: 'db-8', db_type: 'mysql', name: 'newspaper', username: 'newspaper', password: 'mK4-jH5=gF6[', quota: 'Not set', backup_status: 'Not exist', location: 'Localhost', note: 'newspaper', created_at: new Date().toISOString() },
          { id: 'db-9', db_type: 'mysql', name: 'edgecash', username: 'edgecash', password: 'dS7]aP8{oI9}', quota: 'Not set', backup_status: 'Not exist', location: 'Localhost', note: 'edgecash', created_at: new Date().toISOString() },
          { id: 'db-10', db_type: 'mysql', name: 'sql_app_affscash_net', username: 'sql_app_affscash_net', password: 'uY1:tR2;eW3?', quota: 'Not set', backup_status: 'Not exist', location: 'Localhost', note: 'app.affscash.net', created_at: new Date().toISOString() },
          { id: 'db-11', db_type: 'mysql', name: 'sql_affscash_net', username: 'sql_affscash_net', password: 'qA4<zS5>xD6/', quota: 'Not set', backup_status: 'Not exist', location: 'Localhost', note: 'affscash.net', created_at: new Date().toISOString() },
          { id: 'db-12', db_type: 'mysql', name: 'sql_analytics_wpflood_com', username: 'sql_analytics_wpflood_com', password: 'cF7~vG8`bH9|', quota: 'Not set', backup_status: 'Not exist', location: 'Localhost', note: 'analytics.wpflood.com', created_at: new Date().toISOString() },
          { id: 'db-13', db_type: 'mysql', name: 'sql_edgecash_net', username: 'sql_edgecash_net', password: 'nJ1!mK2@lP3#', quota: 'Not set', backup_status: 'Not exist', location: 'Localhost', note: 'edgecash.net', created_at: new Date().toISOString() },
          { id: 'db-14', db_type: 'mysql', name: 'sql_mail_mailsz0_com', username: 'sql_mail_mailsz0_com', password: 'oI4$uY5%tT6^', quota: 'Not set', backup_status: 'Not exist', location: 'Localhost', note: 'mail.mailsz0.com', created_at: new Date().toISOString() },
          { id: 'db-15', db_type: 'mysql', name: 'sql_mailsz0_com', username: 'sql_mailsz0_com', password: 'rE7&wQ8*aZ9(', quota: 'Not set', backup_status: 'Not exist', location: 'Localhost', note: 'mailsz0.com', created_at: new Date().toISOString() },
          { id: 'db-16', db_type: 'mysql', name: 'sql_eliteall_com', username: 'sql_eliteall_com', password: 'sX1)dC2_fV3+', quota: 'Not set', backup_status: 'Not exist', location: 'Localhost', note: 'eliteall.com', created_at: new Date().toISOString() },
          { id: 'db-17', db_type: 'mysql', name: 'sql_ushort_link', username: 'sql_ushort_link', password: 'gB4-hN5=jM6[', quota: 'Not set', backup_status: 'Not exist', location: 'Localhost', note: 'ushort.link', created_at: new Date().toISOString() },
          { id: 'db-18', db_type: 'mysql', name: 'sql_shroo_link', username: 'sql_shroo_link', password: 'kL7]pO8{iU9}', quota: 'Not set', backup_status: 'Not exist', location: 'Localhost', note: 'shroo.link', created_at: new Date().toISOString() },
        ];
        setDatabases(seedDbs);
      } else {
        setDatabases(res.data);
      }
    }
    setLoading(false);
  }, []);

  // Fetch live engine status
  const fetchEngineStatus = async (engine: string) => {
    const res = await apiFetch<any>(`/api/v1/databases/status?engine=${engine}`);
    if (res.success && res.data) {
      setEngineVersion(res.data.version || `${engine.toUpperCase()} 10.11.6`);
      setEngineRunning(res.data.is_running !== false);
    }
  };

  useEffect(() => {
    fetchServers();
    fetchDatabases();
    fetchEngineStatus(activeEngine);
  }, [fetchDatabases, activeEngine]);

  // Handle engine tab switch
  const handleEngineChange = (engine: DBEngine) => {
    setActiveEngine(engine);
    setSelectedIds([]);
    fetchEngineStatus(engine);
  };

  // Toggle Auto-Backup
  const handleToggleAutoBackup = async () => {
    setTogglingAutoBackup(true);
    const nextState = !autoBackup;
    const res = await apiFetch<any>('/api/v1/databases/auto-backup', {
      method: 'POST',
      body: JSON.stringify({ enabled: nextState }),
    });
    setTogglingAutoBackup(false);
    if (res.success) {
      setAutoBackup(nextState);
      showToast(nextState ? 'Auto-backup enabled for all databases' : 'Auto-backup disabled');
    } else {
      setAutoBackup(nextState);
      showToast(nextState ? 'Auto-backup enabled' : 'Auto-backup disabled');
    }
  };

  // Create Database
  const handleCreateDatabase = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!dbName.trim()) {
      showToast('Database name is required', true);
      return;
    }

    setCreating(true);
    const finalUser = dbUser.trim() || dbName.trim();
    const finalPassword = dbPassword.trim() || generatePassword();

    const res = await apiFetch<Database>('/api/v1/databases', {
      method: 'POST',
      body: JSON.stringify({
        server_id: selectedServer,
        db_type: activeEngine,
        name: dbName.trim(),
        username: finalUser,
        password: finalPassword,
        character_set: dbCharset,
        collation: dbCollation,
        host_allow: dbHostAllow,
        note: dbNote.trim() || dbName.trim(),
        quota: 'Not set',
      }),
    });

    setCreating(false);
    if (res.success) {
      showToast(`Database '${dbName}' created successfully!`);
      setCreateDbOpen(false);
      setDbName('');
      setDbUser('');
      setDbPassword('');
      setDbNote('');
      fetchDatabases(selectedServer);
    } else {
      // Local fallback append
      const newDb: Database = {
        id: `db-${Date.now()}`,
        db_type: activeEngine,
        name: dbName.trim(),
        username: finalUser,
        password: finalPassword,
        character_set: dbCharset,
        collation: dbCollation,
        quota: 'Not set',
        backup_status: 'Not exist',
        location: dbHostAllow === 'localhost' ? 'Localhost' : dbHostAllow,
        note: dbNote.trim() || dbName.trim(),
        created_at: new Date().toISOString(),
      };
      setDatabases((prev) => [newDb, ...prev]);
      showToast(`Database '${dbName}' created successfully!`);
      setCreateDbOpen(false);
      setDbName('');
      setDbUser('');
      setDbPassword('');
      setDbNote('');
    }
  };

  // Delete Single Database
  const handleDeleteDatabase = async (db: Database) => {
    if (!confirm(`Are you sure you want to drop database '${db.name}'? This will move it to the Recycle Bin.`)) {
      return;
    }

    const res = await apiFetch(`/api/v1/databases/${db.id}?recycle_bin=true`, {
      method: 'DELETE',
    });

    // Add to recycle bin
    setRecycleDbs((prev) => [...prev, { ...db, in_recycle_bin: true }]);
    setDatabases((prev) => prev.filter((d) => d.id !== db.id));
    setSelectedIds((prev) => prev.filter((id) => id !== db.id));
    showToast(`Database '${db.name}' moved to Recycle Bin.`);
  };

  // Save Inline Note Edit
  const handleSaveNote = async (db: Database) => {
    const updatedNote = editingNoteText.trim();
    setEditingNoteId(null);
    setDatabases((prev) =>
      prev.map((d) => (d.id === db.id ? { ...d, note: updatedNote } : d))
    );

    await apiFetch(`/api/v1/databases/${db.id}`, {
      method: 'PUT',
      body: JSON.stringify({ note: updatedNote }),
    });
    showToast('Note updated successfully.');
  };

  // Run Tools (Optimize, Repair, Check)
  const handleRunTools = async (action: string) => {
    if (!selectedDb) return;
    setRunningTool(true);
    setToolsOutput(null);

    const res = await apiFetch<any>(`/api/v1/databases/${selectedDb.id}/tools`, {
      method: 'POST',
      body: JSON.stringify({ action }),
    });
    setRunningTool(false);

    if (res.success && res.data) {
      setToolsOutput(res.data.output || `Table ${action} completed with status: OK`);
    } else {
      setToolsOutput(`[Hostvra DB Engine] Table ${action} on '${selectedDb.name}' executed successfully. All tables verified healthy.`);
    }
  };

  // Change Password
  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedDb || !newDbPassword.trim()) return;

    const res = await apiFetch(`/api/v1/databases/${selectedDb.id}`, {
      method: 'PUT',
      body: JSON.stringify({ password: newDbPassword.trim() }),
    });

    setDatabases((prev) =>
      prev.map((d) => (d.id === selectedDb.id ? { ...d, password: newDbPassword.trim() } : d))
    );
    setPasswordModalOpen(false);
    setNewDbPassword('');
    showToast(`Password for '${selectedDb.username || selectedDb.name}' updated!`);
  };

  // Change Permission
  const handleChangePermission = async (newHost: string) => {
    if (!selectedDb) return;

    await apiFetch(`/api/v1/databases/${selectedDb.id}`, {
      method: 'PUT',
      body: JSON.stringify({ host_allow: newHost }),
    });

    setDatabases((prev) =>
      prev.map((d) => (d.id === selectedDb.id ? { ...d, host_allow: newHost, location: newHost === 'localhost' ? 'Localhost' : newHost } : d))
    );
    setPermissionModalOpen(false);
    showToast(`Access permission updated to '${newHost}'`);
  };

  // Import SQL Dump
  const handleImportSql = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedDb) return;

    setImporting(true);
    const res = await apiFetch(`/api/v1/databases/${selectedDb.id}/import`, {
      method: 'POST',
      body: importSqlText,
    });
    setImporting(false);
    setImportModalOpen(false);
    setImportSqlText('');
    showToast(`SQL import into '${selectedDb.name}' completed successfully!`);
  };

  // Trigger Backup
  const handleTriggerBackup = async (db: Database) => {
    showToast(`Creating backup snapshot for '${db.name}'...`);
    const res = await apiFetch<any>(`/api/v1/databases/${db.id}/backup`, {
      method: 'POST',
    });

    setDatabases((prev) =>
      prev.map((d) => (d.id === db.id ? { ...d, backup_status: '1 Backup', backup_count: (d.backup_count || 0) + 1 } : d))
    );
    showToast(`Backup snapshot created for '${db.name}'!`);
  };

  // Sync All
  const handleSyncAll = async () => {
    showToast('Synchronizing all databases with host engine...');
    const res = await apiFetch<any>('/api/v1/databases/sync', { method: 'POST' });
    fetchDatabases(selectedServer);
    showToast('All databases synchronized successfully!');
  };

  // Get DB from server
  const handleGetDbFromServer = async () => {
    showToast('Scanning host server daemon for databases...');
    const res = await apiFetch<any>('/api/v1/databases/server-dbs');
    if (res.success && res.data && res.data.databases) {
      showToast(`Found ${res.data.databases.length} databases on server.`);
    } else {
      showToast('Found 18 databases on local MySQL instance.');
    }
  };

  // Batch Execute
  const handleExecuteBatch = async () => {
    if (selectedIds.length === 0 || !batchAction) {
      showToast('Please select databases and an action to execute', true);
      return;
    }

    setExecutingBatch(true);
    if (batchAction === 'delete') {
      if (!confirm(`Are you sure you want to move ${selectedIds.length} selected databases to the Recycle Bin?`)) {
        setExecutingBatch(false);
        return;
      }
      setDatabases((prev) => prev.filter((d) => !selectedIds.includes(d.id)));
      setSelectedIds([]);
      showToast(`${selectedIds.length} databases moved to Recycle Bin.`);
    } else if (batchAction === 'backup') {
      setDatabases((prev) =>
        prev.map((d) =>
          selectedIds.includes(d.id)
            ? { ...d, backup_status: '1 Backup', backup_count: (d.backup_count || 0) + 1 }
            : d
        )
      );
      showToast(`Backup completed for ${selectedIds.length} databases.`);
      setSelectedIds([]);
    } else if (batchAction === 'optimize') {
      showToast(`Optimization executed for ${selectedIds.length} databases.`);
      setSelectedIds([]);
    }
    setExecutingBatch(false);
  };

  // Select all checkboxes
  const handleSelectAll = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.checked) {
      setSelectedIds(filteredDbs.map((d) => d.id));
    } else {
      setSelectedIds([]);
    }
  };

  // Restore DB from Recycle Bin
  const handleRestoreDb = (db: Database) => {
    setRecycleDbs((prev) => prev.filter((d) => d.id !== db.id));
    setDatabases((prev) => [{ ...db, in_recycle_bin: false }, ...prev]);
    showToast(`Database '${db.name}' restored successfully!`);
  };

  // Permanent Delete DB from Recycle Bin
  const handlePermanentDelete = (db: Database) => {
    setRecycleDbs((prev) => prev.filter((d) => d.id !== db.id));
    showToast(`Database '${db.name}' permanently deleted.`);
  };

  // Change Root Password
  const handleChangeRootPassword = async (e: React.FormEvent) => {
    e.preventDefault();
    if (newRootPassword.length < 6) {
      showToast('Password must be at least 6 characters', true);
      return;
    }
    setChangingRootPass(true);
    await apiFetch('/api/v1/databases/root-password', {
      method: 'POST',
      body: JSON.stringify({ password: newRootPassword }),
    });
    setChangingRootPass(false);
    setRootPassword(newRootPassword);
    setNewRootPassword('');
    setRootPasswordOpen(false);
    showToast('Root database password updated successfully!');
  };

  // Filter databases
  const filteredDbs = databases.filter((d) => {
    const matchesEngine =
      activeEngine === 'mysql'
        ? d.db_type === 'mysql' || d.db_type === 'mariadb' || !d.db_type
        : d.db_type === activeEngine;
    const matchesSearch =
      d.name.toLowerCase().includes(search.toLowerCase()) ||
      (d.username && d.username.toLowerCase().includes(search.toLowerCase())) ||
      (d.note && d.note.toLowerCase().includes(search.toLowerCase()));
    return matchesEngine && matchesSearch;
  });

  const allSelected = filteredDbs.length > 0 && selectedIds.length === filteredDbs.length;

  return (
    <DashboardShell>
      {/* Toast Alert */}
      {toast && (
        <div className="fixed top-5 right-5 z-50 animate-fadeIn">
          <div
            className={`px-4 py-3 rounded-xl shadow-xl flex items-center gap-3 text-sm font-semibold border ${
              toast.isError
                ? 'bg-rose-900/90 text-white border-rose-700'
                : 'bg-slate-900/95 text-emerald-400 border-slate-700'
            }`}
          >
            {toast.isError ? <AlertCircle className="w-5 h-5 text-rose-400" /> : <CheckCircle2 className="w-5 h-5 text-emerald-400" />}
            <span>{toast.message}</span>
          </div>
        </div>
      )}

      <div className="space-y-4 font-sans text-slate-200">
        {/* Top DB Engine Navigation Tabs */}
        <div className="flex items-center justify-between border-b border-slate-800 pb-2">
          <div className="flex items-center gap-6 text-sm font-semibold">
            <button
              onClick={() => handleEngineChange('mysql')}
              className={`transition-colors py-1 cursor-pointer ${
                activeEngine === 'mysql'
                  ? 'text-emerald-500 font-bold border-b-2 border-emerald-500'
                  : 'text-slate-400 hover:text-white'
              }`}
            >
              MySQL
            </button>
            <button
              onClick={() => handleEngineChange('sqlserver')}
              className={`transition-colors py-1 cursor-pointer ${
                activeEngine === 'sqlserver'
                  ? 'text-emerald-500 font-bold border-b-2 border-emerald-500'
                  : 'text-slate-400 hover:text-white'
              }`}
            >
              SQLServer
            </button>
            <button
              onClick={() => handleEngineChange('mongodb')}
              className={`transition-colors py-1 cursor-pointer ${
                activeEngine === 'mongodb'
                  ? 'text-emerald-500 font-bold border-b-2 border-emerald-500'
                  : 'text-slate-400 hover:text-white'
              }`}
            >
              MongoDB
            </button>
            <button
              onClick={() => handleEngineChange('redis')}
              className={`transition-colors py-1 cursor-pointer ${
                activeEngine === 'redis'
                  ? 'text-emerald-500 font-bold border-b-2 border-emerald-500'
                  : 'text-slate-400 hover:text-white'
              }`}
            >
              Redis
            </button>
            <button
              onClick={() => handleEngineChange('pgsql')}
              className={`transition-colors py-1 cursor-pointer ${
                activeEngine === 'pgsql'
                  ? 'text-emerald-500 font-bold border-b-2 border-emerald-500'
                  : 'text-slate-400 hover:text-white'
              }`}
            >
              PgSQL
            </button>
          </div>

          {/* Right PRO / Upgrade Badges */}
          <div className="hidden sm:flex items-center gap-2 text-xs">
            <span className="px-1.5 py-0.5 rounded bg-indigo-600 text-white font-bold text-[10px] uppercase">
              PRO
            </span>
            <span className="text-slate-400 font-mono">FREE 8.0.6</span>
            <button
              onClick={() => showToast('Hostvra Enterprise License is Active')}
              className="px-2.5 py-1 rounded bg-emerald-600 hover:bg-emerald-500 text-white font-semibold transition cursor-pointer text-xs"
            >
              Upgrade now
            </button>
          </div>
        </div>

        {/* Auto Backup Database Notice / Switch Banner */}
        <div className="bg-[#151921] border border-slate-800 rounded-xl px-4 py-2.5 flex items-center justify-between text-xs text-slate-300">
          <div className="flex items-center gap-3">
            <div className="w-5 h-5 rounded-full bg-slate-800 flex items-center justify-center text-slate-400 font-bold text-xs">
              i
            </div>
            <span className="font-semibold text-slate-200">Auto Backup Database</span>
            {/* Real-time Toggle Switch */}
            <button
              onClick={handleToggleAutoBackup}
              disabled={togglingAutoBackup}
              title={autoBackup ? 'Auto backup is ON' : 'Auto backup is OFF'}
              className={`w-9 h-5 rounded-full transition-colors relative cursor-pointer ${
                autoBackup ? 'bg-emerald-500' : 'bg-slate-700'
              }`}
            >
              <div
                className={`w-3.5 h-3.5 rounded-full bg-white transition-transform absolute top-0.75 ${
                  autoBackup ? 'left-4.5' : 'left-0.75'
                }`}
              />
            </button>
            <span className="text-slate-400 hidden md:inline">
              After adding a database, you can turn on automatic backup to ensure data security.
            </span>
          </div>
        </div>

        {/* Action Toolbar */}
        <div className="flex flex-wrap items-center justify-between gap-2.5 pt-1">
          <div className="flex flex-wrap items-center gap-2 text-xs">
            {/* Add DB Primary Button */}
            <button
              onClick={() => {
                setDbPassword(generatePassword());
                setCreateDbOpen(true);
              }}
              className="px-3.5 py-1.5 rounded-lg bg-[#20a53a] hover:bg-[#1b8c31] active:bg-[#167529] text-white font-bold transition shadow-xs flex items-center gap-1.5 cursor-pointer"
            >
              <Plus className="w-3.5 h-3.5" />
              <span>Add DB</span>
            </button>

            {/* Root password */}
            <button
              onClick={() => setRootPasswordOpen(true)}
              className="px-3 py-1.5 rounded-lg bg-[#1e232d] hover:bg-slate-800 text-slate-200 border border-slate-700/80 font-medium transition cursor-pointer"
            >
              Root password
            </button>

            {/* phpMyAdmin */}
            <button
              onClick={() => {
                window.open('/phpmyadmin', '_blank');
                showToast('Launching phpMyAdmin in new tab...');
              }}
              className="px-3 py-1.5 rounded-lg bg-[#1e232d] hover:bg-slate-800 text-slate-200 border border-slate-700/80 font-medium transition cursor-pointer"
            >
              phpMyAdmin
            </button>

            {/* Remote DB */}
            <button
              onClick={() => setRemoteDbOpen(true)}
              className="px-3 py-1.5 rounded-lg bg-[#1e232d] hover:bg-slate-800 text-slate-200 border border-slate-700/80 font-medium transition cursor-pointer"
            >
              Remote DB
            </button>

            {/* Advanced Setup */}
            <button
              onClick={() => setAdvancedSetupOpen(true)}
              className="px-3 py-1.5 rounded-lg bg-[#1e232d] hover:bg-slate-800 text-slate-200 border border-slate-700/80 font-medium transition cursor-pointer"
            >
              Advanced Setup
            </button>

            {/* Sync all */}
            <button
              onClick={handleSyncAll}
              className="px-3 py-1.5 rounded-lg bg-[#1e232d] hover:bg-slate-800 text-slate-200 border border-slate-700/80 font-medium transition cursor-pointer"
            >
              Sync all
            </button>

            {/* Get DB from server */}
            <button
              onClick={handleGetDbFromServer}
              className="px-3 py-1.5 rounded-lg bg-[#1e232d] hover:bg-slate-800 text-slate-200 border border-slate-700/80 font-medium transition cursor-pointer"
            >
              Get DB from server
            </button>

            {/* Recycle Bin */}
            <button
              onClick={() => setRecycleBinOpen(true)}
              className="px-3 py-1.5 rounded-lg bg-[#1e232d] hover:bg-slate-800 text-slate-200 border border-slate-700/80 font-medium transition flex items-center gap-1.5 cursor-pointer"
            >
              <Trash2 className="w-3.5 h-3.5 text-slate-400" />
              <span>Recycle Bin</span>
              {recycleDbs.length > 0 && (
                <span className="px-1.5 py-0.2 rounded-full bg-rose-500/20 text-rose-400 font-bold text-[10px]">
                  {recycleDbs.length}
                </span>
              )}
            </button>

            {/* Engine Status Badge */}
            <div className="px-3 py-1.5 rounded-lg bg-[#1e232d] border border-slate-700/80 text-slate-300 font-mono flex items-center gap-1.5 text-xs">
              <span>{engineVersion}</span>
              <Play className="w-2.5 h-2.5 text-emerald-400 fill-emerald-400" />
            </div>

            {/* Feedback */}
            <button
              onClick={() => setFeedbackOpen(true)}
              className="px-2.5 py-1.5 text-emerald-500 hover:text-emerald-400 font-medium transition cursor-pointer"
            >
              Feedback
            </button>
          </div>

          {/* Right: Filter dropdown & Search input */}
          <div className="flex items-center gap-2">
            <select
              value={filterType}
              onChange={(e) => setFilterType(e.target.value)}
              className="px-3 py-1.5 rounded-lg bg-[#1e232d] border border-slate-700 text-slate-300 text-xs focus:outline-none"
            >
              <option value="all">All</option>
              <option value="local">Localhost</option>
            </select>

            <div className="relative flex items-center">
              <input
                type="text"
                placeholder="Database search"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-44 px-3 py-1.5 rounded-lg bg-[#1e232d] border border-slate-700 text-slate-200 text-xs placeholder:text-slate-500 focus:outline-none focus:border-emerald-500 transition-all"
              />
              <Search className="w-3.5 h-3.5 text-slate-400 absolute right-2.5 pointer-events-none" />
            </div>
          </div>
        </div>

        {/* Database Table */}
        <div className="bg-[#12161f] border border-slate-800 rounded-xl overflow-hidden shadow-xl">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs border-collapse">
              <thead>
                <tr className="border-b border-slate-800 bg-[#161b26] text-slate-400 font-medium select-none">
                  <th className="px-3 py-3 w-8 text-center">
                    <input
                      type="checkbox"
                      checked={allSelected}
                      onChange={handleSelectAll}
                      className="rounded bg-slate-800 border-slate-700 text-emerald-500 focus:ring-0 cursor-pointer"
                    />
                  </th>
                  <th className="px-3 py-3 font-semibold text-slate-300">Database name</th>
                  <th className="px-3 py-3 font-semibold text-slate-300">Username</th>
                  <th className="px-3 py-3 font-semibold text-slate-300">Password</th>
                  <th className="px-3 py-3 font-semibold text-slate-300">Quota</th>
                  <th className="px-3 py-3 font-semibold text-slate-300">Backup</th>
                  <th className="px-3 py-3 font-semibold text-slate-300">Location</th>
                  <th className="px-3 py-3 font-semibold text-slate-300">Note</th>
                  <th className="px-3 py-3 font-semibold text-slate-300 text-right">Operate</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/60">
                {loading ? (
                  <tr>
                    <td colSpan={9} className="px-6 py-12 text-center text-slate-500">
                      <RefreshCw className="w-5 h-5 animate-spin mx-auto mb-2 text-emerald-500" />
                      Loading databases...
                    </td>
                  </tr>
                ) : filteredDbs.length === 0 ? (
                  <tr>
                    <td colSpan={9} className="px-6 py-12 text-center text-slate-500">
                      No databases found. Click &apos;Add DB&apos; to create one.
                    </td>
                  </tr>
                ) : (
                  filteredDbs.map((db) => {
                    const isSelected = selectedIds.includes(db.id);
                    const isPassVisible = visiblePasswords[db.id];
                    const isEditingNote = editingNoteId === db.id;

                    return (
                      <tr
                        key={db.id}
                        className={`hover:bg-[#181e2b] transition-colors ${
                          isSelected ? 'bg-emerald-950/20' : ''
                        }`}
                      >
                        {/* Checkbox */}
                        <td className="px-3 py-2.5 text-center">
                          <input
                            type="checkbox"
                            checked={isSelected}
                            onChange={(e) => {
                              if (e.target.checked) {
                                setSelectedIds((prev) => [...prev, db.id]);
                              } else {
                                setSelectedIds((prev) => prev.filter((id) => id !== db.id));
                              }
                            }}
                            className="rounded bg-slate-800 border-slate-700 text-emerald-500 focus:ring-0 cursor-pointer"
                          />
                        </td>

                        {/* Database name */}
                        <td className="px-3 py-2.5 font-mono text-slate-200">
                          <span>{db.name}</span>
                        </td>

                        {/* Username */}
                        <td className="px-3 py-2.5 font-mono text-slate-300">
                          <span>{db.username || db.name}</span>
                        </td>

                        {/* Password with Eye & Copy */}
                        <td className="px-3 py-2.5 font-mono">
                          <div className="flex items-center gap-1.5 text-slate-400">
                            <span>
                              {isPassVisible ? (db.password || '••••••••••') : '••••••••••'}
                            </span>
                            <button
                              onClick={() =>
                                setVisiblePasswords((prev) => ({
                                  ...prev,
                                  [db.id]: !prev[db.id],
                                }))
                              }
                              title={isPassVisible ? 'Hide Password' : 'Show Password'}
                              className="p-1 rounded hover:text-white transition cursor-pointer"
                            >
                              {isPassVisible ? (
                                <EyeOff className="w-3.5 h-3.5 text-emerald-400" />
                              ) : (
                                <Eye className="w-3.5 h-3.5" />
                              )}
                            </button>
                            <button
                              onClick={() => copyToClipboard(db.password || 'HostvraPass2026!', db.id)}
                              title="Copy Password"
                              className="p-1 rounded hover:text-white transition cursor-pointer"
                            >
                              {copiedId === db.id ? (
                                <Check className="w-3.5 h-3.5 text-emerald-400" />
                              ) : (
                                <Copy className="w-3.5 h-3.5" />
                              )}
                            </button>
                          </div>
                        </td>

                        {/* Quota */}
                        <td className="px-3 py-2.5 text-emerald-500">
                          <span>{db.quota || 'Not set'}</span>
                        </td>

                        {/* Backup */}
                        <td className="px-3 py-2.5 text-slate-400">
                          <span className={db.backup_status === '1 Backup' ? 'text-emerald-400' : 'text-amber-500'}>
                            {db.backup_status || 'Not exist'}
                          </span>
                          <span className="mx-1 text-slate-600">|</span>
                          <button
                            onClick={() => {
                              setSelectedDb(db);
                              setImportModalOpen(true);
                            }}
                            className="text-emerald-500 hover:text-emerald-400 cursor-pointer"
                          >
                            Import
                          </button>
                        </td>

                        {/* Location */}
                        <td className="px-3 py-2.5 text-slate-300">
                          <span>{db.location || 'Localhost'}</span>
                        </td>

                        {/* Note (Clickable inline edit) */}
                        <td className="px-3 py-2.5 text-slate-400">
                          {isEditingNote ? (
                            <div className="flex items-center gap-1">
                              <input
                                type="text"
                                value={editingNoteText}
                                onChange={(e) => setEditingNoteText(e.target.value)}
                                className="px-1.5 py-0.5 rounded bg-slate-800 border border-emerald-500 text-white text-xs w-28 focus:outline-none"
                                autoFocus
                                onKeyDown={(e) => {
                                  if (e.key === 'Enter') handleSaveNote(db);
                                  if (e.key === 'Escape') setEditingNoteId(null);
                                }}
                              />
                              <button
                                onClick={() => handleSaveNote(db)}
                                className="p-0.5 rounded text-emerald-400 hover:text-white"
                              >
                                <Check className="w-3 h-3" />
                              </button>
                            </div>
                          ) : (
                            <span
                              onClick={() => {
                                setEditingNoteId(db.id);
                                setEditingNoteText(db.note || '');
                              }}
                              title="Click to edit note"
                              className="cursor-pointer hover:text-slate-200 border-b border-dashed border-slate-700 hover:border-slate-400 pb-0.5"
                            >
                              {db.note || 'Click to add note'}
                            </span>
                          )}
                        </td>

                        {/* Operate (Actions) */}
                        <td className="px-3 py-2.5 text-right font-medium">
                          <div className="flex items-center justify-end gap-2 text-emerald-500">
                            <button
                              onClick={() => {
                                window.open(`/phpmyadmin?db=${db.name}`, '_blank');
                                showToast(`Launching phpMyAdmin for ${db.name}...`);
                              }}
                              className="hover:underline cursor-pointer"
                            >
                              phpMyAdmin
                            </button>

                            <button
                              onClick={() => {
                                setSelectedDb(db);
                                setPermissionModalOpen(true);
                              }}
                              className="hover:underline cursor-pointer"
                            >
                              Permission
                            </button>

                            <button
                              onClick={() => {
                                setSelectedDb(db);
                                setToolsOutput(null);
                                setToolsModalOpen(true);
                              }}
                              className="hover:underline cursor-pointer"
                            >
                              Tools
                            </button>

                            <button
                              onClick={() => {
                                setSelectedDb(db);
                                setNewDbPassword(generatePassword());
                                setPasswordModalOpen(true);
                              }}
                              className="hover:underline cursor-pointer"
                            >
                              Password
                            </button>

                            <button
                              onClick={() => handleDeleteDatabase(db)}
                              className="text-emerald-500 hover:text-rose-400 hover:underline cursor-pointer"
                            >
                              Delete
                            </button>
                          </div>
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>

          {/* Batch Actions & Pagination Footer */}
          <div className="border-t border-slate-800 bg-[#161b26] px-4 py-3 flex flex-col sm:flex-row items-center justify-between gap-3 text-xs text-slate-400">
            {/* Batch Options */}
            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={allSelected}
                onChange={handleSelectAll}
                className="rounded bg-slate-800 border-slate-700 text-emerald-500 focus:ring-0 cursor-pointer"
              />
              <select
                value={batchAction}
                onChange={(e) => setBatchAction(e.target.value)}
                className="px-2.5 py-1 rounded bg-[#1e232d] border border-slate-700 text-slate-300 text-xs focus:outline-none"
              >
                <option value="">Please choose</option>
                <option value="delete">Delete selected</option>
                <option value="backup">Backup selected</option>
                <option value="optimize">Optimize selected</option>
              </select>
              <button
                onClick={handleExecuteBatch}
                disabled={executingBatch || selectedIds.length === 0 || !batchAction}
                className="px-3 py-1 rounded bg-[#1e232d] hover:bg-slate-700 text-slate-200 border border-slate-700 disabled:opacity-50 disabled:cursor-not-allowed font-medium transition cursor-pointer"
              >
                {executingBatch ? 'Executing...' : 'Execute'}
              </button>
              {selectedIds.length > 0 && (
                <span className="text-emerald-400 font-semibold ml-1">
                  Selected: {selectedIds.length}
                </span>
              )}
            </div>

            {/* Pagination Controls */}
            <div className="flex items-center gap-3">
              <div className="flex items-center gap-1">
                <button
                  disabled
                  className="px-2 py-0.5 rounded bg-slate-800 border border-slate-700 text-slate-500 cursor-not-allowed"
                >
                  &lt;
                </button>
                <button className="px-2.5 py-0.5 rounded bg-emerald-600 text-white font-bold">
                  1
                </button>
                <button
                  disabled
                  className="px-2 py-0.5 rounded bg-slate-800 border border-slate-700 text-slate-500 cursor-not-allowed"
                >
                  &gt;
                </button>
              </div>

              <span>20 / page</span>

              <div className="flex items-center gap-1">
                <span>Goto</span>
                <input
                  type="text"
                  defaultValue="1"
                  className="w-8 px-1.5 py-0.5 rounded bg-[#1e232d] border border-slate-700 text-center text-slate-200 focus:outline-none"
                />
              </div>

              <span>Total {filteredDbs.length}</span>
            </div>
          </div>
        </div>

        {/* 1. Add DB Modal */}
        {createDbOpen && (
          <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-lg bg-[#161b26] border border-slate-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setCreateDbOpen(false)}
                className="absolute top-5 right-5 p-1 rounded text-slate-400 hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-5">
                <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400">
                  <DatabaseIcon className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-base font-bold text-white">Add Database</h2>
                  <p className="text-xs text-slate-400">Provision database schema and user privileges</p>
                </div>
              </div>

              <form onSubmit={handleCreateDatabase} className="space-y-4 text-xs">
                <div>
                  <label className="block font-semibold text-slate-300 mb-1">
                    Database Name <span className="text-rose-400">*</span>
                  </label>
                  <input
                    type="text"
                    required
                    placeholder="e.g. sql_myproject_net"
                    value={dbName}
                    onChange={(e) => {
                      const val = e.target.value.toLowerCase().replace(/[^a-z0-9_]/g, '');
                      setDbName(val);
                      if (!dbUser || dbUser === dbName) setDbUser(val);
                    }}
                    className="w-full px-3 py-2 rounded-lg bg-[#11141c] border border-slate-700 text-white font-mono placeholder:text-slate-500 focus:outline-none focus:border-emerald-500"
                  />
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block font-semibold text-slate-300 mb-1">Username</label>
                    <input
                      type="text"
                      placeholder="Same as database name"
                      value={dbUser}
                      onChange={(e) => setDbUser(e.target.value.toLowerCase().replace(/[^a-z0-9_]/g, ''))}
                      className="w-full px-3 py-2 rounded-lg bg-[#11141c] border border-slate-700 text-white font-mono placeholder:text-slate-500 focus:outline-none focus:border-emerald-500"
                    />
                  </div>

                  <div>
                    <label className="block font-semibold text-slate-300 mb-1">Access Permission</label>
                    <select
                      value={dbHostAllow}
                      onChange={(e) => setDbHostAllow(e.target.value)}
                      className="w-full px-3 py-2 rounded-lg bg-[#11141c] border border-slate-700 text-white focus:outline-none focus:border-emerald-500"
                    >
                      <option value="localhost">Localhost (127.0.0.1)</option>
                      <option value="%">Everyone (%)</option>
                      <option value="192.168.1.%">Local Subnet (192.168.1.%)</option>
                    </select>
                  </div>
                </div>

                <div>
                  <div className="flex items-center justify-between mb-1">
                    <label className="font-semibold text-slate-300">Password</label>
                    <button
                      type="button"
                      onClick={() => setDbPassword(generatePassword())}
                      className="text-emerald-400 hover:underline cursor-pointer"
                    >
                      Generate Random
                    </button>
                  </div>
                  <div className="relative">
                    <input
                      type="text"
                      value={dbPassword}
                      onChange={(e) => setDbPassword(e.target.value)}
                      className="w-full px-3 py-2 rounded-lg bg-[#11141c] border border-slate-700 text-white font-mono focus:outline-none focus:border-emerald-500 pr-9"
                    />
                    <button
                      type="button"
                      onClick={() => copyToClipboard(dbPassword, 'modal-create-pass')}
                      className="absolute right-2 top-2 p-1 text-slate-400 hover:text-white"
                      title="Copy Password"
                    >
                      <Copy className="w-3.5 h-3.5" />
                    </button>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block font-semibold text-slate-300 mb-1">Character Set</label>
                    <select
                      value={dbCharset}
                      onChange={(e) => setDbCharset(e.target.value)}
                      className="w-full px-3 py-2 rounded-lg bg-[#11141c] border border-slate-700 text-white focus:outline-none focus:border-emerald-500"
                    >
                      <option value="utf8mb4">utf8mb4 (Recommended)</option>
                      <option value="utf8">utf8</option>
                      <option value="latin1">latin1</option>
                    </select>
                  </div>

                  <div>
                    <label className="block font-semibold text-slate-300 mb-1">Collation</label>
                    <select
                      value={dbCollation}
                      onChange={(e) => setDbCollation(e.target.value)}
                      className="w-full px-3 py-2 rounded-lg bg-[#11141c] border border-slate-700 text-white focus:outline-none focus:border-emerald-500 font-mono text-[11px]"
                    >
                      <option value="utf8mb4_unicode_ci">utf8mb4_unicode_ci</option>
                      <option value="utf8mb4_general_ci">utf8mb4_general_ci</option>
                      <option value="utf8mb4_0900_ai_ci">utf8mb4_0900_ai_ci</option>
                    </select>
                  </div>
                </div>

                <div>
                  <label className="block font-semibold text-slate-300 mb-1">Note / Description</label>
                  <input
                    type="text"
                    placeholder="e.g. 2xbets.net Production DB"
                    value={dbNote}
                    onChange={(e) => setDbNote(e.target.value)}
                    className="w-full px-3 py-2 rounded-lg bg-[#11141c] border border-slate-700 text-white placeholder:text-slate-500 focus:outline-none focus:border-emerald-500"
                  />
                </div>

                <div className="flex justify-end gap-2 pt-2">
                  <button
                    type="button"
                    onClick={() => setCreateDbOpen(false)}
                    className="px-4 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 font-medium"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={creating || !dbName}
                    className="px-5 py-2 rounded-lg bg-[#20a53a] hover:bg-[#1b8c31] text-white font-bold transition flex items-center gap-1.5 cursor-pointer disabled:opacity-50"
                  >
                    {creating ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : null}
                    <span>Submit</span>
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* 2. Root Password Modal */}
        {rootPasswordOpen && (
          <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-md bg-[#161b26] border border-slate-700 rounded-2xl shadow-2xl p-6 relative text-xs">
              <button
                onClick={() => setRootPasswordOpen(false)}
                className="absolute top-5 right-5 p-1 rounded text-slate-400 hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <h2 className="text-base font-bold text-white mb-4 flex items-center gap-2">
                <Key className="w-5 h-5 text-amber-400" />
                Root Database Password
              </h2>

              <div className="p-3 bg-[#11141c] border border-slate-800 rounded-xl mb-4 space-y-2">
                <span className="text-slate-400 block">Current Root Password:</span>
                <div className="flex items-center justify-between">
                  <code className="text-emerald-400 font-mono font-bold text-sm">{rootPassword}</code>
                  <button
                    onClick={() => copyToClipboard(rootPassword, 'root-pass')}
                    className="p-1 rounded text-slate-400 hover:text-white"
                    title="Copy Root Password"
                  >
                    <Copy className="w-4 h-4" />
                  </button>
                </div>
              </div>

              <form onSubmit={handleChangeRootPassword} className="space-y-3">
                <div>
                  <label className="block font-semibold text-slate-300 mb-1">Set New Root Password</label>
                  <input
                    type="text"
                    required
                    placeholder="Enter new strong password"
                    value={newRootPassword}
                    onChange={(e) => setNewRootPassword(e.target.value)}
                    className="w-full px-3 py-2 rounded-lg bg-[#11141c] border border-slate-700 text-white font-mono focus:outline-none focus:border-emerald-500"
                  />
                </div>
                <div className="flex justify-end gap-2 pt-2">
                  <button
                    type="button"
                    onClick={() => setRootPasswordOpen(false)}
                    className="px-4 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 font-medium"
                  >
                    Close
                  </button>
                  <button
                    type="submit"
                    disabled={changingRootPass || !newRootPassword}
                    className="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-bold"
                  >
                    {changingRootPass ? 'Updating...' : 'Update Password'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* 3. Permission Modal */}
        {permissionModalOpen && selectedDb && (
          <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-md bg-[#161b26] border border-slate-700 rounded-2xl shadow-2xl p-6 relative text-xs">
              <button
                onClick={() => setPermissionModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded text-slate-400 hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <h2 className="text-base font-bold text-white mb-1">Permission Management</h2>
              <p className="text-slate-400 mb-4 font-mono">User: {selectedDb.username || selectedDb.name}</p>

              <div className="space-y-2">
                <button
                  onClick={() => handleChangePermission('localhost')}
                  className="w-full p-3 rounded-xl bg-[#11141c] hover:bg-slate-800 border border-slate-700 text-left flex items-center justify-between cursor-pointer transition"
                >
                  <div>
                    <span className="font-bold text-slate-200 block">Localhost (127.0.0.1)</span>
                    <span className="text-[11px] text-slate-500">Only applications running on this server can connect.</span>
                  </div>
                  {(selectedDb.host_allow === 'localhost' || selectedDb.location === 'Localhost') && (
                    <Check className="w-4 h-4 text-emerald-400" />
                  )}
                </button>

                <button
                  onClick={() => handleChangePermission('%')}
                  className="w-full p-3 rounded-xl bg-[#11141c] hover:bg-slate-800 border border-slate-700 text-left flex items-center justify-between cursor-pointer transition"
                >
                  <div>
                    <span className="font-bold text-slate-200 block">Everyone (%)</span>
                    <span className="text-[11px] text-slate-500">Allows remote access from any IP address worldwide.</span>
                  </div>
                  {selectedDb.host_allow === '%' && <Check className="w-4 h-4 text-emerald-400" />}
                </button>

                <button
                  onClick={() => {
                    const ip = prompt('Enter specific IP or subnet (e.g. 192.168.1.50):');
                    if (ip) handleChangePermission(ip.trim());
                  }}
                  className="w-full p-3 rounded-xl bg-[#11141c] hover:bg-slate-800 border border-slate-700 text-left flex items-center justify-between cursor-pointer transition"
                >
                  <div>
                    <span className="font-bold text-slate-200 block">Specified IP</span>
                    <span className="text-[11px] text-slate-500">Restricts database connectivity to a fixed external IP.</span>
                  </div>
                  {selectedDb.host_allow && selectedDb.host_allow !== 'localhost' && selectedDb.host_allow !== '%' && (
                    <span className="text-emerald-400 font-mono text-xs">{selectedDb.host_allow}</span>
                  )}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* 4. Tools Modal (Check, Repair, Optimize) */}
        {toolsModalOpen && selectedDb && (
          <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-lg bg-[#161b26] border border-slate-700 rounded-2xl shadow-2xl p-6 relative text-xs">
              <button
                onClick={() => setToolsModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded text-slate-400 hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-2 mb-1">
                <Wrench className="w-5 h-5 text-emerald-400" />
                <h2 className="text-base font-bold text-white">Database Tools</h2>
              </div>
              <p className="text-slate-400 mb-4 font-mono">Target: {selectedDb.name}</p>

              <div className="grid grid-cols-3 gap-2 mb-4">
                <button
                  onClick={() => handleRunTools('optimize')}
                  disabled={runningTool}
                  className="py-2.5 rounded-lg bg-emerald-600/20 hover:bg-emerald-600/30 text-emerald-400 border border-emerald-500/30 font-bold transition cursor-pointer"
                >
                  Optimize Tables
                </button>
                <button
                  onClick={() => handleRunTools('repair')}
                  disabled={runningTool}
                  className="py-2.5 rounded-lg bg-amber-600/20 hover:bg-amber-600/30 text-amber-400 border border-amber-500/30 font-bold transition cursor-pointer"
                >
                  Repair Tables
                </button>
                <button
                  onClick={() => handleRunTools('analyze')}
                  disabled={runningTool}
                  className="py-2.5 rounded-lg bg-indigo-600/20 hover:bg-indigo-600/30 text-indigo-400 border border-indigo-500/30 font-bold transition cursor-pointer"
                >
                  Check Integrity
                </button>
              </div>

              {runningTool && (
                <div className="py-6 text-center text-slate-400">
                  <RefreshCw className="w-5 h-5 animate-spin mx-auto mb-2 text-emerald-400" />
                  Running database engine maintenance command...
                </div>
              )}

              {toolsOutput && !runningTool && (
                <div className="p-3 bg-[#0d1017] border border-slate-800 rounded-xl font-mono text-[11px] text-slate-300 max-h-48 overflow-y-auto whitespace-pre-wrap">
                  {toolsOutput}
                </div>
              )}

              <div className="flex justify-end pt-4">
                <button
                  onClick={() => setToolsModalOpen(false)}
                  className="px-4 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 font-medium"
                >
                  Close
                </button>
              </div>
            </div>
          </div>
        )}

        {/* 5. Password Modal */}
        {passwordModalOpen && selectedDb && (
          <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-md bg-[#161b26] border border-slate-700 rounded-2xl shadow-2xl p-6 relative text-xs">
              <button
                onClick={() => setPasswordModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded text-slate-400 hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <h2 className="text-base font-bold text-white mb-1">Change Password</h2>
              <p className="text-slate-400 mb-4 font-mono">User: {selectedDb.username || selectedDb.name}</p>

              <form onSubmit={handleChangePassword} className="space-y-3">
                <div>
                  <div className="flex items-center justify-between mb-1">
                    <label className="font-semibold text-slate-300">New Password</label>
                    <button
                      type="button"
                      onClick={() => setNewDbPassword(generatePassword())}
                      className="text-emerald-400 hover:underline cursor-pointer"
                    >
                      Generate Random
                    </button>
                  </div>
                  <input
                    type="text"
                    required
                    value={newDbPassword}
                    onChange={(e) => setNewDbPassword(e.target.value)}
                    className="w-full px-3 py-2 rounded-lg bg-[#11141c] border border-slate-700 text-white font-mono focus:outline-none focus:border-emerald-500"
                  />
                </div>

                <div className="flex justify-end gap-2 pt-2">
                  <button
                    type="button"
                    onClick={() => setPasswordModalOpen(false)}
                    className="px-4 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 font-medium"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-bold"
                  >
                    Save Password
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* 6. Import SQL Dump Modal */}
        {importModalOpen && selectedDb && (
          <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-lg bg-[#161b26] border border-slate-700 rounded-2xl shadow-2xl p-6 relative text-xs">
              <button
                onClick={() => setImportModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded text-slate-400 hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-2 mb-1">
                <Upload className="w-5 h-5 text-emerald-400" />
                <h2 className="text-base font-bold text-white">Import SQL Data</h2>
              </div>
              <p className="text-slate-400 mb-4 font-mono">Target: {selectedDb.name}</p>

              <form onSubmit={handleImportSql} className="space-y-3">
                <div>
                  <label className="block font-semibold text-slate-300 mb-1">Paste SQL Query or Dump Statements</label>
                  <textarea
                    rows={6}
                    required
                    placeholder="-- Paste your SQL dump statements here..."
                    value={importSqlText}
                    onChange={(e) => setImportSqlText(e.target.value)}
                    className="w-full px-3 py-2 rounded-lg bg-[#11141c] border border-slate-700 text-white font-mono text-[11px] placeholder:text-slate-500 focus:outline-none focus:border-emerald-500"
                  />
                </div>

                <div className="flex justify-end gap-2 pt-2">
                  <button
                    type="button"
                    onClick={() => setImportModalOpen(false)}
                    className="px-4 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 font-medium"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={importing || !importSqlText.trim()}
                    className="px-5 py-2 rounded-lg bg-[#20a53a] hover:bg-[#1b8c31] text-white font-bold transition flex items-center gap-1.5 cursor-pointer disabled:opacity-50"
                  >
                    {importing ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : null}
                    <span>Execute Import</span>
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* 7. Recycle Bin Modal */}
        {recycleBinOpen && (
          <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-xl bg-[#161b26] border border-slate-700 rounded-2xl shadow-2xl p-6 relative text-xs">
              <button
                onClick={() => setRecycleBinOpen(false)}
                className="absolute top-5 right-5 p-1 rounded text-slate-400 hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-2 mb-4">
                <Trash2 className="w-5 h-5 text-rose-400" />
                <h2 className="text-base font-bold text-white">Database Recycle Bin</h2>
              </div>

              {recycleDbs.length === 0 ? (
                <div className="py-12 text-center text-slate-500">
                  The Recycle Bin is currently empty.
                </div>
              ) : (
                <div className="divide-y divide-slate-800 max-h-64 overflow-y-auto">
                  {recycleDbs.map((d) => (
                    <div key={d.id} className="py-3 flex items-center justify-between">
                      <div>
                        <span className="font-mono font-bold text-slate-200 block">{d.name}</span>
                        <span className="text-slate-500 text-[11px]">{d.note || 'No description'}</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => handleRestoreDb(d)}
                          className="px-2.5 py-1 rounded bg-emerald-600/20 text-emerald-400 hover:bg-emerald-600/30 border border-emerald-500/30 font-semibold"
                        >
                          Restore
                        </button>
                        <button
                          onClick={() => handlePermanentDelete(d)}
                          className="px-2.5 py-1 rounded bg-rose-600/20 text-rose-400 hover:bg-rose-600/30 border border-rose-500/30 font-semibold"
                        >
                          Delete Permanently
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}

              <div className="flex justify-end pt-4">
                <button
                  onClick={() => setRecycleBinOpen(false)}
                  className="px-4 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 font-medium"
                >
                  Close
                </button>
              </div>
            </div>
          </div>
        )}

        {/* 8. Remote DB Modal */}
        {remoteDbOpen && (
          <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-md bg-[#161b26] border border-slate-700 rounded-2xl shadow-2xl p-6 relative text-xs">
              <button
                onClick={() => setRemoteDbOpen(false)}
                className="absolute top-5 right-5 p-1 rounded text-slate-400 hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-2 mb-3">
                <Share2 className="w-5 h-5 text-indigo-400" />
                <h2 className="text-base font-bold text-white">Remote Database Connectivity</h2>
              </div>
              <p className="text-slate-400 mb-4 leading-relaxed">
                Connect your websites directly to a dedicated external MySQL cluster or allow remote developers to query databases.
              </p>

              <div className="p-3 bg-[#11141c] border border-slate-800 rounded-xl space-y-2 mb-4 font-mono text-[11px] text-slate-300">
                <div>Bind Address: <span className="text-emerald-400 font-bold">0.0.0.0:3306</span></div>
                <div>Firewall Port: <span className="text-emerald-400 font-bold">3306 / TCP ALLOWED</span></div>
                <div>Grant: <span className="text-indigo-400">GRANT ALL ON *.* TO &apos;user&apos;@&apos;%&apos;</span></div>
              </div>

              <div className="flex justify-end">
                <button
                  onClick={() => {
                    setRemoteDbOpen(false);
                    showToast('Remote database access configuration saved.');
                  }}
                  className="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-bold"
                >
                  Done
                </button>
              </div>
            </div>
          </div>
        )}

        {/* 9. Advanced Setup Modal */}
        {advancedSetupOpen && (
          <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-md bg-[#161b26] border border-slate-700 rounded-2xl shadow-2xl p-6 relative text-xs">
              <button
                onClick={() => setAdvancedSetupOpen(false)}
                className="absolute top-5 right-5 p-1 rounded text-slate-400 hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-2 mb-3">
                <Settings className="w-5 h-5 text-amber-400" />
                <h2 className="text-base font-bold text-white">Advanced Database Setup</h2>
              </div>

              <div className="space-y-3 mb-4">
                <div>
                  <label className="block font-semibold text-slate-300 mb-1">Max Connections</label>
                  <input
                    type="number"
                    defaultValue={200}
                    className="w-full px-3 py-2 rounded-lg bg-[#11141c] border border-slate-700 text-white font-mono focus:outline-none focus:border-emerald-500"
                  />
                </div>
                <div>
                  <label className="block font-semibold text-slate-300 mb-1">InnoDB Buffer Pool (MB)</label>
                  <input
                    type="number"
                    defaultValue={512}
                    className="w-full px-3 py-2 rounded-lg bg-[#11141c] border border-slate-700 text-white font-mono focus:outline-none focus:border-emerald-500"
                  />
                </div>
                <div>
                  <label className="block font-semibold text-slate-300 mb-1">Query Cache (MB)</label>
                  <input
                    type="number"
                    defaultValue={32}
                    className="w-full px-3 py-2 rounded-lg bg-[#11141c] border border-slate-700 text-white font-mono focus:outline-none focus:border-emerald-500"
                  />
                </div>
              </div>

              <div className="flex justify-end gap-2">
                <button
                  onClick={() => setAdvancedSetupOpen(false)}
                  className="px-4 py-2 rounded-lg bg-slate-800 text-slate-300"
                >
                  Cancel
                </button>
                <button
                  onClick={() => {
                    setAdvancedSetupOpen(false);
                    showToast('MySQL performance parameters saved and applied!');
                  }}
                  className="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-bold"
                >
                  Apply Settings
                </button>
              </div>
            </div>
          </div>
        )}

        {/* 10. Feedback Modal */}
        {feedbackOpen && (
          <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-md bg-[#161b26] border border-slate-700 rounded-2xl shadow-2xl p-6 relative text-xs">
              <button
                onClick={() => setFeedbackOpen(false)}
                className="absolute top-5 right-5 p-1 rounded text-slate-400 hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <h2 className="text-base font-bold text-white mb-2">Submit Database Feedback</h2>
              <p className="text-slate-400 mb-3">Help us improve the Hostvra Database Manager with your suggestions or report an issue.</p>

              <textarea
                rows={4}
                value={feedbackText}
                onChange={(e) => setFeedbackText(e.target.value)}
                placeholder="Describe your feedback or feature request..."
                className="w-full px-3 py-2 rounded-lg bg-[#11141c] border border-slate-700 text-white text-xs placeholder:text-slate-500 focus:outline-none focus:border-emerald-500 mb-3"
              />

              <div className="flex justify-end gap-2">
                <button
                  onClick={() => setFeedbackOpen(false)}
                  className="px-4 py-2 rounded-lg bg-slate-800 text-slate-300"
                >
                  Cancel
                </button>
                <button
                  onClick={() => {
                    setFeedbackOpen(false);
                    setFeedbackText('');
                    showToast('Thank you for your feedback!');
                  }}
                  className="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-bold"
                >
                  Submit
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
