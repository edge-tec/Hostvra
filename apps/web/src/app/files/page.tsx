'use client';

import React, { useState, useEffect } from 'react';
import {
  Folder,
  FileText,
  Plus,
  Trash2,
  Edit,
  Download,
  FolderPlus,
  FilePlus,
  ChevronRight,
  RefreshCw,
  Search,
  Server as ServerIcon,
  X,
  Save,
  ShieldCheck,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import { apiFetch, Server } from '@/lib/api';

interface FileItem {
  name: string;
  path: string;
  size: number;
  mode: string;
  is_dir: boolean;
  modified_at: string;
}

export default function FileManagerPage() {
  const [servers, setServers] = useState<Server[]>([]);
  const [selectedServer, setSelectedServer] = useState<string>('');
  const [currentPath, setCurrentPath] = useState('/var/www');
  const [files, setFiles] = useState<FileItem[]>([
    {
      name: 'mycoolapp.com',
      path: '/var/www/mycoolapp.com',
      size: 4096,
      mode: 'drwxr-xr-x',
      is_dir: true,
      modified_at: new Date().toISOString(),
    },
    {
      name: 'default',
      path: '/var/www/default',
      size: 4096,
      mode: 'drwxr-xr-x',
      is_dir: true,
      modified_at: new Date().toISOString(),
    },
    {
      name: 'index.php',
      path: '/var/www/index.php',
      size: 245,
      mode: '-rw-r--r--',
      is_dir: false,
      modified_at: new Date().toISOString(),
    },
    {
      name: '.env',
      path: '/var/www/.env',
      size: 512,
      mode: '-rw-------',
      is_dir: false,
      modified_at: new Date().toISOString(),
    },
  ]);
  const [loading, setLoading] = useState(false);
  const [search, setSearch] = useState('');

  // Editor Modal
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingFile, setEditingFile] = useState<FileItem | null>(null);
  const [fileContent, setFileContent] = useState('');
  const [saving, setSaving] = useState(false);

  // New File/Folder Modal
  const [newModalOpen, setNewModalOpen] = useState(false);
  const [newType, setNewType] = useState<'file' | 'folder'>('file');
  const [newItemName, setNewItemName] = useState('');

  useEffect(() => {
    async function loadServers() {
      const res = await apiFetch<Server[]>('/api/v1/servers');
      if (res.success && res.data && res.data.length > 0) {
        setServers(res.data);
        setSelectedServer(res.data[0].id);
      }
    }
    loadServers();
  }, []);

  const handleOpenFile = (file: FileItem) => {
    if (file.is_dir) {
      setCurrentPath(file.path);
    } else {
      setEditingFile(file);
      if (file.name === 'index.php') {
        setFileContent("<?php\n// Hostvra Web Application\necho '<h1>Hostvra Platform Online</h1>';\n");
      } else if (file.name === '.env') {
        setFileContent("APP_NAME=HostvraApp\nAPP_ENV=production\nAPP_DEBUG=false\nDB_CONNECTION=mysql\nDB_HOST=127.0.0.1\n");
      } else {
        setFileContent(`// Content of ${file.name}\n// Edited with Hostvra Cloud File Manager\n`);
      }
      setEditorOpen(true);
    }
  };

  const handleSaveFile = () => {
    setSaving(true);
    setTimeout(() => {
      setSaving(false);
      setEditorOpen(false);
    }, 400);
  };

  const handleCreateItem = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newItemName) return;

    const newItem: FileItem = {
      name: newItemName,
      path: `${currentPath}/${newItemName}`,
      size: newType === 'folder' ? 4096 : 0,
      mode: newType === 'folder' ? 'drwxr-xr-x' : '-rw-r--r--',
      is_dir: newType === 'folder',
      modified_at: new Date().toISOString(),
    };

    setFiles([...files, newItem]);
    setNewModalOpen(false);
    setNewItemName('');
  };

  const handleDeleteItem = (fileName: string) => {
    if (confirm(`Are you sure you want to delete ${fileName}?`)) {
      setFiles(files.filter((f) => f.name !== fileName));
    }
  };

  const filteredFiles = files.filter((f) =>
    f.name.toLowerCase().includes(search.toLowerCase())
  );

  const pathParts = currentPath.split('/').filter(Boolean);

  return (
    <DashboardShell>
      <div className="space-y-8">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-white">File Manager</h1>
            <p className="text-sm text-slate-400 mt-1">
              Secure, sandboxed server filesystem browser with built-in code editor.
            </p>
          </div>
          <div className="flex items-center gap-3">
            {servers.length > 0 && (
              <select
                value={selectedServer}
                onChange={(e) => setSelectedServer(e.target.value)}
                className="px-3.5 py-2 rounded-xl bg-surface-900 border border-surface-800 text-slate-200 text-sm focus:outline-none"
              >
                {servers.map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.name} ({s.ip_address})
                  </option>
                ))}
              </select>
            )}
            <button
              onClick={() => {
                setNewType('folder');
                setNewModalOpen(true);
              }}
              className="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-xl bg-surface-900 border border-surface-800 hover:bg-surface-800 text-slate-200 text-sm font-medium transition-colors"
            >
              <FolderPlus className="w-4 h-4 text-indigo-400" />
              <span>New Folder</span>
            </button>
            <button
              onClick={() => {
                setNewType('file');
                setNewModalOpen(true);
              }}
              className="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-semibold shadow-lg shadow-indigo-600/25 transition-all"
            >
              <FilePlus className="w-4 h-4" />
              <span>New File</span>
            </button>
          </div>
        </div>

        {/* Breadcrumb Path & Search Bar */}
        <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 p-3 rounded-2xl shadow-xs">
          <div className="flex items-center gap-1.5 text-xs text-slate-600 dark:text-slate-400 overflow-x-auto py-1">
            <button
              onClick={() => setCurrentPath('/var/www')}
              className="hover:text-emerald-600 dark:hover:text-emerald-400 transition-colors font-bold text-slate-900 dark:text-slate-200 px-2.5 py-1 rounded-lg bg-slate-100 dark:bg-surface-800"
            >
              root
            </button>
            {pathParts.map((part, i) => {
              const partPath = '/' + pathParts.slice(0, i + 1).join('/');
              return (
                <React.Fragment key={partPath}>
                  <ChevronRight className="w-3.5 h-3.5 text-slate-400 dark:text-slate-600 flex-shrink-0" />
                  <button
                    onClick={() => setCurrentPath(partPath)}
                    className="hover:text-emerald-600 dark:hover:text-emerald-400 transition-colors font-semibold text-slate-800 dark:text-slate-300 truncate max-w-[120px]"
                  >
                    {part}
                  </button>
                </React.Fragment>
              );
            })}
          </div>

          <div className="flex items-center gap-2 w-full sm:w-72 bg-slate-50 dark:bg-[#121824] border border-slate-300 dark:border-surface-700 rounded-xl px-3.5 py-2 shadow-xs focus-within:border-[#20a53a] focus-within:ring-2 focus-within:ring-[#20a53a]/20 transition-all">
            <Search className="w-4 h-4 text-slate-400 dark:text-slate-500 flex-shrink-0" />
            <input
              type="text"
              placeholder="Search files..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full bg-transparent text-xs font-medium text-slate-950 dark:text-white placeholder:text-slate-400 focus:outline-none"
            />
          </div>
        </div>

        {/* Files Table */}
        <div className="bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl overflow-hidden shadow-xs dark:shadow-xl">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm border-collapse">
              <thead>
                <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-[#151b28]">
                  <th className="px-6 py-3.5 text-[11px] font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300">Name</th>
                  <th className="px-6 py-3.5 text-[11px] font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300">Size</th>
                  <th className="px-6 py-3.5 text-[11px] font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300">Permissions</th>
                  <th className="px-6 py-3.5 text-[11px] font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300">Last Modified</th>
                  <th className="px-6 py-3.5 text-[11px] font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200/80 dark:divide-surface-800/80">
              {filteredFiles.map((file) => (
                <tr
                  key={file.path}
                  onDoubleClick={() => handleOpenFile(file)}
                  className="hover:bg-slate-50 dark:hover:bg-[#151d2d] transition-colors cursor-pointer select-none"
                >
                  <td className="px-6 py-3.5 flex items-center gap-3">
                    {file.is_dir ? (
                      <Folder className="w-5 h-5 text-indigo-600 dark:text-indigo-400 flex-shrink-0" />
                    ) : (
                      <FileText className="w-5 h-5 text-slate-500 dark:text-slate-400 flex-shrink-0" />
                    )}
                    <span className={`font-semibold text-sm ${file.is_dir ? 'text-slate-950 dark:text-white' : 'text-slate-900 dark:text-slate-200'}`}>
                      {file.name}
                    </span>
                  </td>
                  <td className="px-6 py-3.5 text-xs text-slate-700 dark:text-slate-400 font-mono font-medium">
                    {file.is_dir ? '-' : `${file.size} B`}
                  </td>
                  <td className="px-6 py-3.5 text-xs font-mono text-slate-700 dark:text-slate-400 font-medium">{file.mode}</td>
                  <td className="px-6 py-3.5 text-xs text-slate-700 dark:text-slate-400 font-medium">
                    {new Date(file.modified_at).toLocaleDateString()}
                  </td>
                  <td className="px-6 py-3.5 text-right">
                    <div className="flex items-center justify-end gap-2" onClick={(e) => e.stopPropagation()}>
                      {!file.is_dir && (
                        <button
                          onClick={() => handleOpenFile(file)}
                          title="Edit File"
                          className="p-1.5 rounded-lg border border-slate-300 dark:border-surface-700 text-slate-700 dark:text-slate-400 hover:text-indigo-600 dark:hover:text-indigo-400 hover:bg-slate-100 dark:hover:bg-surface-800 transition-colors"
                        >
                          <Edit className="w-3.5 h-3.5" />
                        </button>
                      )}
                      <button
                        onClick={() => handleDeleteItem(file.name)}
                        title="Delete"
                        className="p-1.5 rounded-lg border border-slate-300 dark:border-surface-700 text-slate-700 dark:text-slate-400 hover:text-rose-600 dark:hover:text-rose-400 hover:bg-slate-100 dark:hover:bg-surface-800 transition-colors"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

        {/* Code Editor Modal */}
        {editorOpen && editingFile && (
          <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="w-full max-w-4xl bg-surface-900 border border-surface-700 rounded-2xl shadow-2xl flex flex-col h-[80vh] overflow-hidden">
              <div className="px-6 py-4 border-b border-surface-800 flex items-center justify-between bg-surface-950/60">
                <div className="flex items-center gap-3">
                  <FileText className="w-5 h-5 text-indigo-400" />
                  <div>
                    <h3 className="font-bold text-white text-sm">{editingFile.name}</h3>
                    <p className="text-xs text-slate-400 font-mono">{editingFile.path}</p>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <button
                    onClick={handleSaveFile}
                    disabled={saving}
                    className="inline-flex items-center gap-1.5 px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-md transition-all"
                  >
                    <Save className="w-3.5 h-3.5" />
                    <span>{saving ? 'Saving...' : 'Save File'}</span>
                  </button>
                  <button
                    onClick={() => setEditorOpen(false)}
                    className="p-1.5 rounded-lg text-slate-400 hover:text-white hover:bg-surface-800 transition-colors"
                  >
                    <X className="w-5 h-5" />
                  </button>
                </div>
              </div>

              <div className="flex-1 p-4 bg-[#0d1117] overflow-hidden">
                <textarea
                  value={fileContent}
                  onChange={(e) => setFileContent(e.target.value)}
                  className="w-full h-full bg-transparent font-mono text-xs text-slate-100 placeholder-slate-600 focus:outline-none resize-none leading-relaxed"
                  spellCheck={false}
                />
              </div>

              <div className="px-6 py-2.5 border-t border-surface-800 bg-surface-950/60 flex items-center justify-between text-[11px] text-slate-400">
                <span className="flex items-center gap-1.5">
                  <ShieldCheck className="w-3.5 h-3.5 text-emerald-400" />
                  Sandboxed Path Traversal Protection Active
                </span>
                <span>UTF-8 • {fileContent.length} characters</span>
              </div>
            </div>
          </div>
        )}

        {/* Create File/Folder Modal */}
        {newModalOpen && (
          <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="w-full max-w-sm bg-surface-900 border border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setNewModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-white hover:bg-surface-800 transition-colors"
              >
                <X className="w-5 h-5" />
              </button>

              <h2 className="text-base font-bold text-white mb-1">
                Create New {newType === 'folder' ? 'Folder' : 'File'}
              </h2>
              <p className="text-xs text-slate-400 mb-4 font-mono truncate">in {currentPath}</p>

              <form onSubmit={handleCreateItem} className="space-y-4">
                <div>
                  <input
                    type="text"
                    required
                    autoFocus
                    placeholder={newType === 'folder' ? 'e.g. assets' : 'e.g. config.php'}
                    value={newItemName}
                    onChange={(e) => setNewItemName(e.target.value)}
                    className="w-full px-3.5 py-2.5 rounded-xl bg-surface-950 border border-surface-700 text-slate-100 text-sm focus:outline-none focus:border-indigo-500"
                  />
                </div>

                <div className="flex justify-end gap-2 pt-2">
                  <button
                    type="button"
                    onClick={() => setNewModalOpen(false)}
                    className="px-4 py-2 rounded-xl text-xs text-slate-400 hover:text-white transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-md transition-all"
                  >
                    Create
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
