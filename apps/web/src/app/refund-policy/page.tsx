import React from 'react';
import Link from 'next/link';

export const metadata = {
  title: 'Refund Policy — Hostvra Cloud Hosting',
  description: 'Hostvra hosting refund eligibility, domain registration exclusions, and credit wallet refunds.',
};

export default function RefundPolicyPage() {
  return (
    <main className="min-h-screen bg-slate-50 dark:bg-slate-950 text-slate-800 dark:text-slate-200 py-12 px-4">
      <article className="max-w-3xl mx-auto bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-3xl p-8 space-y-6 shadow-sm">
        <div className="flex items-center justify-between border-b border-slate-100 dark:border-slate-800 pb-4">
          <h1 className="text-2xl font-extrabold text-slate-900 dark:text-white">Refund Policy</h1>
          <Link href="/" className="text-xs font-bold text-emerald-600 hover:underline">&larr; Home</Link>
        </div>
        <section className="space-y-3 text-sm leading-relaxed">
          <h2 className="text-base font-bold text-slate-900 dark:text-white">1. Shared &amp; Cloud Hosting Plans</h2>
          <p>New shared and cloud hosting orders may request a refund within 14 days of initial purchase via a support ticket if the service does not meet operational specifications.</p>
          <h2 className="text-base font-bold text-slate-900 dark:text-white">2. Non-Refundable Items</h2>
          <p>Domain name registrations, renewals, transfers, SSL certificates issued by external authorities, and accounts terminated for Acceptable Use violations are non-refundable.</p>
          <h2 className="text-base font-bold text-slate-900 dark:text-white">3. Cryptocurrency Payments</h2>
          <p>Approved refunds for cryptocurrency payments are credited to your Hostvra Account Credit Wallet in USD equivalent at the time of invoice settlement.</p>
        </section>
      </article>
    </main>
  );
}
