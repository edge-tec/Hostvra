'use client';

import React, { useState } from 'react';
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
} from 'lucide-react';
import { AppPackage } from '@/lib/api';
import { getAppLaunchTarget, isAppPinned, togglePinApp } from '@/lib/appstore-utils';

interface AppControlModalProps {
  app: AppPackage | null;
  isOpen: boolean;
  onClose: () => void;
  onServiceControl?: (app: AppPackage, action: 'start' | 'stop' | 'restart') => Promise<void>;
  isActing?: boolean;
}

export function AppControlModal({
  app,
  isOpen,
  onClose,
  onServiceControl,
  isActing = false,
}: AppControlModalProps) {
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const [pinned, setPinned] = useState<boolean>(app ? isAppPinned(app.id) : false);

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
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm animate-fadeIn">
      <div className="bg-surface-900 border border-surface-750 rounded-2xl w-full max-w-xl shadow-2xl overflow-hidden flex flex-col max-h-[90vh]">
        {/* Header */}
        <div className="p-6 border-b border-surface-800 flex items-start justify-between gap-4">
          <div className="flex items-center gap-4">
            <div className="w-12 h-12 rounded-xl bg-surface-800 border border-surface-700 flex items-center justify-center shadow-inner flex-shrink-0">
              {getCategoryIcon(app.category)}
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h3 className="text-lg font-bold text-white tracking-tight">{app.display_name}</h3>
                <span className="text-[11px] font-mono px-2 py-0.5 rounded-full bg-surface-800 text-slate-300 border border-surface-700">
                  v{app.version}
                </span>
              </div>
              <div className="flex items-center gap-2 mt-1">
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

        {/* Content Body */}
        <div className="p-6 space-y-6 overflow-y-auto">
          {/* Description */}
          <p className="text-xs text-slate-300 leading-relaxed bg-surface-950/40 p-3.5 rounded-xl border border-surface-800/80">
            {app.description}
          </p>

          {/* Live Status & Quick Action Card */}
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
                  className={`px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition-all shadow-sm ${
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
                  className="px-2.5 py-1.5 rounded-lg bg-surface-800 hover:bg-surface-700 text-slate-300 hover:text-white text-xs font-medium border border-surface-700 flex items-center gap-1 transition-all disabled:opacity-50"
                  title="Restart Service"
                >
                  <RotateCw className="w-3.5 h-3.5" />
                  Restart
                </button>
              </div>
            )}
          </div>

          {/* Launch Options Section */}
          <div className="space-y-3">
            <h4 className="text-xs font-bold uppercase tracking-wider text-slate-400 flex items-center gap-2">
              <Zap className="w-3.5 h-3.5 text-amber-400" />
              Launch & Management Options
            </h4>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              {/* Option 1: Web GUI or Dedicated Hostvra Route */}
              {target.type === 'route' && target.url && (
                <Link
                  href={target.url}
                  onClick={onClose}
                  className="p-3.5 rounded-xl bg-indigo-600/10 hover:bg-indigo-600/20 border border-indigo-500/30 text-indigo-300 hover:text-indigo-200 transition-all flex items-center justify-between group shadow-sm"
                >
                  <div className="flex items-center gap-2.5">
                    <Boxes className="w-4 h-4 text-indigo-400" />
                    <span className="text-xs font-semibold">{target.label}</span>
                  </div>
                  <ExternalLink className="w-3.5 h-3.5 opacity-70 group-hover:opacity-100 transition-opacity" />
                </Link>
              )}

              {target.type === 'external' && target.url && (
                <a
                  href={target.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="p-3.5 rounded-xl bg-emerald-600/10 hover:bg-emerald-600/20 border border-emerald-500/30 text-emerald-300 hover:text-emerald-200 transition-all flex items-center justify-between group shadow-sm"
                >
                  <div className="flex items-center gap-2.5">
                    <ExternalLink className="w-4 h-4 text-emerald-400" />
                    <div>
                      <span className="text-xs font-semibold block">{target.label}</span>
                      <span className="text-[10px] text-slate-400 font-mono">{target.badge}</span>
                    </div>
                  </div>
                  <ExternalLink className="w-3.5 h-3.5 opacity-70 group-hover:opacity-100 transition-opacity" />
                </a>
              )}

              {/* Option 2: Open in Web Terminal */}
              <Link
                href={`/terminal?cmd=${encodeURIComponent(target.terminalCmd || `${app.id} --help`)}`}
                onClick={onClose}
                className="p-3.5 rounded-xl bg-surface-800 hover:bg-surface-750 border border-surface-700 text-slate-200 transition-all flex items-center justify-between group shadow-sm"
              >
                <div className="flex items-center gap-2.5">
                  <Terminal className="w-4 h-4 text-emerald-400" />
                  <div>
                    <span className="text-xs font-semibold block">Open in Terminal</span>
                    <span className="text-[10px] text-slate-400 font-mono">
                      {target.terminalCmd || 'CLI command'}
                    </span>
                  </div>
                </div>
                <ExternalLink className="w-3.5 h-3.5 opacity-70 group-hover:opacity-100 transition-opacity" />
              </Link>
            </div>
          </div>

          {/* System Paths and Details */}
          <div className="space-y-3 pt-2 border-t border-surface-800">
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

        {/* Footer with Pin to Dashboard */}
        <div className="p-4 bg-surface-950 border-t border-surface-800 flex items-center justify-between">
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
