'use client';

import React, { useState, useEffect } from 'react';
import {
  X,
  FileText,
  Download,
  Trash2,
  Edit3,
  Lock,
  Copy,
  Check,
  ExternalLink,
  Eye,
  FileCode,
  Image as ImageIcon,
  Film,
  Music,
  Archive,
  Info,
  Calendar,
  User,
  Shield,
  HardDrive,
} from 'lucide-react';
import { FileItem } from '@/app/files/page';
import { apiFetch } from '@/lib/api';

interface FilePreviewPanelProps {
  item: FileItem | null;
  onClose: () => void;
  onEdit: (item: FileItem) => void;
  onDownload: (item: FileItem) => void;
  onDelete: (item: FileItem) => void;
  onChmod: (item: FileItem) => void;
  onRename: (item: FileItem) => void;
  activeDomain?: string;
}

export const FilePreviewPanel: React.FC<FilePreviewPanelProps> = ({
  item,
  onClose,
  onEdit,
  onDownload,
  onDelete,
  onChmod,
  onRename,
  activeDomain,
}) => {
  const [copiedPath, setCopiedPath] = useState(false);
  const [copiedUrl, setCopiedUrl] = useState(false);
  const [textContent, setTextContent] = useState<string | null>(null);
  const [loadingContent, setLoadingContent] = useState(false);

  const ext = (item?.extension || '').toLowerCase();
  const isImage = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'ico', 'bmp'].includes(ext);
  const isVideo = ['mp4', 'webm', 'mov', 'mkv'].includes(ext);
  const isAudio = ['mp3', 'wav', 'ogg', 'm4a'].includes(ext);
  const isPdf = ext === 'pdf';
  const isCode = [
    'php', 'js', 'jsx', 'ts', 'tsx', 'html', 'htm', 'css', 'scss', 'json',
    'sql', 'sh', 'bash', 'env', 'conf', 'ini', 'md', 'txt', 'xml', 'yaml', 'yml'
  ].includes(ext);
  const isArchive = ['zip', 'tar', 'gz', 'bz2', 'rar', '7z'].includes(ext);

  // Compute public URL if within domain root
  let publicUrl = '';
  if (item && activeDomain && !item.is_dir) {
    const parts = item.path.split(activeDomain);
    if (parts.length > 1) {
      let sub = parts[1].replace(/\\/g, '/');
      if (sub.startsWith('/public')) sub = sub.replace('/public', '');
      publicUrl = `https://${activeDomain}${sub}`;
    }
  }

  // Fetch small text snippet for code/text files
  useEffect(() => {
    if (item && !item.is_dir && isCode) {
      setLoadingContent(true);
      setTextContent(null);
      apiFetch<{ content: string }>(`/api/v1/files/content?path=${encodeURIComponent(item.path)}`)
        .then((res) => {
          if (res.success && res.data) {
            setTextContent(res.data.content);
          }
        })
        .catch(() => setTextContent(null))
        .finally(() => setLoadingContent(false));
    } else {
      setTextContent(null);
    }
  }, [item?.path, isCode]);

  if (!item) return null;

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  };

  const handleCopy = (text: string, isUrl = false) => {
    navigator.clipboard.writeText(text);
    if (isUrl) {
      setCopiedUrl(true);
      setTimeout(() => setCopiedUrl(false), 2000);
    } else {
      setCopiedPath(true);
      setTimeout(() => setCopiedPath(false), 2000);
    }
  };

  return (
    <div className="w-80 md:w-88 bg-white dark:bg-surface-900 border-l border-slate-200 dark:border-surface-800 flex flex-col h-full shrink-0 shadow-sm animate-fadeIn">
      {/* Header */}
      <div className="flex items-center justify-between p-3.5 border-b border-slate-200 dark:border-surface-800">
        <div className="flex items-center gap-2">
          <Info className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
          <h3 className="text-xs font-bold text-slate-900 dark:text-white uppercase tracking-wider">
            File Details
          </h3>
        </div>
        <button
          type="button"
          onClick={onClose}
          className="p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-surface-800 transition"
        >
          <X className="w-4 h-4" />
        </button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-4 space-y-5 text-xs">
        {/* Visual Preview Box */}
        <div className="w-full aspect-video rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 flex items-center justify-center overflow-hidden relative shadow-inner">
          {item.is_dir ? (
            <div className="text-center p-4">
              <div className="w-12 h-12 rounded-xl bg-amber-100 dark:bg-amber-950/40 text-amber-600 flex items-center justify-center mx-auto mb-2">
                <Info className="w-6 h-6" />
              </div>
              <p className="font-bold text-slate-800 dark:text-white truncate max-w-[200px]">
                {item.name}
              </p>
              <p className="text-[11px] text-slate-400">Directory Folder</p>
            </div>
          ) : isImage ? (
            <img
              src={`/api/v1/files/download?path=${encodeURIComponent(item.path)}`}
              alt={item.name}
              className="w-full h-full object-contain p-2"
              loading="lazy"
            />
          ) : isVideo ? (
            <video
              src={`/api/v1/files/download?path=${encodeURIComponent(item.path)}`}
              controls
              className="w-full h-full object-contain"
            />
          ) : isAudio ? (
            <div className="p-4 w-full">
              <Music className="w-8 h-8 text-sky-500 mx-auto mb-2" />
              <audio
                src={`/api/v1/files/download?path=${encodeURIComponent(item.path)}`}
                controls
                className="w-full h-8"
              />
            </div>
          ) : isPdf ? (
            <div className="text-center p-4">
              <FileText className="w-10 h-10 text-rose-500 mx-auto mb-1" />
              <p className="text-xs font-bold text-slate-700 dark:text-slate-300">PDF Document</p>
              <a
                href={`/api/v1/files/download?path=${encodeURIComponent(item.path)}`}
                target="_blank"
                rel="noreferrer"
                className="text-[11px] text-emerald-600 hover:underline mt-1 inline-flex items-center gap-1"
              >
                <span>Open in Tab</span>
                <ExternalLink className="w-3 h-3" />
              </a>
            </div>
          ) : isCode && textContent ? (
            <div className="w-full h-full p-2.5 overflow-hidden font-mono text-[10px] text-slate-700 dark:text-slate-300 bg-slate-900 text-slate-200">
              <pre className="whitespace-pre-wrap line-clamp-6">{textContent}</pre>
            </div>
          ) : (
            <div className="text-center p-4">
              <FileCode className="w-10 h-10 text-emerald-600 mx-auto mb-1" />
              <p className="text-xs font-bold text-slate-700 dark:text-slate-300 uppercase">
                {ext ? `${ext} File` : 'Binary File'}
              </p>
            </div>
          )}
        </div>

        {/* File Name & Path */}
        <div>
          <h4 className="font-bold text-slate-900 dark:text-white break-words text-sm mb-1">
            {item.name}
          </h4>
          <div className="flex items-center justify-between text-[11px] text-slate-500 bg-slate-50 dark:bg-surface-800 p-2 rounded-lg border border-slate-200 dark:border-surface-700">
            <span className="font-mono truncate mr-2 select-all">{item.path}</span>
            <button
              type="button"
              onClick={() => handleCopy(item.path)}
              title="Copy Full Path"
              className="p-1 rounded text-slate-400 hover:text-slate-700 dark:hover:text-white shrink-0"
            >
              {copiedPath ? <Check className="w-3.5 h-3.5 text-emerald-500" /> : <Copy className="w-3.5 h-3.5" />}
            </button>
          </div>
        </div>

        {/* Public URL if available */}
        {publicUrl && (
          <div>
            <span className="text-[11px] text-slate-400 font-medium mb-1 block">Public URL</span>
            <div className="flex items-center justify-between text-[11px] text-emerald-600 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-950/30 p-2 rounded-lg border border-emerald-200 dark:border-emerald-800">
              <a
                href={publicUrl}
                target="_blank"
                rel="noreferrer"
                className="truncate hover:underline font-mono"
              >
                {publicUrl}
              </a>
              <button
                type="button"
                onClick={() => handleCopy(publicUrl, true)}
                className="p-1 rounded text-emerald-600 hover:text-emerald-700 shrink-0"
              >
                {copiedUrl ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
              </button>
            </div>
          </div>
        )}

        {/* Quick Action Buttons Toolbar */}
        <div className="grid grid-cols-2 gap-2 pt-1">
          {!item.is_dir && (
            <button
              type="button"
              onClick={() => onEdit(item)}
              className="flex items-center justify-center gap-1.5 py-1.5 px-3 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold transition shadow-xs cursor-pointer"
            >
              <Edit3 className="w-3.5 h-3.5" />
              <span>Edit File</span>
            </button>
          )}

          {!item.is_dir && (
            <button
              type="button"
              onClick={() => onDownload(item)}
              className="flex items-center justify-center gap-1.5 py-1.5 px-3 rounded-lg bg-slate-100 hover:bg-slate-200 dark:bg-surface-800 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-200 font-bold transition cursor-pointer"
            >
              <Download className="w-3.5 h-3.5" />
              <span>Download</span>
            </button>
          )}

          <button
            type="button"
            onClick={() => onRename(item)}
            className="flex items-center justify-center gap-1.5 py-1.5 px-3 rounded-lg border border-slate-200 dark:border-surface-700 hover:bg-slate-100 dark:hover:bg-surface-800 text-slate-700 dark:text-slate-300 font-medium transition cursor-pointer"
          >
            <span>Rename (F2)</span>
          </button>

          <button
            type="button"
            onClick={() => onChmod(item)}
            className="flex items-center justify-center gap-1.5 py-1.5 px-3 rounded-lg border border-slate-200 dark:border-surface-700 hover:bg-slate-100 dark:hover:bg-surface-800 text-slate-700 dark:text-slate-300 font-medium transition cursor-pointer"
          >
            <Lock className="w-3.5 h-3.5 text-amber-500" />
            <span>Permissions</span>
          </button>

          <button
            type="button"
            onClick={() => onDelete(item)}
            className="col-span-2 flex items-center justify-center gap-1.5 py-1.5 px-3 rounded-lg bg-rose-50 hover:bg-rose-100 dark:bg-rose-950/40 text-rose-600 font-bold transition border border-rose-200 dark:border-rose-900 cursor-pointer"
          >
            <Trash2 className="w-3.5 h-3.5" />
            <span>Move to Trash</span>
          </button>
        </div>

        {/* Detailed Metadata Grid */}
        <div className="space-y-2.5 pt-2 border-t border-slate-200 dark:border-surface-800 text-[11px]">
          <div className="flex items-center justify-between">
            <span className="text-slate-400 flex items-center gap-1.5">
              <HardDrive className="w-3.5 h-3.5" />
              <span>Size</span>
            </span>
            <span className="font-mono font-bold text-slate-800 dark:text-slate-200">
              {formatBytes(item.size)}
            </span>
          </div>

          <div className="flex items-center justify-between">
            <span className="text-slate-400 flex items-center gap-1.5">
              <Shield className="w-3.5 h-3.5" />
              <span>Permissions</span>
            </span>
            <span className="font-mono font-bold text-emerald-600 dark:text-emerald-400">
              {item.perm_octal || '0644'} ({item.mode})
            </span>
          </div>

          <div className="flex items-center justify-between">
            <span className="text-slate-400 flex items-center gap-1.5">
              <User className="w-3.5 h-3.5" />
              <span>Owner / Group</span>
            </span>
            <span className="font-mono text-slate-700 dark:text-slate-300">
              {item.owner}:{item.group} ({item.uid}:{item.gid})
            </span>
          </div>

          <div className="flex items-center justify-between">
            <span className="text-slate-400 flex items-center gap-1.5">
              <Calendar className="w-3.5 h-3.5" />
              <span>Last Modified</span>
            </span>
            <span className="text-slate-700 dark:text-slate-300">
              {new Date(item.modified_at).toLocaleString()}
            </span>
          </div>
        </div>
      </div>
    </div>
  );
};
