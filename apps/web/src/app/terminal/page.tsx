'use client';

import React, { useState, useEffect, useRef } from 'react';
import {
  Terminal as TerminalIcon,
  Play,
  RotateCcw,
  Trash2,
  Copy,
  Check,
  Server,
  Folder,
  User,
  Cpu,
  Clock,
  CheckCircle2,
  AlertCircle,
  HelpCircle,
  Maximize2,
  Minimize2,
  ArrowRight,
  Sparkles,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import { apiFetch, TerminalInfo, TerminalExecutionResult } from '@/lib/api';

interface TerminalEntry {
  id: string;
  command: string;
  cwd: string;
  stdout: string;
  stderr: string;
  exitCode: number;
  durationMs: number;
  timestamp: string;
}

export default function TerminalPage() {
  const [info, setInfo] = useState<TerminalInfo | null>(null);
  const [cwd, setCwd] = useState<string>('/root/Hostvra');
  const [command, setCommand] = useState('');
  const [history, setHistory] = useState<TerminalEntry[]>([]);
  const [commandHistory, setCommandHistory] = useState<string[]>([]);
  const [historyIndex, setHistoryIndex] = useState<number>(-1);
  const [isExecuting, setIsExecuting] = useState(false);
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [autoScroll, setAutoScroll] = useState(true);

  const inputRef = useRef<HTMLInputElement>(null);
  const terminalEndRef = useRef<HTMLDivElement>(null);
  const terminalContainerRef = useRef<HTMLDivElement>(null);

  // Fetch initial terminal environment metadata
  useEffect(() => {
    async function loadInfo() {
      const res = await apiFetch<TerminalInfo>('/api/v1/terminal/info');
      if (res.success && res.data) {
        setInfo(res.data);
        if (res.data.default_cwd) {
          setCwd(res.data.default_cwd);
        }
      }
    }
    loadInfo();

    if (typeof window !== 'undefined') {
      const params = new URLSearchParams(window.location.search);
      const initialCmd = params.get('cmd');
      if (initialCmd) {
        setCommand(initialCmd);
      }
    }
  }, []);

  // Auto-scroll terminal to bottom when new entries arrive
  useEffect(() => {
    if (autoScroll && terminalEndRef.current) {
      terminalEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [history, isExecuting, autoScroll]);

  // Focus input automatically
  useEffect(() => {
    inputRef.current?.focus();
  }, [isExecuting]);

  // Execute terminal command
  const handleExecute = async (cmdToRun?: string) => {
    const rawCmd = (cmdToRun !== undefined ? cmdToRun : command).trim();
    if (!rawCmd || isExecuting) return;

    // Built-in client commands
    if (rawCmd === 'clear') {
      setHistory([]);
      setCommand('');
      return;
    }

    if (rawCmd === 'help') {
      setHistory((prev) => [
        ...prev,
        {
          id: Math.random().toString(),
          command: 'help',
          cwd,
          stdout: `Hostvra Integrated Web Terminal
================================
Available shortcuts & capabilities:
  - Any server command: git, npm, systemctl, docker, apt, etc.
  - cd <path>          : Navigate between directories (retains active state)
  - clear              : Clear terminal screen output
  - Up / Down Arrow    : Browse command history
  - Ctrl + L           : Clear console buffer
  - Pre-built chips    : Quick execution badges above the prompt`,
          stderr: '',
          exitCode: 0,
          durationMs: 0,
          timestamp: new Date().toLocaleTimeString(),
        },
      ]);
      setCommand('');
      return;
    }

    setIsExecuting(true);
    setCommandHistory((prev) => [...prev, rawCmd]);
    setHistoryIndex(-1);

    const startTs = new Date().toLocaleTimeString();

    try {
      const res = await apiFetch<TerminalExecutionResult>('/api/v1/terminal/execute', {
        method: 'POST',
        body: JSON.stringify({
          command: rawCmd,
          cwd,
        }),
      });

      if (res.success && res.data) {
        const result = res.data;
        if (result.cwd) {
          setCwd(result.cwd);
        }
        setHistory((prev) => [
          ...prev,
          {
            id: Math.random().toString(),
            command: rawCmd,
            cwd: result.cwd || cwd,
            stdout: result.stdout,
            stderr: result.stderr,
            exitCode: result.exit_code,
            durationMs: result.duration_ms,
            timestamp: startTs,
          },
        ]);
      } else {
        setHistory((prev) => [
          ...prev,
          {
            id: Math.random().toString(),
            command: rawCmd,
            cwd,
            stdout: '',
            stderr: res.error?.message || 'Execution failed or rejected by server API',
            exitCode: 1,
            durationMs: 0,
            timestamp: startTs,
          },
        ]);
      }
    } catch (err: any) {
      setHistory((prev) => [
        ...prev,
        {
          id: Math.random().toString(),
          command: rawCmd,
          cwd,
          stdout: '',
          stderr: err.message || 'Network communication error',
          exitCode: 1,
          durationMs: 0,
          timestamp: startTs,
        },
      ]);
    } finally {
      setIsExecuting(false);
      setCommand('');
      setTimeout(() => inputRef.current?.focus(), 50);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleExecute();
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (commandHistory.length === 0) return;
      const nextIndex = historyIndex === -1 ? commandHistory.length - 1 : Math.max(0, historyIndex - 1);
      setHistoryIndex(nextIndex);
      setCommand(commandHistory[nextIndex]);
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (commandHistory.length === 0 || historyIndex === -1) return;
      const nextIndex = historyIndex + 1;
      if (nextIndex >= commandHistory.length) {
        setHistoryIndex(-1);
        setCommand('');
      } else {
        setHistoryIndex(nextIndex);
        setCommand(commandHistory[nextIndex]);
      }
    } else if (e.ctrlKey && e.key === 'l') {
      e.preventDefault();
      setHistory([]);
    }
  };

  const handleCopyOutput = (id: string, text: string) => {
    navigator.clipboard.writeText(text);
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 2000);
  };

  const quickCommands = [
    { label: 'Git Status', cmd: 'git status' },
    { label: 'Git Pull', cmd: 'git pull origin main' },
    { label: 'API Service', cmd: 'systemctl status hostvra-api' },
    { label: 'Web UI Service', cmd: 'systemctl status hostvra-web' },
    { label: 'Uptime', cmd: 'uptime' },
    { label: 'Memory (RAM)', cmd: 'free -m' },
    { label: 'Disk Space', cmd: 'df -h' },
    { label: 'Docker Containers', cmd: 'docker ps' },
  ];

  return (
    <DashboardShell>
      <div className={`space-y-6 ${isFullscreen ? 'fixed inset-4 z-50 bg-slate-950 p-6 rounded-2xl shadow-2xl border border-slate-800 flex flex-col' : ''}`}>
        {/* Header Title and Controls */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-slate-900 dark:text-white flex items-center gap-2.5">
              <span className="p-2 rounded-xl bg-indigo-500/10 text-indigo-500 dark:text-indigo-400">
                <TerminalIcon className="w-6 h-6" />
              </span>
              Web Terminal
              <span className="text-xs px-2.5 py-0.5 rounded-full bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 font-medium flex items-center gap-1.5">
                <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />
                Root Console
              </span>
            </h1>
            <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
              Execute live server maintenance commands, git updates, and diagnostics directly from the browser.
            </p>
          </div>

          {/* Node Metadata Badges */}
          <div className="flex flex-wrap items-center gap-2">
            <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-surface-100 dark:bg-surface-800 border border-surface-200 dark:border-surface-700 text-xs text-slate-600 dark:text-slate-300">
              <Server className="w-3.5 h-3.5 text-indigo-500" />
              <span className="font-mono font-medium">{info?.hostname || 'localhost'}</span>
            </div>
            <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-surface-100 dark:bg-surface-800 border border-surface-200 dark:border-surface-700 text-xs text-slate-600 dark:text-slate-300">
              <User className="w-3.5 h-3.5 text-indigo-500" />
              <span className="font-mono font-medium">{info?.user || 'root'}</span>
            </div>
            <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-surface-100 dark:bg-surface-800 border border-surface-200 dark:border-surface-700 text-xs text-slate-600 dark:text-slate-300">
              <Folder className="w-3.5 h-3.5 text-indigo-500" />
              <span className="font-mono font-medium truncate max-w-[150px]">{cwd}</span>
            </div>
          </div>
        </div>

        {/* Quick Action Chips */}
        <div className="flex items-center gap-2 overflow-x-auto pb-1 text-xs">
          <span className="text-slate-400 font-medium text-[11px] uppercase tracking-wider whitespace-nowrap mr-1 flex items-center gap-1">
            <Sparkles className="w-3 h-3 text-indigo-400" /> Quick:
          </span>
          {quickCommands.map((q) => (
            <button
              key={q.cmd}
              onClick={() => handleExecute(q.cmd)}
              disabled={isExecuting}
              className="px-2.5 py-1 rounded-md bg-surface-100 dark:bg-surface-800 hover:bg-indigo-50 dark:hover:bg-indigo-950/40 hover:text-indigo-600 dark:hover:text-indigo-400 text-slate-600 dark:text-slate-300 border border-surface-200 dark:border-surface-700 whitespace-nowrap transition-all duration-150 active:scale-95 disabled:opacity-50"
            >
              {q.label}
            </button>
          ))}
        </div>

        {/* Terminal Window Box */}
        <div
          ref={terminalContainerRef}
          className={`flex-1 rounded-2xl overflow-hidden shadow-2xl border border-slate-800 bg-[#0c1017] flex flex-col font-mono text-[13px] ${
            isFullscreen ? 'min-h-0' : 'min-h-[580px] max-h-[75vh]'
          }`}
        >
          {/* Mac-style Window Top Bar */}
          <div className="h-10 bg-[#161b22] px-4 border-b border-slate-800 flex items-center justify-between select-none">
            <div className="flex items-center gap-2">
              <span className="w-3 h-3 rounded-full bg-[#ff5f56] inline-block shadow-sm" />
              <span className="w-3 h-3 rounded-full bg-[#ffbd2e] inline-block shadow-sm" />
              <span className="w-3 h-3 rounded-full bg-[#27c93f] inline-block shadow-sm" />
              <span className="ml-3 text-xs text-slate-400 font-sans font-medium flex items-center gap-1.5">
                <TerminalIcon className="w-3.5 h-3.5 text-indigo-400" />
                hostvra-terminal — bash — {info?.hostname || 'node'}
              </span>
            </div>

            <div className="flex items-center gap-2">
              <button
                onClick={() => setAutoScroll(!autoScroll)}
                title={autoScroll ? 'Auto-scroll is Enabled' : 'Auto-scroll is Paused'}
                className={`text-xs px-2 py-0.5 rounded font-sans transition-colors ${
                  autoScroll
                    ? 'text-emerald-400 bg-emerald-500/10'
                    : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                Auto-scroll
              </button>
              <button
                onClick={() => setHistory([])}
                title="Clear Terminal Output (Ctrl+L)"
                className="p-1 rounded text-slate-400 hover:text-white hover:bg-slate-800 transition-colors"
              >
                <Trash2 className="w-3.5 h-3.5" />
              </button>
              <button
                onClick={() => setIsFullscreen(!isFullscreen)}
                title={isFullscreen ? 'Exit Fullscreen' : 'Fullscreen'}
                className="p-1 rounded text-slate-400 hover:text-white hover:bg-slate-800 transition-colors"
              >
                {isFullscreen ? <Minimize2 className="w-3.5 h-3.5" /> : <Maximize2 className="w-3.5 h-3.5" />}
              </button>
            </div>
          </div>

          {/* Terminal Body Screen */}
          <div
            onClick={() => inputRef.current?.focus()}
            className="flex-1 overflow-y-auto p-4 space-y-4 cursor-text font-mono selection:bg-indigo-500 selection:text-white"
          >
            {/* Terminal Splash Welcome */}
            <div className="text-slate-500 text-xs pb-3 border-b border-slate-800/80 leading-relaxed space-y-1">
              <p className="text-indigo-400 font-semibold">Hostvra Core OS Terminal v1.0.0 (x86_64-linux)</p>
              <p>Type <span className="text-amber-300">help</span> for guidance, <span className="text-amber-300">clear</span> to wipe buffer, or run bash commands directly.</p>
              <p className="text-slate-600">Active Node: {info?.hostname || 'localhost'} | Working Directory: {cwd}</p>
            </div>

            {/* Execution History */}
            {history.map((entry) => (
              <div key={entry.id} className="space-y-1 group">
                {/* Prompt & Command Header */}
                <div className="flex items-center justify-between text-xs pt-1">
                  <div className="flex items-center gap-1.5 flex-wrap">
                    <span className="text-emerald-400 font-semibold">{info?.user || 'root'}@{info?.hostname || 'hostvra'}</span>
                    <span className="text-slate-500">:</span>
                    <span className="text-indigo-400 font-medium">{entry.cwd}</span>
                    <span className="text-slate-400">#</span>
                    <span className="text-white font-semibold">{entry.command}</span>
                  </div>

                  <div className="flex items-center gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                    <span className="text-[11px] text-slate-500 flex items-center gap-1">
                      <Clock className="w-3 h-3" />
                      {entry.durationMs}ms
                    </span>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        handleCopyOutput(entry.id, entry.stdout || entry.stderr);
                      }}
                      className="text-slate-400 hover:text-white p-1 rounded hover:bg-slate-800"
                      title="Copy Output"
                    >
                      {copiedId === entry.id ? (
                        <Check className="w-3.5 h-3.5 text-emerald-400" />
                      ) : (
                        <Copy className="w-3.5 h-3.5" />
                      )}
                    </button>
                    {entry.exitCode === 0 ? (
                      <span className="text-[10px] px-1.5 py-0.5 rounded bg-emerald-500/20 text-emerald-400">
                        exit 0
                      </span>
                    ) : (
                      <span className="text-[10px] px-1.5 py-0.5 rounded bg-rose-500/20 text-rose-400">
                        exit {entry.exitCode}
                      </span>
                    )}
                  </div>
                </div>

                {/* Command Output */}
                {entry.stdout && (
                  <pre className="text-slate-300 whitespace-pre-wrap break-all pl-2 leading-relaxed bg-[#080b10]/60 p-2.5 rounded-lg border border-slate-900/80">
                    {entry.stdout}
                  </pre>
                )}

                {entry.stderr && (
                  <pre
                    className={`whitespace-pre-wrap break-all pl-2 leading-relaxed p-2.5 rounded-lg border ${
                      entry.exitCode === 0
                        ? 'text-slate-400 bg-slate-900/40 border-slate-800/80'
                        : 'text-rose-400 bg-rose-950/20 border-rose-900/30'
                    }`}
                  >
                    {entry.stderr}
                  </pre>
                )}
              </div>
            ))}

            {/* In-Flight Command Spinner Indicator */}
            {isExecuting && (
              <div className="flex items-center gap-2 text-xs text-indigo-400 py-1 pl-1">
                <span className="w-2 h-2 rounded-full bg-indigo-500 animate-ping" />
                <span>Executing on host...</span>
              </div>
            )}

            {/* Active Interactive Command Prompt Line */}
            <div className="flex items-center gap-2 text-xs pt-2">
              <span className="text-emerald-400 font-semibold whitespace-nowrap">
                {info?.user || 'root'}@{info?.hostname || 'hostvra'}
              </span>
              <span className="text-slate-500">:</span>
              <span className="text-indigo-400 font-medium whitespace-nowrap max-w-[200px] truncate">
                {cwd}
              </span>
              <span className="text-slate-400">#</span>
              <input
                ref={inputRef}
                type="text"
                value={command}
                onChange={(e) => setCommand(e.target.value)}
                onKeyDown={handleKeyDown}
                disabled={isExecuting}
                placeholder={isExecuting ? 'Command in progress...' : 'Type a command (e.g. git status, systemctl status)...'}
                className="flex-1 bg-transparent text-white focus:outline-none placeholder:text-slate-600 font-mono text-[13px] disabled:opacity-50"
                autoFocus
                spellCheck={false}
                autoComplete="off"
              />
              <button
                onClick={() => handleExecute()}
                disabled={!command.trim() || isExecuting}
                className="px-2.5 py-1 rounded bg-indigo-600 hover:bg-indigo-500 disabled:opacity-30 text-white text-xs font-sans font-medium flex items-center gap-1 transition-all"
              >
                <Play className="w-3 h-3 fill-current" />
                Run
              </button>
            </div>

            <div ref={terminalEndRef} />
          </div>
        </div>
      </div>
    </DashboardShell>
  );
}
