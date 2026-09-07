'use client';

import React, { useState, useEffect, useRef } from 'react';
import {
  Folder,
  FileText,
  Trash2,
  Edit,
  Download,
  FolderPlus,
  FilePlus,
  ChevronRight,
  RefreshCw,
  Search,
  X,
  Save,
  ShieldCheck,
  UploadCloud,
  Archive,
  FolderArchive,
  ArrowUp,
  FileCode,
  Lock,
  Copy,
  AlertCircle,
  CheckCircle2,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import { apiFetch, getApiBaseUrl } from '@/lib/api';

export interface FileItem {
  name: string;
  path: string;
  size: number;
  mode: string;
  perm_octal: string;
  owner: string;
  group: string;
  uid: number;
  gid: number;
  is_dir: boolean;
  modified_at: string;
  extension: string;
}

interface FileListResponse {
  current_path: string;
  items: FileItem[];
  count: number;
}

export default function FileManagerPage() {
  const [currentPath, setCurrentPath] = useState('/var/www');
  const [files, setFiles] = useState<FileItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [search, setSearch] = useState('');

  // Editor Modal State
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingFile, setEditingFile] = useState<FileItem | null>(null);
  const [fileContent, setFileContent] = useState('');
  const [savingFile, setSavingFile] = useState(false);
  const [saveSuccess, setSaveSuccess] = useState(false);

  // New Folder / File Modal
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [createType, setCreateType] = useState<'file' | 'folder'>('file');
  const [createName, setCreateName] = useState('');
  const [creating, setCreating] = useState(false);

  // Upload Modal State
  const [uploadModalOpen, setUploadModalOpen] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Rename Modal State
  const [renameModalOpen, setRenameModalOpen] = useState(false);
  const [itemToRename, setItemToRename] = useState<FileItem | null>(null);
  const [newName, setNewName] = useState('');

  // Permissions Modal State
  const [permModalOpen, setPermModalOpen] = useState(false);
  const [itemForPerm, setItemForPerm] = useState<FileItem | null>(null);
  const [permMode, setPermMode] = useState('0755');

  // Archive & Extract States
  const [archiveModalOpen, setArchiveModalOpen] = useState(false);
  const [itemToArchive, setItemToArchive] = useState<FileItem | null>(null);
  const [archiveFormat, setArchiveFormat] = useState<'zip' | 'tar.gz'>('zip');
  const [isProcessingArchive, setIsProcessingArchive] = useState(false);

  // Fetch directory listing from real backend
  const fetchDirectory = async (path: string) => {
    try {
      setLoading(true);
      setErrorMsg(null);
      const res = await apiFetch<FileListResponse>(`/api/v1/files/list?path=${encodeURIComponent(path)}`);
      if (res.success && res.data) {
        setFiles(res.data.items || []);
        setCurrentPath(res.data.current_path || path);
      } else {
        setErrorMsg(res.error?.message || 'Failed to load directory');
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Error communicating with host filesystem API');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchDirectory(currentPath);
  }, []);

  // Open file or directory
  const handleItemClick = async (file: FileItem) => {
    if (file.is_dir) {
      fetchDirectory(file.path);
    } else {
      // Open text editor for editable files
      try {
        setEditingFile(file);
        setSaveSuccess(false);
        const res = await apiFetch<{ content: string }>(`/api/v1/files/content?path=${encodeURIComponent(file.path)}`);
        if (res.success && res.data) {
          setFileContent(res.data.content);
          setEditorOpen(true);
        } else {
          alert(res.error?.message || 'Failed to read file content');
        }
      } catch (err: any) {
        alert(err.message || 'Cannot open file');
      }
    }
  };

  // Save File Content with atomic write and .bak snapshot
  const handleSaveContent = async () => {
    if (!editingFile) return;
    try {
      setSavingFile(true);
      const res = await apiFetch('/api/v1/files/content', {
        method: 'PUT',
        body: JSON.stringify({
          path: editingFile.path,
          content: fileContent,
        }),
      });

      if (res.success) {
        setSaveSuccess(true);
        setTimeout(() => setSaveSuccess(false), 3000);
        fetchDirectory(currentPath);
      } else {
        alert(res.error?.message || 'Failed to save file');
      }
    } catch (err: any) {
      alert(err.message || 'Failed to save file');
    } finally {
      setSavingFile(false);
    }
  };

  // Keyboard shortcut Ctrl+S / Cmd+S in Editor
  const handleEditorKeyDown = (e: React.KeyboardEvent) => {
    if ((e.ctrlKey || e.metaKey) && e.key === 's') {
      e.preventDefault();
      handleSaveContent();
    }
  };

  // Create new folder or file
  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!createName.trim()) return;

    try {
      setCreating(true);
      const targetPath = `${currentPath.replace(/\/$/, '')}/${createName.trim()}`;

      if (createType === 'folder') {
        const res = await apiFetch('/api/v1/files/mkdir', {
          method: 'POST',
          body: JSON.stringify({ path: targetPath }),
        });
        if (!res.success) throw new Error(res.error?.message || 'Failed to create directory');
      } else {
        const res = await apiFetch('/api/v1/files/content', {
          method: 'PUT',
          body: JSON.stringify({ path: targetPath, content: '' }),
        });
        if (!res.success) throw new Error(res.error?.message || 'Failed to create file');
      }

      setCreateModalOpen(false);
      setCreateName('');
      fetchDirectory(currentPath);
    } catch (err: any) {
      alert(err.message || 'Creation failed');
    } finally {
      setCreating(false);
    }
  };

  // Upload file stream
  const handleUploadSubmit = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const uploadedFiles = e.target.files;
    if (!uploadedFiles || uploadedFiles.length === 0) return;

    setUploading(true);
    setUploadProgress(10);

    for (let i = 0; i < uploadedFiles.length; i++) {
      const f = uploadedFiles[i];
      const formData = new FormData();
      formData.append('path', currentPath);
      formData.append('file', f);

      try {
        const token = localStorage.getItem('hostvra_token');
        const baseUrl = getApiBaseUrl();
        const res = await fetch(`${baseUrl}/api/v1/files/upload`, {
          method: 'POST',
          headers: token ? { Authorization: `Bearer ${token}` } : {},
          body: formData,
        });

        if (!res.ok) {
          const errJson = await res.json().catch(() => ({}));
          throw new Error(errJson.error?.message || 'Upload rejected by host server');
        }

        setUploadProgress(Math.round(((i + 1) / uploadedFiles.length) * 100));
      } catch (err: any) {
        alert(`Error uploading ${f.name}: ${err.message}`);
      }
    }

    setUploading(false);
    setUploadModalOpen(false);
    setUploadProgress(0);
    if (fileInputRef.current) fileInputRef.current.value = '';
    fetchDirectory(currentPath);
  };

  // Delete item
  const handleDelete = async (file: FileItem) => {
    if (!confirm(`Are you sure you want to permanently delete "${file.name}"?`)) return;

    try {
      const res = await apiFetch(`/api/v1/files/delete?path=${encodeURIComponent(file.path)}`, {
        method: 'DELETE',
      });
      if (res.success) {
        fetchDirectory(currentPath);
      } else {
        alert(res.error?.message || 'Delete failed');
      }
    } catch (err: any) {
      alert(err.message || 'Delete failed');
    }
  };

  // Rename item
  const handleRenameSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!itemToRename || !newName.trim()) return;

    const parentDir = currentPath.replace(/\/$/, '');
    const targetPath = `${parentDir}/${newName.trim()}`;

    try {
      const res = await apiFetch('/api/v1/files/rename', {
        method: 'POST',
        body: JSON.stringify({
          old_path: itemToRename.path,
          new_path: targetPath,
        }),
      });

      if (res.success) {
        setRenameModalOpen(false);
        setItemToRename(null);
        setNewName('');
        fetchDirectory(currentPath);
      } else {
        alert(res.error?.message || 'Rename failed');
      }
    } catch (err: any) {
      alert(err.message || 'Rename failed');
    }
  };

  // Permissions submit
  const handlePermSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!itemForPerm) return;

    try {
      const res = await apiFetch('/api/v1/files/permissions', {
        method: 'POST',
        body: JSON.stringify({
          path: itemForPerm.path,
          mode: permMode,
        }),
      });

      if (res.success) {
        setPermModalOpen(false);
        setItemForPerm(null);
        fetchDirectory(currentPath);
      } else {
        alert(res.error?.message || 'Permissions change failed');
      }
    } catch (err: any) {
      alert(err.message || 'Permissions change failed');
    }
  };

  // Archive item
  const handleArchiveSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!itemToArchive) return;

    setIsProcessingArchive(true);
    const ext = archiveFormat === 'tar.gz' ? '.tar.gz' : '.zip';
    const destPath = `${itemToArchive.path}${ext}`;

    try {
      const res = await apiFetch('/api/v1/files/archive', {
        method: 'POST',
        body: JSON.stringify({
          paths: [itemToArchive.path],
          dest_path: destPath,
          format: archiveFormat,
        }),
      });

      if (res.success) {
        setArchiveModalOpen(false);
        setItemToArchive(null);
        fetchDirectory(currentPath);
      } else {
        alert(res.error?.message || 'Compression failed');
      }
    } catch (err: any) {
      alert(err.message || 'Compression failed');
    } finally {
      setIsProcessingArchive(false);
    }
  };

  // Extract archive
  const handleExtract = async (file: FileItem) => {
    if (!confirm(`Extract archive "${file.name}" into current directory?`)) return;

    try {
      const res = await apiFetch('/api/v1/files/extract', {
        method: 'POST',
        body: JSON.stringify({
          archive_path: file.path,
          dest_dir: currentPath,
        }),
      });

      if (res.success) {
        fetchDirectory(currentPath);
      } else {
        alert(res.error?.message || 'Extraction failed');
      }
    } catch (err: any) {
      alert(err.message || 'Extraction failed');
    }
  };

  // Download file
  const handleDownload = (file: FileItem) => {
    const token = localStorage.getItem('hostvra_token');
    const baseUrl = getApiBaseUrl();
    const downloadUrl = `${baseUrl}/api/v1/files/download?path=${encodeURIComponent(file.path)}`;
    
    // Create an authenticated temporary anchor or open in new tab
    const a = document.createElement('a');
    a.href = downloadUrl;
    a.download = file.name;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  };

  // Navigate up one level
  const handleNavigateUp = () => {
    const parts = currentPath.split('/').filter(Boolean);
    if (parts.length <= 1) {
      fetchDirectory('/');
    } else {
      const upPath = '/' + parts.slice(0, parts.length - 1).join('/');
      fetchDirectory(upPath);
    }
  };

  // Format file size
  const formatSize = (bytes: number, isDir: boolean) => {
    if (isDir) return '--';
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
  };

  const filteredFiles = files.filter((f) =>
    f.name.toLowerCase().includes(search.toLowerCase())
  );

  const pathParts = currentPath.split('/').filter(Boolean);

  return (
    <DashboardShell>
      <div className="space-y-6">
        {/* Top Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-xl sm:text-2xl font-bold tracking-tight text-slate-900 dark:text-white">
                File Manager
              </h1>
              <span className="text-[10px] px-2 py-0.5 rounded-full bg-emerald-100 dark:bg-emerald-950/60 text-emerald-700 dark:text-emerald-300 font-bold border border-emerald-500/20">
                Live Server Filesystem
              </span>
            </div>
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
              Sandboxed server filesystem browser with atomic save, .bak protection, chmod, and drag-and-drop upload.
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-2.5">
            <button
              onClick={() => setUploadModalOpen(true)}
              className="inline-flex items-center gap-2 px-3.5 py-2 rounded-xl bg-white hover:bg-slate-50 dark:bg-surface-800 dark:hover:bg-surface-700 text-slate-800 dark:text-slate-100 text-xs font-bold border border-slate-300 dark:border-surface-700 transition shadow-xs cursor-pointer"
            >
              <UploadCloud className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
              <span>Upload</span>
            </button>
            <button
              onClick={() => {
                setCreateType('folder');
                setCreateModalOpen(true);
              }}
              className="inline-flex items-center gap-2 px-3.5 py-2 rounded-xl bg-white hover:bg-slate-50 dark:bg-surface-800 dark:hover:bg-surface-700 text-slate-800 dark:text-slate-100 text-xs font-bold border border-slate-300 dark:border-surface-700 transition shadow-xs cursor-pointer"
            >
              <FolderPlus className="w-4 h-4 text-indigo-600 dark:text-indigo-400" />
              <span>New Folder</span>
            </button>
            <button
              onClick={() => {
                setCreateType('file');
                setCreateModalOpen(true);
              }}
              className="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white text-xs font-bold transition shadow-sm cursor-pointer"
            >
              <FilePlus className="w-4 h-4 text-white" />
              <span>New File</span>
            </button>
            <button
              onClick={() => fetchDirectory(currentPath)}
              className="p-2 rounded-xl bg-white hover:bg-slate-50 dark:bg-surface-800 dark:hover:bg-surface-700 border border-slate-300 dark:border-surface-700 text-slate-700 dark:text-slate-300 transition shadow-xs cursor-pointer"
              title="Refresh Directory"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            </button>
          </div>
        </div>

        {/* Quick Jump Shortcuts */}
        <div className="flex flex-wrap items-center gap-2 text-xs">
          <span className="text-slate-400 font-medium">Quick Jump:</span>
          {[
            { label: '/var/www (Websites)', path: '/var/www' },
            { label: '/home', path: '/home' },
            { label: '/etc/nginx', path: '/etc/nginx' },
            { label: '/etc/php', path: '/etc/php' },
            { label: '/var/log', path: '/var/log' },
            { label: '/tmp', path: '/tmp' },
          ].map((sc) => (
            <button
              key={sc.path}
              onClick={() => fetchDirectory(sc.path)}
              className={`px-2.5 py-1 rounded-lg font-mono text-[11px] transition ${
                currentPath.startsWith(sc.path)
                  ? 'bg-emerald-600 text-white font-bold'
                  : 'bg-slate-100 dark:bg-surface-800 hover:bg-slate-200 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-300'
              }`}
            >
              {sc.label}
            </button>
          ))}
        </div>

        {/* Breadcrumb Path & Search Bar */}
        <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 bg-white dark:bg-[#121824] border border-slate-200 dark:border-surface-800 p-2.5 rounded-2xl shadow-xs">
          <div className="flex items-center gap-1.5 text-xs text-slate-600 dark:text-slate-400 overflow-x-auto py-1">
            <button
              onClick={handleNavigateUp}
              disabled={currentPath === '/'}
              className="p-1 rounded bg-slate-100 dark:bg-surface-800 hover:bg-slate-200 dark:hover:bg-surface-700 text-slate-600 dark:text-slate-300 disabled:opacity-40"
              title="Up one level"
            >
              <ArrowUp className="w-3.5 h-3.5" />
            </button>

            <button
              onClick={() => fetchDirectory('/')}
              className="hover:text-emerald-600 dark:hover:text-emerald-400 transition-colors font-bold text-slate-900 dark:text-slate-200 px-2 py-0.5 rounded bg-slate-100 dark:bg-surface-800"
            >
              root /
            </button>

            {pathParts.map((part, i) => {
              const partPath = '/' + pathParts.slice(0, i + 1).join('/');
              return (
                <React.Fragment key={partPath}>
                  <ChevronRight className="w-3.5 h-3.5 text-slate-400 flex-shrink-0" />
                  <button
                    onClick={() => fetchDirectory(partPath)}
                    className="hover:text-emerald-600 dark:hover:text-emerald-400 transition-colors font-semibold text-slate-800 dark:text-slate-300 truncate max-w-[140px]"
                  >
                    {part}
                  </button>
                </React.Fragment>
              );
            })}
          </div>

          <div className="flex items-center gap-2 w-full sm:w-64 bg-slate-50 dark:bg-surface-900 border border-slate-300 dark:border-surface-700 rounded-xl px-3 py-1.5 shadow-xs focus-within:border-emerald-500 focus-within:ring-2 focus-within:ring-emerald-500/20 transition-all">
            <Search className="w-3.5 h-3.5 text-slate-400 flex-shrink-0" />
            <input
              type="text"
              placeholder="Search current folder..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full bg-transparent text-xs text-slate-900 dark:text-white placeholder:text-slate-400 focus:outline-none"
            />
          </div>
        </div>

        {/* Error Notification */}
        {errorMsg && (
          <div className="p-3.5 rounded-xl bg-rose-50 dark:bg-rose-950/30 border border-rose-200 dark:border-rose-900/40 text-rose-700 dark:text-rose-400 text-xs flex items-center gap-2">
            <AlertCircle className="w-4 h-4 flex-shrink-0" />
            <span>{errorMsg}</span>
          </div>
        )}

        {/* Files Table */}
        <div className="bg-white dark:bg-[#121824] border border-slate-200 dark:border-surface-800 rounded-2xl overflow-hidden shadow-xs">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs border-collapse">
              <thead>
                <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-[#151b28]">
                  <th className="px-5 py-3 font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300">Name</th>
                  <th className="px-5 py-3 font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300">Size</th>
                  <th className="px-5 py-3 font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300">Permissions</th>
                  <th className="px-5 py-3 font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300">Owner/Group</th>
                  <th className="px-5 py-3 font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300">Last Modified</th>
                  <th className="px-5 py-3 font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 dark:divide-surface-800">
                {loading ? (
                  <tr>
                    <td colSpan={6} className="px-5 py-8 text-center text-slate-400">
                      <RefreshCw className="w-5 h-5 animate-spin mx-auto mb-2 text-emerald-500" />
                      Loading directory contents from server...
                    </td>
                  </tr>
                ) : filteredFiles.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="px-5 py-8 text-center text-slate-400">
                      <Folder className="w-6 h-6 mx-auto mb-1.5 opacity-40" />
                      This directory is currently empty.
                    </td>
                  </tr>
                ) : (
                  filteredFiles.map((file) => {
                    const isArchive =
                      file.name.endsWith('.zip') || file.name.endsWith('.tar.gz') || file.name.endsWith('.tgz');

                    return (
                      <tr
                        key={file.path}
                        className="hover:bg-slate-50 dark:hover:bg-surface-800/50 transition cursor-pointer group"
                        onDoubleClick={() => handleItemClick(file)}
                      >
                        {/* File Name & Icon */}
                        <td className="px-5 py-2.5 font-medium text-slate-900 dark:text-slate-200">
                          <div className="flex items-center gap-2.5">
                            {file.is_dir ? (
                              <Folder className="w-4 h-4 text-amber-500 flex-shrink-0" />
                            ) : isArchive ? (
                              <FolderArchive className="w-4 h-4 text-purple-500 flex-shrink-0" />
                            ) : file.name.endsWith('.bak') ? (
                              <ShieldCheck className="w-4 h-4 text-emerald-500 flex-shrink-0" />
                            ) : (
                              <FileCode className="w-4 h-4 text-indigo-400 flex-shrink-0" />
                            )}
                            <button
                              onClick={() => handleItemClick(file)}
                              className="text-left font-semibold text-slate-800 dark:text-slate-200 hover:text-emerald-600 dark:hover:text-emerald-400 transition truncate max-w-xs sm:max-w-md"
                            >
                              {file.name}
                            </button>
                          </div>
                        </td>

                        {/* File Size */}
                        <td className="px-5 py-2.5 font-mono text-slate-500 dark:text-slate-400">
                          {formatSize(file.size, file.is_dir)}
                        </td>

                        {/* Permissions (Octal) */}
                        <td className="px-5 py-2.5 font-mono text-slate-600 dark:text-slate-300">
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              setItemForPerm(file);
                              setPermMode(file.perm_octal || '0755');
                              setPermModalOpen(true);
                            }}
                            className="px-2 py-0.5 rounded bg-slate-100 dark:bg-surface-800 hover:border-emerald-500 border border-transparent transition text-[11px]"
                            title="Click to change permissions"
                          >
                            {file.perm_octal}
                          </button>
                        </td>

                        {/* Owner / Group */}
                        <td className="px-5 py-2.5 text-slate-500 dark:text-slate-400 font-mono text-[11px]">
                          {file.owner}:{file.group}
                        </td>

                        {/* Modified Time */}
                        <td className="px-5 py-2.5 text-slate-500 dark:text-slate-400 whitespace-nowrap">
                          {file.modified_at ? new Date(file.modified_at).toLocaleString() : '--'}
                        </td>

                        {/* Actions */}
                        <td className="px-5 py-2.5 text-right">
                          <div className="flex items-center justify-end gap-1.5">
                            {!file.is_dir && (
                              <button
                                onClick={(e) => {
                                  e.stopPropagation();
                                  handleItemClick(file);
                                }}
                                className="p-1.5 rounded-lg bg-white dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-slate-600 dark:text-slate-300 hover:text-emerald-600 dark:hover:text-emerald-400 hover:bg-emerald-50 dark:hover:bg-emerald-950/30 hover:border-emerald-200 transition shadow-2xs cursor-pointer"
                                title="Edit File"
                              >
                                <Edit className="w-3.5 h-3.5" />
                              </button>
                            )}

                            {!file.is_dir && (
                              <button
                                onClick={(e) => {
                                  e.stopPropagation();
                                  handleDownload(file);
                                }}
                                className="p-1.5 rounded-lg bg-white dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-slate-600 dark:text-slate-300 hover:text-indigo-600 dark:hover:text-indigo-400 hover:bg-indigo-50 dark:hover:bg-indigo-950/30 hover:border-indigo-200 transition shadow-2xs cursor-pointer"
                                title="Download File"
                              >
                                <Download className="w-3.5 h-3.5" />
                              </button>
                            )}

                            {isArchive && (
                              <button
                                onClick={(e) => {
                                  e.stopPropagation();
                                  handleExtract(file);
                                }}
                                className="p-1.5 rounded-lg bg-white dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-purple-600 dark:text-purple-400 hover:bg-purple-50 dark:hover:bg-purple-950/30 hover:border-purple-200 transition shadow-2xs cursor-pointer"
                                title="Extract Archive"
                              >
                                <Archive className="w-3.5 h-3.5" />
                              </button>
                            )}

                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                setItemToArchive(file);
                                setArchiveModalOpen(true);
                              }}
                              className="p-1.5 rounded-lg bg-white dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-slate-600 dark:text-slate-300 hover:text-purple-600 dark:hover:text-purple-400 hover:bg-purple-50 dark:hover:bg-purple-950/30 hover:border-purple-200 transition shadow-2xs cursor-pointer"
                              title="Compress / Archive"
                            >
                              <Archive className="w-3.5 h-3.5" />
                            </button>

                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                setItemToRename(file);
                                setNewName(file.name);
                                setRenameModalOpen(true);
                              }}
                              className="p-1.5 rounded-lg bg-white dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-slate-600 dark:text-slate-300 hover:text-amber-600 dark:hover:text-amber-400 hover:bg-amber-50 dark:hover:bg-amber-950/30 hover:border-amber-200 transition shadow-2xs cursor-pointer"
                              title="Rename"
                            >
                              <Copy className="w-3.5 h-3.5" />
                            </button>

                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                handleDelete(file);
                              }}
                              className="p-1.5 rounded-lg bg-white dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-slate-500 hover:text-rose-600 dark:hover:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-950/30 hover:border-rose-200 transition shadow-2xs cursor-pointer"
                              title="Delete Permanently"
                            >
                              <Trash2 className="w-3.5 h-3.5" />
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
        </div>
      </div>

      {/* Real Code Editor Modal */}
      {editorOpen && editingFile && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-6 bg-black/70 backdrop-blur-xs animate-fadeIn">
          <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl w-full max-w-5xl shadow-2xl flex flex-col h-[85vh] overflow-hidden">
            {/* Modal Header */}
            <div className="px-5 py-3.5 border-b border-slate-200 dark:border-surface-800 flex items-center justify-between bg-slate-50 dark:bg-surface-800/60">
              <div className="flex items-center gap-2.5 min-w-0">
                <FileCode className="w-4 h-4 text-emerald-500 flex-shrink-0" />
                <span className="font-mono font-bold text-xs text-slate-800 dark:text-slate-200 truncate">
                  {editingFile.path}
                </span>
                <span className="text-[10px] text-slate-400 font-mono">
                  ({formatSize(editingFile.size, false)})
                </span>
                {saveSuccess && (
                  <span className="flex items-center gap-1 text-[11px] text-emerald-600 dark:text-emerald-400 font-bold bg-emerald-50 dark:bg-emerald-950/50 px-2 py-0.5 rounded">
                    <CheckCircle2 className="w-3 h-3" />
                    Saved & Backup (.bak) updated
                  </span>
                )}
              </div>

              <div className="flex items-center gap-2 flex-shrink-0">
                <button
                  onClick={handleSaveContent}
                  disabled={savingFile}
                  className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-xs transition shadow-xs disabled:opacity-50"
                  title="Save (Cmd+S / Ctrl+S)"
                >
                  <Save className="w-3.5 h-3.5" />
                  <span>{savingFile ? 'Saving...' : 'Save File'}</span>
                </button>
                <button
                  onClick={() => setEditorOpen(false)}
                  className="w-7 h-7 rounded-lg bg-slate-200 dark:bg-surface-700 text-slate-500 hover:text-slate-900 dark:hover:text-white flex items-center justify-center transition"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>
            </div>

            {/* Editor Textarea */}
            <div className="flex-1 p-0 relative bg-slate-950 text-slate-100 font-mono text-xs overflow-hidden flex">
              <textarea
                value={fileContent}
                onChange={(e) => setFileContent(e.target.value)}
                onKeyDown={handleEditorKeyDown}
                spellCheck={false}
                className="w-full h-full p-4 bg-transparent resize-none focus:outline-none font-mono text-xs leading-relaxed text-slate-200 selection:bg-emerald-500/30 overflow-auto"
                placeholder="File content..."
              />
            </div>

            {/* Modal Footer */}
            <div className="px-5 py-2.5 border-t border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-surface-800/40 text-[11px] text-slate-500 dark:text-slate-400 flex items-center justify-between">
              <span>Press <kbd className="px-1.5 py-0.5 rounded bg-slate-200 dark:bg-surface-700 text-slate-800 dark:text-slate-200 font-bold">Ctrl+S</kbd> or <kbd className="px-1.5 py-0.5 rounded bg-slate-200 dark:bg-surface-700 text-slate-800 dark:text-slate-200 font-bold">Cmd+S</kbd> to save. Automatic backup is created on disk.</span>
              <span>Lines: {fileContent.split('\n').length} · Chars: {fileContent.length}</span>
            </div>
          </div>
        </div>
      )}

      {/* New Folder / File Modal */}
      {createModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs animate-fadeIn">
          <form onSubmit={handleCreate} className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl w-full max-w-md shadow-2xl p-6 space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-bold text-slate-900 dark:text-white flex items-center gap-2">
                {createType === 'folder' ? <FolderPlus className="w-4 h-4 text-indigo-500" /> : <FilePlus className="w-4 h-4 text-emerald-500" />}
                Create New {createType === 'folder' ? 'Folder' : 'File'}
              </h3>
              <button
                type="button"
                onClick={() => setCreateModalOpen(false)}
                className="w-7 h-7 rounded-full bg-slate-100 dark:bg-surface-800 text-slate-500 hover:text-slate-900 dark:hover:text-white flex items-center justify-center"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-semibold text-slate-700 dark:text-slate-300">
                Name:
              </label>
              <input
                type="text"
                autoFocus
                required
                value={createName}
                onChange={(e) => setCreateName(e.target.value)}
                placeholder={createType === 'folder' ? 'e.g. public_html' : 'e.g. index.php or .env'}
                className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-xs text-slate-900 dark:text-white focus:outline-none focus:border-emerald-500"
              />
              <p className="text-[10px] text-slate-400">
                Target path: {currentPath.replace(/\/$/, '')}/{createName || '...'}
              </p>
            </div>

            <div className="flex justify-end gap-2 pt-2">
              <button
                type="button"
                onClick={() => setCreateModalOpen(false)}
                className="px-3.5 py-1.5 rounded-lg border border-slate-300 dark:border-surface-700 text-xs font-medium text-slate-600 dark:text-slate-300"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={creating || !createName.trim()}
                className="px-4 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold transition shadow-xs disabled:opacity-50"
              >
                {creating ? 'Creating...' : 'Create'}
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Upload Modal */}
      {uploadModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs animate-fadeIn">
          <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl w-full max-w-md shadow-2xl p-6 space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-bold text-slate-900 dark:text-white flex items-center gap-2">
                <UploadCloud className="w-4 h-4 text-emerald-500" />
                Upload to Server
              </h3>
              <button
                onClick={() => setUploadModalOpen(false)}
                className="w-7 h-7 rounded-full bg-slate-100 dark:bg-surface-800 text-slate-500 hover:text-slate-900 dark:hover:text-white flex items-center justify-center"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="p-6 border-2 border-dashed border-slate-300 dark:border-surface-700 rounded-xl text-center space-y-3 bg-slate-50 dark:bg-surface-800/40">
              <UploadCloud className="w-8 h-8 mx-auto text-emerald-500" />
              <div>
                <p className="text-xs font-bold text-slate-800 dark:text-slate-200">
                  Select files to upload
                </p>
                <p className="text-[11px] text-slate-400 mt-0.5">
                  Files will be uploaded directly to: <span className="font-mono text-emerald-500">{currentPath}</span>
                </p>
              </div>

              <input
                ref={fileInputRef}
                type="file"
                multiple
                onChange={handleUploadSubmit}
                className="hidden"
                id="file-upload-input"
              />
              <label
                htmlFor="file-upload-input"
                className="inline-block px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold cursor-pointer transition shadow-xs"
              >
                {uploading ? `Uploading (${uploadProgress}%)...` : 'Choose Files'}
              </label>
            </div>

            {uploading && (
              <div className="w-full bg-slate-200 dark:bg-surface-700 h-2 rounded-full overflow-hidden">
                <div
                  className="bg-emerald-500 h-full transition-all duration-200"
                  style={{ width: `${uploadProgress}%` }}
                />
              </div>
            )}
          </div>
        </div>
      )}

      {/* Rename Modal */}
      {renameModalOpen && itemToRename && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs animate-fadeIn">
          <form onSubmit={handleRenameSubmit} className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl w-full max-w-md shadow-2xl p-6 space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-bold text-slate-900 dark:text-white flex items-center gap-2">
                <Copy className="w-4 h-4 text-amber-500" />
                Rename Item
              </h3>
              <button
                type="button"
                onClick={() => setRenameModalOpen(false)}
                className="w-7 h-7 rounded-full bg-slate-100 dark:bg-surface-800 text-slate-500 hover:text-slate-900 dark:hover:text-white flex items-center justify-center"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-semibold text-slate-700 dark:text-slate-300">
                New Name:
              </label>
              <input
                type="text"
                autoFocus
                required
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-xs text-slate-900 dark:text-white focus:outline-none focus:border-emerald-500 font-mono"
              />
            </div>

            <div className="flex justify-end gap-2 pt-2">
              <button
                type="button"
                onClick={() => setRenameModalOpen(false)}
                className="px-3.5 py-1.5 rounded-lg border border-slate-300 dark:border-surface-700 text-xs font-medium text-slate-600 dark:text-slate-300"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={!newName.trim() || newName === itemToRename.name}
                className="px-4 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold transition shadow-xs disabled:opacity-50"
              >
                Rename
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Permissions (chmod) Modal */}
      {permModalOpen && itemForPerm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs animate-fadeIn">
          <form onSubmit={handlePermSubmit} className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl w-full max-w-md shadow-2xl p-6 space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-bold text-slate-900 dark:text-white flex items-center gap-2">
                <Lock className="w-4 h-4 text-emerald-500" />
                File Permissions (chmod)
              </h3>
              <button
                type="button"
                onClick={() => setPermModalOpen(false)}
                className="w-7 h-7 rounded-full bg-slate-100 dark:bg-surface-800 text-slate-500 hover:text-slate-900 dark:hover:text-white flex items-center justify-center"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <p className="text-xs text-slate-500 font-mono truncate">
              {itemForPerm.path}
            </p>

            <div className="space-y-1.5">
              <label className="text-xs font-semibold text-slate-700 dark:text-slate-300">
                Octal Permission Mode:
              </label>
              <input
                type="text"
                autoFocus
                required
                value={permMode}
                onChange={(e) => setPermMode(e.target.value)}
                placeholder="e.g. 0755 or 0644"
                className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-xs text-slate-900 dark:text-white focus:outline-none focus:border-emerald-500 font-mono"
              />
              <div className="flex gap-2 pt-1 text-[11px]">
                <button
                  type="button"
                  onClick={() => setPermMode('0644')}
                  className="px-2 py-0.5 rounded bg-slate-100 dark:bg-surface-800 hover:bg-slate-200 text-slate-700 dark:text-slate-300"
                >
                  0644 (Standard File)
                </button>
                <button
                  type="button"
                  onClick={() => setPermMode('0755')}
                  className="px-2 py-0.5 rounded bg-slate-100 dark:bg-surface-800 hover:bg-slate-200 text-slate-700 dark:text-slate-300"
                >
                  0755 (Standard Folder / Executable)
                </button>
              </div>
            </div>

            <div className="flex justify-end gap-2 pt-2">
              <button
                type="button"
                onClick={() => setPermModalOpen(false)}
                className="px-3.5 py-1.5 rounded-lg border border-slate-300 dark:border-surface-700 text-xs font-medium text-slate-600 dark:text-slate-300"
              >
                Cancel
              </button>
              <button
                type="submit"
                className="px-4 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold transition shadow-xs"
              >
                Apply Permissions
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Archive Modal */}
      {archiveModalOpen && itemToArchive && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs animate-fadeIn">
          <form onSubmit={handleArchiveSubmit} className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl w-full max-w-md shadow-2xl p-6 space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-bold text-slate-900 dark:text-white flex items-center gap-2">
                <Archive className="w-4 h-4 text-purple-500" />
                Compress / Archive
              </h3>
              <button
                type="button"
                onClick={() => setArchiveModalOpen(false)}
                className="w-7 h-7 rounded-full bg-slate-100 dark:bg-surface-800 text-slate-500 hover:text-slate-900 dark:hover:text-white flex items-center justify-center"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <p className="text-xs text-slate-500 truncate">
              Compress: <span className="font-mono text-slate-800 dark:text-slate-200">{itemToArchive.name}</span>
            </p>

            <div className="space-y-1.5">
              <label className="text-xs font-semibold text-slate-700 dark:text-slate-300">
                Format:
              </label>
              <div className="flex gap-4 text-xs">
                <label className="flex items-center gap-2 cursor-pointer text-slate-800 dark:text-slate-200">
                  <input
                    type="radio"
                    name="archiveFormat"
                    value="zip"
                    checked={archiveFormat === 'zip'}
                    onChange={() => setArchiveFormat('zip')}
                  />
                  <span>.ZIP (Universal Standard)</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer text-slate-800 dark:text-slate-200">
                  <input
                    type="radio"
                    name="archiveFormat"
                    value="tar.gz"
                    checked={archiveFormat === 'tar.gz'}
                    onChange={() => setArchiveFormat('tar.gz')}
                  />
                  <span>.tar.gz (Linux Gzip)</span>
                </label>
              </div>
            </div>

            <div className="flex justify-end gap-2 pt-2">
              <button
                type="button"
                onClick={() => setArchiveModalOpen(false)}
                className="px-3.5 py-1.5 rounded-lg border border-slate-300 dark:border-surface-700 text-xs font-medium text-slate-600 dark:text-slate-300"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={isProcessingArchive}
                className="px-4 py-1.5 rounded-lg bg-purple-600 hover:bg-purple-500 text-white text-xs font-bold transition shadow-xs disabled:opacity-50"
              >
                {isProcessingArchive ? 'Compressing...' : 'Start Compression'}
              </button>
            </div>
          </form>
        </div>
      )}
    </DashboardShell>
  );
}
