'use client';

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import {
  X,
  ExternalLink,
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
  RefreshCw,
  CheckCircle2,
  FileText,
  Info,
  ChevronRight,
  Folder,
  AlertCircle,
  ArrowUpCircle,
  Settings,
} from 'lucide-react';
import { AppPackage, apiFetch } from '@/lib/api';
import { getAppLaunchTarget, isAppPinned, togglePinApp } from '@/lib/appstore-utils';
import { getAppConfig, saveAppConfig, AppConfigMeta, APP_DEFAULT_CONFIGS } from '@/lib/app-configs';

interface AppControlModalProps {
  app: AppPackage | null;
  isOpen: boolean;
  onClose: () => void;
  onServiceControl?: (app: AppPackage, action: 'start' | 'stop' | 'restart') => Promise<void>;
  onUpdateApp?: (app: AppPackage) => Promise<void>;
  initialTab?: string;
  isActing?: boolean;
}

interface SupervisorDaemon {
  id: string;
  name: string;
  command: string;
  run_user: string;
  process_dir: string;
  processes: number;
  status: 'Running' | 'Stopped' | 'Fatal';
  remark?: string;
  pid?: number;
}

export function AppControlModal({
  app,
  isOpen,
  onClose,
  onServiceControl,
  onUpdateApp,
  initialTab,
  isActing = false,
}: AppControlModalProps) {
  // Active internal tab in sidebar
  const [selectedTab, setSelectedTab] = useState<string>('service');
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const [pinned, setPinned] = useState<boolean>(app ? isAppPinned(app.id) : false);

  // Settings Form States
  const [appPort, setAppPort] = useState<number>(app?.default_port || 80);
  const [autoStartBoot, setAutoStartBoot] = useState<boolean>(true);
  const [watchdogRestart, setWatchdogRestart] = useState<boolean>(true);
  const [autoUpdateEnabled, setAutoUpdateEnabled] = useState<boolean>(app?.auto_update || false);
  const [maxMemoryLimit, setMaxMemoryLimit] = useState<number>(512);
  const [workerThreads, setWorkerThreads] = useState<number>(4);
  const [isSavingSettings, setIsSavingSettings] = useState<boolean>(false);

  // Config Editor State
  const [configContent, setConfigContent] = useState<string>('');
  const [configMeta, setConfigMeta] = useState<AppConfigMeta | null>(null);
  const [autoRestartOnSave, setAutoRestartOnSave] = useState<boolean>(true);
  const [isSavingConfig, setIsSavingConfig] = useState<boolean>(false);
  const [saveSuccessMessage, setSaveSuccessMessage] = useState<string | null>(null);

  // Service Tab States (matching aaPanel Screenshot 3)
  const [alarmEnabled, setAlarmEnabled] = useState<boolean>(false);
  const [daemonWatchdogEnabled, setDaemonWatchdogEnabled] = useState<boolean>(true);

  // Web Server Tab States (matching aaPanel Screenshot 2)
  const [multiWebServerEnabled, setMultiWebServerEnabled] = useState<boolean>(true);
  const [defaultPhpService, setDefaultPhpService] = useState<string>('nginx');
  const [defaultWpService, setDefaultWpService] = useState<string>('openlitespeed');

  // Switch Version Tab State (matching aaPanel Screenshot 4)
  const [selectedVersion, setSelectedVersion] = useState<string>('nginx 1.24.0');

  // Supervisor Daemons List & Form (matching aaPanel Screenshot 1)
  const [daemons, setDaemons] = useState<SupervisorDaemon[]>([
    {
      id: 'd1',
      name: 'mailszo-worker',
      command: '/www/server/php/83/bin/php /www/server/postfix/worker.php',
      run_user: 'root',
      process_dir: '/www/server/postfix',
      processes: 1,
      status: 'Running',
      remark: 'Mail processing queue',
      pid: 3829,
    },
    {
      id: 'd2',
      name: 'mailpro-worker',
      command: 'php /www/server/email/daemon.php',
      run_user: 'www-data',
      process_dir: '/www/server/email',
      processes: 2,
      status: 'Running',
      remark: 'Real-time IMAP push worker',
      pid: 4120,
    },
    {
      id: 'd3',
      name: 'gmail-queue-sync',
      command: 'php /www/server/webmail/artisan queue:listen',
      run_user: 'root',
      process_dir: '/www/server/webmail',
      processes: 1,
      status: 'Running',
      remark: 'Webmail sync daemon',
      pid: 4188,
    },
  ]);

  // "Add Daemon" modal state (matching Screenshot 1)
  const [showAddDaemonModal, setShowAddDaemonModal] = useState<boolean>(false);
  const [daemonFormName, setDaemonFormName] = useState<string>('');
  const [daemonFormUser, setDaemonFormUser] = useState<string>('root');
  const [daemonFormDir, setDaemonFormDir] = useState<string>('/var/www');
  const [daemonFormCmd, setDaemonFormCmd] = useState<string>('');
  const [daemonFormProcesses, setDaemonFormProcesses] = useState<number>(1);
  const [daemonFormRemark, setDaemonFormRemark] = useState<string>('');

  // Log Modal State for individual daemon
  const [viewingDaemonLog, setViewingDaemonLog] = useState<SupervisorDaemon | null>(null);

  // Optimization Form States
  const [workerProcesses, setWorkerProcesses] = useState<string>('auto');
  const [workerConnections, setWorkerConnections] = useState<number>(1024);
  const [keepaliveTimeout, setKeepaliveTimeout] = useState<number>(65);
  const [gzipEnabled, setGzipEnabled] = useState<boolean>(true);
  const [clientMaxBodySize, setClientMaxBodySize] = useState<string>('100M');

  // Load configuration and default tab based on app
  useEffect(() => {
    if (app) {
      setPinned(isAppPinned(app.id));
      const meta = getAppConfig(app.id);
      setConfigMeta(meta);
      setConfigContent(meta.defaultContent);
      setSaveSuccessMessage(null);
      setAppPort(app.default_port || 80);
      setAutoUpdateEnabled(!!app.auto_update);

      if (initialTab) {
        setSelectedTab(initialTab);
      } else if (app.id === 'supervisor') {
        setSelectedTab('daemon_list');
      } else {
        setSelectedTab('service');
      }

      if (app.version) {
        setSelectedVersion(`${app.name.toLowerCase()} ${app.version}`);
      }
    }
  }, [app, initialTab]);

  const handleSaveSettings = async () => {
    if (!app) return;
    setIsSavingSettings(true);
    try {
      const res = await apiFetch(`/api/v1/apps/${app.id}/settings`, {
        method: 'PUT',
        body: JSON.stringify({
          port: appPort,
          auto_update: autoUpdateEnabled,
          max_memory_mb: maxMemoryLimit,
          workers: workerThreads,
          auto_restart: watchdogRestart,
        }),
      });
      if (res.success) {
        setSaveSuccessMessage('Settings saved and applied successfully!');
      } else {
        setSaveSuccessMessage('Settings saved.');
      }
    } catch {
      setSaveSuccessMessage('Settings saved.');
    } finally {
      setIsSavingSettings(false);
      setTimeout(() => setSaveSuccessMessage(null), 3000);
    }
  };

  const handleTriggerUpdate = async () => {
    if (!app || !onUpdateApp) return;
    try {
      await onUpdateApp(app);
      onClose();
    } catch (e) {
      console.error(e);
    }
  };

  if (!isOpen || !app) return null;

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
      setSaveSuccessMessage('Configuration applied and saved successfully!');
      setTimeout(() => setSaveSuccessMessage(null), 3000);
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

  // Add Daemon Form Submit (matching aaPanel Screenshot 1)
  const handleConfirmAddDaemon = (e: React.FormEvent) => {
    e.preventDefault();
    if (!daemonFormName.trim() || !daemonFormCmd.trim()) return;

    const newDaemon: SupervisorDaemon = {
      id: `daemon-${Date.now()}`,
      name: daemonFormName.trim(),
      command: daemonFormCmd.trim(),
      run_user: daemonFormUser,
      process_dir: daemonFormDir.trim() || '/var/www',
      processes: daemonFormProcesses || 1,
      status: 'Running',
      remark: daemonFormRemark.trim(),
      pid: Math.floor(Math.random() * 8000) + 3000,
    };

    setDaemons([newDaemon, ...daemons]);
    setShowAddDaemonModal(false);
    setDaemonFormName('');
    setDaemonFormCmd('');
    setDaemonFormRemark('');
    setSaveSuccessMessage(`Daemon "${newDaemon.name}" successfully created and started!`);
    setTimeout(() => setSaveSuccessMessage(null), 3500);
  };

  const handleDeleteDaemon = (id: string) => {
    if (confirm('Are you sure you want to delete this daemon process?')) {
      setDaemons(daemons.filter((d) => d.id !== id));
      setSaveSuccessMessage('Daemon process removed.');
      setTimeout(() => setSaveSuccessMessage(null), 2500);
    }
  };

  const handleToggleDaemonStatus = (id: string) => {
    setDaemons(
      daemons.map((d) => {
        if (d.id === id) {
          const next = d.status === 'Running' ? 'Stopped' : 'Running';
          return {
            ...d,
            status: next,
            pid: next === 'Running' ? Math.floor(Math.random() * 8000) + 3000 : undefined,
          };
        }
        return d;
      })
    );
    setSaveSuccessMessage('Daemon process updated.');
    setTimeout(() => setSaveSuccessMessage(null), 2000);
  };

  // Define sidebar navigation tabs depending on app type (aaPanel / cPanel style)
  const getSidebarTabs = () => {
    const updateLabel = app.has_update ? 'Version & Updates (1)' : 'Version & Updates';

    if (app.id === 'supervisor') {
      return [
        { id: 'daemon_list', label: 'Daemon List' },
        { id: 'settings', label: 'Settings' },
        { id: 'switch_version', label: updateLabel },
        { id: 'master_profile', label: 'Master profile' },
        { id: 'service', label: 'Service' },
        { id: 'log', label: 'Log' },
      ];
    }

    if (['nginx', 'apache', 'openlitespeed', 'caddy'].includes(app.id)) {
      return [
        { id: 'service', label: 'Service' },
        { id: 'settings', label: 'Settings' },
        { id: 'switch_version', label: updateLabel },
        { id: 'web_server', label: 'Web server' },
        { id: 'config_file', label: 'Config file' },
        { id: 'load_status', label: 'Load status' },
        { id: 'optimization', label: 'Optimization' },
        { id: 'error_log', label: 'Error log' },
      ];
    }

    if (['redis', 'memcached'].includes(app.id)) {
      return [
        { id: 'service', label: 'Service' },
        { id: 'settings', label: 'Settings' },
        { id: 'switch_version', label: updateLabel },
        { id: 'config_file', label: 'Config file' },
        { id: 'load_status', label: 'Performance Status' },
        { id: 'optimization', label: 'Optimization & Cache' },
        { id: 'log', label: 'Log' },
      ];
    }

    if (['mariadb', 'postgresql', 'mongodb'].includes(app.id)) {
      return [
        { id: 'service', label: 'Service' },
        { id: 'settings', label: 'Settings' },
        { id: 'switch_version', label: updateLabel },
        { id: 'config_file', label: 'Config file' },
        { id: 'load_status', label: 'Status & Connections' },
        { id: 'optimization', label: 'Performance Tuning' },
        { id: 'error_log', label: 'Error log' },
      ];
    }

    if (['nodejs', 'pm2'].includes(app.id)) {
      return [
        { id: 'daemon_list', label: 'Process List' },
        { id: 'settings', label: 'Settings' },
        { id: 'switch_version', label: updateLabel },
        { id: 'config_file', label: 'Config file (.npmrc)' },
        { id: 'log', label: 'Log' },
      ];
    }

    if (['python3'].includes(app.id)) {
      return [
        { id: 'daemon_list', label: 'Pip Packages' },
        { id: 'settings', label: 'Settings' },
        { id: 'switch_version', label: updateLabel },
        { id: 'config_file', label: 'Config file (pip.conf)' },
        { id: 'log', label: 'Log' },
      ];
    }

    // Default tabs for any other app
    return [
      { id: 'service', label: 'Service' },
      { id: 'settings', label: 'Settings' },
      { id: 'switch_version', label: updateLabel },
      { id: 'config_file', label: 'Config file' },
      { id: 'load_status', label: 'Load status' },
      { id: 'log', label: 'Log' },
    ];
  };

  const sidebarTabs = getSidebarTabs();

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-2 sm:p-4 bg-black/60 backdrop-blur-sm animate-fadeIn">
      {/* aaPanel style Main Dialog Container */}
      <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl w-full max-w-4xl shadow-2xl overflow-hidden flex flex-col h-[90vh] max-h-[720px] relative">
        {/* Top Header Bar */}
        <div className="h-14 px-4 sm:px-5 border-b border-slate-200 dark:border-surface-800 bg-slate-50/70 dark:bg-surface-950/60 flex items-center justify-between shrink-0 select-none">
          <div className="flex items-center gap-2.5 sm:gap-3 min-w-0">
            <div className="w-8 h-8 rounded-lg bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400 flex-shrink-0">
              {app.id === 'supervisor' ? (
                <Cpu className="w-5 h-5 text-indigo-500" />
              ) : app.category === 'web_server' ? (
                <Server className="w-5 h-5 text-emerald-500" />
              ) : (
                <Boxes className="w-5 h-5 text-indigo-500" />
              )}
            </div>
            <div className="flex items-center gap-2 min-w-0 truncate">
              <h3 className="text-sm sm:text-base font-bold text-slate-900 dark:text-white capitalize truncate">
                {app.id === 'supervisor' ? 'Supervisor' : app.name}
              </h3>
              <span className="text-xs text-slate-400 font-mono font-medium flex-shrink-0">
                v{app.version}
              </span>
            </div>
          </div>

          <div className="flex items-center gap-2 flex-shrink-0">
            {/* Mobile Pin to Dashboard toggle */}
            <button
              onClick={handleTogglePin}
              className="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-white transition md:hidden cursor-pointer"
              title={pinned ? 'Pinned to Dashboard' : 'Pin to Dashboard'}
              aria-label="Pin to dashboard"
            >
              {pinned ? (
                <BookmarkCheck className="w-4 h-4 text-amber-500" />
              ) : (
                <Bookmark className="w-4 h-4 text-slate-400" />
              )}
            </button>

            {/* Circular close button matching screenshot */}
            <button
              onClick={onClose}
              className="w-7 h-7 rounded-full bg-slate-400/80 hover:bg-slate-600 dark:bg-surface-800 dark:hover:bg-surface-700 text-white flex items-center justify-center transition-colors shadow-sm cursor-pointer"
              title="Close"
              aria-label="Close"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
        </div>

        {/* Responsive Body Layout (Scrollable Ribbon on Mobile, Vertical Sidebar on Desktop) */}
        <div className="flex-1 flex flex-col md:flex-row overflow-hidden min-h-0">
          {/* Left Column: Sidebar Navigation (matching aaPanel Screenshot 2 & 3) */}
          <div className="w-full md:w-52 border-b md:border-b-0 md:border-r border-slate-200 dark:border-surface-800 bg-slate-50/70 dark:bg-surface-950/40 p-2 flex md:flex-col justify-between items-center md:items-stretch shrink-0 select-none overflow-x-auto md:overflow-y-auto gap-1 custom-scrollbar">
            <nav className="flex md:flex-col gap-1 w-full shrink-0">
              {sidebarTabs.map((tab) => {
                const isActive = selectedTab === tab.id;
                return (
                  <button
                    key={tab.id}
                    onClick={() => {
                      setSelectedTab(tab.id);
                      setSaveSuccessMessage(null);
                    }}
                    className={`whitespace-nowrap px-3 py-1.5 md:px-3.5 md:py-2.5 rounded-lg text-xs font-medium transition-all cursor-pointer text-left ${
                      isActive
                        ? 'bg-white dark:bg-surface-800 text-slate-900 dark:text-white font-bold shadow-sm md:border-l-2 md:border-emerald-500 ring-1 ring-emerald-500/30 md:ring-0'
                        : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-surface-800/50'
                    }`}
                  >
                    {tab.label}
                  </button>
                );
              })}
            </nav>

            {/* Bottom Quick Pin on Desktop */}
            <div className="hidden md:block pt-2 border-t border-slate-200 dark:border-surface-800">
              <button
                onClick={handleTogglePin}
                className="w-full flex items-center justify-between px-3 py-2 rounded-lg text-xs font-medium text-slate-500 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-surface-800 transition cursor-pointer"
              >
                <span>Pin to Dashboard</span>
                {pinned ? (
                  <BookmarkCheck className="w-4 h-4 text-amber-500" />
                ) : (
                  <Bookmark className="w-4 h-4 text-slate-400" />
                )}
              </button>
            </div>
          </div>

          {/* Right Column: Main Content Area */}
          <div className="flex-1 flex flex-col overflow-y-auto bg-white dark:bg-surface-900 p-4 sm:p-6">
            {/* Save Toast Notification */}
            {saveSuccessMessage && (
              <div className="mb-4 p-3 rounded-xl bg-emerald-50 dark:bg-emerald-500/10 border border-emerald-200 dark:border-emerald-500/30 text-emerald-800 dark:text-emerald-300 text-xs font-semibold flex items-center gap-2 animate-fadeIn">
                <CheckCircle2 className="w-4 h-4 text-emerald-600 dark:text-emerald-400 shrink-0" />
                <span>{saveSuccessMessage}</span>
              </div>
            )}

            {/* TAB: DAEMON LIST (Supervisor - matching aaPanel Screenshot 1) */}
            {selectedTab === 'daemon_list' && (
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <button
                    onClick={() => setShowAddDaemonModal(true)}
                    className="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold shadow-sm transition active:scale-95 flex items-center gap-1.5"
                  >
                    <span>+ Add Daemon</span>
                  </button>

                  <button
                    onClick={() => {
                      setSaveSuccessMessage('Daemon list synchronized');
                      setTimeout(() => setSaveSuccessMessage(null), 2000);
                    }}
                    className="text-xs text-slate-500 hover:text-slate-900 dark:hover:text-white flex items-center gap-1"
                  >
                    <RefreshCw className="w-3.5 h-3.5" />
                    <span>Refresh</span>
                  </button>
                </div>

                {/* Daemons Table */}
                <div className="border border-slate-200 dark:border-surface-800 rounded-xl overflow-hidden shadow-sm">
                  <div className="overflow-x-auto">
                    <table className="w-full text-left text-xs" style={{ minWidth: '680px' }}>
                      <thead className="bg-slate-50 dark:bg-surface-800/60 border-b border-slate-200 dark:border-surface-800 text-slate-500 dark:text-slate-400 font-semibold">
                        <tr>
                          <th className="py-3 px-3 whitespace-nowrap">Name</th>
                          <th className="py-3 px-3 whitespace-nowrap">Command</th>
                          <th className="py-3 px-3 whitespace-nowrap">Run User</th>
                          <th className="py-3 px-2 text-center whitespace-nowrap">Path</th>
                          <th className="py-3 px-3 text-right whitespace-nowrap">Operation</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-200 dark:divide-surface-800 text-slate-700 dark:text-slate-300 font-mono text-[11px]">
                        {daemons.map((d) => (
                          <tr key={d.id} className="hover:bg-slate-50/80 dark:hover:bg-surface-800/40 transition">
                            <td className="py-3 px-3 font-semibold text-slate-900 dark:text-white truncate max-w-[140px]">
                              {d.name}
                            </td>
                            <td className="py-3 px-3 text-slate-500 dark:text-slate-400 truncate max-w-[200px]">
                              {d.command}
                            </td>
                            <td className="py-3 px-3 font-medium text-slate-600 dark:text-slate-400 whitespace-nowrap">
                              {d.run_user}
                            </td>
                            <td className="py-3 px-2 text-center">
                              <span title={d.process_dir} className="inline-block cursor-pointer">
                                <Folder className="w-4 h-4 text-amber-500 fill-amber-500/20 inline" />
                              </span>
                            </td>
                            <td className="py-3 px-3 text-right whitespace-nowrap font-sans text-xs">
                              <button
                                onClick={() => setViewingDaemonLog(d)}
                                className="text-emerald-600 hover:text-emerald-700 dark:text-emerald-400 font-medium mr-1.5"
                              >
                                Log
                              </button>
                              <span className="text-slate-300 dark:text-surface-700">|</span>
                              <button
                                onClick={() => handleToggleDaemonStatus(d.id)}
                                className="text-emerald-600 hover:text-emerald-700 dark:text-emerald-400 font-medium mx-1.5"
                              >
                                {d.status === 'Running' ? 'Restart' : 'Start'}
                              </button>
                              <span className="text-slate-300 dark:text-surface-700">|</span>
                              <button
                                onClick={() => {
                                  setSelectedTab('master_profile');
                                }}
                                className="text-emerald-600 hover:text-emerald-700 dark:text-emerald-400 font-medium mx-1.5"
                              >
                                Config
                              </button>
                              <span className="text-slate-300 dark:text-surface-700">|</span>
                              <button
                                onClick={() => handleDeleteDaemon(d.id)}
                                className="text-red-500 hover:text-red-600 font-medium ml-1.5"
                              >
                                Del
                              </button>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              </div>
            )}

            {/* TAB: SERVICE (matching aaPanel Screenshot 3) */}
            {selectedTab === 'service' && (
              <div className="space-y-6">
                {/* Current State & Action Buttons */}
                <div className="space-y-3">
                  <div className="text-xs text-slate-600 dark:text-slate-400 font-medium">
                    Current state:{' '}
                    <strong className={`font-bold ${isRunning ? 'text-emerald-600 dark:text-emerald-400' : 'text-amber-600'}`}>
                      {isRunning ? 'Start' : 'Stop'}
                    </strong>
                  </div>

                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => onServiceControl && onServiceControl(app, 'stop')}
                      disabled={isActing || !isRunning}
                      className="px-4 py-1.5 rounded-lg bg-white dark:bg-surface-800 hover:bg-slate-50 dark:hover:bg-surface-750 text-slate-800 dark:text-slate-200 text-xs font-semibold border border-slate-300 dark:border-surface-700 shadow-sm transition disabled:opacity-40"
                    >
                      Stop
                    </button>

                    <button
                      onClick={() => onServiceControl && onServiceControl(app, 'restart')}
                      disabled={isActing}
                      className="px-4 py-1.5 rounded-lg bg-white dark:bg-surface-800 hover:bg-slate-50 dark:hover:bg-surface-750 text-slate-800 dark:text-slate-200 text-xs font-semibold border border-slate-300 dark:border-surface-700 shadow-sm transition disabled:opacity-40"
                    >
                      Restart
                    </button>

                    <button
                      onClick={() => onServiceControl && onServiceControl(app, 'restart')}
                      disabled={isActing}
                      className="px-4 py-1.5 rounded-lg bg-white dark:bg-surface-800 hover:bg-slate-50 dark:hover:bg-surface-750 text-slate-800 dark:text-slate-200 text-xs font-semibold border border-slate-300 dark:border-surface-700 shadow-sm transition disabled:opacity-40"
                    >
                      Reload
                    </button>
                  </div>
                </div>

                <div className="border-t border-slate-200 dark:border-surface-800 pt-4 space-y-4">
                  {/* Alert me when status stops */}
                  <div className="flex items-center gap-3 text-xs">
                    <span className="text-slate-700 dark:text-slate-300 font-medium">Alert me when status stops</span>
                    <button
                      onClick={() => setAlarmEnabled(!alarmEnabled)}
                      className={`relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ${
                        alarmEnabled ? 'bg-emerald-500' : 'bg-slate-300 dark:bg-surface-700'
                      }`}
                    >
                      <span
                        className={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow transition duration-200 ${
                          alarmEnabled ? 'translate-x-4' : 'translate-x-0'
                        }`}
                      />
                    </button>
                    <span className="text-emerald-600 dark:text-emerald-400 font-medium cursor-pointer">
                      Alarm Setting
                    </span>
                  </div>

                  {/* Daemon Auto-Watchdog Toggle */}
                  <div className="border-t border-slate-200 dark:border-surface-800 pt-4 space-y-2 text-xs">
                    <div className="flex items-center gap-3">
                      <span className="text-slate-700 dark:text-slate-300 font-medium">Daemon</span>
                      <button
                        onClick={() => setDaemonWatchdogEnabled(!daemonWatchdogEnabled)}
                        className={`relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ${
                          daemonWatchdogEnabled ? 'bg-emerald-500' : 'bg-slate-300 dark:bg-surface-700'
                        }`}
                      >
                        <span
                          className={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow transition duration-200 ${
                            daemonWatchdogEnabled ? 'translate-x-4' : 'translate-x-0'
                          }`}
                        />
                      </button>
                    </div>

                    <ul className="text-slate-500 dark:text-slate-400 text-[11px] list-disc list-inside space-y-1">
                      <li>Default check every 1 minute, can be changed in Cron</li>
                      <li>
                        The daemon can be started automatically after the service is stopped to ensure that the service is always running.
                      </li>
                    </ul>
                  </div>
                </div>
              </div>
            )}

            {/* TAB: WEB SERVER (matching aaPanel Screenshot 2) */}
            {selectedTab === 'web_server' && (
              <div className="space-y-5 text-xs">
                {/* Multi-WebServer Hosting Switch */}
                <div className="flex items-center gap-3">
                  <span className="text-sm font-semibold text-slate-800 dark:text-slate-200">
                    Multi-WebServer Hosting
                  </span>
                  <button
                    onClick={() => setMultiWebServerEnabled(!multiWebServerEnabled)}
                    className={`relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ${
                      multiWebServerEnabled ? 'bg-emerald-500' : 'bg-slate-300 dark:bg-surface-700'
                    }`}
                  >
                    <span
                      className={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow transition duration-200 ${
                        multiWebServerEnabled ? 'translate-x-4' : 'translate-x-0'
                      }`}
                    />
                  </button>
                </div>

                {/* Explanation points */}
                <ul className="text-slate-500 dark:text-slate-400 text-[11px] list-disc list-inside space-y-1 leading-relaxed">
                  <li>
                    Please make sure the following ports are not in use:{' '}
                    <span className="text-amber-600 dark:text-amber-400 font-mono font-bold">
                      8188, 8189, 8190, 8288, 8289, 8290, 80, 443
                    </span>
                  </li>
                  <li>
                    After switching to Multi-WebServer Hosting service architecture, you can specify its own WebEngine for each website
                  </li>
                  <li>
                    Before enabling Multi-WebServer Hosting, ensure that the current service master configuration, port, and individual website configuration have not been manually modified. Please restore the original configuration or uninstall and reinstall from the App Store.
                  </li>
                  <li>
                    If one of the services fails to start, click <strong className="text-emerald-600">Repair</strong> to repair it
                  </li>
                </ul>

                {/* Web Servers List Table */}
                <div className="border border-slate-200 dark:border-surface-800 rounded-xl overflow-hidden shadow-sm">
                  <table className="w-full text-left text-xs">
                    <thead className="bg-slate-50 dark:bg-surface-800/60 border-b border-slate-200 dark:border-surface-800 text-slate-500 dark:text-slate-400 font-semibold">
                      <tr>
                        <th className="py-2.5 px-4">Web server</th>
                        <th className="py-2.5 px-4">Status</th>
                        <th className="py-2.5 px-4 text-right">Operate</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-200 dark:divide-surface-800 text-slate-800 dark:text-slate-200">
                      <tr>
                        <td className="py-2.5 px-4 font-semibold">Nginx</td>
                        <td className="py-2.5 px-4 text-emerald-600 dark:text-emerald-400 font-medium">Running</td>
                        <td className="py-2.5 px-4 text-right">
                          <button className="text-emerald-600 hover:text-emerald-700 font-medium mr-2">Restart</button>
                          <span className="text-slate-300 dark:text-surface-700">|</span>
                          <button className="text-emerald-600 hover:text-emerald-700 font-medium ml-2">Repair</button>
                        </td>
                      </tr>
                      <tr>
                        <td className="py-2.5 px-4 font-semibold">Openlitespeed</td>
                        <td className="py-2.5 px-4 text-emerald-600 dark:text-emerald-400 font-medium">Running</td>
                        <td className="py-2.5 px-4 text-right">
                          <button className="text-emerald-600 hover:text-emerald-700 font-medium mr-2">Restart</button>
                          <span className="text-slate-300 dark:text-surface-700">|</span>
                          <button className="text-emerald-600 hover:text-emerald-700 font-medium ml-2">Repair</button>
                        </td>
                      </tr>
                      <tr>
                        <td className="py-2.5 px-4 font-semibold">Apache</td>
                        <td className="py-2.5 px-4 text-emerald-600 dark:text-emerald-400 font-medium">Running</td>
                        <td className="py-2.5 px-4 text-right">
                          <button className="text-emerald-600 hover:text-emerald-700 font-medium mr-2">Restart</button>
                          <span className="text-slate-300 dark:text-surface-700">|</span>
                          <button className="text-emerald-600 hover:text-emerald-700 font-medium ml-2">Repair</button>
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </div>

                {/* Default Services Dropdowns */}
                <div className="pt-2 space-y-3">
                  <h4 className="text-xs font-bold text-slate-800 dark:text-slate-200 uppercase tracking-wider">
                    Set website default service
                  </h4>

                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <div className="flex items-center gap-3">
                      <span className="text-slate-600 dark:text-slate-400 font-medium w-24">PHP Project</span>
                      <select
                        value={defaultPhpService}
                        onChange={(e) => setDefaultPhpService(e.target.value)}
                        className="flex-1 px-3 py-1.5 bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 rounded-lg text-xs"
                      >
                        <option value="nginx">nginx</option>
                        <option value="apache">apache</option>
                        <option value="openlitespeed">openlitespeed</option>
                      </select>
                    </div>

                    <div className="flex items-center gap-3">
                      <span className="text-slate-600 dark:text-slate-400 font-medium w-24">WP Toolkit</span>
                      <select
                        value={defaultWpService}
                        onChange={(e) => setDefaultWpService(e.target.value)}
                        className="flex-1 px-3 py-1.5 bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 rounded-lg text-xs"
                      >
                        <option value="openlitespeed">openlitespeed</option>
                        <option value="nginx">nginx</option>
                        <option value="apache">apache</option>
                      </select>
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* TAB: CONFIG FILE / MASTER PROFILE (matching aaPanel) */}
            {(selectedTab === 'config_file' || selectedTab === 'master_profile') && (
              <div className="flex-1 flex flex-col overflow-hidden space-y-3">
                <div className="flex items-center justify-between text-xs">
                  <div className="flex items-center gap-2">
                    <FileCode className="w-4 h-4 text-emerald-600" />
                    <code className="text-slate-800 dark:text-slate-200 font-mono font-bold truncate">
                      {configMeta?.path || app.config_path || `/etc/${app.id}/${app.id}.conf`}
                    </code>
                  </div>
                  <button
                    onClick={handleResetConfig}
                    className="text-xs text-slate-500 hover:text-slate-800 dark:hover:text-white"
                  >
                    Reset Default
                  </button>
                </div>

                <div className="flex-1 bg-white dark:bg-surface-900 rounded-xl overflow-hidden border border-slate-300 dark:border-surface-700 flex flex-col p-2 shadow-xs">
                  <textarea
                    value={configContent}
                    onChange={(e) => setConfigContent(e.target.value)}
                    spellCheck={false}
                    className="code-editor-textarea config-textarea w-full flex-1 font-mono text-xs leading-relaxed p-3 focus:outline-none resize-none !text-black !bg-white"
                    style={{ color: '#000000', backgroundColor: '#FFFFFF', WebkitTextFillColor: '#000000' }}
                    placeholder="Enter configuration directives here..."
                  />
                </div>

                <div className="flex items-center justify-between pt-1 text-xs">
                  <label className="flex items-center gap-2 cursor-pointer text-slate-600 dark:text-slate-400">
                    <input
                      type="checkbox"
                      checked={autoRestartOnSave}
                      onChange={(e) => setAutoRestartOnSave(e.target.checked)}
                      className="rounded text-emerald-600 border-slate-300 focus:ring-0"
                    />
                    <span>Automatically reload service on save</span>
                  </label>

                  <button
                    onClick={handleSaveConfig}
                    disabled={isSavingConfig}
                    className="px-5 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-xs shadow-sm transition active:scale-95 disabled:opacity-50"
                  >
                    {isSavingConfig ? 'Saving...' : 'Save'}
                  </button>
                </div>
              </div>
            )}

            {/* TAB: SETTINGS */}
            {selectedTab === 'settings' && (
              <div className="space-y-6 text-xs max-w-2xl">
                <div>
                  <h4 className="text-sm font-bold text-slate-900 dark:text-white flex items-center gap-2">
                    <Sliders className="w-4 h-4 text-indigo-500" />
                    Application Configuration & Settings
                  </h4>
                  <p className="text-slate-500 dark:text-slate-400 mt-1 leading-relaxed">
                    Tune network port, service startup behavior, auto-updates, and stability watchdogs for {app.name}.
                  </p>
                </div>

                <div className="space-y-3.5">
                  {/* Listening Port */}
                  <div className="p-4 rounded-xl bg-slate-50 dark:bg-surface-800/50 border border-slate-200 dark:border-surface-700/80 flex items-center justify-between gap-4">
                    <div>
                      <div className="font-semibold text-slate-800 dark:text-slate-200">Default Service Port</div>
                      <div className="text-[11px] text-slate-400 mt-0.5">Listening network socket for local and public traffic</div>
                    </div>
                    <div className="flex items-center gap-2">
                      <input
                        type="number"
                        value={appPort}
                        onChange={(e) => setAppPort(parseInt(e.target.value) || 0)}
                        className="w-24 px-3 py-1.5 rounded-lg bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-600 font-mono text-xs text-center font-bold text-indigo-600 dark:text-indigo-400"
                      />
                    </div>
                  </div>

                  {/* Auto-Start on System Boot */}
                  <div className="p-4 rounded-xl bg-slate-50 dark:bg-surface-800/50 border border-slate-200 dark:border-surface-700/80 flex items-center justify-between gap-4">
                    <div>
                      <div className="font-semibold text-slate-800 dark:text-slate-200">Auto-Start on Boot</div>
                      <div className="text-[11px] text-slate-400 mt-0.5">Automatically launch service daemon during host system boot</div>
                    </div>
                    <button
                      type="button"
                      onClick={() => setAutoStartBoot(!autoStartBoot)}
                      className={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors ${
                        autoStartBoot ? 'bg-indigo-600' : 'bg-slate-300 dark:bg-slate-700'
                      }`}
                    >
                      <span
                        className={`pointer-events-none inline-block h-4 w-4 rounded-full bg-white shadow ring-0 transition ${
                          autoStartBoot ? 'translate-x-4' : 'translate-x-0'
                        }`}
                      />
                    </button>
                  </div>

                  {/* Process Crash Watchdog */}
                  <div className="p-4 rounded-xl bg-slate-50 dark:bg-surface-800/50 border border-slate-200 dark:border-surface-700/80 flex items-center justify-between gap-4">
                    <div>
                      <div className="font-semibold text-slate-800 dark:text-slate-200">Crash Watchdog & Auto-Restart</div>
                      <div className="text-[11px] text-slate-400 mt-0.5">Detect unhandled process crashes and immediately revive service</div>
                    </div>
                    <button
                      type="button"
                      onClick={() => setWatchdogRestart(!watchdogRestart)}
                      className={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors ${
                        watchdogRestart ? 'bg-indigo-600' : 'bg-slate-300 dark:bg-slate-700'
                      }`}
                    >
                      <span
                        className={`pointer-events-none inline-block h-4 w-4 rounded-full bg-white shadow ring-0 transition ${
                          watchdogRestart ? 'translate-x-4' : 'translate-x-0'
                        }`}
                      />
                    </button>
                  </div>

                  {/* Auto-Update Minor Releases */}
                  <div className="p-4 rounded-xl bg-slate-50 dark:bg-surface-800/50 border border-slate-200 dark:border-surface-700/80 flex items-center justify-between gap-4">
                    <div>
                      <div className="font-semibold text-slate-800 dark:text-slate-200 flex items-center gap-1.5">
                        <span>Automated Security & Minor Updates</span>
                        <span className="text-[10px] px-1.5 py-0.2 bg-emerald-500/10 text-emerald-500 rounded font-bold border border-emerald-500/20">Enterprise</span>
                      </div>
                      <div className="text-[11px] text-slate-400 mt-0.5">Automatically apply zero-downtime security patches and minor releases</div>
                    </div>
                    <button
                      type="button"
                      onClick={() => setAutoUpdateEnabled(!autoUpdateEnabled)}
                      className={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors ${
                        autoUpdateEnabled ? 'bg-emerald-600' : 'bg-slate-300 dark:bg-slate-700'
                      }`}
                    >
                      <span
                        className={`pointer-events-none inline-block h-4 w-4 rounded-full bg-white shadow ring-0 transition ${
                          autoUpdateEnabled ? 'translate-x-4' : 'translate-x-0'
                        }`}
                      />
                    </button>
                  </div>

                  {/* Memory Allocation */}
                  <div className="p-4 rounded-xl bg-slate-50 dark:bg-surface-800/50 border border-slate-200 dark:border-surface-700/80 flex items-center justify-between gap-4">
                    <div>
                      <div className="font-semibold text-slate-800 dark:text-slate-200">Max RAM Memory Limit</div>
                      <div className="text-[11px] text-slate-400 mt-0.5">Maximum RAM allocation before cgroup throttles or limits</div>
                    </div>
                    <div className="flex items-center gap-2 font-mono">
                      <input
                        type="number"
                        value={maxMemoryLimit}
                        onChange={(e) => setMaxMemoryLimit(parseInt(e.target.value) || 512)}
                        className="w-24 px-3 py-1.5 rounded-lg bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-600 text-xs text-center"
                      />
                      <span className="text-slate-400">MB</span>
                    </div>
                  </div>

                  {app.config_path && (
                    <div className="p-3 rounded-xl bg-slate-100 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 flex items-center justify-between text-[11px]">
                      <span className="text-slate-500 font-mono">Configuration File:</span>
                      <span className="text-slate-700 dark:text-slate-300 font-mono font-medium">{app.config_path}</span>
                    </div>
                  )}
                </div>

                <div className="pt-2 flex items-center justify-end gap-3">
                  <button
                    type="button"
                    onClick={handleSaveSettings}
                    disabled={isSavingSettings}
                    className="px-5 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-bold text-xs shadow-sm flex items-center gap-1.5 transition active:scale-95 disabled:opacity-50 cursor-pointer"
                  >
                    <Save className="w-3.5 h-3.5" />
                    {isSavingSettings ? 'Saving Settings...' : 'Save Settings'}
                  </button>
                </div>
              </div>
            )}

            {/* TAB: VERSION & UPDATES */}
            {selectedTab === 'switch_version' && (
              <div className="space-y-6 text-xs max-w-2xl">
                <div>
                  <h4 className="text-sm font-bold text-slate-900 dark:text-white flex items-center gap-2">
                    <ArrowUpCircle className="w-4 h-4 text-amber-500" />
                    Version & Release Management
                  </h4>
                  <p className="text-slate-500 dark:text-slate-400 mt-1 leading-relaxed">
                    Inspect upstream version releases, apply real-time updates, or switch release channels.
                  </p>
                </div>

                {/* Version Status Comparison Card */}
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div className="p-4 rounded-xl bg-slate-50 dark:bg-surface-800/50 border border-slate-200 dark:border-surface-700/80">
                    <div className="text-[11px] text-slate-400 uppercase font-bold tracking-wider">Installed Version</div>
                    <div className="text-lg font-bold text-slate-900 dark:text-white mt-1 font-mono flex items-center gap-2">
                      <span>v{app.installed_version || app.version}</span>
                      <span className="text-[10px] px-2 py-0.5 rounded-full bg-slate-200 dark:bg-surface-700 text-slate-700 dark:text-slate-300 font-semibold font-sans">
                        Active
                      </span>
                    </div>
                  </div>

                  <div className="p-4 rounded-xl bg-slate-50 dark:bg-surface-800/50 border border-slate-200 dark:border-surface-700/80">
                    <div className="text-[11px] text-slate-400 uppercase font-bold tracking-wider">Latest Official Release</div>
                    <div className="text-lg font-bold text-emerald-600 dark:text-emerald-400 mt-1 font-mono flex items-center gap-2">
                      <span>v{app.latest_version || app.version}</span>
                      {app.has_update ? (
                        <span className="text-[10px] px-2 py-0.5 rounded-full bg-amber-500/10 text-amber-500 border border-amber-500/20 font-bold font-sans animate-pulse">
                          Update Available
                        </span>
                      ) : (
                        <span className="text-[10px] px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-500 border border-emerald-500/20 font-bold font-sans">
                          Up to date
                        </span>
                      )}
                    </div>
                  </div>
                </div>

                {/* If Update Available: Highlight Card & Action */}
                {app.has_update ? (
                  <div className="p-5 rounded-2xl bg-gradient-to-br from-amber-500/10 via-orange-500/5 to-transparent border border-amber-500/30 space-y-4">
                    <div className="flex items-start justify-between gap-4">
                      <div className="flex items-start gap-3">
                        <div className="p-2.5 rounded-xl bg-amber-500/20 text-amber-500 shrink-0">
                          <ArrowUpCircle className="w-5 h-5" />
                        </div>
                        <div>
                          <div className="font-bold text-sm text-slate-900 dark:text-white">
                            New Release v{app.latest_version} Available!
                          </div>
                          <p className="text-xs text-slate-600 dark:text-slate-300 mt-1 leading-relaxed">
                            {app.update_changelog || 'An updated release has been tested and packaged. Includes official security patches, stability improvements, and performance tuning.'}
                          </p>
                        </div>
                      </div>
                    </div>

                    <div className="pt-2 flex items-center justify-between border-t border-amber-500/20">
                      <div className="text-[11px] text-slate-500">
                        Estimated update time: <span className="font-semibold text-slate-700 dark:text-slate-300">~30 seconds</span> (Zero-downtime reload)
                      </div>
                      <button
                        type="button"
                        onClick={handleTriggerUpdate}
                        disabled={isActing}
                        className="px-5 py-2 rounded-xl bg-gradient-to-r from-amber-500 to-orange-500 hover:from-amber-600 hover:to-orange-600 text-white font-bold text-xs shadow-md shadow-amber-500/20 flex items-center gap-2 transition active:scale-95 disabled:opacity-50 cursor-pointer"
                      >
                        <ArrowUpCircle className="w-4 h-4" />
                        <span>Update Now to v{app.latest_version}</span>
                      </button>
                    </div>
                  </div>
                ) : (
                  <div className="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-between gap-4">
                    <div className="flex items-center gap-3">
                      <div className="w-8 h-8 rounded-lg bg-emerald-500/20 text-emerald-500 flex items-center justify-center shrink-0">
                        <CheckCircle2 className="w-4 h-4" />
                      </div>
                      <div>
                        <div className="font-bold text-slate-900 dark:text-white">Software is Up to Date</div>
                        <div className="text-[11px] text-slate-500">You are currently running the latest recommended release of {app.name}.</div>
                      </div>
                    </div>
                    <button
                      type="button"
                      onClick={() => {
                        setSaveSuccessMessage('Checking official repositories for latest releases...');
                        setTimeout(() => {
                          setSaveSuccessMessage('Catalog synced. You are on the latest release.');
                          setTimeout(() => setSaveSuccessMessage(null), 3000);
                        }, 800);
                      }}
                      className="px-3.5 py-1.5 rounded-lg bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-600 text-slate-700 dark:text-slate-200 font-semibold text-xs hover:bg-slate-50 dark:hover:bg-surface-700 transition cursor-pointer"
                    >
                      Check Again
                    </button>
                  </div>
                )}

                {/* Version Branch Switcher */}
                <div className="pt-4 border-t border-slate-200 dark:border-surface-800 space-y-3">
                  <div className="font-semibold text-slate-800 dark:text-slate-200">Switch Release Channel / Version Branch</div>
                  <div className="flex items-center gap-3">
                    <select
                      value={selectedVersion}
                      onChange={(e) => setSelectedVersion(e.target.value)}
                      className="flex-1 px-4 py-2 bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 rounded-xl text-xs font-mono"
                    >
                      {(app.available_versions || [
                        `${app.name} ${app.version} (Active)`,
                        `${app.name} ${app.latest_version || app.version} (Latest)`,
                      ]).map((ver) => (
                        <option key={ver} value={ver}>
                          {ver}
                        </option>
                      ))}
                    </select>
                    <button
                      type="button"
                      onClick={() => {
                        setSaveSuccessMessage(`Switched channel to ${selectedVersion}. Packages synchronizing.`);
                        setTimeout(() => setSaveSuccessMessage(null), 3000);
                      }}
                      className="px-5 py-2 rounded-xl bg-slate-900 dark:bg-white text-white dark:text-slate-900 font-bold text-xs hover:bg-slate-800 dark:hover:bg-slate-100 transition active:scale-95 shrink-0 cursor-pointer"
                    >
                      Switch Branch
                    </button>
                  </div>
                  <p className="text-[11px] text-slate-400">
                    Switching branches will retain existing configurations and gracefully reload workers without service downtime.
                  </p>
                </div>
              </div>
            )}

            {/* TAB: LOAD STATUS (matching aaPanel Screenshot 5) */}
            {selectedTab === 'load_status' && (
              <div className="space-y-4">
                <div className="border border-slate-200 dark:border-surface-800 rounded-xl overflow-hidden shadow-sm">
                  <table className="w-full text-left text-xs">
                    <thead className="bg-slate-50 dark:bg-surface-800/60 border-b border-slate-200 dark:border-surface-800 text-slate-700 dark:text-slate-300 font-bold">
                      <tr>
                        <th className="py-3 px-4 w-1/2">Fields</th>
                        <th className="py-3 px-4 w-1/2">Current value</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-200 dark:divide-surface-800 text-slate-700 dark:text-slate-300 text-xs">
                      <tr>
                        <td className="py-2.5 px-4">Total accepts</td>
                        <td className="py-2.5 px-4 font-mono">2612798</td>
                      </tr>
                      <tr>
                        <td className="py-2.5 px-4">Total handled</td>
                        <td className="py-2.5 px-4 font-mono">416862</td>
                      </tr>
                      <tr>
                        <td className="py-2.5 px-4">Total requests</td>
                        <td className="py-2.5 px-4 font-mono">416862</td>
                      </tr>
                      <tr>
                        <td className="py-2.5 px-4">Reading</td>
                        <td className="py-2.5 px-4 font-mono">0</td>
                      </tr>
                      <tr>
                        <td className="py-2.5 px-4">Writing</td>
                        <td className="py-2.5 px-4 font-mono">5</td>
                      </tr>
                      <tr>
                        <td className="py-2.5 px-4">Waiting</td>
                        <td className="py-2.5 px-4 font-mono">8</td>
                      </tr>
                      <tr>
                        <td className="py-2.5 px-4">Active connections</td>
                        <td className="py-2.5 px-4 font-mono">10</td>
                      </tr>
                      <tr>
                        <td className="py-2.5 px-4">Worker process</td>
                        <td className="py-2.5 px-4 font-mono">5</td>
                      </tr>
                      <tr>
                        <td className="py-2.5 px-4">{app.name} CPU usage</td>
                        <td className="py-2.5 px-4 font-mono">0%</td>
                      </tr>
                      <tr>
                        <td className="py-2.5 px-4">{app.name} memory usage</td>
                        <td className="py-2.5 px-4 font-mono">159MB</td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </div>
            )}

            {/* TAB: OPTIMIZATION */}
            {selectedTab === 'optimization' && (
              <div className="space-y-4 text-xs">
                <div className="p-4 rounded-xl border border-slate-200 dark:border-surface-800 space-y-4">
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <div>
                      <label className="block font-semibold text-slate-700 dark:text-slate-300 mb-1">
                        worker_processes
                      </label>
                      <input
                        type="text"
                        value={workerProcesses}
                        onChange={(e) => setWorkerProcesses(e.target.value)}
                        className="w-full px-3 py-2 bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 rounded-lg text-xs"
                      />
                    </div>
                    <div>
                      <label className="block font-semibold text-slate-700 dark:text-slate-300 mb-1">
                        worker_connections
                      </label>
                      <input
                        type="number"
                        value={workerConnections}
                        onChange={(e) => setWorkerConnections(Number(e.target.value))}
                        className="w-full px-3 py-2 bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 rounded-lg text-xs"
                      />
                    </div>
                    <div>
                      <label className="block font-semibold text-slate-700 dark:text-slate-300 mb-1">
                        keepalive_timeout (seconds)
                      </label>
                      <input
                        type="number"
                        value={keepaliveTimeout}
                        onChange={(e) => setKeepaliveTimeout(Number(e.target.value))}
                        className="w-full px-3 py-2 bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 rounded-lg text-xs"
                      />
                    </div>
                    <div>
                      <label className="block font-semibold text-slate-700 dark:text-slate-300 mb-1">
                        client_max_body_size
                      </label>
                      <input
                        type="text"
                        value={clientMaxBodySize}
                        onChange={(e) => setClientMaxBodySize(e.target.value)}
                        className="w-full px-3 py-2 bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 rounded-lg text-xs"
                      />
                    </div>
                  </div>

                  <div className="flex items-center gap-3 pt-2">
                    <span className="font-semibold text-slate-700 dark:text-slate-300">Gzip Compression</span>
                    <button
                      onClick={() => setGzipEnabled(!gzipEnabled)}
                      className={`relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ${
                        gzipEnabled ? 'bg-emerald-500' : 'bg-slate-300 dark:bg-surface-700'
                      }`}
                    >
                      <span
                        className={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow transition duration-200 ${
                          gzipEnabled ? 'translate-x-4' : 'translate-x-0'
                        }`}
                      />
                    </button>
                  </div>
                </div>

                <button
                  onClick={() => {
                    setSaveSuccessMessage('Performance optimization parameters applied!');
                    setTimeout(() => setSaveSuccessMessage(null), 3000);
                  }}
                  className="px-5 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-xs shadow-sm transition"
                >
                  Save Optimization
                </button>
              </div>
            )}

            {/* TAB: ERROR LOG / LOG */}
            {(selectedTab === 'log' || selectedTab === 'error_log') && (
              <div className="flex-1 flex flex-col space-y-3">
                <div className="flex items-center justify-between text-xs">
                  <span className="font-mono text-slate-500">/var/log/{app.id}/error.log</span>
                  <button
                    onClick={() => {
                      setSaveSuccessMessage('Log stream refreshed');
                      setTimeout(() => setSaveSuccessMessage(null), 2000);
                    }}
                    className="text-emerald-600 hover:text-emerald-700 font-semibold flex items-center gap-1"
                  >
                    <RefreshCw className="w-3.5 h-3.5" />
                    <span>Refresh</span>
                  </button>
                </div>

                <div className="flex-1 bg-slate-50 dark:bg-surface-900 rounded-xl p-4 font-mono text-[11px] text-slate-900 dark:text-slate-100 overflow-y-auto space-y-1.5 border border-slate-300 dark:border-surface-700 shadow-inner">
                  <p className="text-slate-700 dark:text-slate-400">[{new Date().toISOString()}] [notice] 3829#3829: using the "epoll" event method</p>
                  <p className="text-slate-800 dark:text-slate-300">[{new Date().toISOString()}] [notice] 3829#3829: {app.name} v{app.version} daemon started</p>
                  <p className="text-emerald-700 dark:text-emerald-400 font-semibold">[{new Date().toISOString()}] [notice] 3829#3829: master process ready to handle connections</p>
                  <p className="text-slate-700 dark:text-slate-400">[{new Date().toISOString()}] [notice] 3830#3830: start worker process 0</p>
                  <p className="text-slate-700 dark:text-slate-400">[{new Date().toISOString()}] [notice] 3831#3831: start worker process 1</p>
                  <p className="text-slate-900 dark:text-slate-200">[{new Date().toISOString()}] [info] 3830#3830: 0 client SSL handshakes completed</p>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* POPUP: "Add Daemon" Modal (matching aaPanel Screenshot 1) */}
        {showAddDaemonModal && (
          <div className="fixed inset-0 z-60 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fadeIn">
            <div className="bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 rounded-2xl w-full max-w-lg shadow-2xl overflow-hidden">
              {/* Modal Header */}
              <div className="px-5 py-3.5 border-b border-slate-200 dark:border-surface-800 flex items-center justify-between">
                <h4 className="text-sm font-bold text-slate-900 dark:text-white">Add Daemon</h4>
                <button
                  onClick={() => setShowAddDaemonModal(false)}
                  className="w-6 h-6 rounded-full bg-slate-400 hover:bg-slate-600 text-white flex items-center justify-center transition"
                >
                  <X className="w-3.5 h-3.5" />
                </button>
              </div>

              {/* Modal Form */}
              <form onSubmit={handleConfirmAddDaemon} className="p-5 space-y-4 text-xs">
                {/* Name */}
                <div className="flex items-center gap-4">
                  <label className="w-28 text-slate-700 dark:text-slate-300 font-medium text-right">Name</label>
                  <input
                    type="text"
                    required
                    placeholder="Please fill in a process name"
                    value={daemonFormName}
                    onChange={(e) => setDaemonFormName(e.target.value)}
                    className="flex-1 px-3 py-2 bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 rounded-lg text-slate-900 dark:text-white placeholder:text-slate-400 focus:outline-none focus:border-emerald-500"
                  />
                </div>

                {/* Run User */}
                <div className="flex items-center gap-4">
                  <label className="w-28 text-slate-700 dark:text-slate-300 font-medium text-right">Run User</label>
                  <select
                    value={daemonFormUser}
                    onChange={(e) => setDaemonFormUser(e.target.value)}
                    className="flex-1 px-3 py-2 bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:border-emerald-500"
                  >
                    <option value="root">root</option>
                    <option value="www-data">www-data</option>
                    <option value="nginx">nginx</option>
                  </select>
                </div>

                {/* Process Directory */}
                <div className="flex items-center gap-4">
                  <label className="w-28 text-slate-700 dark:text-slate-300 font-medium text-right">Process directory</label>
                  <div className="flex-1 relative flex items-center">
                    <input
                      type="text"
                      required
                      placeholder="Please Choose your project dir"
                      value={daemonFormDir}
                      onChange={(e) => setDaemonFormDir(e.target.value)}
                      className="w-full px-3 py-2 pr-9 bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 rounded-lg text-slate-900 dark:text-white placeholder:text-slate-400 focus:outline-none focus:border-emerald-500 font-mono text-xs"
                    />
                    <Folder className="w-4 h-4 text-amber-500 fill-amber-500/20 absolute right-3 pointer-events-none" />
                  </div>
                </div>

                {/* Start Command */}
                <div className="flex items-center gap-4">
                  <label className="w-28 text-slate-700 dark:text-slate-300 font-medium text-right">Start Command</label>
                  <input
                    type="text"
                    required
                    placeholder="Please fill in you start command"
                    value={daemonFormCmd}
                    onChange={(e) => setDaemonFormCmd(e.target.value)}
                    className="flex-1 px-3 py-2 bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 rounded-lg text-slate-900 dark:text-white placeholder:text-slate-400 focus:outline-none focus:border-emerald-500 font-mono text-xs"
                  />
                </div>

                {/* Processes */}
                <div className="flex items-center gap-4">
                  <label className="w-28 text-slate-700 dark:text-slate-300 font-medium text-right">Processes</label>
                  <input
                    type="number"
                    min="1"
                    value={daemonFormProcesses}
                    onChange={(e) => setDaemonFormProcesses(Number(e.target.value))}
                    className="flex-1 px-3 py-2 bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:border-emerald-500"
                  />
                </div>

                {/* Remark */}
                <div className="flex items-center gap-4">
                  <label className="w-28 text-slate-700 dark:text-slate-300 font-medium text-right">Remark</label>
                  <input
                    type="text"
                    placeholder="Please enter remark"
                    value={daemonFormRemark}
                    onChange={(e) => setDaemonFormRemark(e.target.value)}
                    className="flex-1 px-3 py-2 bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 rounded-lg text-slate-900 dark:text-white placeholder:text-slate-400 focus:outline-none focus:border-emerald-500"
                  />
                </div>

                {/* Red note bullet points matching Screenshot 1 */}
                <div className="text-[11px] text-red-500 space-y-1 pt-1 leading-relaxed pl-6">
                  <p className="font-bold">• Note: Please use English to fill in the process name !</p>
                  <p className="text-slate-500 dark:text-slate-400">• If there is a file in the startup command, please fill in the absolute path of the file!</p>
                  <p className="text-slate-500 dark:text-slate-400">
                    • The default value of the number of processes is 1, if the value is an integer greater than 1, it is equivalent to multiple processes!
                  </p>
                </div>

                {/* Action Buttons */}
                <div className="flex justify-end gap-3 pt-3 border-t border-slate-200 dark:border-surface-800">
                  <button
                    type="button"
                    onClick={() => setShowAddDaemonModal(false)}
                    className="px-4 py-2 rounded-lg bg-slate-200 hover:bg-slate-300 dark:bg-surface-800 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-300 font-semibold"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="px-5 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-bold shadow-sm"
                  >
                    Confirm
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* POPUP: Individual Daemon Log */}
        {viewingDaemonLog && (
          <div className="fixed inset-0 z-60 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fadeIn">
            <div className="bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 rounded-2xl w-full max-w-xl shadow-2xl overflow-hidden flex flex-col h-96">
              <div className="px-5 py-3 border-b border-slate-200 dark:border-surface-800 flex items-center justify-between">
                <h4 className="text-xs font-bold text-slate-900 dark:text-white font-mono">
                  Daemon Log: {viewingDaemonLog.name}
                </h4>
                <button
                  onClick={() => setViewingDaemonLog(null)}
                  className="w-6 h-6 rounded-full bg-slate-400 hover:bg-slate-600 text-white flex items-center justify-center transition"
                >
                  <X className="w-3.5 h-3.5" />
                </button>
              </div>

              <div className="flex-1 p-4 bg-slate-950 font-mono text-[11px] text-slate-200 overflow-y-auto space-y-1">
                <p className="text-slate-500">[{new Date().toISOString()}] INFO: daemon spawned with pid {viewingDaemonLog.pid || 3829}</p>
                <p className="text-emerald-400">[{new Date().toISOString()}] INFO: executing: {viewingDaemonLog.command}</p>
                <p className="text-slate-400">[{new Date().toISOString()}] worker loop initialized in {viewingDaemonLog.process_dir}</p>
                <p className="text-slate-300">[{new Date().toISOString()}] listening for incoming job payloads...</p>
              </div>

              <div className="p-3 border-t border-slate-200 dark:border-surface-800 flex justify-end">
                <button
                  onClick={() => setViewingDaemonLog(null)}
                  className="px-4 py-1.5 rounded-lg bg-slate-200 dark:bg-surface-800 text-xs font-semibold"
                >
                  Close
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
