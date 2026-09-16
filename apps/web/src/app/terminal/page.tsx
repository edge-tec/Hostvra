'use client';

import React, { useState, useEffect, useRef } from 'react';
import {
  Terminal as TerminalIcon,
  Trash2,
  Copy,
  Check,
  Server,
  Folder,
  User,
  Clock,
  Maximize2,
  Minimize2,
  Sparkles,
  Layers,
  Palette,
  Type,
  X,
  Minus,
  Plus,
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
  aborted?: boolean;
}

// Convert ANSI escape codes to styled React elements
function renderAnsi(text: string): React.ReactNode {
  if (!text) return null;

  // Match ANSI escape codes like \u001b[32m or \033[1;34m
  const parts = text.split(/(\u001b\[[0-9;]*m)/g);
  if (parts.length === 1) return text;

  const colorMap: Record<string, string> = {
    '0': '', // reset
    '1': 'font-bold',
    '2': 'opacity-60',
    '3': 'italic',
    '4': 'underline',
    // Standard Foreground
    '30': 'text-slate-900 dark:text-black',
    '31': 'text-rose-500 font-semibold',
    '32': 'text-emerald-400 font-semibold',
    '33': 'text-amber-400 font-semibold',
    '34': 'text-sky-400 font-semibold',
    '35': 'text-fuchsia-400 font-semibold',
    '36': 'text-cyan-400 font-semibold',
    '37': 'text-slate-200',
    // High Intensity Foreground
    '90': 'text-slate-500',
    '91': 'text-red-400 font-bold',
    '92': 'text-emerald-300 font-bold',
    '93': 'text-yellow-300 font-bold',
    '94': 'text-blue-300 font-bold',
    '95': 'text-pink-400 font-bold',
    '96': 'text-cyan-300 font-bold',
    '97': 'text-white font-bold',
  };

  let activeStyles = new Set<string>();
  const elements: React.ReactNode[] = [];

  for (let i = 0; i < parts.length; i++) {
    const part = parts[i];
    const match = part.match(/^\u001b\[([0-9;]*)m$/);
    if (match) {
      const codes = (match[1] || '0').split(';');
      for (const code of codes) {
        if (code === '0' || code === '') {
          activeStyles.clear();
        } else if (colorMap[code]) {
          activeStyles.add(colorMap[code]);
        }
      }
    } else if (part) {
      const className = Array.from(activeStyles).join(' ');
      elements.push(
        className ? (
          <span key={i} className={className}>
            {part}
          </span>
        ) : (
          part
        )
      );
    }
  }

  return elements;
}

// Shorten /root or /home/user paths to ~ like real Unix shells
function formatPath(path: string): string {
  if (!path) return '~';
  if (path === '/root') return '~';
  if (path.startsWith('/root/')) return '~/' + path.slice(6);
  if (path.startsWith('/home/')) {
    const parts = path.split('/');
    if (parts.length >= 3) {
      return '~' + path.slice(parts.slice(0, 3).join('/').length);
    }
  }
  return path;
}

type TerminalTheme = 'macos' | 'ubuntu' | 'matrix';

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
  const [theme, setTheme] = useState<TerminalTheme>('macos');
  const [fontSize, setFontSize] = useState<number>(13);

  const inputRef = useRef<HTMLInputElement>(null);
  const terminalEndRef = useRef<HTMLDivElement>(null);
  const terminalContainerRef = useRef<HTMLDivElement>(null);
  const abortControllerRef = useRef<AbortController | null>(null);

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
  }, [history, isExecuting, autoScroll, command]);

  // Keep focus on the terminal prompt automatically
  useEffect(() => {
    inputRef.current?.focus();
  }, [isExecuting, history]);

  const handleCancel = () => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }
    setIsExecuting(false);
    setHistory((prev) => [
      ...prev,
      {
        id: Math.random().toString(),
        command: command || (commandHistory[commandHistory.length - 1] || ''),
        cwd,
        stdout: '',
        stderr: '^C',
        exitCode: 130,
        durationMs: 0,
        timestamp: new Date().toLocaleTimeString(),
        aborted: true,
      },
    ]);
    setCommand('');
  };

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
          stdout: `Hostvra Cloud OS Shell (x86_64-linux)
==============================================
Available commands & features:
  • Any Linux command : git, npm, pm2, systemctl, mariadb, nginx, docker, etc.
  • cd <directory>    : Navigate directories (state is preserved across commands)
  • clear / Ctrl+L    : Clear terminal screen
  • Ctrl+C            : Interrupt running process or cancel line
  • Up / Down Arrow   : Browse command history
  • Tab               : Autocomplete shell commands
  • Quick chips       : Click predefined buttons above to run diagnostic commands`,
          stderr: '',
          exitCode: 0,
          durationMs: 0,
          timestamp: new Date().toLocaleTimeString(),
        },
      ]);
      setCommand('');
      return;
    }

    if (rawCmd === 'history') {
      const historyList = commandHistory
        .map((cmd, idx) => `  ${String(idx + 1).padStart(4, ' ')}  ${cmd}`)
        .join('\n');
      setHistory((prev) => [
        ...prev,
        {
          id: Math.random().toString(),
          command: 'history',
          cwd,
          stdout: historyList || 'No command history recorded yet.',
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

    const controller = new AbortController();
    abortControllerRef.current = controller;
    const startTs = new Date().toLocaleTimeString();

    try {
      const res = await apiFetch<TerminalExecutionResult>('/api/v1/terminal/execute', {
        method: 'POST',
        signal: controller.signal,
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
            stderr: res.error?.message || 'bash: command failed or rejected by server API',
            exitCode: 1,
            durationMs: 0,
            timestamp: startTs,
          },
        ]);
      }
    } catch (err: any) {
      if (err.name === 'AbortError' || err.message?.includes('aborted')) {
        return;
      }
      let errMsg = err.message || 'Network communication error';
      if (
        errMsg === 'Load failed' ||
        errMsg.includes('Failed to fetch') ||
        errMsg.includes('NetworkError')
      ) {
        errMsg =
          'Connection reset or closed by host server. If you executed a service restart (e.g. systemctl restart hostvra-web), the server restarted. Please refresh your browser page.';
      }
      setHistory((prev) => [
        ...prev,
        {
          id: Math.random().toString(),
          command: rawCmd,
          cwd,
          stdout: '',
          stderr: errMsg,
          exitCode: 1,
          durationMs: 0,
          timestamp: startTs,
        },
      ]);
    } finally {
      setIsExecuting(false);
      abortControllerRef.current = null;
      setCommand('');
      setTimeout(() => inputRef.current?.focus(), 30);
    }
  };

  const commonCommands = [
    'systemctl status',
    'systemctl restart',
    'git status',
    'git pull origin main',
    'git log -n 5',
    'npm run build',
    'pm2 status',
    'pm2 restart all',
    'docker ps',
    'df -h',
    'free -m',
    'uptime',
    'ls -la',
    'cat',
    'nano',
    'mkdir',
    'chmod',
    'chown',
    'mariadb',
    'nginx -t',
    'journalctl -xeu',
    'clear',
    'help',
  ];

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    // Ctrl + C (cancel command or abort current line)
    if (e.ctrlKey && (e.key === 'c' || e.key === 'C')) {
      e.preventDefault();
      if (isExecuting) {
        handleCancel();
      } else {
        // Echo ^C and create fresh line
        setHistory((prev) => [
          ...prev,
          {
            id: Math.random().toString(),
            command: command,
            cwd,
            stdout: '',
            stderr: '^C',
            exitCode: 130,
            durationMs: 0,
            timestamp: new Date().toLocaleTimeString(),
            aborted: true,
          },
        ]);
        setCommand('');
      }
      return;
    }

    // Ctrl + L (clear buffer)
    if (e.ctrlKey && (e.key === 'l' || e.key === 'L')) {
      e.preventDefault();
      setHistory([]);
      return;
    }

    // Tab (Autocompletion)
    if (e.key === 'Tab') {
      e.preventDefault();
      const current = command.trim();
      if (!current) return;
      const match = commonCommands.find((c) => c.startsWith(current) && c !== current);
      if (match) {
        setCommand(match);
      }
      return;
    }

    // Enter (Execute)
    if (e.key === 'Enter') {
      e.preventDefault();
      handleExecute();
      return;
    }

    // History navigation with Up/Down arrows
    if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (commandHistory.length === 0) return;
      const nextIndex =
        historyIndex === -1 ? commandHistory.length - 1 : Math.max(0, historyIndex - 1);
      setHistoryIndex(nextIndex);
      setCommand(commandHistory[nextIndex]);
      return;
    }

    if (e.key === 'ArrowDown') {
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
      return;
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

  // Theme styling configurations
  const themeStyles = {
    macos: {
      bg: 'bg-[#18181b]',
      headerBg: 'bg-[#27272a]/90 border-b border-[#3f3f46]',
      text: 'text-slate-100',
      userColor: 'text-emerald-400',
      pathColor: 'text-sky-400',
      promptChar: 'text-slate-300',
      caretColor: '#34d399',
    },
    ubuntu: {
      bg: 'bg-[#300a24]',
      headerBg: 'bg-[#3e1130] border-b border-[#5a1b47]',
      text: 'text-[#f5f5f5]',
      userColor: 'text-[#8ae234]',
      pathColor: 'text-[#729fcf]',
      promptChar: 'text-white',
      caretColor: '#8ae234',
    },
    matrix: {
      bg: 'bg-[#0a0e14]',
      headerBg: 'bg-[#0f141c] border-b border-[#1f2937]',
      text: 'text-emerald-300',
      userColor: 'text-emerald-400 font-bold',
      pathColor: 'text-cyan-400 font-bold',
      promptChar: 'text-emerald-500 font-bold',
      caretColor: '#10b981',
    },
  }[theme];

  const currentUser = info?.user || 'root';
  const currentHost = info?.hostname || 'vmi3561516';
  const isRoot = currentUser === 'root';

  return (
    <DashboardShell>
      <div
        className={`space-y-5 transition-all ${
          isFullscreen
            ? 'fixed inset-0 z-50 bg-slate-950 p-4 sm:p-6 flex flex-col'
            : ''
        }`}
      >
        {/* Header Title and Quick Badges */}
        {!isFullscreen && (
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div>
              <h1 className="text-2xl font-bold tracking-tight text-slate-900 dark:text-white flex items-center gap-2.5">
                <span className="p-2 rounded-xl bg-indigo-500/10 text-indigo-500 dark:text-indigo-400">
                  <TerminalIcon className="w-5 h-5" />
                </span>
                Web Terminal
                <span className="text-[11px] px-2.5 py-0.5 rounded-full bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 font-semibold flex items-center gap-1.5">
                  <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />
                  Root Shell
                </span>
              </h1>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                Authentic Unix terminal stream with bash execution, ANSI color rendering, and directory tracking.
              </p>
            </div>

            {/* Server Badges */}
            <div className="flex flex-wrap items-center gap-2 text-xs">
              <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-white dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-slate-700 dark:text-slate-200 shadow-2xs">
                <Server className="w-3.5 h-3.5 text-indigo-500" />
                <span className="font-mono font-semibold">{currentHost}</span>
              </div>
              <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-white dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-slate-700 dark:text-slate-200 shadow-2xs">
                <User className="w-3.5 h-3.5 text-emerald-500" />
                <span className="font-mono font-semibold">{currentUser}</span>
              </div>
              <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-white dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-slate-700 dark:text-slate-200 shadow-2xs">
                <Folder className="w-3.5 h-3.5 text-sky-500" />
                <span className="font-mono font-semibold truncate max-w-[150px]">{formatPath(cwd)}</span>
              </div>
            </div>
          </div>
        )}

        {/* Quick Command Chips */}
        {!isFullscreen && (
          <div className="flex items-center gap-1.5 overflow-x-auto pb-1 text-xs no-scrollbar">
            <span className="text-slate-500 dark:text-slate-400 font-bold text-[11px] uppercase tracking-wider whitespace-nowrap mr-1 flex items-center gap-1">
              <Sparkles className="w-3.5 h-3.5 text-indigo-500" /> Quick:
            </span>
            {quickCommands.map((q) => (
              <button
                key={q.cmd}
                onClick={() => handleExecute(q.cmd)}
                disabled={isExecuting}
                className="px-2.5 py-1 rounded-lg bg-white hover:bg-slate-50 dark:bg-surface-800 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-200 border border-slate-200 dark:border-surface-700 text-[11px] font-mono whitespace-nowrap transition-all shadow-2xs active:scale-95 disabled:opacity-40 cursor-pointer"
              >
                {q.label}
              </button>
            ))}
          </div>
        )}

        {/* Real Mac / Linux Terminal Window */}
        <div
          ref={terminalContainerRef}
          className={`flex-1 rounded-2xl overflow-hidden shadow-2xl border border-slate-800/90 ${themeStyles.bg} flex flex-col font-mono select-text transition-colors duration-200 ${
            isFullscreen ? 'h-full min-h-0' : 'min-h-[580px] max-h-[76vh]'
          }`}
          style={{ fontSize: `${fontSize}px` }}
        >
          {/* Authentic macOS Window Titlebar with Traffic Lights */}
          <div className={`h-10 px-4 ${themeStyles.headerBg} flex items-center justify-between select-none`}>
            {/* Window Traffic Lights */}
            <div className="flex items-center gap-2 group/dots">
              <button
                type="button"
                onClick={() => setHistory([])}
                title="Close / Clear Terminal (Ctrl+L)"
                className="w-3 h-3 rounded-full bg-[#ff5f56] hover:brightness-110 flex items-center justify-center transition cursor-pointer shadow-xs"
              >
                <X className="w-2 h-2 text-black/70 opacity-0 group-hover/dots:opacity-100 transition-opacity" />
              </button>
              <button
                type="button"
                onClick={() => setHistory([])}
                title="Minimize Buffer"
                className="w-3 h-3 rounded-full bg-[#ffbd2e] hover:brightness-110 flex items-center justify-center transition cursor-pointer shadow-xs"
              >
                <Minus className="w-2 h-2 text-black/70 opacity-0 group-hover/dots:opacity-100 transition-opacity" />
              </button>
              <button
                type="button"
                onClick={() => setIsFullscreen(!isFullscreen)}
                title={isFullscreen ? 'Exit Fullscreen' : 'Maximize / Fullscreen'}
                className="w-3 h-3 rounded-full bg-[#27c93f] hover:brightness-110 flex items-center justify-center transition cursor-pointer shadow-xs"
              >
                <Plus className="w-2 h-2 text-black/70 opacity-0 group-hover/dots:opacity-100 transition-opacity" />
              </button>
            </div>

            {/* Window Title (Center) */}
            <div className="text-xs text-slate-300 font-mono font-medium flex items-center gap-2 truncate max-w-[280px] sm:max-w-md">
              <span className="text-emerald-400 font-bold">{currentUser}@{currentHost}</span>
              <span className="text-slate-500">:</span>
              <span className="text-sky-400 font-semibold">{formatPath(cwd)}</span>
              <span className="text-slate-500 hidden sm:inline">— bash — 80×24</span>
            </div>

            {/* Terminal Controls (Right) */}
            <div className="flex items-center gap-1.5 text-xs">
              {/* Theme Selector */}
              <div className="flex items-center bg-black/40 rounded-lg p-0.5 border border-white/10 text-[10px]">
                {(['macos', 'ubuntu', 'matrix'] as TerminalTheme[]).map((t) => (
                  <button
                    key={t}
                    type="button"
                    onClick={() => setTheme(t)}
                    className={`px-2 py-0.5 rounded capitalize transition cursor-pointer ${
                      theme === t
                        ? 'bg-white/20 text-white font-bold'
                        : 'text-slate-400 hover:text-white'
                    }`}
                  >
                    {t === 'macos' ? 'macOS' : t === 'ubuntu' ? 'Linux' : 'Matrix'}
                  </button>
                ))}
              </div>

              {/* Font Size Adjust */}
              <div className="hidden sm:flex items-center gap-1 bg-black/40 px-1.5 py-0.5 rounded-lg border border-white/10 text-[11px] text-slate-300">
                <button
                  type="button"
                  onClick={() => setFontSize((f) => Math.max(11, f - 1))}
                  className="px-1 hover:text-white cursor-pointer"
                  title="Smaller Font"
                >
                  A-
                </button>
                <span className="text-[9px] text-slate-500 font-mono">{fontSize}</span>
                <button
                  type="button"
                  onClick={() => setFontSize((f) => Math.min(18, f + 1))}
                  className="px-1 hover:text-white cursor-pointer"
                  title="Larger Font"
                >
                  A+
                </button>
              </div>

              {/* Clear Output */}
              <button
                type="button"
                onClick={() => setHistory([])}
                title="Clear Terminal Output (Ctrl+L)"
                className="p-1 rounded-lg text-slate-400 hover:text-white hover:bg-white/10 transition cursor-pointer"
              >
                <Trash2 className="w-3.5 h-3.5" />
              </button>

              {/* Fullscreen Toggle */}
              <button
                type="button"
                onClick={() => setIsFullscreen(!isFullscreen)}
                title={isFullscreen ? 'Exit Fullscreen' : 'Fullscreen'}
                className="p-1 rounded-lg text-slate-400 hover:text-white hover:bg-white/10 transition cursor-pointer"
              >
                {isFullscreen ? <Minimize2 className="w-3.5 h-3.5" /> : <Maximize2 className="w-3.5 h-3.5" />}
              </button>
            </div>
          </div>

          {/* Terminal Viewport / Screen (Click anywhere to focus) */}
          <div
            onClick={() => inputRef.current?.focus()}
            className="flex-1 overflow-y-auto p-4 space-y-2 cursor-text leading-relaxed font-mono selection:bg-emerald-500/40 selection:text-white"
          >
            {/* Authentic Unix Welcome Header */}
            <div className="text-xs text-slate-500 space-y-0.5 pb-2 border-b border-white/5 select-text">
              <p className="text-slate-400">Last login: {new Date().toLocaleDateString()} on pts/0</p>
              <p className="text-slate-500">
                Hostvra Cloud OS ({info?.os || 'GNU/Linux'} {info?.arch || 'x86_64'}) • {info?.shell || 'bash'}
              </p>
              <p className="text-slate-600">
                Type <span className="text-amber-400">help</span> for shortcuts, <span className="text-amber-400">clear</span> (or Ctrl+L) to wipe buffer.
              </p>
            </div>

            {/* Historical Command & Output Log */}
            {history.map((entry) => (
              <div key={entry.id} className="space-y-0.5 leading-snug group/entry">
                {/* Command Prompt Line */}
                <div className="flex items-center justify-between flex-wrap gap-x-2">
                  <div className="flex items-baseline flex-wrap">
                    <span className={`${themeStyles.userColor} font-bold mr-0.5`}>
                      {currentUser}@{currentHost}
                    </span>
                    <span className="text-slate-500 mr-0.5">:</span>
                    <span className={`${themeStyles.pathColor} font-semibold mr-1.5`}>
                      {formatPath(entry.cwd)}
                    </span>
                    <span className={`${themeStyles.promptChar} font-bold mr-2`}>
                      {isRoot ? '#' : '$'}
                    </span>
                    <span className="text-white font-medium break-all">{entry.command}</span>
                    {entry.aborted && <span className="text-rose-400 font-bold ml-1">^C</span>}
                  </div>

                  <div className="opacity-0 group-hover/entry:opacity-100 transition-opacity flex items-center gap-2 text-[10px] text-slate-500">
                    {entry.durationMs > 0 && <span>{entry.durationMs}ms</span>}
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        handleCopyOutput(entry.id, entry.stdout || entry.stderr);
                      }}
                      className="text-slate-400 hover:text-white p-0.5 rounded cursor-pointer"
                      title="Copy Output"
                    >
                      {copiedId === entry.id ? (
                        <Check className="w-3 h-3 text-emerald-400" />
                      ) : (
                        <Copy className="w-3 h-3" />
                      )}
                    </button>
                    {entry.exitCode !== 0 && !entry.aborted && (
                      <span className="text-rose-400 bg-rose-950/40 px-1 rounded border border-rose-800/40 font-mono">
                        [{entry.exitCode}]
                      </span>
                    )}
                  </div>
                </div>

                {/* Raw Stdout Stream */}
                {entry.stdout && (
                  <div className="whitespace-pre-wrap break-all text-slate-200 py-0.5 pl-0 leading-relaxed select-text font-mono">
                    {renderAnsi(entry.stdout)}
                  </div>
                )}

                {/* Raw Stderr Stream */}
                {entry.stderr && !entry.aborted && (
                  <div
                    className={`whitespace-pre-wrap break-all py-0.5 pl-0 leading-relaxed select-text font-mono ${
                      entry.exitCode === 0 ? 'text-slate-300' : 'text-rose-400'
                    }`}
                  >
                    {renderAnsi(entry.stderr)}
                  </div>
                )}
              </div>
            ))}

            {/* Currently Executing Spinner Banner */}
            {isExecuting && (
              <div className="flex items-center justify-between text-xs py-1 px-2.5 rounded-lg bg-emerald-950/30 border border-emerald-800/40 text-emerald-400 my-1">
                <div className="flex items-center gap-2">
                  <span className="w-2 h-2 rounded-full bg-emerald-400 animate-ping" />
                  <span>Running command on host...</span>
                </div>
                <button
                  type="button"
                  onClick={handleCancel}
                  className="flex items-center gap-1 px-2 py-0.5 text-[11px] rounded bg-rose-950 hover:bg-rose-900 border border-rose-800 text-rose-300 font-mono cursor-pointer transition shadow-2xs"
                >
                  <span>Interrupt</span>
                  <kbd className="text-[9px] bg-black/40 px-1 rounded">Ctrl+C</kbd>
                </button>
              </div>
            )}

            {/* Active Inline Command Prompt (Seamless terminal stream like Mac/Linux) */}
            <div className="flex items-baseline flex-wrap leading-snug pt-1">
              <span className={`${themeStyles.userColor} font-bold mr-0.5 select-none`}>
                {currentUser}@{currentHost}
              </span>
              <span className="text-slate-500 mr-0.5 select-none">:</span>
              <span className={`${themeStyles.pathColor} font-semibold mr-1.5 select-none`}>
                {formatPath(cwd)}
              </span>
              <span className={`${themeStyles.promptChar} font-bold mr-2 select-none`}>
                {isRoot ? '#' : '$'}
              </span>

              {/* Native Continuous Command Input with Blinking Cursor */}
              <div className="relative inline-flex items-center flex-1 min-w-[180px]">
                <input
                  ref={inputRef}
                  type="text"
                  value={command}
                  onChange={(e) => setCommand(e.target.value)}
                  onKeyDown={handleKeyDown}
                  disabled={isExecuting}
                  className="w-full bg-transparent text-white font-mono focus:outline-none border-none p-0 m-0 shadow-none leading-none"
                  style={{
                    caretColor: themeStyles.caretColor,
                    fontSize: `${fontSize}px`,
                  }}
                  autoFocus
                  spellCheck={false}
                  autoComplete="off"
                  autoCorrect="off"
                  autoCapitalize="off"
                />
              </div>
            </div>

            <div ref={terminalEndRef} />
          </div>
        </div>
      </div>
    </DashboardShell>
  );
}
