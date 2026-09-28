'use client';

import { useState, useEffect } from 'react';
import { useWebmail } from '@/context/WebmailContext';
import {
  Users,
  UserPlus,
  Search,
  Mail,
  Phone,
  Trash2,
  Edit2,
  Check,
  AlertCircle,
  X,
  Send,
} from 'lucide-react';

export default function WebmailContactsPage() {
  const { contacts, fetchContacts, saveContact, deleteContact, openCompose } = useWebmail();
  const [searchQuery, setSearchQuery] = useState('');
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);

  // Form State
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [phone, setPhone] = useState('');
  const [company, setCompany] = useState('');
  const [groupName, setGroupName] = useState('');
  const [notes, setNotes] = useState('');

  const [notification, setNotification] = useState<{ text: string; isError?: boolean } | null>(null);

  useEffect(() => {
    fetchContacts();
  }, []);

  const showNotify = (text: string, isError = false) => {
    setNotification({ text, isError });
    setTimeout(() => setNotification(null), 3500);
  };

  const openAddModal = () => {
    setEditingId(null);
    setName('');
    setEmail('');
    setPhone('');
    setCompany('');
    setGroupName('');
    setNotes('');
    setIsModalOpen(true);
  };

  const openEditModal = (c: any) => {
    setEditingId(c.id);
    setName(c.name || '');
    setEmail(c.email || '');
    setPhone(c.phone || '');
    setCompany(c.company || '');
    setGroupName(c.group_name || '');
    setNotes(c.notes || '');
    setIsModalOpen(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name || !email) {
      showNotify('Name and email are required', true);
      return;
    }

    const ok = await saveContact({
      id: editingId || undefined,
      name,
      email,
      phone,
      company,
      group_name: groupName,
      notes,
    });

    if (ok) {
      showNotify(editingId ? 'Contact updated successfully' : 'Contact created successfully');
      setIsModalOpen(false);
    } else {
      showNotify('Failed to save contact', true);
    }
  };

  const handleDelete = async (id: string, contactName: string) => {
    if (!confirm(`Are you sure you want to delete contact "${contactName}"?`)) return;
    const ok = await deleteContact(id);
    if (ok) {
      showNotify('Contact deleted successfully');
    } else {
      showNotify('Failed to delete contact', true);
    }
  };

  const handleComposeTo = (contactEmail: string, contactName: string) => {
    openCompose({
      to: `"${contactName}" <${contactEmail}>`,
      subject: '',
      bodyHTML: '',
    });
  };

  const filteredContacts = contacts.filter((c) => {
    const q = searchQuery.toLowerCase();
    return (
      c.name?.toLowerCase().includes(q) ||
      c.email?.toLowerCase().includes(q) ||
      c.company?.toLowerCase().includes(q) ||
      c.group_name?.toLowerCase().includes(q)
    );
  });

  return (
    <div className="flex-1 overflow-y-auto bg-slate-950 p-6 md:p-8">
      <div className="max-w-6xl mx-auto space-y-6">
        {/* Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-4 border-b border-slate-800">
          <div>
            <h1 className="text-2xl font-bold text-white flex items-center gap-2.5">
              <Users className="w-6 h-6 text-sky-400" />
              Contacts & Address Book
            </h1>
            <p className="text-xs text-slate-400 mt-1">
              Manage your personal contacts and autocomplete recipients when composing messages.
            </p>
          </div>

          <button
            onClick={openAddModal}
            className="flex items-center gap-2 px-4 py-2.5 bg-sky-500 hover:bg-sky-400 text-white rounded-xl text-xs font-semibold shadow-lg shadow-sky-500/20 transition-all self-start sm:self-auto"
          >
            <UserPlus className="w-4 h-4" />
            Add Contact
          </button>
        </div>

        {/* Notifications */}
        {notification && (
          <div
            className={`p-3.5 rounded-xl text-xs flex items-center gap-2 animate-in fade-in ${
              notification.isError
                ? 'bg-rose-950/80 border border-rose-500/40 text-rose-200'
                : 'bg-emerald-950/80 border border-emerald-500/40 text-emerald-200'
            }`}
          >
            {notification.isError ? (
              <AlertCircle className="w-4 h-4 text-rose-400 shrink-0" />
            ) : (
              <Check className="w-4 h-4 text-emerald-400 shrink-0" />
            )}
            {notification.text}
          </div>
        )}

        {/* Search Bar */}
        <div className="relative">
          <Search className="absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400 w-4 h-4" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search contacts by name, email, company or group..."
            className="w-full bg-slate-900 border border-slate-800 rounded-xl pl-10 pr-4 py-2.5 text-xs text-white placeholder-slate-500 focus:outline-none focus:border-sky-500"
          />
        </div>

        {/* Contacts Grid */}
        {filteredContacts.length === 0 ? (
          <div className="text-center py-16 bg-slate-900/40 border border-slate-800/80 rounded-2xl">
            <Users className="w-12 h-12 text-slate-600 mx-auto mb-3" />
            <p className="text-sm font-medium text-slate-300">
              {searchQuery ? 'No contacts match your search query.' : 'Your address book is empty.'}
            </p>
            <p className="text-xs text-slate-500 mt-1">
              Add contacts to easily autocomplete recipient emails in the composer.
            </p>
            {!searchQuery && (
              <button
                onClick={openAddModal}
                className="mt-4 inline-flex items-center gap-2 px-4 py-2 bg-sky-500 hover:bg-sky-400 text-white rounded-xl text-xs font-semibold shadow-lg shadow-sky-500/20 transition-all"
              >
                <UserPlus className="w-3.5 h-3.5" />
                Add First Contact
              </button>
            )}
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {filteredContacts.map((c) => (
              <div
                key={c.id}
                className="bg-slate-900/70 border border-slate-800/80 hover:border-slate-700/80 rounded-2xl p-4.5 space-y-3.5 transition-all flex flex-col justify-between group"
              >
                <div className="space-y-2.5">
                  <div className="flex items-start justify-between gap-2">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-full bg-gradient-to-tr from-sky-600 to-indigo-600 flex items-center justify-center text-white font-bold text-sm shadow-md shadow-sky-950">
                        {c.name.slice(0, 1).toUpperCase()}
                      </div>
                      <div>
                        <h3 className="text-xs font-bold text-white group-hover:text-sky-300 transition-colors">
                          {c.name}
                        </h3>
                        {c.company && (
                          <span className="text-[11px] text-slate-400 block">{c.company}</span>
                        )}
                      </div>
                    </div>

                    {c.group_name && (
                      <span className="text-[10px] px-2 py-0.5 rounded-full bg-slate-800 text-slate-300 border border-slate-700">
                        {c.group_name}
                      </span>
                    )}
                  </div>

                  <div className="space-y-1.5 pt-1 text-xs text-slate-300">
                    <div className="flex items-center gap-2">
                      <Mail className="w-3.5 h-3.5 text-sky-400 shrink-0" />
                      <span className="truncate">{c.email}</span>
                    </div>
                    {c.phone && (
                      <div className="flex items-center gap-2">
                        <Phone className="w-3.5 h-3.5 text-slate-500 shrink-0" />
                        <span className="truncate text-slate-400">{c.phone}</span>
                      </div>
                    )}
                  </div>
                </div>

                <div className="pt-3 border-t border-slate-800/60 flex items-center justify-between gap-2">
                  <button
                    onClick={() => handleComposeTo(c.email, c.name)}
                    className="flex items-center gap-1.5 px-3 py-1.5 bg-sky-500/10 hover:bg-sky-500 text-sky-400 hover:text-white rounded-lg text-xs font-medium transition-colors"
                  >
                    <Send className="w-3 h-3" />
                    Compose
                  </button>

                  <div className="flex items-center gap-1">
                    <button
                      onClick={() => openEditModal(c)}
                      className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-800 rounded-lg transition-colors"
                      title="Edit contact"
                    >
                      <Edit2 className="w-3.5 h-3.5" />
                    </button>
                    <button
                      onClick={() => handleDelete(c.id, c.name)}
                      className="p-1.5 text-slate-400 hover:text-rose-400 hover:bg-rose-950/40 rounded-lg transition-colors"
                      title="Delete contact"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Modal: Add / Edit Contact */}
        {isModalOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm animate-in fade-in">
            <div className="bg-slate-900 border border-slate-800 rounded-2xl max-w-lg w-full p-6 space-y-4 shadow-2xl">
              <div className="flex items-center justify-between pb-3 border-b border-slate-800">
                <h3 className="text-sm font-bold text-white flex items-center gap-2">
                  <UserPlus className="w-4 h-4 text-sky-400" />
                  {editingId ? 'Edit Contact' : 'Add New Contact'}
                </h3>
                <button
                  onClick={() => setIsModalOpen(false)}
                  className="p-1 text-slate-400 hover:text-white rounded-lg"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              <form onSubmit={handleSubmit} className="space-y-4">
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3.5">
                  <div className="space-y-1">
                    <label className="text-xs font-medium text-slate-300">
                      Full Name <span className="text-sky-400">*</span>
                    </label>
                    <input
                      type="text"
                      required
                      placeholder="e.g. John Doe"
                      value={name}
                      onChange={(e) => setName(e.target.value)}
                      className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3 py-2 text-xs text-white focus:outline-none focus:border-sky-500"
                    />
                  </div>

                  <div className="space-y-1">
                    <label className="text-xs font-medium text-slate-300">
                      Email Address <span className="text-sky-400">*</span>
                    </label>
                    <input
                      type="email"
                      required
                      placeholder="e.g. john@example.com"
                      value={email}
                      onChange={(e) => setEmail(e.target.value)}
                      className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3 py-2 text-xs text-white focus:outline-none focus:border-sky-500"
                    />
                  </div>

                  <div className="space-y-1">
                    <label className="text-xs font-medium text-slate-300">Phone (Optional)</label>
                    <input
                      type="text"
                      placeholder="e.g. +880 1712 345678"
                      value={phone}
                      onChange={(e) => setPhone(e.target.value)}
                      className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3 py-2 text-xs text-white focus:outline-none focus:border-sky-500"
                    />
                  </div>

                  <div className="space-y-1">
                    <label className="text-xs font-medium text-slate-300">Company (Optional)</label>
                    <input
                      type="text"
                      placeholder="e.g. Acme Corp"
                      value={company}
                      onChange={(e) => setCompany(e.target.value)}
                      className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3 py-2 text-xs text-white focus:outline-none focus:border-sky-500"
                    />
                  </div>

                  <div className="space-y-1 sm:col-span-2">
                    <label className="text-xs font-medium text-slate-300">Group / Category (Optional)</label>
                    <input
                      type="text"
                      placeholder="e.g. Clients, Team, Vendors"
                      value={groupName}
                      onChange={(e) => setGroupName(e.target.value)}
                      className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3 py-2 text-xs text-white focus:outline-none focus:border-sky-500"
                    />
                  </div>

                  <div className="space-y-1 sm:col-span-2">
                    <label className="text-xs font-medium text-slate-300">Notes (Optional)</label>
                    <textarea
                      rows={2}
                      placeholder="Additional notes about this contact..."
                      value={notes}
                      onChange={(e) => setNotes(e.target.value)}
                      className="w-full bg-slate-950 border border-slate-800 rounded-xl p-2.5 text-xs text-white focus:outline-none focus:border-sky-500"
                    />
                  </div>
                </div>

                <div className="pt-3 border-t border-slate-800 flex items-center justify-end gap-2">
                  <button
                    type="button"
                    onClick={() => setIsModalOpen(false)}
                    className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-xl text-xs font-medium transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="px-4 py-2 bg-sky-500 hover:bg-sky-400 text-white rounded-xl text-xs font-semibold shadow-lg shadow-sky-500/20 transition-all"
                  >
                    {editingId ? 'Save Changes' : 'Add Contact'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
