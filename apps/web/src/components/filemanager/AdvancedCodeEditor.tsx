'use client';

import React, { useState, useEffect, useRef } from 'react';
import {
  FileCode,
  Save,
  X,
  Check,
  Search,
  Replace,
  Maximize2,
  Minimize2,
  WrapText,
  Type,
  Code2,
  AlertCircle,
  Copy,
  ChevronDown
} from 'lucide-react';
import { FileItem } from '@/app/files/page';

interface AdvancedCodeEditorProps {
  isOpen: boolean;
  file: FileItem | null;
  content: string;
  onContentChange: (newContent: string) => void;
  onSave: () => Promise<void>;
  onClose: () => void;
  saving: boolean;
  saveSuccess: boolean;
}

type EditorTheme = 'dark' | 'emerald' | 'light';

export const AdvancedCodeEditor: React.FC<AdvancedCodeEditorProps> = ({
  isOpen,
  file,
  content,
  onContentChange,
  onSave,
  onClose,
  saving,
  saveSuccess,
}) => {
  const [initialContent, setInitialContent] = useState('');
  const [isDirty, setIsDirty] = useState(false);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [wordWrap, setWordWrap] = useState(false);
  const [fontSize, setFontSize] = useState<number>(13);
  const [theme, setTheme] = useState<EditorTheme>('dark');
  const [showSearch, setShowSearch] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [replaceQuery, setReplaceQuery] = useState('');
  const [matchCount, setMatchCount] = useState(0);
  const [copied, setCopied] = useState(false);

  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const lineNumbersRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (file) {
      setInitialContent(content);
      setIsDirty(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [file?.path]);

  useEffect(() => {
    setIsDirty(content !== initialContent);
  }, [content, initialContent]);

  // Sync scroll between line numbers and textarea
  const handleScroll = () => {
    if (textareaRef.current && lineNumbersRef.current) {
      lineNumbersRef.current.scrollTop = textareaRef.current.scrollTop;
    }
  };

  // Detect file language
  const getLanguage = () => {
    if (!file) return 'Text';
    const ext = (file.extension || file.name.split('.').pop() || '').toLowerCase().replace(/^\./, '');
    switch (ext) {
      case 'php': return 'PHP';
      case 'js': return 'JavaScript';
      case 'ts': return 'TypeScript';
      case 'tsx': return 'TypeScript React';
      case 'jsx': return 'React JSX';
      case 'json': return 'JSON';
      case 'html': return 'HTML';
      case 'css': return 'CSS';
      case 'scss': return 'SCSS';
      case 'sql': return 'SQL';
      case 'env': return 'DotEnv';
      case 'htaccess': return 'Apache .htaccess';
      case 'conf': return 'Nginx Conf';
      case 'sh': return 'Shell Script';
      case 'yaml':
      case 'yml': return 'YAML';
      case 'xml': return 'XML';
      case 'md': return 'Markdown';
      case 'py': return 'Python';
      case 'go': return 'Go';
      default: return ext.toUpperCase() || 'Plain Text';
    }
  };

  // Search logic
  useEffect(() => {
    if (!searchQuery) {
      setMatchCount(0);
      return;
    }
    const regex = new RegExp(searchQuery.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'gi');
    const matches = content.match(regex);
    setMatchCount(matches ? matches.length : 0);
  }, [searchQuery, content]);

  const handleReplace = () => {
    if (!searchQuery) return;
    const regex = new RegExp(searchQuery.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'i');
    const updated = content.replace(regex, replaceQuery);
    onContentChange(updated);
  };

  const handleReplaceAll = () => {
    if (!searchQuery) return;
    const regex = new RegExp(searchQuery.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'gi');
    const updated = content.replace(regex, replaceQuery);
    onContentChange(updated);
  };

  const handleClose = () => {
    if (isDirty) {
      if (confirm('You have unsaved changes in this file. Are you sure you want to discard them?')) {
        onClose();
      }
    } else {
      onClose();
    }
  };

  if (!isOpen || !file) return null;

  const lines = content.split('\n');
  const lineCount = lines.length;

  // Theme styles
  const getThemeStyles = () => {
    switch (theme) {
      case 'emerald':
        return {
          bg: 'bg-[#031510]',
          text: 'text-emerald-300',
          gutterBg: 'bg-[#020d0a]',
          gutterText: 'text-emerald-800',
          border: 'border-emerald-900/50',
          highlight: 'selection:bg-emerald-800 selection:text-white',
        };
      case 'light':
        return {
          bg: 'bg-white',
          text: 'text-slate-900',
          gutterBg: 'bg-slate-100',
          gutterText: 'text-slate-400',
          border: 'border-slate-300',
          highlight: 'selection:bg-emerald-200 selection:text-slate-900',
        };
      case 'dark':
      default:
        return {
          bg: 'bg-[#0d1117]',
          text: 'text-slate-200',
          gutterBg: 'bg-[#080b0f]',
          gutterText: 'text-slate-600',
          border: 'border-slate-800',
          highlight: 'selection:bg-blue-800 selection:text-white',
        };
    }
  };

  const themeStyles = getThemeStyles();

  return (
    <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-xs flex items-center justify-center p-2 sm:p-4 animate-fadeIn">
      <div
        className={`w-full ${
          isFullscreen ? 'h-full max-w-none rounded-none' : 'max-w-6xl h-[92vh] rounded-2xl'
        } bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 shadow-2xl flex flex-col overflow-hidden font-sans transition-all`}
      >
        {/* Editor Main Header */}
        <div className="px-5 py-2.5 border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-surface-850 flex items-center justify-between gap-3">
          <div className="flex items-center gap-2.5 min-w-0">
            <div className="w-8 h-8 rounded-lg bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600 shrink-0">
              <FileCode className="w-4 h-4" />
            </div>
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <span className="font-bold text-xs text-slate-900 dark:text-white truncate">
                  {file.name}
                </span>
                {isDirty && (
                  <span className="w-2 h-2 rounded-full bg-amber-500 shrink-0" title="Unsaved changes" />
                )}
                <span className="px-2 py-0.2 rounded-md text-[10px] font-bold bg-slate-200 dark:bg-surface-800 text-slate-700 dark:text-slate-300 uppercase">
                  {getLanguage()}
                </span>
              </div>
              <p className="text-[10px] text-slate-400 font-mono truncate max-w-md">
                {file.path}
              </p>
            </div>
          </div>

          {/* Action buttons */}
          <div className="flex items-center gap-2 shrink-0">
            {saveSuccess && (
              <span className="text-xs text-emerald-600 font-bold flex items-center gap-1 animate-fadeIn">
                <Check className="w-3.5 h-3.5" />
                <span className="hidden sm:inline">Saved</span>
              </span>
            )}

            <button
              type="button"
              onClick={onSave}
              disabled={saving}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white font-bold text-xs transition shadow-xs disabled:opacity-50 cursor-pointer"
            >
              <Save className="w-3.5 h-3.5" />
              <span>{saving ? 'Saving...' : 'Save (Ctrl+S)'}</span>
            </button>

            <button
              type="button"
              onClick={() => setIsFullscreen(!isFullscreen)}
              className="p-1.5 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-surface-800 transition"
              title={isFullscreen ? 'Exit Fullscreen' : 'Fullscreen'}
            >
              {isFullscreen ? <Minimize2 className="w-4 h-4" /> : <Maximize2 className="w-4 h-4" />}
            </button>

            <button
              type="button"
              onClick={handleClose}
              className="p-1.5 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-surface-800 transition"
              title="Close Editor"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Editor Controls Sub-Bar */}
        <div className="px-5 py-1.5 border-b border-slate-200 dark:border-surface-800 bg-white dark:bg-surface-900 flex flex-wrap items-center justify-between gap-3 text-xs">
          {/* Left: Search Toggle & Utilities */}
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => setShowSearch(!showSearch)}
              className={`px-2.5 py-1 rounded-lg border text-[11px] font-semibold flex items-center gap-1 transition ${
                showSearch
                  ? 'bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600 border-emerald-300 dark:border-emerald-800'
                  : 'border-slate-200 dark:border-surface-700 text-slate-600 dark:text-slate-300 hover:bg-slate-50'
              }`}
            >
              <Search className="w-3 h-3" />
              <span>Find & Replace</span>
            </button>

            <button
              type="button"
              onClick={() => setWordWrap(!wordWrap)}
              className={`px-2.5 py-1 rounded-lg border text-[11px] font-semibold flex items-center gap-1 transition ${
                wordWrap
                  ? 'bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600 border-emerald-300 dark:border-emerald-800'
                  : 'border-slate-200 dark:border-surface-700 text-slate-600 dark:text-slate-300 hover:bg-slate-50'
              }`}
            >
              <WrapText className="w-3 h-3" />
              <span>Wrap Lines</span>
            </button>

            <button
              type="button"
              onClick={() => {
                navigator.clipboard.writeText(content);
                setCopied(true);
                setTimeout(() => setCopied(false), 2000);
              }}
              className="px-2.5 py-1 rounded-lg border border-slate-200 dark:border-surface-700 text-slate-600 dark:text-slate-300 hover:bg-slate-50 text-[11px] font-semibold flex items-center gap-1 transition"
            >
              {copied ? <Check className="w-3 h-3 text-emerald-600" /> : <Copy className="w-3 h-3" />}
              <span>{copied ? 'Copied' : 'Copy All'}</span>
            </button>
          </div>

          {/* Right: Font Size & Theme */}
          <div className="flex items-center gap-3">
            {/* Font size */}
            <div className="flex items-center gap-1 text-[11px] text-slate-500">
              <span className="font-semibold">Size:</span>
              <button
                type="button"
                onClick={() => setFontSize(Math.max(10, fontSize - 1))}
                className="px-1.5 py-0.5 rounded border border-slate-200 dark:border-surface-700 hover:bg-slate-100 font-mono font-bold"
              >
                -
              </button>
              <span className="font-mono text-slate-800 dark:text-slate-200 w-5 text-center">{fontSize}</span>
              <button
                type="button"
                onClick={() => setFontSize(Math.min(22, fontSize + 1))}
                className="px-1.5 py-0.5 rounded border border-slate-200 dark:border-surface-700 hover:bg-slate-100 font-mono font-bold"
              >
                +
              </button>
            </div>

            {/* Theme switcher */}
            <div className="flex items-center gap-1 text-[11px] text-slate-500">
              <span className="font-semibold">Theme:</span>
              <select
                value={theme}
                onChange={(e) => setTheme(e.target.value as EditorTheme)}
                className="bg-slate-100 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 rounded-lg px-2 py-0.5 text-slate-800 dark:text-slate-200 font-medium focus:outline-none cursor-pointer"
              >
                <option value="dark">Midnight Dark</option>
                <option value="emerald">Matrix Emerald</option>
                <option value="light">Paper Light</option>
              </select>
            </div>
          </div>
        </div>

        {/* Find & Replace Bar */}
        {showSearch && (
          <div className="px-5 py-2 bg-slate-100 dark:bg-surface-850 border-b border-slate-200 dark:border-surface-800 flex flex-wrap items-center gap-2 text-xs animate-slideDown">
            <div className="flex items-center gap-1 bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 rounded-lg px-2 py-1">
              <Search className="w-3.5 h-3.5 text-slate-400" />
              <input
                type="text"
                placeholder="Find text..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-36 sm:w-48 bg-transparent text-slate-900 dark:text-white font-mono text-[11px] focus:outline-none"
              />
              {searchQuery && (
                <span className="text-[10px] text-slate-400 font-mono pl-1">
                  {matchCount} match{matchCount === 1 ? '' : 'es'}
                </span>
              )}
            </div>

            <div className="flex items-center gap-1 bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 rounded-lg px-2 py-1">
              <Replace className="w-3.5 h-3.5 text-slate-400" />
              <input
                type="text"
                placeholder="Replace with..."
                value={replaceQuery}
                onChange={(e) => setReplaceQuery(e.target.value)}
                className="w-36 sm:w-48 bg-transparent text-slate-900 dark:text-white font-mono text-[11px] focus:outline-none"
              />
            </div>

            <button
              type="button"
              onClick={handleReplace}
              disabled={!searchQuery || matchCount === 0}
              className="px-2.5 py-1 rounded-lg bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 font-semibold text-[11px] hover:bg-slate-50 disabled:opacity-50"
            >
              Replace
            </button>

            <button
              type="button"
              onClick={handleReplaceAll}
              disabled={!searchQuery || matchCount === 0}
              className="px-2.5 py-1 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-semibold text-[11px] disabled:opacity-50"
            >
              Replace All
            </button>
          </div>
        )}

        {/* Code Workspace with Line Numbers */}
        <div className={`flex-1 flex overflow-hidden ${themeStyles.bg} relative font-mono`}>
          {/* Line Numbers Gutter */}
          <div
            ref={lineNumbersRef}
            className={`w-14 shrink-0 py-4 select-none overflow-hidden text-right pr-3 font-mono border-r ${themeStyles.border} ${themeStyles.gutterBg} ${themeStyles.gutterText}`}
            style={{ fontSize: `${fontSize}px`, lineHeight: '1.6' }}
          >
            {Array.from({ length: lineCount }).map((_, i) => (
              <div key={i}>{i + 1}</div>
            ))}
          </div>

          {/* Textarea Code Body */}
          <textarea
            ref={textareaRef}
            value={content}
            onChange={(e) => onContentChange(e.target.value)}
            onScroll={handleScroll}
            onKeyDown={(e) => {
              if ((e.ctrlKey || e.metaKey) && e.key === 's') {
                e.preventDefault();
                onSave();
              }
              // Tab support: insert 2 spaces
              if (e.key === 'Tab') {
                e.preventDefault();
                const start = e.currentTarget.selectionStart;
                const end = e.currentTarget.selectionEnd;
                const newContent = content.substring(0, start) + '  ' + content.substring(end);
                onContentChange(newContent);
                setTimeout(() => {
                  if (textareaRef.current) {
                    textareaRef.current.selectionStart = textareaRef.current.selectionEnd = start + 2;
                  }
                }, 0);
              }
            }}
            wrap={wordWrap ? 'on' : 'off'}
            spellCheck={false}
            className={`flex-1 p-4 bg-transparent ${themeStyles.text} ${themeStyles.highlight} focus:outline-none resize-none leading-relaxed overflow-auto`}
            style={{
              fontSize: `${fontSize}px`,
              lineHeight: '1.6',
              tabSize: 2,
              whiteSpace: wordWrap ? 'pre-wrap' : 'pre',
            }}
          />
        </div>

        {/* Editor Status Bar Footer */}
        <div className="px-5 py-2 border-t border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-surface-850 flex items-center justify-between text-xs text-slate-500 font-mono">
          <div className="flex items-center gap-4">
            <span>{lineCount} lines</span>
            <span>{content.length.toLocaleString()} characters</span>
            <span>UTF-8</span>
          </div>

          <div className="flex items-center gap-3">
            {isDirty && (
              <span className="text-amber-600 dark:text-amber-400 font-bold">Unsaved Changes</span>
            )}
            <span>Press Ctrl+S to save • Tab for indent</span>
          </div>
        </div>
      </div>
    </div>
  );
};
