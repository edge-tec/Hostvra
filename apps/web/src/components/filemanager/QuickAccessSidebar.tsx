'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Star,
  Clock,
  Globe,
  Folder,
  Trash2,
  Archive,
  Upload,
  Download,
  ChevronRight,
  ChevronDown,
  Search,
  Plus,
  X,
  HardDrive,
  FolderTree,
  Tag,
  RefreshCw,
  FolderPlus,
} from 'lucide-react';
import { apiFetch } from '@/lib/api';

export interface QuickAccessFavorite {
  id: string;
  path: string;
  name: string;
  domain: string;
  color?: string;
  created_at: string;
}

export interface QuickAccessRecent {
  id: string;
  path: string;
  domain: string;
  last_accessed_at: string;
}

export interface QuickAccessDomain {
  id: string;
  domain: string;
  document_root: string;
  status: string;
}

export interface QuickAccessFolderLabel {
  id: string;
  domain: string;
  path: string;
  color: string;
  label: string;
}

export interface TreeNodeData {
  name: string;
  path: string;
  is_dir: boolean;
  child_count: number;
  children?: TreeNodeData[];
}

interface QuickAccessData {
  favorites: QuickAccessFavorite[];
  recent: QuickAccessRecent[];
  domains: QuickAccessDomain[];
  folder_labels: QuickAccessFolderLabel[];
  root_directory: string;
  shortcuts: Array<{ name: string; path: string; icon: string }>;
  trash_stats: { count: number; bytes: number };
}

interface QuickAccessSidebarProps {
  currentPath: string;
  onNavigate: (path: string, domain?: string) => void;
  activeDomain: string;
  onDomainChange: (domain: string) => void;
  refreshKey?: number;
  isMobileOpen: boolean;
  onCloseMobile: () => void;
  onTrashClick: () => void;
  isTrashActive: boolean;
  onDropItem?: (targetPath: string, isTrash?: boolean) => void;
  onDeleteFolder?: (path: string) => void;
}

export const QuickAccessSidebar: React.FC<QuickAccessSidebarProps> = ({
  currentPath,
  onNavigate,
  activeDomain,
  onDomainChange,
  refreshKey = 0,
  isMobileOpen,
  onCloseMobile,
  onTrashClick,
  isTrashActive,
  onDropItem,
  onDeleteFolder,
}) => {
  const [data, setData] = useState<QuickAccessData | null>(null);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');

  // Sections collapse toggles
  const [expandedSections, setExpandedSections] = useState({
    favorites: true,
    recent: true,
    domains: true,
    shortcuts: true,
    tree: true,
  });

  // Tree nodes state & expanded paths
  const [treeExpanded, setTreeExpanded] = useState<Record<string, boolean>>({});
  const [treeChildren, setTreeChildren] = useState<Record<string, TreeNodeData[]>>({});
  const [treeLoading, setTreeLoading] = useState<Record<string, boolean>>({});

  // Drag over highlighting for folders
  const [dragOverPath, setDragOverPath] = useState<string | null>(null);

  const toggleSection = (section: keyof typeof expandedSections) => {
    setExpandedSections((prev) => ({ ...prev, [section]: !prev[section] }));
  };

  const fetchQuickAccess = useCallback(async () => {
    try {
      setLoading(true);
      const res = await apiFetch<QuickAccessData>('/api/v1/filemanager/quick-access');
      if (res.success && res.data) {
        setData(res.data);
      }
    } catch (err) {
      console.error('Failed to load Quick Access', err);
    } finally {
      setLoading(false);
    }
  }, []);

  // Load tree children on expand or refresh
  const fetchTreeNodeChildren = useCallback(async (path: string) => {
    try {
      setTreeLoading((prev) => ({ ...prev, [path]: true }));
      const res = await apiFetch<{ nodes: TreeNodeData[] }>(
        `/api/v1/filemanager/tree?path=${encodeURIComponent(path)}`
      );
      if (res.success && res.data) {
        setTreeChildren((prev) => ({ ...prev, [path]: res.data?.nodes || [] }));
      }
    } catch (err) {
      console.error('Failed to load tree node:', path, err);
      setTreeExpanded((prev) => ({ ...prev, [path]: false }));
    } finally {
      setTreeLoading((prev) => ({ ...prev, [path]: false }));
    }
  }, []);

  const toggleTreeNode = async (path: string) => {
    const isCurrentlyExpanded = !!treeExpanded[path];
    if (isCurrentlyExpanded) {
      setTreeExpanded((prev) => ({ ...prev, [path]: false }));
      return;
    }

    setTreeExpanded((prev) => ({ ...prev, [path]: true }));

    // Fetch subdirectories if not loaded yet
    if (!treeChildren[path]) {
      await fetchTreeNodeChildren(path);
    }
  };

  useEffect(() => {
    fetchQuickAccess();
    if (refreshKey > 0) {
      const rootPath = data?.root_directory || '/var/www';
      fetchTreeNodeChildren(rootPath);
      setTreeExpanded((currentExpanded) => {
        Object.keys(currentExpanded).forEach((p) => {
          if (currentExpanded[p] && p !== rootPath) {
            fetchTreeNodeChildren(p);
          }
        });
        return currentExpanded;
      });
    }
  }, [fetchQuickAccess, refreshKey, fetchTreeNodeChildren, data?.root_directory]);

  // Initial root tree load
  useEffect(() => {
    const rootPath = data?.root_directory || '/var/www';
    if (!treeChildren[rootPath] && !treeLoading[rootPath]) {
      toggleTreeNode(rootPath);
    }
  }, [data?.root_directory]);

  // Handle Drag & Drop to sidebar targets
  const handleDragOver = (e: React.DragEvent, path: string) => {
    e.preventDefault();
    e.stopPropagation();
    setDragOverPath(path);
  };

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragOverPath(null);
  };

  const handleDrop = (e: React.DragEvent, targetPath: string, isTrash = false) => {
    e.preventDefault();
    e.stopPropagation();
    setDragOverPath(null);
    if (onDropItem) {
      onDropItem(targetPath, isTrash);
    }
  };

  // Color label mapper
  const getColorClass = (color?: string) => {
    switch (color) {
      case 'emerald':
        return 'bg-emerald-500';
      case 'rose':
        return 'bg-rose-500';
      case 'amber':
        return 'bg-amber-500';
      case 'purple':
        return 'bg-purple-500';
      case 'indigo':
        return 'bg-indigo-500';
      case 'blue':
      default:
        return 'bg-sky-500';
    }
  };

  // Search filtering
  const q = search.toLowerCase().trim();
  const filteredFavorites = (data?.favorites || []).filter(
    (f) => !q || f.name.toLowerCase().includes(q) || f.path.toLowerCase().includes(q)
  );
  const filteredRecent = (data?.recent || []).filter(
    (r) => !q || r.path.toLowerCase().includes(q)
  );
  const filteredDomains = (data?.domains || []).filter(
    (d) => !q || d.domain.toLowerCase().includes(q) || d.document_root.toLowerCase().includes(q)
  );

  const formatBytes = (bytes: number) => {
    if (!bytes || bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  };

  // Render a recursive tree node
  const renderTreeNode = (node: TreeNodeData, depth = 0) => {
    const isExpanded = !!treeExpanded[node.path];
    const isLoading = !!treeLoading[node.path];
    const isSelected = currentPath === node.path && !isTrashActive;
    const isDragOver = dragOverPath === node.path;
    const children = treeChildren[node.path] || [];

    // Find label if any
    const label = data?.folder_labels.find((l) => l.path === node.path);

    return (
      <div key={node.path} className="select-none text-xs">
        <div
          onDragOver={(e) => handleDragOver(e, node.path)}
          onDragLeave={handleDragLeave}
          onDrop={(e) => handleDrop(e, node.path)}
          onClick={() => {
            onNavigate(node.path);
            if (node.child_count > 0 && !isExpanded) {
              toggleTreeNode(node.path);
            }
          }}
          onContextMenu={(e) => {
            e.preventDefault();
            e.stopPropagation();
            if (onDeleteFolder) {
              onDeleteFolder(node.path);
            }
          }}
          style={{ paddingLeft: `${depth * 14 + 10}px` }}
          className={`group flex items-center justify-between py-1.5 pr-2 rounded-lg cursor-pointer transition ${
            isSelected
              ? 'bg-emerald-50 text-emerald-900 font-semibold border-l-2 border-emerald-600 dark:bg-emerald-950/40 dark:text-emerald-200'
              : 'text-slate-700 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-surface-800'
          } ${isDragOver ? 'ring-2 ring-emerald-500 bg-emerald-100/60' : ''}`}
        >
          <div className="flex items-center gap-1.5 min-w-0">
            {node.child_count > 0 ? (
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  toggleTreeNode(node.path);
                }}
                className="p-0.5 rounded hover:bg-slate-200 dark:hover:bg-surface-700 text-slate-400 hover:text-slate-600 transition"
              >
                {isLoading ? (
                  <RefreshCw className="w-3 h-3 animate-spin text-emerald-600" />
                ) : isExpanded ? (
                  <ChevronDown className="w-3.5 h-3.5" />
                ) : (
                  <ChevronRight className="w-3.5 h-3.5" />
                )}
              </button>
            ) : (
              <span className="w-4" />
            )}

            <Folder
              className={`w-3.5 h-3.5 shrink-0 ${
                label ? getColorClass(label.color).replace('bg-', 'text-') : 'text-amber-500'
              }`}
            />
            <span className="truncate">{node.name}</span>
          </div>

          <div className="flex items-center gap-1">
            {label && (
              <span
                className={`w-2 h-2 rounded-full ${getColorClass(label.color)}`}
                title={label.label || 'Tagged folder'}
              />
            )}
            {node.child_count > 0 && (
              <span className={`text-[10px] text-slate-400 font-mono px-1 rounded bg-slate-100 dark:bg-surface-800 ${onDeleteFolder ? 'group-hover:hidden' : ''}`}>
                {node.child_count}
              </span>
            )}
            {onDeleteFolder && (
              <button
                type="button"
                title={`Delete ${node.name}`}
                onClick={(e) => {
                  e.stopPropagation();
                  onDeleteFolder(node.path);
                }}
                className="hidden group-hover:flex items-center justify-center p-1 rounded-md text-slate-400 hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/40 transition shrink-0"
              >
                <Trash2 className="w-3.5 h-3.5" />
              </button>
            )}
          </div>
        </div>

        {isExpanded && children.length > 0 && (
          <div className="border-l border-slate-200 dark:border-surface-800 ml-4">
            {children.map((child) => renderTreeNode(child, depth + 1))}
          </div>
        )}
      </div>
    );
  };

  return (
    <>
      {/* Mobile backdrop */}
      {isMobileOpen && (
        <div
          onClick={onCloseMobile}
          className="fixed inset-0 z-40 bg-black/40 backdrop-blur-xs md:hidden"
        />
      )}

      <aside
        className={`fixed md:static inset-y-0 left-0 z-50 w-72 md:w-64 lg:w-72 bg-white dark:bg-surface-900 border-r border-slate-200 dark:border-surface-800 flex flex-col transition-transform duration-200 ease-in-out shrink-0 ${
          isMobileOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'
        }`}
      >
        {/* Sidebar Header & Quick Filter */}
        <div className="p-3 border-b border-slate-200 dark:border-surface-800">
          <div className="flex items-center justify-between mb-2.5">
            <div className="flex items-center gap-2">
              <div className="w-7 h-7 rounded-lg bg-emerald-600 text-white flex items-center justify-center font-bold text-xs shadow-xs">
                HV
              </div>
              <div>
                <h3 className="text-xs font-bold text-slate-900 dark:text-white uppercase tracking-wider">
                  Quick Access
                </h3>
                <p className="text-[10px] text-slate-500">Explorer & Roots</p>
              </div>
            </div>

            <div className="flex items-center gap-1">
              <button
                type="button"
                onClick={fetchQuickAccess}
                title="Refresh Quick Access"
                className="p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-surface-800 transition"
              >
                <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
              </button>
              <button
                type="button"
                onClick={onCloseMobile}
                className="md:hidden p-1 rounded-lg text-slate-400 hover:text-slate-700"
              >
                <X className="w-4 h-4" />
              </button>
            </div>
          </div>

          {/* Quick Search */}
          <div className="relative">
            <Search className="w-3.5 h-3.5 absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search folders & domains..."
              className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-slate-800 dark:text-slate-200 focus:outline-hidden focus:ring-1 focus:ring-emerald-500"
            />
            {search && (
              <button
                type="button"
                onClick={() => setSearch('')}
                className="absolute right-2 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600"
              >
                <X className="w-3 h-3" />
              </button>
            )}
          </div>
        </div>

        {/* Scrollable Content */}
        <div className="flex-1 overflow-y-auto p-2.5 space-y-4 text-xs font-sans">
          {/* SECTION 1: FAVORITES */}
          <div>
            <button
              type="button"
              onClick={() => toggleSection('favorites')}
              className="w-full flex items-center justify-between py-1 px-1.5 text-slate-500 hover:text-slate-900 dark:hover:text-white font-bold text-[11px] uppercase tracking-wider transition"
            >
              <span className="flex items-center gap-1.5">
                <Star className="w-3.5 h-3.5 text-amber-500 fill-amber-500" />
                <span>Favorites</span>
              </span>
              <div className="flex items-center gap-1">
                <span className="text-[10px] font-mono text-slate-400">
                  {filteredFavorites.length}
                </span>
                {expandedSections.favorites ? (
                  <ChevronDown className="w-3 h-3 text-slate-400" />
                ) : (
                  <ChevronRight className="w-3 h-3 text-slate-400" />
                )}
              </div>
            </button>

            {expandedSections.favorites && (
              <div className="mt-1 space-y-0.5">
                {filteredFavorites.length === 0 ? (
                  <p className="px-2 py-1 text-[11px] text-slate-400 italic">
                    No pinned favorites. Star folders from context menu.
                  </p>
                ) : (
                  filteredFavorites.map((fav) => {
                    const isSelected = currentPath === fav.path && !isTrashActive;
                    return (
                      <div
                        key={fav.id}
                        onClick={() => onNavigate(fav.path, fav.domain)}
                        onDragOver={(e) => handleDragOver(e, fav.path)}
                        onDragLeave={handleDragLeave}
                        onDrop={(e) => handleDrop(e, fav.path)}
                        className={`group flex items-center justify-between px-2 py-1.5 rounded-lg cursor-pointer transition ${
                          isSelected
                            ? 'bg-emerald-50 text-emerald-900 font-semibold border-l-2 border-emerald-600 dark:bg-emerald-950/40 dark:text-emerald-200'
                            : 'text-slate-700 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-surface-800'
                        } ${dragOverPath === fav.path ? 'ring-2 ring-emerald-500 bg-emerald-100/60' : ''}`}
                      >
                        <div className="flex items-center gap-2 min-w-0">
                          <Folder
                            className={`w-3.5 h-3.5 shrink-0 ${
                              fav.color ? getColorClass(fav.color).replace('bg-', 'text-') : 'text-amber-500'
                            }`}
                          />
                          <span className="truncate">{fav.name}</span>
                        </div>
                        <div className="flex items-center gap-1">
                          {fav.domain && (
                            <span className={`text-[10px] text-slate-400 truncate max-w-[80px] ${onDeleteFolder ? 'group-hover:hidden' : ''}`}>
                              {fav.domain}
                            </span>
                          )}
                          {onDeleteFolder && (
                            <button
                              type="button"
                              title={`Delete ${fav.name}`}
                              onClick={(e) => {
                                e.stopPropagation();
                                onDeleteFolder(fav.path);
                              }}
                              className="hidden group-hover:flex items-center justify-center p-1 rounded-md text-slate-400 hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/40 transition shrink-0"
                            >
                              <Trash2 className="w-3.5 h-3.5" />
                            </button>
                          )}
                        </div>
                      </div>
                    );
                  })
                )}
              </div>
            )}
          </div>

          {/* SECTION 2: MY DOMAINS */}
          <div>
            <button
              type="button"
              onClick={() => toggleSection('domains')}
              className="w-full flex items-center justify-between py-1 px-1.5 text-slate-500 hover:text-slate-900 dark:hover:text-white font-bold text-[11px] uppercase tracking-wider transition"
            >
              <span className="flex items-center gap-1.5">
                <Globe className="w-3.5 h-3.5 text-blue-600" />
                <span>My Domains</span>
              </span>
              <div className="flex items-center gap-1">
                <span className="text-[10px] font-mono text-slate-400">
                  {filteredDomains.length}
                </span>
                {expandedSections.domains ? (
                  <ChevronDown className="w-3 h-3 text-slate-400" />
                ) : (
                  <ChevronRight className="w-3 h-3 text-slate-400" />
                )}
              </div>
            </button>

            {expandedSections.domains && (
              <div className="mt-1 space-y-0.5">
                {filteredDomains.length === 0 ? (
                  <p className="px-2 py-1 text-[11px] text-slate-400 italic">No websites configured</p>
                ) : (
                  filteredDomains.map((dom) => {
                    const isSelected = activeDomain === dom.domain && !isTrashActive;
                    return (
                      <div
                        key={dom.id}
                        onClick={() => {
                          onDomainChange(dom.domain);
                          onNavigate(dom.document_root, dom.domain);
                        }}
                        onDragOver={(e) => handleDragOver(e, dom.document_root)}
                        onDragLeave={handleDragLeave}
                        onDrop={(e) => handleDrop(e, dom.document_root)}
                        className={`group flex items-center justify-between px-2 py-1.5 rounded-lg cursor-pointer transition ${
                          isSelected
                            ? 'bg-blue-50 text-blue-900 font-semibold border-l-2 border-blue-600 dark:bg-blue-950/40 dark:text-blue-200'
                            : 'text-slate-700 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-surface-800'
                        } ${dragOverPath === dom.document_root ? 'ring-2 ring-blue-500 bg-blue-100/60' : ''}`}
                      >
                        <div className="flex items-center gap-2 min-w-0">
                          <Globe className="w-3.5 h-3.5 text-blue-500 shrink-0" />
                          <span className="truncate">{dom.domain}</span>
                        </div>
                        <span
                          className={`w-1.5 h-1.5 rounded-full shrink-0 ${
                            dom.status === 'active' ? 'bg-emerald-500' : 'bg-slate-400'
                          }`}
                        />
                      </div>
                    );
                  })
                )}
              </div>
            )}
          </div>

          {/* SECTION 3: SYSTEM SHORTCUTS & TRASH */}
          <div>
            <button
              type="button"
              onClick={() => toggleSection('shortcuts')}
              className="w-full flex items-center justify-between py-1 px-1.5 text-slate-500 hover:text-slate-900 dark:hover:text-white font-bold text-[11px] uppercase tracking-wider transition"
            >
              <span className="flex items-center gap-1.5">
                <HardDrive className="w-3.5 h-3.5 text-slate-600 dark:text-slate-400" />
                <span>Shortcuts</span>
              </span>
              {expandedSections.shortcuts ? (
                <ChevronDown className="w-3 h-3 text-slate-400" />
              ) : (
                <ChevronRight className="w-3 h-3 text-slate-400" />
              )}
            </button>

            {expandedSections.shortcuts && (
              <div className="mt-1 space-y-0.5">
                {/* Root Directory */}
                <div
                  onClick={() => onNavigate(data?.root_directory || '/var/www')}
                  className={`flex items-center justify-between px-2 py-1.5 rounded-lg cursor-pointer transition ${
                    currentPath === (data?.root_directory || '/var/www') && !isTrashActive
                      ? 'bg-emerald-50 text-emerald-900 font-semibold border-l-2 border-emerald-600'
                      : 'text-slate-700 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-surface-800'
                  }`}
                >
                  <div className="flex items-center gap-2 min-w-0">
                    <Folder className="w-3.5 h-3.5 text-amber-500" />
                    <span>Root Directory</span>
                  </div>
                  <span className="text-[10px] text-slate-400 font-mono">
                    {data?.root_directory || '/var/www'}
                  </span>
                </div>

                {/* Backups */}
                <div
                  onClick={() => onNavigate('/var/backups')}
                  className={`flex items-center justify-between px-2 py-1.5 rounded-lg cursor-pointer transition ${
                    currentPath === '/var/backups' && !isTrashActive
                      ? 'bg-emerald-50 text-emerald-900 font-semibold border-l-2 border-emerald-600'
                      : 'text-slate-700 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-surface-800'
                  }`}
                >
                  <div className="flex items-center gap-2 min-w-0">
                    <Archive className="w-3.5 h-3.5 text-purple-500" />
                    <span>Backups</span>
                  </div>
                </div>

                {/* Uploads */}
                <div
                  onClick={() => onNavigate(data?.root_directory + '/uploads')}
                  className="flex items-center justify-between px-2 py-1.5 rounded-lg cursor-pointer text-slate-700 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-surface-800"
                >
                  <div className="flex items-center gap-2 min-w-0">
                    <Upload className="w-3.5 h-3.5 text-sky-500" />
                    <span>Uploads</span>
                  </div>
                </div>

                {/* Downloads */}
                <div
                  onClick={() => onNavigate(data?.root_directory + '/downloads')}
                  className="flex items-center justify-between px-2 py-1.5 rounded-lg cursor-pointer text-slate-700 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-surface-800"
                >
                  <div className="flex items-center gap-2 min-w-0">
                    <Download className="w-3.5 h-3.5 text-emerald-500" />
                    <span>Downloads</span>
                  </div>
                </div>

                {/* TRASH BIN ITEM WITH DRAG TARGET */}
                <div
                  onClick={onTrashClick}
                  onDragOver={(e) => handleDragOver(e, 'TRASH_BIN')}
                  onDragLeave={handleDragLeave}
                  onDrop={(e) => handleDrop(e, 'TRASH_BIN', true)}
                  className={`flex items-center justify-between px-2 py-1.5 rounded-lg cursor-pointer transition ${
                    isTrashActive
                      ? 'bg-rose-50 text-rose-900 font-semibold border-l-2 border-rose-600 dark:bg-rose-950/40 dark:text-rose-200'
                      : 'text-slate-700 hover:bg-rose-50/60 dark:text-slate-300 hover:text-rose-600'
                  } ${dragOverPath === 'TRASH_BIN' ? 'ring-2 ring-rose-500 bg-rose-100 scale-102 font-bold' : ''}`}
                >
                  <div className="flex items-center gap-2 min-w-0">
                    <Trash2 className="w-3.5 h-3.5 text-rose-500" />
                    <span>Trash Bin</span>
                  </div>
                  <div className="flex items-center gap-1.5">
                    {data?.trash_stats?.count ? (
                      <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-rose-100 text-rose-700 font-mono font-bold dark:bg-rose-900/60 dark:text-rose-300">
                        {data.trash_stats.count}
                      </span>
                    ) : null}
                    {data?.trash_stats?.bytes ? (
                      <span className="text-[10px] text-slate-400 font-mono">
                        {formatBytes(data.trash_stats.bytes)}
                      </span>
                    ) : null}
                  </div>
                </div>
              </div>
            )}
          </div>

          {/* SECTION 4: RECENT FOLDERS */}
          <div>
            <button
              type="button"
              onClick={() => toggleSection('recent')}
              className="w-full flex items-center justify-between py-1 px-1.5 text-slate-500 hover:text-slate-900 dark:hover:text-white font-bold text-[11px] uppercase tracking-wider transition"
            >
              <span className="flex items-center gap-1.5">
                <Clock className="w-3.5 h-3.5 text-slate-600 dark:text-slate-400" />
                <span>Recent Folders</span>
              </span>
              <div className="flex items-center gap-1">
                <span className="text-[10px] font-mono text-slate-400">{filteredRecent.length}</span>
                {expandedSections.recent ? (
                  <ChevronDown className="w-3 h-3 text-slate-400" />
                ) : (
                  <ChevronRight className="w-3 h-3 text-slate-400" />
                )}
              </div>
            </button>

            {expandedSections.recent && (
              <div className="mt-1 space-y-0.5">
                {filteredRecent.length === 0 ? (
                  <p className="px-2 py-1 text-[11px] text-slate-400 italic">No recent history</p>
                ) : (
                  filteredRecent.slice(0, 6).map((rec) => (
                    <div
                      key={rec.id}
                      onClick={() => onNavigate(rec.path, rec.domain)}
                      className={`group flex items-center justify-between px-2 py-1.5 rounded-lg cursor-pointer transition ${
                        currentPath === rec.path && !isTrashActive
                          ? 'bg-slate-100 font-semibold dark:bg-surface-800'
                          : 'text-slate-700 hover:bg-slate-50 dark:text-slate-300 dark:hover:bg-surface-800'
                      }`}
                    >
                      <div className="flex items-center gap-2 min-w-0">
                        <Folder className="w-3.5 h-3.5 text-slate-400 shrink-0" />
                        <span className="truncate">{filepathBase(rec.path)}</span>
                      </div>
                      <div className="flex items-center gap-1">
                        <span className={`text-[10px] text-slate-400 font-mono truncate max-w-[90px] ${onDeleteFolder ? 'group-hover:hidden' : ''}`}>
                          {rec.domain || filepathDir(rec.path)}
                        </span>
                        {onDeleteFolder && (
                          <button
                            type="button"
                            title={`Delete ${filepathBase(rec.path)}`}
                            onClick={(e) => {
                              e.stopPropagation();
                              onDeleteFolder(rec.path);
                            }}
                            className="hidden group-hover:flex items-center justify-center p-1 rounded-md text-slate-400 hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/40 transition shrink-0"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        )}
                      </div>
                    </div>
                  ))
                )}
              </div>
            )}
          </div>

          {/* SECTION 5: FOLDER EXPLORER TREE */}
          <div>
            <button
              type="button"
              onClick={() => toggleSection('tree')}
              className="w-full flex items-center justify-between py-1 px-1.5 text-slate-500 hover:text-slate-900 dark:hover:text-white font-bold text-[11px] uppercase tracking-wider transition"
            >
              <span className="flex items-center gap-1.5">
                <FolderTree className="w-3.5 h-3.5 text-emerald-600" />
                <span>Folder Tree</span>
              </span>
              {expandedSections.tree ? (
                <ChevronDown className="w-3 h-3 text-slate-400" />
              ) : (
                <ChevronRight className="w-3 h-3 text-slate-400" />
              )}
            </button>

            {expandedSections.tree && (
              <div className="mt-1 space-y-0.5">
                {treeChildren[data?.root_directory || '/var/www']?.map((node) =>
                  renderTreeNode(node, 0)
                )}
              </div>
            )}
          </div>
        </div>

        {/* Sidebar Footer info */}
        <div className="p-3 border-t border-slate-200 dark:border-surface-800 text-[11px] text-slate-400 flex items-center justify-between">
          <span>HostVra v3.0</span>
          <span className="text-emerald-600 dark:text-emerald-400 font-medium">Enterprise Explorer</span>
        </div>
      </aside>
    </>
  );
};

function filepathBase(p: string): string {
  if (!p) return '';
  const parts = p.replace(/\\/g, '/').split('/').filter(Boolean);
  return parts[parts.length - 1] || p;
}

function filepathDir(p: string): string {
  if (!p) return '';
  const parts = p.replace(/\\/g, '/').split('/').filter(Boolean);
  if (parts.length <= 1) return '/';
  return parts.slice(0, -1).join('/');
}
