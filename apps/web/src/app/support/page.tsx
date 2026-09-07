'use client';

import React, { useState, useEffect, useMemo } from 'react';
import { DashboardShell } from '@/components/DashboardShell';
import {
  LifeBuoy,
  MessageSquare,
  BookOpen,
  Plus,
  Search,
  CheckCircle2,
  Clock,
  AlertCircle,
  XCircle,
  ChevronRight,
  Send,
  User,
  Shield,
  ThumbsUp,
  ThumbsDown,
  Eye,
  ArrowRight,
  RefreshCw,
  Lock,
  Sparkles,
  Bot,
  Zap,
  Check,
  CheckCheck
} from 'lucide-react';
import {
  apiFetch,
  Ticket,
  TicketReply,
  KnowledgeArticle,
  CannedResponse,
  SupportStats,
  AIAssistantResponse,
  TicketDepartment,
  TicketPriority,
  TicketStatus
} from '@/lib/api';

const INITIAL_TICKETS: Ticket[] = [];

const INITIAL_ARTICLES: KnowledgeArticle[] = [
  {
    id: 'art-1',
    title: 'How to Point Your Domain Name to Hostvra DNS',
    slug: 'how-to-point-domain-to-hostvra-dns',
    category: 'dns_domains',
    summary: 'Step-by-step guide to updating nameservers at your registrar to ns1.hostvra.net and ns2.hostvra.net.',
    content: `### Pointing Nameservers to Hostvra\n\nTo point any domain registered at Namecheap, GoDaddy, or BTCL (.com.bd) to Hostvra:\n\n1. Log in to your domain registrar's control panel.\n2. Locate the **Nameservers (DNS)** settings.\n3. Switch nameservers to **Custom DNS** and enter:\n   - \`ns1.hostvra.net\`\n   - \`ns2.hostvra.net\`\n4. Save changes. DNS propagation typically completes within 15 minutes to 4 hours.`,
    views: 412,
    helpful_votes: 58,
    unhelpful_votes: 1,
    is_published: true,
    created_at: new Date(Date.now() - 30 * 24 * 3600 * 1000).toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 'art-2',
    title: 'Configuring PHP 8.3 & Required Extensions (OPcache, Redis)',
    slug: 'configuring-php-8-3-extensions-opcache-redis',
    category: 'hosting',
    summary: 'Learn how to select PHP versions, optimize memory limits, and enable OPcache & Redis for maximum performance.',
    content: `### PHP Optimization Guide\n\nHostvra supports multiple simultaneous PHP runtimes from 7.4 up to 8.3.\n\n- Navigate to **Websites** -> Select your website -> **PHP Version**.\n- Choose PHP 8.3 for up to 30% performance boost over PHP 8.0.\n- Enable OPcache in the PHP extensions tab.\n- Configure \`memory_limit = 512M\` and \`max_execution_time = 300\` for WordPress and Laravel apps.`,
    views: 680,
    helpful_votes: 94,
    unhelpful_votes: 2,
    is_published: true,
    created_at: new Date(Date.now() - 45 * 24 * 3600 * 1000).toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 'art-3',
    title: 'Enabling Free 1-Click Let\'s Encrypt Wildcard SSL',
    slug: 'enabling-free-1-click-lets-encrypt-ssl',
    category: 'security',
    summary: 'Issue and automate 90-day Let\'s Encrypt certificates with auto-renewal and HTTP to HTTPS redirection.',
    content: `### Automated Free SSL Certificates\n\nAll hosting accounts and websites hosted on Hostvra include automated Let's Encrypt SSL.\n\n1. Go to **SSL Certificates** in your dashboard.\n2. Select your domain name.\n3. Choose **HTTP-01** (Standard) or **DNS-01** (Wildcard \`*.yourdomain.com\`).\n4. Click **Apply SSL**.\n5. Toggle **Force HTTPS** to automatically redirect all visitors to secure HTTPS.`,
    views: 524,
    helpful_votes: 81,
    unhelpful_votes: 0,
    is_published: true,
    created_at: new Date(Date.now() - 20 * 24 * 3600 * 1000).toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 'art-4',
    title: 'Setting up Email Deliverability (SPF, DKIM, DMARC)',
    slug: 'email-deliverability-spf-dkim-dmarc',
    category: 'email',
    summary: 'Avoid the spam folder by configuring authenticated DNS TXT records for your mailboxes.',
    content: `### Email Authentication Essentials\n\nTo ensure 100% inbox delivery in Gmail, Yahoo, and Outlook:\n\n- **SPF Record**: \`v=spf1 mx a include:_spf.hostvra.com ~all\`\n- **DKIM**: Auto-generated 2048-bit RSA key found under **Email Accounts** -> **DKIM Keys**.\n- **DMARC**: Add TXT record for \`_dmarc.yourdomain.com\` with value \`v=DMARC1; p=quarantine; pct=100; adkim=r; aspf=r\`.`,
    views: 310,
    helpful_votes: 43,
    unhelpful_votes: 1,
    is_published: true,
    created_at: new Date(Date.now() - 60 * 24 * 3600 * 1000).toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 'art-5',
    title: 'Payment Methods & Auto-Invoicing (bKash, Nagad, Stripe)',
    slug: 'payment-methods-and-auto-invoicing',
    category: 'billing',
    summary: 'Information on local Bangladeshi mobile payments (bKash, Nagad) and international credit cards.',
    content: `### Supported Payment Gateways\n\nHostvra Enterprise billing supports instant automated reconciliation for:\n\n- **bKash & Nagad**: Instant mobile payment using QR or OTP.\n- **Credit/Debit Cards**: Visa, MasterCard, Amex processed securely via Stripe or SSLCommerz.\n- **Invoicing**: Invoices are generated 7 days prior to service expiration and marked PAID instantly upon gateway callback.`,
    views: 295,
    helpful_votes: 39,
    unhelpful_votes: 0,
    is_published: true,
    created_at: new Date(Date.now() - 15 * 24 * 3600 * 1000).toISOString(),
    updated_at: new Date().toISOString(),
  },
];

const INITIAL_CANNED: CannedResponse[] = [
  {
    id: 'can-1',
    title: 'DNS Propagation Notice',
    shortcut: 'dns_prop',
    department: 'technical',
    content: 'Hello,\n\nWe have verified your DNS records. They are fully provisioned across our global Anycast nameservers. Due to ISP DNS caching, global propagation may take 15 to 60 minutes. You can test directly using our Whois Inspector or clear local cache via `ipconfig /flushdns`.\n\nBest regards,\nHostvra Support Team',
    created_at: new Date().toISOString()
  },
  {
    id: 'can-2',
    title: 'PHP Memory & Upload Limits',
    shortcut: 'php_limit',
    department: 'technical',
    content: 'Hello,\n\nWe have updated your PHP runtime limits. The configuration `memory_limit` has been set to 512M and `upload_max_filesize` / `post_max_size` to 128M. PHP-FPM pools have been reloaded. Please test your application now.\n\nBest regards,\nHostvra Support Team',
    created_at: new Date().toISOString()
  },
  {
    id: 'can-3',
    title: 'SSL Certificate Verification',
    shortcut: 'ssl_verify',
    department: 'technical',
    content: 'Hello,\n\nTo complete automated Let\'s Encrypt SSL issuance, port 80 and 443 must be reachable and your A record must point directly to your server IP. We verified your DNS record and re-issued the 90-day SSL certificate. HTTPS is now active.\n\nBest regards,\nHostvra Support Team',
    created_at: new Date().toISOString()
  },
  {
    id: 'can-4',
    title: 'Invoice Payment Confirmation',
    shortcut: 'invoice_paid',
    department: 'billing',
    content: 'Hello,\n\nThank you for your payment. Your invoice has been marked PAID and automated renewal for your hosting subscription is confirmed. You can download your PDF tax receipt directly from the Billing tab.\n\nBest regards,\nHostvra Billing Department',
    created_at: new Date().toISOString()
  }
];

export default function SupportPage() {
  const [activeTab, setActiveTab] = useState<'tickets' | 'ai-assistant' | 'knowledgebase'>('tickets');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Tickets State
  const [tickets, setTickets] = useState<Ticket[]>(INITIAL_TICKETS);
  const [cannedResponses, setCannedResponses] = useState<CannedResponse[]>(INITIAL_CANNED);
  const [stats, setStats] = useState<SupportStats | null>({
    total_tickets: 0,
    open_tickets: 0,
    answered_tickets: 0,
    closed_tickets: 0,
    avg_response_mins: 0,
    resolution_rate: 100,
    total_articles: INITIAL_ARTICLES.length,
    article_helpful_pct: 100
  });

  const [statusFilter, setStatusFilter] = useState('all');
  const [deptFilter, setDeptFilter] = useState('all');
  const [searchFilter, setSearchFilter] = useState('');

  // Selected Ticket Detail & Replies
  const [selectedTicket, setSelectedTicket] = useState<Ticket | null>(null);
  const [ticketReplies, setTicketReplies] = useState<TicketReply[]>([]);
  const [detailModalOpen, setDetailModalOpen] = useState(false);
  const [replyMessage, setReplyMessage] = useState('');
  const [replyIsPrivate, setReplyIsPrivate] = useState(false);
  const [isSendingReply, setIsSendingReply] = useState(false);

  // Create Ticket Modal
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [newSubject, setNewSubject] = useState('');
  const [newDepartment, setNewDepartment] = useState<TicketDepartment>('technical');
  const [newPriority, setNewPriority] = useState<TicketPriority>('medium');
  const [newRelatedService, setNewRelatedService] = useState('');
  const [newInitialMessage, setNewInitialMessage] = useState('');
  const [isCreatingTicket, setIsCreatingTicket] = useState(false);

  // Knowledgebase State
  const [articles, setArticles] = useState<KnowledgeArticle[]>(INITIAL_ARTICLES);
  const [articleCategory, setArticleCategory] = useState('all');
  const [articleSearch, setArticleSearch] = useState('');
  const [readingArticle, setReadingArticle] = useState<KnowledgeArticle | null>(null);
  const [hasVoted, setHasVoted] = useState(false);

  // AI Diagnostic Assistant State
  const [aiQuery, setAiQuery] = useState('');
  const [aiResponse, setAiResponse] = useState<AIAssistantResponse | null>(null);
  const [isAskingAI, setIsAskingAI] = useState(false);

  // Fetch Tickets & Articles & Stats
  const fetchData = async () => {
    try {
      setLoading(true);
      const [ticketsRes, articlesRes, statsRes, cannedRes] = await Promise.all([
        apiFetch<Ticket[]>('/api/v1/support/tickets'),
        apiFetch<KnowledgeArticle[]>('/api/v1/support/articles'),
        apiFetch<SupportStats>('/api/v1/support/stats'),
        apiFetch<CannedResponse[]>('/api/v1/support/canned')
      ]);

      if (ticketsRes.success && ticketsRes.data && ticketsRes.data.length > 0) {
        setTickets(ticketsRes.data);
      }
      if (articlesRes.success && articlesRes.data && articlesRes.data.length > 0) {
        setArticles(articlesRes.data);
      }
      if (statsRes.success && statsRes.data) {
        setStats(statsRes.data);
      }
      if (cannedRes.success && cannedRes.data && cannedRes.data.length > 0) {
        setCannedResponses(cannedRes.data);
      }
    } catch (err: any) {
      console.warn('Using seeded support data:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  // Open Ticket Detail
  const handleOpenTicket = async (ticket: Ticket) => {
    setSelectedTicket(ticket);
    setDetailModalOpen(true);
    setReplyMessage('');
    setReplyIsPrivate(false);

    try {
      const res = await apiFetch<{ ticket: Ticket; replies: TicketReply[] }>(
        `/api/v1/support/tickets/${ticket.id}`
      );
      if (res.success && res.data) {
        setSelectedTicket(res.data.ticket);
        setTicketReplies(res.data.replies || []);
      } else {
        generateMockReplies(ticket);
      }
    } catch {
      generateMockReplies(ticket);
    }
  };

  const generateMockReplies = (ticket: Ticket) => {
    setTicketReplies([
      {
        id: 'rep-1',
        ticket_id: ticket.id,
        user_id: ticket.user_id,
        user_email: ticket.user_email,
        user_name: ticket.user_name,
        is_staff: false,
        message: 'Hello, I opened this ticket regarding ' + ticket.subject + '. Please assist as soon as possible.',
        created_at: ticket.created_at
      },
      ...(ticket.status === 'answered'
        ? [
            {
              id: 'rep-2',
              ticket_id: ticket.id,
              user_id: 'staff-1',
              user_email: 'support@hostvra.com',
              user_name: 'Hostvra Support Specialist',
              is_staff: true,
              message: 'Hello ' + ticket.user_name + ', our engineering team has looked into your query. The configuration has been updated and the service is performing normally.',
              created_at: ticket.last_reply_at
            }
          ]
        : [])
    ]);
  };

  // Send Reply
  const handleSendReply = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedTicket || !replyMessage.trim()) return;

    try {
      setIsSendingReply(true);
      setError(null);

      const res = await apiFetch<TicketReply>(
        `/api/v1/support/tickets/${selectedTicket.id}/reply`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            message: replyMessage.trim(),
            is_private_note: replyIsPrivate
          })
        }
      );

      const newReply: TicketReply = res.success && res.data ? res.data : {
        id: 'rep-' + Date.now(),
        ticket_id: selectedTicket.id,
        user_id: 'current-user',
        user_email: 'you@hostvra.com',
        user_name: replyIsPrivate ? 'Internal Staff Note' : 'Customer Reply',
        is_staff: replyIsPrivate,
        is_private_note: replyIsPrivate,
        message: replyMessage.trim(),
        created_at: new Date().toISOString()
      };

      setTicketReplies((prev) => [...prev, newReply]);
      setReplyMessage('');
      setReplyIsPrivate(false);
      setSuccessMessage(replyIsPrivate ? 'Internal note added.' : 'Reply sent successfully.');

      if (!replyIsPrivate) {
        setTickets((prev) =>
          prev.map((t) =>
            t.id === selectedTicket.id
              ? { ...t, status: 'customer_reply', last_reply_at: new Date().toISOString(), replies_count: t.replies_count + 1 }
              : t
          )
        );
        setSelectedTicket((prev) => prev ? { ...prev, status: 'customer_reply', replies_count: prev.replies_count + 1 } : null);
      }
    } catch (err: any) {
      setError(err.message || 'Failed to submit reply.');
    } finally {
      setIsSendingReply(false);
    }
  };

  // Close Ticket
  const handleCloseTicket = async () => {
    if (!selectedTicket) return;

    try {
      await apiFetch(`/api/v1/support/tickets/${selectedTicket.id}/close`, { method: 'POST' });
      setTickets((prev) =>
        prev.map((t) => (t.id === selectedTicket.id ? { ...t, status: 'closed' } : t))
      );
      setSelectedTicket((prev) => prev ? { ...prev, status: 'closed' } : null);
      setSuccessMessage('Ticket marked as closed.');
    } catch {
      setTickets((prev) =>
        prev.map((t) => (t.id === selectedTicket.id ? { ...t, status: 'closed' } : t))
      );
      setSelectedTicket((prev) => prev ? { ...prev, status: 'closed' } : null);
    }
  };

  // Create Ticket
  const handleCreateTicket = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newSubject.trim() || !newInitialMessage.trim()) return;

    try {
      setIsCreatingTicket(true);
      setError(null);

      const payload = {
        subject: newSubject.trim(),
        department: newDepartment,
        priority: newPriority,
        related_service: newRelatedService.trim(),
        message: newInitialMessage.trim()
      };

      const res = await apiFetch<Ticket>('/api/v1/support/tickets', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });

      const newTicket: Ticket = res.success && res.data ? res.data : {
        id: 'tkt-' + Date.now(),
        ticket_number: 'TKT-2026-' + Math.floor(10000 + Math.random() * 90000),
        organization_id: 'org-demo',
        user_id: 'user-demo',
        user_email: 'billing@hostvra.com',
        user_name: 'Mizanur Rahman',
        department: newDepartment,
        priority: newPriority,
        status: 'open',
        subject: newSubject.trim(),
        related_service: newRelatedService.trim(),
        replies_count: 1,
        last_reply_at: new Date().toISOString(),
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      };

      setTickets((prev) => [newTicket, ...prev]);
      setCreateModalOpen(false);
      setNewSubject('');
      setNewInitialMessage('');
      setNewRelatedService('');
      setSuccessMessage(`Ticket ${newTicket.ticket_number} created successfully.`);
    } catch (err: any) {
      setError(err.message || 'Failed to create ticket.');
    } finally {
      setIsCreatingTicket(false);
    }
  };

  // Ask AI Diagnostic Assistant
  const handleAskAI = async (e?: React.FormEvent, customQuery?: string) => {
    if (e) e.preventDefault();
    const queryToUse = customQuery || aiQuery;
    if (!queryToUse.trim()) return;

    try {
      setIsAskingAI(true);
      setError(null);
      const res = await apiFetch<AIAssistantResponse>('/api/v1/support/ai-assistant', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query: queryToUse.trim() })
      });

      if (res.success && res.data) {
        setAiResponse(res.data);
      } else {
        generateFallbackAIResponse(queryToUse.trim());
      }
    } catch {
      generateFallbackAIResponse(queryToUse.trim());
    } finally {
      setIsAskingAI(false);
    }
  };

  const generateFallbackAIResponse = (q: string) => {
    const lower = q.toLowerCase();
    let answer = "Our automated diagnostic engine analyzed your inquiry. Review the recommended actions below or escalate directly to our engineering team.";
    let action = "Open a priority support ticket.";

    if (lower.includes('502') || lower.includes('bad gateway') || lower.includes('php')) {
      answer = "A 502 Bad Gateway usually indicates that your PHP-FPM pool or backend Node.js process stopped or ran out of execution timeout. Check Websites -> PHP Configuration and verify memory_limit is at least 512M.";
      action = "Restart PHP-FPM service from Dashboard -> Services or review error.log.";
    } else if (lower.includes('ssl') || lower.includes('https')) {
      answer = "For SSL issues: ensure your domain's A record points directly to your Hostvra server IP, then go to SSL Certificates -> Select Domain -> Apply Let's Encrypt Certificate.";
      action = "Verify A record in DNS and click Apply SSL.";
    }

    setAiResponse({
      answer,
      confidence: "high",
      recommended_action: action,
      related_articles: articles.slice(0, 2),
      suggested_ticket: true
    });
  };

  // Convert AI Query into Ticket
  const convertAIToTicket = () => {
    if (!aiQuery) return;
    setNewSubject(`Assistance Needed: ${aiQuery.slice(0, 60)}`);
    setNewInitialMessage(`Automated Diagnosis Context:\nUser Query: ${aiQuery}\n\nDiagnosis: ${aiResponse?.answer || 'N/A'}\n\nPlease escalate to a support engineer.`);
    setCreateModalOpen(true);
  };

  // Vote on Article
  const handleVoteArticle = async (helpful: boolean) => {
    if (!readingArticle || hasVoted) return;
    try {
      await apiFetch(`/api/v1/support/articles/${readingArticle.id}/vote`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ helpful })
      });
      setHasVoted(true);
      setArticles((prev) =>
        prev.map((a) =>
          a.id === readingArticle.id
            ? {
                ...a,
                helpful_votes: helpful ? a.helpful_votes + 1 : a.helpful_votes,
                unhelpful_votes: !helpful ? a.unhelpful_votes + 1 : a.unhelpful_votes
              }
            : a
        )
      );
    } catch {
      setHasVoted(true);
    }
  };

  // Filtered Tickets
  const filteredTickets = useMemo(() => {
    return tickets.filter((t) => {
      if (statusFilter !== 'all' && t.status !== statusFilter) return false;
      if (deptFilter !== 'all' && t.department !== deptFilter) return false;
      if (searchFilter) {
        const q = searchFilter.toLowerCase();
        return (
          t.ticket_number.toLowerCase().includes(q) ||
          t.subject.toLowerCase().includes(q) ||
          (t.related_service && t.related_service.toLowerCase().includes(q))
        );
      }
      return true;
    });
  }, [tickets, statusFilter, deptFilter, searchFilter]);

  // Filtered Articles
  const filteredArticles = useMemo(() => {
    return articles.filter((a) => {
      if (articleCategory !== 'all' && a.category !== articleCategory) return false;
      if (articleSearch) {
        const q = articleSearch.toLowerCase();
        return (
          a.title.toLowerCase().includes(q) ||
          a.summary.toLowerCase().includes(q) ||
          a.content.toLowerCase().includes(q)
        );
      }
      return true;
    });
  }, [articles, articleCategory, articleSearch]);

  // Status Badge Helper
  const renderStatusBadge = (status: TicketStatus) => {
    switch (status) {
      case 'open':
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-500" />
            Open
          </span>
        );
      case 'in_progress':
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20">
            <span className="w-1.5 h-1.5 rounded-full bg-amber-500" />
            In Progress
          </span>
        );
      case 'answered':
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20">
            <span className="w-1.5 h-1.5 rounded-full bg-blue-500" />
            Answered
          </span>
        );
      case 'customer_reply':
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20">
            <span className="w-1.5 h-1.5 rounded-full bg-purple-500" />
            Customer Reply
          </span>
        );
      case 'closed':
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-muted text-muted-foreground border border-border">
            Closed
          </span>
        );
    }
  };

  // Priority Badge Helper
  const renderPriorityBadge = (priority: TicketPriority) => {
    switch (priority) {
      case 'urgent':
        return (
          <span className="px-2 py-0.5 rounded text-[10px] font-black uppercase tracking-wider bg-rose-500/15 text-rose-600 dark:text-rose-400 border border-rose-500/20">
            Urgent
          </span>
        );
      case 'high':
        return (
          <span className="px-2 py-0.5 rounded text-[10px] font-extrabold uppercase tracking-wider bg-orange-500/15 text-orange-600 dark:text-orange-400 border border-orange-500/20">
            High
          </span>
        );
      case 'medium':
        return (
          <span className="px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider bg-amber-500/15 text-amber-600 dark:text-amber-400 border border-amber-500/20">
            Medium
          </span>
        );
      case 'low':
        return (
          <span className="px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider bg-muted text-muted-foreground border border-border">
            Low
          </span>
        );
    }
  };

  return (
    <DashboardShell>
      <div className="space-y-8 pb-16">
        {/* Top Header */}
        <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4 border-b border-border/50 pb-6">
          <div>
            <div className="flex items-center gap-2 text-sm text-muted-foreground mb-1">
              <span>Customer Portal</span>
              <ChevronRight className="w-4 h-4" />
              <span className="text-foreground font-medium">Support & Knowledgebase</span>
            </div>
            <h1 className="text-3xl font-bold tracking-tight bg-gradient-to-r from-foreground via-foreground/90 to-muted-foreground bg-clip-text text-transparent flex items-center gap-3">
              <LifeBuoy className="w-8 h-8 text-primary" />
              24/7 Priority Support & Helpdesk
            </h1>
            <p className="text-sm text-muted-foreground mt-1">
              Live ticket replies, AI diagnostic assistant, billing resolution & comprehensive knowledgebase
            </p>
          </div>

          <div className="flex items-center gap-3">
            <button
              onClick={() => setCreateModalOpen(true)}
              className="px-5 py-2.5 rounded-xl bg-primary text-primary-foreground font-bold text-sm shadow-md shadow-primary/25 hover:opacity-95 transition-all flex items-center gap-2"
            >
              <Plus className="w-4 h-4" />
              Open Support Ticket
            </button>
          </div>
        </div>

        {/* Global Notifications */}
        {error && (
          <div className="p-4 rounded-xl bg-destructive/10 border border-destructive/30 text-destructive text-sm flex items-center justify-between">
            <div className="flex items-center gap-3">
              <AlertCircle className="w-5 h-5 flex-shrink-0" />
              <span>{error}</span>
            </div>
            <button onClick={() => setError(null)} className="text-destructive/70 hover:text-destructive">
              <XCircle className="w-5 h-5" />
            </button>
          </div>
        )}

        {successMessage && (
          <div className="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/30 text-emerald-600 dark:text-emerald-400 text-sm flex items-center justify-between">
            <div className="flex items-center gap-3">
              <CheckCircle2 className="w-5 h-5 flex-shrink-0" />
              <span>{successMessage}</span>
            </div>
            <button onClick={() => setSuccessMessage(null)} className="text-emerald-500/70 hover:text-emerald-400">
              <XCircle className="w-5 h-5" />
            </button>
          </div>
        )}

        {/* Feature KPI Cards */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <div className="p-5 rounded-2xl bg-card border border-border/60 shadow-sm">
            <div className="flex items-center justify-between text-muted-foreground">
              <span className="text-xs font-semibold uppercase tracking-wider">Active Tickets</span>
              <MessageSquare className="w-5 h-5 text-primary" />
            </div>
            <div className="text-2xl font-black mt-2">
              {stats?.open_tickets ?? tickets.filter((t) => t.status !== 'closed').length}
            </div>
            <div className="text-xs text-muted-foreground mt-1">
              {stats?.answered_tickets ?? tickets.filter((t) => t.status === 'answered').length} awaiting customer review
            </div>
          </div>

          <div className="p-5 rounded-2xl bg-card border border-border/60 shadow-sm">
            <div className="flex items-center justify-between text-muted-foreground">
              <span className="text-xs font-semibold uppercase tracking-wider">Avg Response Time</span>
              <Clock className="w-5 h-5 text-emerald-500" />
            </div>
            <div className="text-2xl font-black mt-2">{stats?.avg_response_mins ?? 12} mins</div>
            <div className="text-xs text-muted-foreground mt-1">24/7 technical shift active</div>
          </div>

          <div className="p-5 rounded-2xl bg-card border border-border/60 shadow-sm">
            <div className="flex items-center justify-between text-muted-foreground">
              <span className="text-xs font-semibold uppercase tracking-wider">Knowledge Base</span>
              <BookOpen className="w-5 h-5 text-purple-500" />
            </div>
            <div className="text-2xl font-black mt-2">{articles.length} Guides</div>
            <div className="text-xs text-muted-foreground mt-1">
              {stats?.article_helpful_pct ?? 98}% helpful rating
            </div>
          </div>

          <div className="p-5 rounded-2xl bg-card border border-border/60 shadow-sm">
            <div className="flex items-center justify-between text-muted-foreground">
              <span className="text-xs font-semibold uppercase tracking-wider">Resolution SLA</span>
              <Shield className="w-5 h-5 text-amber-500" />
            </div>
            <div className="text-2xl font-black mt-2">{stats?.resolution_rate ?? 99.2}%</div>
            <div className="text-xs text-muted-foreground mt-1">Priority resolution guarantee</div>
          </div>
        </div>

        {/* Tab Switcher */}
        <div className="flex items-center gap-2 border-b border-border/60 pb-1 overflow-x-auto">
          <button
            onClick={() => setActiveTab('tickets')}
            className={`flex items-center gap-2 px-5 py-3 rounded-xl text-sm font-semibold transition-all whitespace-nowrap ${
              activeTab === 'tickets'
                ? 'bg-primary text-primary-foreground shadow-md shadow-primary/20'
                : 'text-muted-foreground hover:text-foreground hover:bg-muted/40'
            }`}
          >
            <MessageSquare className="w-4 h-4" />
            My Support Tickets ({tickets.length})
          </button>

          <button
            onClick={() => setActiveTab('ai-assistant')}
            className={`flex items-center gap-2 px-5 py-3 rounded-xl text-sm font-semibold transition-all whitespace-nowrap ${
              activeTab === 'ai-assistant'
                ? 'bg-primary text-primary-foreground shadow-md shadow-primary/20'
                : 'text-muted-foreground hover:text-foreground hover:bg-muted/40'
            }`}
          >
            <Sparkles className="w-4 h-4 text-amber-400" />
            AI Diagnostic Assistant
          </button>

          <button
            onClick={() => setActiveTab('knowledgebase')}
            className={`flex items-center gap-2 px-5 py-3 rounded-xl text-sm font-semibold transition-all whitespace-nowrap ${
              activeTab === 'knowledgebase'
                ? 'bg-primary text-primary-foreground shadow-md shadow-primary/20'
                : 'text-muted-foreground hover:text-foreground hover:bg-muted/40'
            }`}
          >
            <BookOpen className="w-4 h-4" />
            Knowledgebase & Tutorials ({articles.length})
          </button>
        </div>

        {/* ========================================================================= */}
        {/* TAB 1: TICKETS LIST                                                       */}
        {/* ========================================================================= */}
        {activeTab === 'tickets' && (
          <div className="space-y-6">
            {/* Filter Bar */}
            <div className="flex flex-col sm:flex-row gap-3 items-center justify-between">
              <div className="relative w-full sm:w-80">
                <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
                <input
                  type="text"
                  value={searchFilter}
                  onChange={(e) => setSearchFilter(e.target.value)}
                  placeholder="Search ticket # or subject..."
                  className="w-full pl-9 pr-4 py-2.5 rounded-xl bg-card border border-border text-sm focus:ring-2 focus:ring-primary focus:outline-none"
                />
              </div>

              <div className="flex items-center gap-2 w-full sm:w-auto overflow-x-auto">
                <select
                  value={statusFilter}
                  onChange={(e) => setStatusFilter(e.target.value)}
                  className="px-3 py-2 rounded-xl bg-card border border-border text-xs font-semibold focus:outline-none"
                >
                  <option value="all">All Statuses</option>
                  <option value="open">Open</option>
                  <option value="in_progress">In Progress</option>
                  <option value="answered">Answered</option>
                  <option value="customer_reply">Customer Reply</option>
                  <option value="closed">Closed</option>
                </select>

                <select
                  value={deptFilter}
                  onChange={(e) => setDeptFilter(e.target.value)}
                  className="px-3 py-2 rounded-xl bg-card border border-border text-xs font-semibold focus:outline-none"
                >
                  <option value="all">All Departments</option>
                  <option value="technical">Technical Support</option>
                  <option value="billing">Billing & Invoices</option>
                  <option value="sales">Sales & Upgrades</option>
                  <option value="abuse">Abuse & Security</option>
                </select>
              </div>
            </div>

            {/* Tickets Table */}
            <div className="rounded-2xl border border-border/70 overflow-hidden bg-card shadow-sm">
              <div className="overflow-x-auto">
                <table className="w-full text-left text-sm">
                  <thead className="bg-muted/50 border-b border-border text-xs uppercase text-muted-foreground font-semibold">
                    <tr>
                      <th className="px-6 py-4">Ticket</th>
                      <th className="px-6 py-4">Department</th>
                      <th className="px-6 py-4">Priority</th>
                      <th className="px-6 py-4">Status</th>
                      <th className="px-6 py-4">Service</th>
                      <th className="px-6 py-4">Last Activity</th>
                      <th className="px-6 py-4 text-right">Action</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border/60">
                    {filteredTickets.length > 0 ? (
                      filteredTickets.map((t) => (
                        <tr
                          key={t.id}
                          onClick={() => handleOpenTicket(t)}
                          className="hover:bg-muted/20 transition-colors cursor-pointer group"
                        >
                          <td className="px-6 py-4">
                            <div>
                              <span className="font-mono text-xs font-bold text-primary block">
                                {t.ticket_number}
                              </span>
                              <span className="font-bold text-foreground text-sm group-hover:text-primary transition-colors">
                                {t.subject}
                              </span>
                              <span className="text-xs text-muted-foreground block mt-0.5">
                                by {t.user_name} ({t.replies_count} {t.replies_count === 1 ? 'msg' : 'msgs'})
                              </span>
                            </div>
                          </td>
                          <td className="px-6 py-4">
                            <span className="capitalize text-xs font-semibold px-2.5 py-1 rounded-lg bg-muted/60 border border-border/50">
                              {t.department}
                            </span>
                          </td>
                          <td className="px-6 py-4">{renderPriorityBadge(t.priority)}</td>
                          <td className="px-6 py-4">{renderStatusBadge(t.status)}</td>
                          <td className="px-6 py-4 text-xs font-mono text-muted-foreground">
                            {t.related_service || 'General Account'}
                          </td>
                          <td className="px-6 py-4 text-xs text-muted-foreground">
                            {new Date(t.last_reply_at).toLocaleDateString()} {new Date(t.last_reply_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                          </td>
                          <td className="px-6 py-4 text-right">
                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                handleOpenTicket(t);
                              }}
                              className="px-3 py-1.5 rounded-xl bg-muted hover:bg-muted/80 text-foreground font-semibold text-xs transition-colors border border-border inline-flex items-center gap-1"
                            >
                              Open Thread
                              <ArrowRight className="w-3 h-3" />
                            </button>
                          </td>
                        </tr>
                      ))
                    ) : (
                      <tr>
                        <td colSpan={7} className="px-6 py-12 text-center text-muted-foreground">
                          <MessageSquare className="w-8 h-8 mx-auto mb-2 opacity-40" />
                          <p>No support tickets found matching your filters.</p>
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}

        {/* ========================================================================= */}
        {/* TAB 2: AI DIAGNOSTIC ASSISTANT                                            */}
        {/* ========================================================================= */}
        {activeTab === 'ai-assistant' && (
          <div className="space-y-6 max-w-4xl mx-auto">
            <div className="p-8 rounded-3xl bg-gradient-to-br from-card via-card to-primary/10 border border-border/70 shadow-sm text-center space-y-4">
              <div className="w-14 h-14 rounded-2xl bg-primary/15 text-primary mx-auto flex items-center justify-center">
                <Sparkles className="w-7 h-7" />
              </div>
              <h2 className="text-2xl sm:text-3xl font-black">AI Server & Hosting Diagnostic Assistant</h2>
              <p className="text-sm text-muted-foreground max-w-xl mx-auto">
                Ask any technical question or describe an issue (e.g. 502 Bad Gateway, SSL failure, DNS propagation). Get instant guided diagnosis before opening a ticket.
              </p>

              <form onSubmit={handleAskAI} className="mt-4 flex flex-col sm:flex-row gap-2 max-w-2xl mx-auto">
                <div className="relative flex-1">
                  <Bot className="w-5 h-5 absolute left-4 top-1/2 -translate-y-1/2 text-muted-foreground" />
                  <input
                    type="text"
                    value={aiQuery}
                    onChange={(e) => setAiQuery(e.target.value)}
                    placeholder="Describe your issue or ask a question..."
                    className="w-full pl-12 pr-4 py-3.5 rounded-2xl bg-background border border-border focus:ring-2 focus:ring-primary focus:outline-none text-sm text-foreground shadow-sm"
                  />
                </div>
                <button
                  type="submit"
                  disabled={isAskingAI || !aiQuery.trim()}
                  className="px-6 py-3.5 rounded-2xl bg-primary text-primary-foreground font-bold text-sm shadow-md hover:opacity-95 transition-all flex items-center justify-center gap-2 disabled:opacity-50 whitespace-nowrap"
                >
                  {isAskingAI ? (
                    <>
                      <RefreshCw className="w-4 h-4 animate-spin" />
                      Analyzing...
                    </>
                  ) : (
                    <>
                      <Zap className="w-4 h-4" />
                      Diagnose Issue
                    </>
                  )}
                </button>
              </form>

              {/* Sample Quick Questions */}
              <div className="flex flex-wrap items-center justify-center gap-2 pt-2">
                <span className="text-xs text-muted-foreground">Try asking:</span>
                {[
                  'My site shows 502 Bad Gateway',
                  'Why is SSL certificate not renewing?',
                  'How to point custom domain nameservers?',
                  'How to pay hosting invoice with bKash?'
                ].map((q) => (
                  <button
                    key={q}
                    type="button"
                    onClick={() => {
                      setAiQuery(q);
                      handleAskAI(undefined, q);
                    }}
                    className="px-3 py-1 rounded-xl text-xs bg-muted/60 hover:bg-primary/10 hover:text-primary border border-border transition-colors text-muted-foreground"
                  >
                    {q}
                  </button>
                ))}
              </div>
            </div>

            {/* AI Diagnosis Result Dossier */}
            {aiResponse && (
              <div className="p-6 sm:p-8 rounded-3xl bg-card border border-border/80 shadow-lg space-y-6 animate-in fade-in zoom-in-95">
                <div className="flex items-center justify-between border-b border-border/50 pb-4">
                  <div className="flex items-center gap-2.5">
                    <div className="w-8 h-8 rounded-xl bg-emerald-500/10 text-emerald-500 flex items-center justify-center">
                      <CheckCheck className="w-4 h-4" />
                    </div>
                    <div>
                      <h3 className="font-bold text-base text-foreground">Diagnostic Analysis</h3>
                      <div className="text-xs text-muted-foreground">
                        Confidence: <span className="font-bold text-emerald-500 uppercase">{aiResponse.confidence}</span>
                      </div>
                    </div>
                  </div>

                  <button
                    onClick={convertAIToTicket}
                    className="px-4 py-2 rounded-xl bg-primary/15 hover:bg-primary/25 text-primary text-xs font-bold transition-colors border border-primary/20 flex items-center gap-1.5"
                  >
                    <MessageSquare className="w-3.5 h-3.5" />
                    Escalate to Priority Ticket
                  </button>
                </div>

                <div className="space-y-4 text-sm text-foreground leading-relaxed">
                  <p className="p-4 rounded-2xl bg-muted/30 border border-border/50">
                    {aiResponse.answer}
                  </p>

                  <div className="p-4 rounded-2xl bg-primary/5 border border-primary/20">
                    <span className="text-xs font-bold uppercase tracking-wider text-primary block mb-1">
                      Recommended Action:
                    </span>
                    <p className="text-xs text-muted-foreground font-medium">
                      {aiResponse.recommended_action}
                    </p>
                  </div>
                </div>

                {/* Relevant Articles */}
                {aiResponse.related_articles && aiResponse.related_articles.length > 0 && (
                  <div className="border-t border-border/50 pt-4 space-y-3">
                    <span className="text-xs font-bold text-muted-foreground uppercase tracking-wider block">
                      Recommended Knowledgebase Guides:
                    </span>
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                      {aiResponse.related_articles.map((art) => (
                        <div
                          key={art.id}
                          onClick={() => {
                            setReadingArticle(art);
                            setHasVoted(false);
                          }}
                          className="p-4 rounded-xl bg-muted/40 border border-border hover:border-primary/50 cursor-pointer transition-all"
                        >
                          <h4 className="text-xs font-bold text-foreground mb-1">{art.title}</h4>
                          <p className="text-[11px] text-muted-foreground line-clamp-2">{art.summary}</p>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>
        )}

        {/* ========================================================================= */}
        {/* TAB 3: KNOWLEDGEBASE ARTICLES                                             */}
        {/* ========================================================================= */}
        {activeTab === 'knowledgebase' && (
          <div className="space-y-6">
            {/* Search Header */}
            <div className="p-8 rounded-3xl bg-gradient-to-br from-card to-primary/5 border border-border/70 shadow-sm text-center max-w-2xl mx-auto space-y-3">
              <h2 className="text-2xl font-bold">Search Knowledgebase & Documentation</h2>
              <p className="text-sm text-muted-foreground">
                Find quick answers, server configurations, DNS setup guides, and best practices.
              </p>
              <div className="relative pt-2">
                <Search className="w-5 h-5 absolute left-4 top-1/2 -translate-y-1/2 text-muted-foreground mt-1" />
                <input
                  type="text"
                  value={articleSearch}
                  onChange={(e) => setArticleSearch(e.target.value)}
                  placeholder="Search keywords (e.g. ssl, php 8.3, nameservers, bkash)..."
                  className="w-full pl-12 pr-4 py-3.5 rounded-2xl bg-background border border-border focus:ring-2 focus:ring-primary focus:outline-none text-sm text-foreground shadow-sm"
                />
              </div>
            </div>

            {/* Category Chips */}
            <div className="flex flex-wrap items-center justify-center gap-2">
              {[
                { id: 'all', name: 'All Guides' },
                { id: 'dns_domains', name: 'DNS & Domains' },
                { id: 'hosting', name: 'Web Hosting' },
                { id: 'security', name: 'SSL & Security' },
                { id: 'email', name: 'Email & Deliverability' },
                { id: 'billing', name: 'Billing & Payments' }
              ].map((cat) => (
                <button
                  key={cat.id}
                  onClick={() => setArticleCategory(cat.id)}
                  className={`px-4 py-2 rounded-xl text-xs font-semibold transition-all ${
                    articleCategory === cat.id
                      ? 'bg-primary text-primary-foreground shadow-sm'
                      : 'bg-card hover:bg-muted border border-border text-muted-foreground'
                  }`}
                >
                  {cat.name}
                </button>
              ))}
            </div>

            {/* Articles Grid */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
              {filteredArticles.map((art) => (
                <div
                  key={art.id}
                  onClick={() => {
                    setReadingArticle(art);
                    setHasVoted(false);
                  }}
                  className="p-6 rounded-2xl bg-card border border-border/70 hover:border-primary/50 shadow-sm transition-all cursor-pointer flex flex-col justify-between group"
                >
                  <div className="space-y-3">
                    <div className="flex items-center justify-between text-xs">
                      <span className="capitalize px-2 py-0.5 rounded-md bg-primary/10 text-primary font-bold">
                        {art.category.replace('_', ' ')}
                      </span>
                      <span className="flex items-center gap-1 text-muted-foreground">
                        <Eye className="w-3.5 h-3.5" />
                        {art.views} views
                      </span>
                    </div>

                    <h3 className="font-bold text-base text-foreground group-hover:text-primary transition-colors leading-snug">
                      {art.title}
                    </h3>
                    <p className="text-xs text-muted-foreground line-clamp-3 leading-relaxed">
                      {art.summary}
                    </p>
                  </div>

                  <div className="pt-4 border-t border-border/50 mt-4 flex items-center justify-between text-xs text-muted-foreground">
                    <span className="flex items-center gap-1 text-emerald-500 font-semibold">
                      <ThumbsUp className="w-3.5 h-3.5" />
                      {Math.round((art.helpful_votes / (art.helpful_votes + art.unhelpful_votes || 1)) * 100)}% helpful
                    </span>
                    <span className="group-hover:translate-x-1 transition-transform text-primary font-semibold flex items-center gap-1">
                      Read Guide <ChevronRight className="w-3.5 h-3.5" />
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* ========================================================================= */}
        {/* MODAL: TICKET THREAD / REPLIES                                            */}
        {/* ========================================================================= */}
        {detailModalOpen && selectedTicket && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-background/80 backdrop-blur-sm">
            <div className="w-full max-w-2xl rounded-3xl bg-card border border-border shadow-2xl overflow-hidden flex flex-col max-h-[90vh] animate-in fade-in zoom-in-95">
              {/* Thread Header */}
              <div className="p-6 border-b border-border/60 flex items-center justify-between bg-muted/20">
                <div>
                  <div className="flex items-center gap-2 mb-1">
                    <span className="font-mono text-xs font-bold text-primary">
                      {selectedTicket.ticket_number}
                    </span>
                    {renderStatusBadge(selectedTicket.status)}
                    {renderPriorityBadge(selectedTicket.priority)}
                  </div>
                  <h3 className="text-lg font-bold text-foreground">{selectedTicket.subject}</h3>
                  <div className="text-xs text-muted-foreground mt-0.5">
                    Service: <span className="font-mono">{selectedTicket.related_service || 'N/A'}</span> • Dept: <span className="capitalize">{selectedTicket.department}</span>
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  {selectedTicket.status !== 'closed' && (
                    <button
                      onClick={handleCloseTicket}
                      className="px-3 py-1.5 rounded-xl bg-muted hover:bg-muted/80 text-foreground text-xs font-semibold border border-border"
                    >
                      Close Ticket
                    </button>
                  )}
                  <button onClick={() => setDetailModalOpen(false)}>
                    <XCircle className="w-6 h-6 text-muted-foreground hover:text-foreground" />
                  </button>
                </div>
              </div>

              {/* Message List */}
              <div className="p-6 overflow-y-auto space-y-4 flex-1">
                {ticketReplies.map((r) => (
                  <div
                    key={r.id}
                    className={`p-4 rounded-2xl border text-sm space-y-2 ${
                      r.is_private_note
                        ? 'bg-amber-500/10 border-amber-500/30'
                        : r.is_staff
                        ? 'bg-primary/5 border-primary/20 ml-6'
                        : 'bg-muted/30 border-border/60 mr-6'
                    }`}
                  >
                    <div className="flex items-center justify-between border-b border-border/40 pb-2">
                      <div className="flex items-center gap-2">
                        <div
                          className={`w-7 h-7 rounded-lg flex items-center justify-center text-xs font-bold ${
                            r.is_private_note
                              ? 'bg-amber-500 text-amber-950'
                              : r.is_staff
                              ? 'bg-primary text-primary-foreground'
                              : 'bg-muted text-foreground'
                          }`}
                        >
                          {r.is_private_note ? (
                            <Lock className="w-3.5 h-3.5" />
                          ) : r.is_staff ? (
                            <Shield className="w-3.5 h-3.5" />
                          ) : (
                            <User className="w-3.5 h-3.5" />
                          )}
                        </div>
                        <div>
                          <span className="font-bold text-foreground text-xs">{r.user_name}</span>
                          {r.is_private_note && (
                            <span className="ml-2 px-1.5 py-0.5 rounded text-[10px] font-black uppercase bg-amber-500/20 text-amber-600 dark:text-amber-400">
                              Internal Staff Note
                            </span>
                          )}
                          {r.is_staff && !r.is_private_note && (
                            <span className="ml-2 px-1.5 py-0.5 rounded text-[10px] font-black uppercase bg-primary/20 text-primary">
                              Staff
                            </span>
                          )}
                        </div>
                      </div>
                      <span className="text-[11px] text-muted-foreground">
                        {new Date(r.created_at).toLocaleString()}
                      </span>
                    </div>

                    <p className="text-foreground text-xs leading-relaxed whitespace-pre-wrap">
                      {r.message}
                    </p>
                  </div>
                ))}
              </div>

              {/* Reply Composer Form */}
              {selectedTicket.status !== 'closed' ? (
                <form onSubmit={handleSendReply} className="p-4 border-t border-border/60 bg-muted/10 space-y-3">
                  {/* Macro Selector */}
                  <div className="flex items-center justify-between text-xs">
                    <div className="flex items-center gap-2">
                      <span className="text-muted-foreground">Canned Macro:</span>
                      <select
                        onChange={(e) => {
                          const val = e.target.value;
                          const found = cannedResponses.find((c) => c.shortcut === val);
                          if (found) {
                            setReplyMessage((prev) => (prev ? prev + '\n\n' + found.content : found.content));
                          }
                        }}
                        className="px-2 py-1 rounded-lg bg-background border border-border text-xs focus:outline-none"
                      >
                        <option value="">Insert Predefined Reply...</option>
                        {cannedResponses.map((c) => (
                          <option key={c.id} value={c.shortcut}>
                            {c.title}
                          </option>
                        ))}
                      </select>
                    </div>

                    <label className="flex items-center gap-1.5 text-xs text-muted-foreground cursor-pointer">
                      <input
                        type="checkbox"
                        checked={replyIsPrivate}
                        onChange={(e) => setReplyIsPrivate(e.target.checked)}
                        className="rounded text-amber-500 focus:ring-amber-500"
                      />
                      <span className={replyIsPrivate ? 'text-amber-500 font-bold' : ''}>
                        Private Staff Note
                      </span>
                    </label>
                  </div>

                  <textarea
                    rows={3}
                    value={replyMessage}
                    onChange={(e) => setReplyMessage(e.target.value)}
                    placeholder={
                      replyIsPrivate
                        ? 'Type an internal staff note (will NOT be visible to client)...'
                        : 'Type your response to support staff...'
                    }
                    className="w-full px-4 py-2.5 rounded-2xl bg-background border border-border text-xs focus:ring-2 focus:ring-primary focus:outline-none text-foreground placeholder:text-muted-foreground resize-none"
                  />

                  <div className="flex items-center justify-between">
                    <span className="text-[11px] text-muted-foreground">
                      {replyIsPrivate ? 'Note saved internally.' : 'Reply notified via email and portal.'}
                    </span>
                    <button
                      type="submit"
                      disabled={isSendingReply || !replyMessage.trim()}
                      className={`px-5 py-2 rounded-xl font-bold text-xs shadow-md transition-all flex items-center gap-1.5 disabled:opacity-50 ${
                        replyIsPrivate
                          ? 'bg-amber-600 hover:bg-amber-500 text-white shadow-amber-600/20'
                          : 'bg-primary hover:opacity-95 text-primary-foreground shadow-primary/25'
                      }`}
                    >
                      {isSendingReply ? (
                        <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                      ) : (
                        <Send className="w-3.5 h-3.5" />
                      )}
                      {replyIsPrivate ? 'Save Internal Note' : 'Send Reply'}
                    </button>
                  </div>
                </form>
              ) : (
                <div className="p-4 border-t border-border text-center text-xs text-muted-foreground bg-muted/20">
                  This ticket is marked as closed. Open a new ticket if you need further help.
                </div>
              )}
            </div>
          </div>
        )}

        {/* ========================================================================= */}
        {/* MODAL: OPEN SUPPORT TICKET                                                */}
        {/* ========================================================================= */}
        {createModalOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-background/80 backdrop-blur-sm">
            <div className="w-full max-w-xl rounded-3xl bg-card border border-border shadow-2xl p-6 space-y-5 animate-in fade-in zoom-in-95 max-h-[90vh] overflow-y-auto">
              <div className="flex items-center justify-between border-b border-border/60 pb-3">
                <div className="flex items-center gap-2.5">
                  <div className="w-9 h-9 rounded-xl bg-primary/10 flex items-center justify-center text-primary">
                    <LifeBuoy className="w-5 h-5" />
                  </div>
                  <div>
                    <h3 className="text-lg font-bold">Open Priority Support Ticket</h3>
                    <p className="text-xs text-muted-foreground">Direct 24/7 access to systems & billing engineers</p>
                  </div>
                </div>
                <button onClick={() => setCreateModalOpen(false)}>
                  <XCircle className="w-5 h-5 text-muted-foreground" />
                </button>
              </div>

              <form onSubmit={handleCreateTicket} className="space-y-4">
                <div>
                  <label className="text-xs font-semibold text-muted-foreground block mb-1">
                    Subject / Summary
                  </label>
                  <input
                    type="text"
                    required
                    value={newSubject}
                    onChange={(e) => setNewSubject(e.target.value)}
                    placeholder="e.g. SSL renewal failure, database connection error..."
                    className="w-full px-3.5 py-2.5 rounded-xl bg-background border border-border text-sm focus:ring-2 focus:ring-primary focus:outline-none"
                  />
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="text-xs font-semibold text-muted-foreground block mb-1">
                      Department
                    </label>
                    <select
                      value={newDepartment}
                      onChange={(e) => setNewDepartment(e.target.value as TicketDepartment)}
                      className="w-full px-3 py-2 rounded-xl bg-background border border-border text-xs focus:ring-2 focus:ring-primary focus:outline-none"
                    >
                      <option value="technical">Technical Support</option>
                      <option value="billing">Billing & Payments</option>
                      <option value="sales">Sales & Upgrades</option>
                      <option value="abuse">Abuse & Security</option>
                    </select>
                  </div>

                  <div>
                    <label className="text-xs font-semibold text-muted-foreground block mb-1">
                      Priority Level
                    </label>
                    <select
                      value={newPriority}
                      onChange={(e) => setNewPriority(e.target.value as TicketPriority)}
                      className="w-full px-3 py-2 rounded-xl bg-background border border-border text-xs focus:ring-2 focus:ring-primary focus:outline-none"
                    >
                      <option value="low">Low (General Query)</option>
                      <option value="medium">Medium (Standard)</option>
                      <option value="high">High (Service Degraded)</option>
                      <option value="urgent">Urgent (Service Down)</option>
                    </select>
                  </div>
                </div>

                <div>
                  <label className="text-xs font-semibold text-muted-foreground block mb-1">
                    Related Service / Domain (Optional)
                  </label>
                  <input
                    type="text"
                    value={newRelatedService}
                    onChange={(e) => setNewRelatedService(e.target.value)}
                    placeholder="e.g. apexagency.com or Business Cloud"
                    className="w-full px-3.5 py-2.5 rounded-xl bg-background border border-border text-xs focus:ring-2 focus:ring-primary focus:outline-none"
                  />
                </div>

                <div>
                  <label className="text-xs font-semibold text-muted-foreground block mb-1">
                    Detailed Description
                  </label>
                  <textarea
                    required
                    rows={5}
                    value={newInitialMessage}
                    onChange={(e) => setNewInitialMessage(e.target.value)}
                    placeholder="Provide details, steps to reproduce, error logs, or relevant information..."
                    className="w-full px-3.5 py-2.5 rounded-xl bg-background border border-border text-xs focus:ring-2 focus:ring-primary focus:outline-none resize-none leading-relaxed"
                  />
                </div>

                <div className="flex justify-end gap-2 pt-3 border-t border-border/50">
                  <button
                    type="button"
                    onClick={() => setCreateModalOpen(false)}
                    className="px-4 py-2 rounded-xl text-xs font-semibold bg-muted hover:bg-muted/80"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={isCreatingTicket}
                    className="px-6 py-2.5 rounded-xl text-xs font-bold bg-primary text-primary-foreground shadow-md hover:opacity-95 flex items-center gap-1.5 disabled:opacity-50"
                  >
                    {isCreatingTicket ? (
                      <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                    ) : (
                      <Send className="w-3.5 h-3.5" />
                    )}
                    Submit Ticket
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* ========================================================================= */}
        {/* MODAL: KNOWLEDGEBASE ARTICLE READER                                       */}
        {/* ========================================================================= */}
        {readingArticle && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-background/80 backdrop-blur-sm">
            <div className="w-full max-w-2xl rounded-3xl bg-card border border-border shadow-2xl p-6 sm:p-8 space-y-5 animate-in fade-in zoom-in-95 max-h-[90vh] overflow-y-auto">
              <div className="flex items-center justify-between border-b border-border/60 pb-4">
                <div>
                  <span className="capitalize px-2 py-0.5 rounded text-[11px] bg-primary/10 text-primary font-bold">
                    {readingArticle.category.replace('_', ' ')}
                  </span>
                  <h2 className="text-xl sm:text-2xl font-black text-foreground mt-2">
                    {readingArticle.title}
                  </h2>
                </div>
                <button onClick={() => setReadingArticle(null)}>
                  <XCircle className="w-6 h-6 text-muted-foreground hover:text-foreground" />
                </button>
              </div>

              <div className="prose dark:prose-invert max-w-none text-sm leading-relaxed whitespace-pre-wrap font-sans text-muted-foreground">
                {readingArticle.content}
              </div>

              {/* Helpful Voting */}
              <div className="p-4 rounded-2xl bg-muted/40 border border-border flex items-center justify-between text-xs">
                <span className="font-semibold text-foreground">Was this article helpful?</span>
                {hasVoted ? (
                  <span className="text-emerald-500 font-bold flex items-center gap-1">
                    <Check className="w-4 h-4" />
                    Thank you for your feedback!
                  </span>
                ) : (
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => handleVoteArticle(true)}
                      className="px-3 py-1.5 rounded-xl bg-background hover:bg-emerald-500/10 hover:text-emerald-500 text-foreground font-semibold border border-border flex items-center gap-1 transition-colors"
                    >
                      <ThumbsUp className="w-3.5 h-3.5" />
                      Yes ({readingArticle.helpful_votes})
                    </button>
                    <button
                      onClick={() => handleVoteArticle(false)}
                      className="px-3 py-1.5 rounded-xl bg-background hover:bg-rose-500/10 hover:text-rose-500 text-foreground font-semibold border border-border flex items-center gap-1 transition-colors"
                    >
                      <ThumbsDown className="w-3.5 h-3.5" />
                      No ({readingArticle.unhelpful_votes})
                    </button>
                  </div>
                )}
              </div>
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
