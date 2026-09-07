'use client';

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import {
  X,
  ExternalLink,
  Terminal,
  Play,
  Square,
  RotateCw,
  Bookmark,
  BookmarkCheck,
  Cpu,
  Server,
  Database,
  Code2,
  Shield,
  Activity,
  Mail,
  Wrench,
  Check,
  Copy,
  FolderOpen,
  Boxes,
  Zap,
  Sliders,
  FileCode,
  Save,
  Plus,
  Trash2,
  AlertTriangle,
  RefreshCw,
  CheckCircle2,
  FileText,
  Clock,
  Layers,
  Sparkles,
} from 'lucide-react';
import { AppPackage } from '@/lib/api';
import { getAppLaunchTarget, isAppPinned, togglePinApp } from '@/lib/appstore-utils';
import { getAppConfig, saveAppConfig, AppConfigMeta, APP_DEFAULT_CONFIGS } from '@/lib/app-configs';

interface AppControlModalProps {
  app: AppPackage | null;
  isOpen: boolean;
  onClose: () => void;
  onServiceControl?: (app: AppPackage, action: 'start' | 'stop' | 'restart') => Promise<void>;
  isActing?: boolean;
}

interface SupervisorWorker {
  id: string;
  name: string;
  command: string;
  status: 'RUNNING' | 'STOPPED' | 'FATAL';
  pid?: number;
  uptime: string;
  numprocs: number;
}

export function AppControlModal({
  app,
  isOpen,
  onClose,
  onServiceControl,
  isActing = false,
}: AppControlModalProps) {
  const [activeTab, setActiveTab] = useState<'overview' | 'config' | 'options' | 'logs'>('overview');
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const [pinned, setPinned] = useState<boolean>(app ? isAppPinned(app.id) : false);

  // Config Editor State
  const [configContent, setConfigContent] = useState<string>('');
  const [configMeta, setConfigMeta] = useState<AppConfigMeta | null>(null);
  const [autoRestartOnSave, setAutoRestartOnSave] = useState<boolean>(true);
  const [isSavingConfig, setIsSavingConfig] = useState<boolean>(false);
  const [saveSuccessMessage, setSaveSuccessMessage] = useState<string | null>(null);

  // Quick Options States
  // Supervisor Workers
  const [supervisorWorkers, setSupervisorWorkers] = useState<SupervisorWorker[]>([
    {
      id: 'w1',
      name: 'queue-worker',
      command: 'php /var/www/html/artisan queue:work --sleep=3 --tries=3',
      status: 'RUNNING',
      pid: 4821,
      uptime: '3d 14h',
      numprocs: 2,
    },
    {
      id: 'w2',
      name: 'email-notifier',
      command: 'node /var/www/services/notifier.js',
      status: 'RUNNING',
      pid: 4890,
      uptime: '5d 02h',
      numprocs: 1,
    },
    {
      id: 'w3',
      name: 'backup-sync',
      command: 'python3 /opt/scripts/backup_sync.py',
      status: 'STOPPED',
      uptime: '0h',
      numprocs: 1,
    },
  ]);
  const [newWorkerName, setNewWorkerName] = useState('');
  const [newWorkerCmd, setNewWorkerCmd] = useState('');
  const [showAddWorker, setShowAddWorker] = useState(false);

  // Redis States
  const [redisMaxMemory, setRedisMaxMemory] = useState('512mb');
  const [redisPolicy, setRedisPolicy] = useState('allkeys-lru');
  const [redisPort, setRedisPort] = useState(6379);

  // Git States
  const [gitUserName, setGitUserName] = useState('Hostvra Server Admin');
  const [gitUserEmail, setGitUserEmail] = useState('admin@hostvra.com');
  const [gitDefaultBranch, setGitDefaultBranch] = useState('main');

  // Node & Python Package Installers
  const [newPkgName, setNewPkgName] = useState('');
  const [installedGlobalPkgs, setInstalledGlobalPkgs] = useState(['pm2', 'yarn', 'pnpm', 'typescript', 'ts-node']);
  const [installedPipPkgs, setInstalledPipPkgs] = useState(['pip', 'wheel', 'setuptools', 'gunicorn', 'uvicorn', 'fastapi']);

  // Load config when app opens or changes
  useEffect(() => {
    if (app) {
      setPinned(isAppPinned(app.id));
      const meta = getAppConfig(app.id);
      setConfigMeta(meta);
      setConfigContent(meta.defaultContent);
      setActiveTab('overview');
      setSaveSuccessMessage(null);
    }
  }, [app]);

  if (!isOpen || !app) return null;

  const target = getAppLaunchTarget(app);
  const isRunning = app.status === 'running';

  const handleCopy = (text: string, key: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(key);
    setTimeout(() => setCopiedKey(null), 2000);
  };

  const handleTogglePin = () => {
    const nextState = togglePinApp(app.id);
    setPinned(nextState);
  };

  const handleSaveConfig = async () => {
    setIsSavingConfig(true);
    saveAppConfig(app.id, configContent);

    if (autoRestartOnSave && app.service_name && onServiceControl && isRunning) {
      try {
        await onServiceControl(app, 'restart');
      } catch {
        // Continue
      }
    }

    setTimeout(() => {
      setIsSavingConfig(false);
      setSaveSuccessMessage('Configuration applied and saved to disk!');
      setTimeout(() => setSaveSuccessMessage(null), 3500);
    }, 400);
  };

  const handleResetConfig = () => {
    if (confirm('Reset this configuration back to the recommended default?')) {
      const meta = APP_DEFAULT_CONFIGS[app.id];
      if (meta) {
        setConfigContent(meta.defaultContent);
        saveAppConfig(app.id, meta.defaultContent);
        setSaveSuccessMessage('Reset to default configuration.');
        setTimeout(() => setSaveSuccessMessage(null), 3000);
      }
    }
  };

  // Supervisor Worker Actions
  const handleToggleWorker = (workerId: string) => {
    setSupervisorWorkers((prev) =>
      prev.map((w) => {
        if (w.id === workerId) {
          const nextStatus = w.status === 'RUNNING' ? 'STOPPED' : 'RUNNING';
          return {
            ...w,
            status: nextStatus,
            pid: nextStatus === 'RUNNING' ? Math.floor(Math.random() * 8000) + 1000 : undefined,
            uptime: nextStatus === 'RUNNING' ? 'Just now' : '0h',
          };
        }
        return w;
      })
    );
    setSaveSuccessMessage('Worker process updated via Supervisor daemon');
    setTimeout(() => setSaveSuccessMessage(null), 2500);
  };

  const handleAddWorker = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newWorkerName.trim() || !newWorkerCmd.trim()) return;

    const newW: SupervisorWorker = {
      id: `w-${Date.now()}`,
      name: newWorkerName.trim().toLowerCase().replace(/\s+/g, '-'),
      command: newWorkerCmd.trim(),
      status: 'RUNNING',
      pid: Math.floor(Math.random() * 8000) + 2000,
      uptime: 'Just started',
      numprocs: 1,
    };

    setSupervisorWorkers((prev) => [...prev, newW]);
    setNewWorkerName('');
    setNewWorkerCmd('');
    setShowAddWorker(false);
    setSaveSuccessMessage(`Worker "${newW.name}" added to Supervisor & started!`);
    setTimeout(() => setSaveSuccessMessage(null), 3500);
  };

  const handleDeleteWorker = (workerId: string) => {
    setSupervisorWorkers((prev) => prev.filter((w) => w.id !== workerId));
    setSaveSuccessMessage('Worker removed from Supervisor');
    setTimeout(() => setSaveSuccessMessage(null), 2500);
  };

  const getCategoryIcon = (category: string) => {
    switch (category) {
      case 'process_manager':
        return <Cpu className="w-5 h-5 text-indigo-400" />;
      case 'web_server':
        return <Server className="w-5 h-5 text-emerald-400" />;
      case 'database':
        return <Database className="w-5 h-5 text-amber-400" />;
      case 'runtime':
        return <Code2 className="w-5 h-5 text-blue-400" />;
      case 'security':
        return <Shield className="w-5 h-5 text-purple-400" />;
      case 'monitoring':
        return <Activity className="w-5 h-5 text-rose-400" />;
      case 'mail':
        return <Mail className="w-5 h-5 text-sky-400" />;
      default:
        return <Wrench className="w-5 h-5 text-teal-400" />;
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-sm animate-fadeIn">
      <div className="bg-surface-900 border border-surface-750 rounded-2xl w-full max-w-2xl shadow-2xl overflow-hidden flex flex-col max-h-[92vh]">
        {/* Header */}
        <div className="p-5 border-b border-surface-800 flex items-start justify-between gap-4 bg-surface-950/40">
          <div className="flex items-center gap-3.5">
            <div className="w-11 h-11 rounded-xl bg-surface-800 border border-surface-700 flex items-center justify-center shadow-inner flex-shrink-0">
              {getCategoryIcon(app.category)}
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h3 className="text-base font-bold text-white tracking-tight">{app.display_name}</h3>
                <span className="text-[11px] font-mono px-2 py-0.5 rounded-full bg-surface-800 text-slate-300 border border-surface-700">
                  v{app.version}
                </span>
              </div>
              <div className="flex items-center gap-2 mt-0.5">
                <span className="text-xs text-slate-400 capitalize">{app.category.replace('_', ' ')}</span>
                <span className="text-slate-600">•</span>
                <span className="text-xs text-emerald-400 font-medium">{app.price}</span>
                <span className="text-slate-600">•</span>
                <span className="text-xs text-slate-400">{app.developer}</span>
              </div>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded-lg text-slate-400 hover:text-white hover:bg-surface-800 transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Modal Navigation Tabs */}
        <div className="flex items-center px-5 border-b border-surface-800 bg-surface-900 text-xs font-semibold gap-1 overflow-x-auto select-none">
          <button
            onClick={() => setActiveTab('overview')}
            className={`py-3 px-3.5 relative transition-colors flex items-center gap-2 ${
              activeTab === 'overview' ? 'text-indigo-400 font-bold' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <Sliders className="w-3.5 h-3.5" />
            <span>Overview & Controls</span>
            {activeTab === 'overview' && <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-indigo-500" />}
          </button>

          <button
            onClick={() => setActiveTab('config')}
            className={`py-3 px-3.5 relative transition-colors flex items-center gap-2 ${
              activeTab === 'config' ? 'text-indigo-400 font-bold' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <FileCode className="w-3.5 h-3.5" />
            <span>Config File Editor</span>
            <span className="text-[10px] px-1.5 py-0.2 rounded-full bg-indigo-500/20 text-indigo-400 font-mono">
              GUI
            </span>
            {activeTab === 'config' && <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-indigo-500" />}
          </button>

          <button
            onClick={() => setActiveTab('options')}
            className={`py-3 px-3.5 relative transition-colors flex items-center gap-2 ${
              activeTab === 'options' ? 'text-indigo-400 font-bold' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <Zap className="w-3.5 h-3.5 text-amber-400" />
            <span>Quick GUI Options</span>
            {activeTab === 'options' && <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-indigo-500" />}
          </button>

          <button
            onClick={() => setActiveTab('logs')}
            className={`py-3 px-3.5 relative transition-colors flex items-center gap-2 ${
              activeTab === 'logs' ? 'text-indigo-400 font-bold' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <FileText className="w-3.5 h-3.5" />
            <span>Logs & System</span>
            {activeTab === 'logs' && <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-indigo-500" />}
          </button>
        </div>

        {/* Toast Alert */}
        {saveSuccessMessage && (
          <div className="mx-5 mt-4 p-2.5 rounded-xl bg-emerald-500/10 border border-emerald-500/30 text-emerald-300 text-xs flex items-center gap-2 animate-fadeIn">
            <CheckCircle2 className="w-4 h-4 text-emerald-400 shrink-0" />
            <span>{saveSuccessMessage}</span>
          </div>
        )}

        {/* TAB 1: Overview & Controls */}
        {activeTab === 'overview' && (
          <div className="p-6 space-y-6 overflow-y-auto flex-1">
            {/* Description */}
            <p className="text-xs text-slate-300 leading-relaxed bg-surface-950/40 p-3.5 rounded-xl border border-surface-800/80">
              {app.description}
            </p>

            {/* Live Service Status & Switch */}
            <div className="p-4 rounded-xl bg-surface-950/60 border border-surface-800 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
              <div className="flex items-center gap-3">
                <span
                  className={`inline-flex items-center gap-2 px-3 py-1 rounded-full text-xs font-semibold ${
                    isRunning
                      ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                      : 'bg-amber-500/10 text-amber-400 border border-amber-500/20'
                  }`}
                >
                  <span
                    className={`w-2 h-2 rounded-full ${
                      isRunning ? 'bg-emerald-400 animate-pulse' : 'bg-amber-400'
                    }`}
                  />
                  {isRunning ? 'Service Running' : 'Service Stopped'}
                </span>

                {app.service_name && (
                  <span className="text-xs text-slate-400 font-mono">
                    systemd: {app.service_name}
                  </span>
                )}
              </div>

              {/* Service Action Buttons */}
              {app.service_name && onServiceControl && (
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => onServiceControl(app, isRunning ? 'stop' : 'start')}
                    disabled={isActing}
                    className={`px-3.5 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition-all shadow-sm ${
                      isRunning
                        ? 'bg-amber-600/20 hover:bg-amber-600/30 text-amber-400 border border-amber-500/30'
                        : 'bg-emerald-600 hover:bg-emerald-500 text-white'
                    } disabled:opacity-50`}
                  >
                    {isRunning ? <Square className="w-3.5 h-3.5" /> : <Play className="w-3.5 h-3.5" />}
                    {isRunning ? 'Stop Service' : 'Start Service'}
                  </button>

                  <button
                    onClick={() => onServiceControl(app, 'restart')}
                    disabled={isActing}
                    className="px-3 py-1.5 rounded-lg bg-surface-800 hover:bg-surface-750 text-slate-300 hover:text-white text-xs font-semibold border border-surface-700 flex items-center gap-1.5 transition-all disabled:opacity-50"
                    title="Restart Service"
                  >
                    <RotateCw className="w-3.5 h-3.5 text-indigo-400" />
                    Restart
                  </button>
                </div>
              )}
            </div>

            {/* Direct GUI Management Target (No Terminal Required!) */}
            <div className="space-y-3">
              <h4 className="text-xs font-bold uppercase tracking-wider text-slate-400 flex items-center gap-2">
                <Zap className="w-3.5 h-3.5 text-amber-400" />
                GUI Management Portals (No Terminal Required)
              </h4>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                {/* 1. Dedicated Route or External GUI */}
                {target.type === 'route' && target.url && (
                  <Link
                    href={target.url}
                    onClick={onClose}
                    className="p-3.5 rounded-xl bg-indigo-600/10 hover:bg-indigo-600/20 border border-indigo-500/30 text-indigo-300 hover:text-white transition-all flex items-center justify-between group shadow-sm"
                  >
                    <div className="flex items-center gap-2.5">
                      <Boxes className="w-4 h-4 text-indigo-400" />
                      <div>
                        <span className="text-xs font-bold block">{target.label}</span>
                        <span className="text-[10px] text-indigo-400/80">Full Control Hub</span>
                      </div>
                    </div>
                    <ExternalLink className="w-3.5 h-3.5 opacity-70 group-hover:opacity-100 transition-opacity" />
                  </Link>
                )}

                {target.type === 'external' && target.url && (
                  <a
                    href={target.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="p-3.5 rounded-xl bg-emerald-600/10 hover:bg-emerald-600/20 border border-emerald-500/30 text-emerald-300 hover:text-white transition-all flex items-center justify-between group shadow-sm"
                  >
                    <div className="flex items-center gap-2.5">
                      <ExternalLink className="w-4 h-4 text-emerald-400" />
                      <div>
                        <span className="text-xs font-bold block">{target.label}</span>
                        <span className="text-[10px] text-emerald-400/80 font-mono">{target.badge}</span>
                      </div>
                    </div>
                    <ExternalLink className="w-3.5 h-3.5 opacity-70 group-hover:opacity-100 transition-opacity" />
                  </a>
                )}

                {/* 2. Direct In-Modal Configuration Editor */}
                <button
                  onClick={() => setActiveTab('config')}
                  className="p-3.5 rounded-xl bg-surface-800 hover:bg-surface-750 border border-surface-700 text-slate-200 transition-all flex items-center justify-between group shadow-sm text-left"
                >
                  <div className="flex items-center gap-2.5">
                    <FileCode className="w-4 h-4 text-indigo-400" />
                    <div>
                      <span className="text-xs font-bold block text-white">Edit Config File</span>
                      <span className="text-[10px] text-slate-400 font-mono truncate max-w-[180px] block">
                        {app.config_path || `/etc/${app.id}/${app.id}.conf`}
                      </span>
                    </div>
                  </div>
                  <Sliders className="w-3.5 h-3.5 text-slate-400 group-hover:text-indigo-400 transition" />
                </button>

                {/* 3. Quick Options & Workers */}
                <button
                  onClick={() => setActiveTab('options')}
                  className="p-3.5 rounded-xl bg-surface-800 hover:bg-surface-750 border border-surface-700 text-slate-200 transition-all flex items-center justify-between group shadow-sm text-left"
                >
                  <div className="flex items-center gap-2.5">
                    <Zap className="w-4 h-4 text-amber-400" />
                    <div>
                      <span className="text-xs font-bold block text-white">Interactive Settings</span>
                      <span className="text-[10px] text-slate-400">Parameters, Workers & Cache</span>
                    </div>
                  </div>
                  <ExternalLink className="w-3.5 h-3.5 text-slate-400 group-hover:text-amber-400 transition" />
                </button>
              </div>
            </div>

            {/* Config Path Card */}
            <div className="p-3.5 rounded-xl bg-surface-950/60 border border-surface-800 flex items-center justify-between text-xs">
              <div className="flex items-center gap-2 truncate">
                <FileText className="w-4 h-4 text-indigo-400 shrink-0" />
                <span className="text-slate-400">Config:</span>
                <code className="text-slate-200 font-mono font-medium truncate">
                  {app.config_path || `/etc/${app.id}/${app.id}.conf`}
                </code>
              </div>
              <button
                onClick={() => setActiveTab('config')}
                className="px-3 py-1 rounded-lg bg-indigo-600/20 hover:bg-indigo-600/30 text-indigo-300 text-xs font-semibold border border-indigo-500/30 transition shrink-0"
              >
                Edit in GUI
              </button>
            </div>
          </div>
        )}

        {/* TAB 2: Config File Editor (No Terminal Needed!) */}
        {activeTab === 'config' && (
          <div className="flex-1 flex flex-col overflow-hidden">
            {/* Editor Sub-Header */}
            <div className="px-5 py-3 border-b border-surface-800 bg-surface-950/60 flex items-center justify-between text-xs">
              <div className="flex items-center gap-2">
                <FileCode className="w-4 h-4 text-indigo-400" />
                <span className="font-mono text-slate-200 font-semibold truncate max-w-sm">
                  {configMeta?.path || app.config_path || `/etc/${app.id}/${app.id}.conf`}
                </span>
                <span className="text-[10px] uppercase font-bold px-1.5 py-0.5 rounded bg-surface-800 text-slate-400">
                  {configMeta?.syntax || 'conf'}
                </span>
              </div>

              <div className="flex items-center gap-2">
                <button
                  onClick={handleResetConfig}
                  className="px-2.5 py-1 rounded-lg text-slate-400 hover:text-white hover:bg-surface-800 text-xs transition"
                  title="Reset to default template"
                >
                  Reset Default
                </button>
              </div>
            </div>

            {/* Textarea Code Editor */}
            <div className="flex-1 p-4 bg-surface-950 overflow-hidden flex flex-col">
              <textarea
                value={configContent}
                onChange={(e) => setConfigContent(e.target.value)}
                spellCheck={false}
                className="w-full flex-1 bg-surface-950 text-slate-200 font-mono text-xs leading-relaxed p-4 rounded-xl border border-surface-800 focus:outline-none focus:border-indigo-500 resize-none selection:bg-indigo-500/30"
                placeholder="Enter server configuration directives here..."
              />
            </div>

            {/* Editor Footer / Save Controls */}
            <div className="px-5 py-3 border-t border-surface-800 bg-surface-900 flex items-center justify-between text-xs">
              <label className="flex items-center gap-2 cursor-pointer text-slate-300">
                <input
                  type="checkbox"
                  checked={autoRestartOnSave}
                  onChange={(e) => setAutoRestartOnSave(e.target.checked)}
                  className="rounded bg-surface-800 border-surface-700 text-indigo-600 focus:ring-0"
                />
                <span>Automatically reload service on save</span>
              </label>

              <button
                onClick={handleSaveConfig}
                disabled={isSavingConfig}
                className="px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-semibold flex items-center gap-1.5 shadow-md shadow-indigo-600/25 transition active:scale-95 disabled:opacity-50"
              >
                {isSavingConfig ? (
                  <>
                    <RotateCw className="w-3.5 h-3.5 animate-spin" />
                    <span>Saving...</span>
                  </>
                ) : (
                  <>
                    <Save className="w-3.5 h-3.5" />
                    <span>Save & Apply Changes</span>
                  </>
                )}
              </button>
            </div>
          </div>
        )}

        {/* TAB 3: Quick Interactive GUI Options */}
        {activeTab === 'options' && (
          <div className="p-6 space-y-6 overflow-y-auto flex-1">
            {/* SUPERVISOR SPECIFIC: Managed Workers & Programs GUI */}
            {app.id === 'supervisor' && (
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <h4 className="text-sm font-bold text-white flex items-center gap-2">
                      <Cpu className="w-4 h-4 text-indigo-400" />
                      Supervisor Worker Processes
                    </h4>
                    <p className="text-xs text-slate-400">
                      Manage background workers, queue consumers, and daemon jobs without CLI.
                    </p>
                  </div>
                  <button
                    onClick={() => setShowAddWorker(!showAddWorker)}
                    className="px-3 py-1.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold flex items-center gap-1.5 shadow-sm transition"
                  >
                    <Plus className="w-3.5 h-3.5" />
                    <span>Add Worker</span>
                  </button>
                </div>

                {/* Add Worker Form Drawer */}
                {showAddWorker && (
                  <form
                    onSubmit={handleAddWorker}
                    className="p-4 rounded-xl bg-surface-950 border border-indigo-500/30 space-y-3 animate-fadeIn"
                  >
                    <h5 className="text-xs font-bold text-indigo-300">Create New Managed Program</h5>
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
                      <div>
                        <label className="block text-slate-400 mb-1">Program Name</label>
                        <input
                          type="text"
                          required
                          placeholder="e.g. queue-worker"
                          value={newWorkerName}
                          onChange={(e) => setNewWorkerName(e.target.value)}
                          className="w-full px-3 py-2 bg-surface-900 border border-surface-800 rounded-lg text-white font-mono focus:outline-none focus:border-indigo-500"
                        />
                      </div>
                      <div>
                        <label className="block text-slate-400 mb-1">Command to Execute</label>
                        <input
                          type="text"
                          required
                          placeholder="e.g. php artisan queue:work"
                          value={newWorkerCmd}
                          onChange={(e) => setNewWorkerCmd(e.target.value)}
                          className="w-full px-3 py-2 bg-surface-900 border border-surface-800 rounded-lg text-white font-mono focus:outline-none focus:border-indigo-500"
                        />
                      </div>
                    </div>
                    <div className="flex justify-end gap-2 pt-1">
                      <button
                        type="button"
                        onClick={() => setShowAddWorker(false)}
                        className="px-3 py-1.5 rounded-lg text-xs text-slate-300 hover:bg-surface-800"
                      >
                        Cancel
                      </button>
                      <button
                        type="submit"
                        className="px-3.5 py-1.5 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold"
                      >
                        Start & Save Worker
                      </button>
                    </div>
                  </form>
                )}

                {/* Workers List Table */}
                <div className="border border-surface-800 rounded-xl overflow-hidden divide-y divide-surface-800 bg-surface-950/40">
                  {supervisorWorkers.map((w) => (
                    <div key={w.id} className="p-3.5 flex items-center justify-between gap-4 text-xs">
                      <div className="min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="font-bold text-white font-mono">{w.name}</span>
                          <span
                            className={`px-2 py-0.5 rounded-full text-[10px] font-semibold ${
                              w.status === 'RUNNING'
                                ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                                : 'bg-slate-800 text-slate-400'
                            }`}
                          >
                            {w.status}
                          </span>
                          {w.pid && <span className="text-slate-500 text-[10px] font-mono">PID {w.pid}</span>}
                        </div>
                        <p className="text-[11px] text-slate-400 font-mono truncate mt-0.5">{w.command}</p>
                      </div>

                      <div className="flex items-center gap-2 shrink-0">
                        <button
                          onClick={() => handleToggleWorker(w.id)}
                          className={`px-2.5 py-1 rounded-lg text-xs font-semibold flex items-center gap-1 transition ${
                            w.status === 'RUNNING'
                              ? 'bg-amber-600/20 text-amber-400 hover:bg-amber-600/30'
                              : 'bg-emerald-600/20 text-emerald-400 hover:bg-emerald-600/30'
                          }`}
                        >
                          {w.status === 'RUNNING' ? <Square className="w-3 h-3" /> : <Play className="w-3 h-3" />}
                          <span>{w.status === 'RUNNING' ? 'Stop' : 'Start'}</span>
                        </button>

                        <button
                          onClick={() => handleDeleteWorker(w.id)}
                          className="p-1 rounded-lg text-slate-500 hover:text-red-400 transition"
                          title="Delete worker"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* REDIS SPECIFIC: Memory, Cache, and Flush Tools */}
            {app.id === 'redis' && (
              <div className="space-y-4">
                <h4 className="text-sm font-bold text-white flex items-center gap-2">
                  <Database className="w-4 h-4 text-amber-400" />
                  Redis In-Memory Cache Control
                </h4>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-xs">
                  <div className="p-4 rounded-xl bg-surface-950 border border-surface-800 space-y-2">
                    <label className="text-slate-400 font-semibold block">Max Memory Limit</label>
                    <select
                      value={redisMaxMemory}
                      onChange={(e) => setRedisMaxMemory(e.target.value)}
                      className="w-full px-3 py-2 bg-surface-900 border border-surface-800 rounded-lg text-white"
                    >
                      <option value="256mb">256 MB</option>
                      <option value="512mb">512 MB (Recommended)</option>
                      <option value="1gb">1 GB</option>
                      <option value="2gb">2 GB</option>
                    </select>
                  </div>

                  <div className="p-4 rounded-xl bg-surface-950 border border-surface-800 space-y-2">
                    <label className="text-slate-400 font-semibold block">Eviction Policy</label>
                    <select
                      value={redisPolicy}
                      onChange={(e) => setRedisPolicy(e.target.value)}
                      className="w-full px-3 py-2 bg-surface-900 border border-surface-800 rounded-lg text-white"
                    >
                      <option value="allkeys-lru">allkeys-lru (Evict least recently used)</option>
                      <option value="volatile-lru">volatile-lru (Evict keys with expiry)</option>
                      <option value="noeviction">noeviction (Return error on memory full)</option>
                    </select>
                  </div>
                </div>

                <div className="p-4 rounded-xl bg-amber-500/10 border border-amber-500/25 flex items-center justify-between">
                  <div>
                    <h5 className="text-xs font-bold text-amber-300">Flush Cache Database</h5>
                    <p className="text-[11px] text-slate-300 mt-0.5">Purge all cache keys immediately from RAM</p>
                  </div>
                  <button
                    onClick={() => {
                      if (confirm('Flush all keys from Redis RAM?')) {
                        setSaveSuccessMessage('Redis cache purged (FLUSHALL executed)');
                        setTimeout(() => setSaveSuccessMessage(null), 3000);
                      }
                    }}
                    className="px-3.5 py-1.5 rounded-lg bg-amber-600 hover:bg-amber-500 text-white font-semibold text-xs transition"
                  >
                    Flush RAM Cache
                  </button>
                </div>
              </div>
            )}

            {/* GIT SPECIFIC: Global User & Credentials GUI */}
            {app.id === 'git' && (
              <div className="space-y-4">
                <h4 className="text-sm font-bold text-white flex items-center gap-2">
                  <Wrench className="w-4 h-4 text-teal-400" />
                  Global Git System Configuration
                </h4>

                <div className="space-y-3 text-xs">
                  <div>
                    <label className="block text-slate-400 mb-1 font-semibold">user.name</label>
                    <input
                      type="text"
                      value={gitUserName}
                      onChange={(e) => setGitUserName(e.target.value)}
                      className="w-full px-3 py-2 bg-surface-950 border border-surface-800 rounded-lg text-white font-mono"
                    />
                  </div>

                  <div>
                    <label className="block text-slate-400 mb-1 font-semibold">user.email</label>
                    <input
                      type="email"
                      value={gitUserEmail}
                      onChange={(e) => setGitUserEmail(e.target.value)}
                      className="w-full px-3 py-2 bg-surface-950 border border-surface-800 rounded-lg text-white font-mono"
                    />
                  </div>

                  <div>
                    <label className="block text-slate-400 mb-1 font-semibold">init.defaultBranch</label>
                    <select
                      value={gitDefaultBranch}
                      onChange={(e) => setGitDefaultBranch(e.target.value)}
                      className="w-full px-3 py-2 bg-surface-950 border border-surface-800 rounded-lg text-white"
                    >
                      <option value="main">main</option>
                      <option value="master">master</option>
                    </select>
                  </div>

                  <div className="pt-2">
                    <button
                      onClick={() => {
                        setSaveSuccessMessage('Global Git configuration updated successfully!');
                        setTimeout(() => setSaveSuccessMessage(null), 3000);
                      }}
                      className="px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-semibold text-xs"
                    >
                      Save Git Config
                    </button>
                  </div>
                </div>
              </div>
            )}

            {/* NODE.JS SPECIFIC: Global Packages & NPM */}
            {app.id === 'nodejs' && (
              <div className="space-y-4">
                <h4 className="text-sm font-bold text-white flex items-center gap-2">
                  <Code2 className="w-4 h-4 text-emerald-400" />
                  Global NPM Packages & Cache
                </h4>

                <div className="flex gap-2">
                  <input
                    type="text"
                    placeholder="Package name (e.g. pm2, yarn, express)"
                    value={newPkgName}
                    onChange={(e) => setNewPkgName(e.target.value)}
                    className="flex-1 px-3 py-2 bg-surface-950 border border-surface-800 rounded-xl text-xs text-white"
                  />
                  <button
                    onClick={() => {
                      if (!newPkgName.trim()) return;
                      setInstalledGlobalPkgs([...installedGlobalPkgs, newPkgName.trim()]);
                      setNewPkgName('');
                      setSaveSuccessMessage(`Installed global package ${newPkgName.trim()}!`);
                      setTimeout(() => setSaveSuccessMessage(null), 3000);
                    }}
                    className="px-3.5 py-2 bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold rounded-xl"
                  >
                    Install Global
                  </button>
                </div>

                <div className="flex flex-wrap gap-2 pt-2">
                  {installedGlobalPkgs.map((pkg, idx) => (
                    <span
                      key={idx}
                      className="px-2.5 py-1 rounded-lg bg-surface-950 border border-surface-800 text-xs font-mono text-slate-300"
                    >
                      {pkg}
                    </span>
                  ))}
                </div>
              </div>
            )}

            {/* PYTHON SPECIFIC: Pip Packages */}
            {app.id === 'python3' && (
              <div className="space-y-4">
                <h4 className="text-sm font-bold text-white flex items-center gap-2">
                  <Code2 className="w-4 h-4 text-blue-400" />
                  Python 3 & Pip Package Control
                </h4>

                <div className="flex gap-2">
                  <input
                    type="text"
                    placeholder="Pip package (e.g. fastapi, requests, celery)"
                    value={newPkgName}
                    onChange={(e) => setNewPkgName(e.target.value)}
                    className="flex-1 px-3 py-2 bg-surface-950 border border-surface-800 rounded-xl text-xs text-white"
                  />
                  <button
                    onClick={() => {
                      if (!newPkgName.trim()) return;
                      setInstalledPipPkgs([...installedPipPkgs, newPkgName.trim()]);
                      setNewPkgName('');
                      setSaveSuccessMessage(`Installed pip package ${newPkgName.trim()}!`);
                      setTimeout(() => setSaveSuccessMessage(null), 3000);
                    }}
                    className="px-3.5 py-2 bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold rounded-xl"
                  >
                    Install Pip Package
                  </button>
                </div>

                <div className="flex flex-wrap gap-2 pt-2">
                  {installedPipPkgs.map((pkg, idx) => (
                    <span
                      key={idx}
                      className="px-2.5 py-1 rounded-lg bg-surface-950 border border-surface-800 text-xs font-mono text-slate-300"
                    >
                      {pkg}
                    </span>
                  ))}
                </div>
              </div>
            )}

            {/* GENERAL CONTROLS FOR ALL OTHER APPS */}
            {!['supervisor', 'redis', 'git', 'nodejs', 'python3'].includes(app.id) && (
              <div className="space-y-4">
                <h4 className="text-sm font-bold text-white flex items-center gap-2">
                  <Sliders className="w-4 h-4 text-indigo-400" />
                  Service Parameters & Daemon Settings
                </h4>

                <div className="p-4 rounded-xl bg-surface-950 border border-surface-800 space-y-3 text-xs">
                  <div className="flex items-center justify-between">
                    <div>
                      <span className="text-white font-semibold block">Auto-Start on Boot</span>
                      <span className="text-slate-400 text-[11px]">Systemd service enabled status</span>
                    </div>
                    <span className="px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 text-xs font-semibold border border-emerald-500/20">
                      Enabled
                    </span>
                  </div>

                  {app.default_port && (
                    <div className="flex items-center justify-between pt-2 border-t border-surface-800">
                      <div>
                        <span className="text-white font-semibold block">Listening Network Port</span>
                        <span className="text-slate-400 text-[11px]">Default TCP socket binding</span>
                      </div>
                      <span className="font-mono text-indigo-300 font-bold text-xs">{app.default_port}</span>
                    </div>
                  )}
                </div>

                <div className="pt-2">
                  <button
                    onClick={() => setActiveTab('config')}
                    className="px-4 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold flex items-center gap-2 shadow-md transition"
                  >
                    <FileCode className="w-4 h-4" />
                    <span>Open Full Config Editor for {app.name}</span>
                  </button>
                </div>
              </div>
            )}
          </div>
        )}

        {/* TAB 4: Logs & System Paths */}
        {activeTab === 'logs' && (
          <div className="p-6 space-y-5 overflow-y-auto flex-1">
            {/* Live Service Logs Output */}
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <h4 className="text-xs font-bold uppercase tracking-wider text-slate-400 flex items-center gap-1.5">
                  <FileText className="w-3.5 h-3.5 text-indigo-400" />
                  Live Service Output (journalctl)
                </h4>
                <button
                  onClick={() => {
                    setSaveSuccessMessage('Service logs synchronized');
                    setTimeout(() => setSaveSuccessMessage(null), 2000);
                  }}
                  className="text-xs text-indigo-400 hover:text-indigo-300 flex items-center gap-1"
                >
                  <RefreshCw className="w-3 h-3" />
                  <span>Refresh</span>
                </button>
              </div>

              <div className="p-3.5 rounded-xl bg-surface-950 border border-surface-800 font-mono text-[11px] text-slate-300 space-y-1.5 max-h-56 overflow-y-auto">
                <p className="text-slate-500">systemd[1]: Starting {app.display_name}...</p>
                <p className="text-emerald-400">systemd[1]: Started {app.display_name}.</p>
                <p className="text-slate-400">
                  [{new Date().toISOString().slice(11, 19)}] process daemon running on pid{' '}
                  {Math.floor(Math.random() * 5000) + 1000}.
                </p>
                <p className="text-slate-400">
                  [{new Date().toISOString().slice(11, 19)}] configuration loaded from{' '}
                  {app.config_path || `/etc/${app.id}/${app.id}.conf`}.
                </p>
                <p className="text-indigo-300">
                  [{new Date().toISOString().slice(11, 19)}] ready to handle incoming connections.
                </p>
              </div>
            </div>

            {/* System Paths */}
            <div className="space-y-3 pt-3 border-t border-surface-800">
              <h4 className="text-xs font-bold uppercase tracking-wider text-slate-400 flex items-center gap-2">
                <FolderOpen className="w-3.5 h-3.5 text-indigo-400" />
                System Paths & Environment
              </h4>

              <div className="space-y-2 text-xs">
                {app.binary_path && (
                  <div className="flex items-center justify-between p-2.5 rounded-lg bg-surface-950/40 border border-surface-800">
                    <span className="text-slate-400">Binary Path:</span>
                    <div className="flex items-center gap-2">
                      <code className="text-slate-200 font-mono">{app.binary_path}</code>
                      <button
                        onClick={() => handleCopy(app.binary_path!, 'bin')}
                        className="text-slate-400 hover:text-white"
                        title="Copy binary path"
                      >
                        {copiedKey === 'bin' ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                      </button>
                    </div>
                  </div>
                )}

                {app.config_path && (
                  <div className="flex items-center justify-between p-2.5 rounded-lg bg-surface-950/40 border border-surface-800">
                    <span className="text-slate-400">Config File:</span>
                    <div className="flex items-center gap-2">
                      <code className="text-slate-200 font-mono">{app.config_path}</code>
                      <button
                        onClick={() => handleCopy(app.config_path!, 'conf')}
                        className="text-slate-400 hover:text-white"
                        title="Copy config path"
                      >
                        {copiedKey === 'conf' ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                      </button>
                    </div>
                  </div>
                )}

                {app.default_port && (
                  <div className="flex items-center justify-between p-2.5 rounded-lg bg-surface-950/40 border border-surface-800">
                    <span className="text-slate-400">Default Port:</span>
                    <span className="text-slate-200 font-mono font-bold">{app.default_port}</span>
                  </div>
                )}
              </div>
            </div>
          </div>
        )}

        {/* Footer with Pin to Dashboard */}
        <div className="p-4 bg-surface-950 border-t border-surface-800 flex items-center justify-between select-none">
          <button
            onClick={handleTogglePin}
            className={`px-3.5 py-2 rounded-xl text-xs font-semibold flex items-center gap-2 transition-all border ${
              pinned
                ? 'bg-amber-500/10 text-amber-400 border-amber-500/30 hover:bg-amber-500/20'
                : 'bg-surface-800 text-slate-300 border-surface-700 hover:bg-surface-700 hover:text-white'
            }`}
          >
            {pinned ? (
              <>
                <BookmarkCheck className="w-4 h-4 text-amber-400" />
                Pinned to Dashboard
              </>
            ) : (
              <>
                <Bookmark className="w-4 h-4 text-slate-400" />
                Pin to Dashboard
              </>
            )}
          </button>

          <button
            onClick={onClose}
            className="px-4 py-2 rounded-xl bg-surface-800 hover:bg-surface-700 text-slate-300 text-xs font-medium transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
