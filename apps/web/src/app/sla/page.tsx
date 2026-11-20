import React from 'react';
import Link from 'next/link';

export const metadata = {
  title: 'Service Level Agreement (SLA) — Hostvra',
  description: 'Hostvra infrastructure availability targets, maintenance windows, and service credits.',
};

export default function SLAPage() {
  return (
    <main className="min-h-screen bg-slate-50 dark:bg-slate-950 text-slate-800 dark:text-slate-200 py-12 px-4">
      <article className="max-w-3xl mx-auto bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-3xl p-8 space-y-6 shadow-sm">
        <div className="flex items-center justify-between border-b border-slate-100 dark:border-slate-800 pb-4">
          <h1 className="text-2xl font-extrabold text-slate-900 dark:text-white">Service Level Agreement (SLA)</h1>
          <Link href="/" className="text-xs font-bold text-emerald-600 hover:underline">&larr; Home</Link>
        </div>
        <section className="space-y-3 text-sm leading-relaxed">
          <h2 className="text-base font-bold text-slate-900 dark:text-white">1. Live Status Monitoring</h2>
          <p>Real-time infrastructure telemetry is published transparently on our <Link href="/status" className="text-emerald-600 underline">Public System Status Page</Link>.</p>
          <h2 className="text-base font-bold text-slate-900 dark:text-white">2. Scheduled Maintenance</h2>
          <p>Routine kernel, web server, and security updates are performed during low-traffic maintenance windows with advance notice whenever possible.</p>
          <h2 className="text-base font-bold text-slate-900 dark:text-white">3. Service Credit Requests</h2>
          <p>Eligible verified unscheduled downtime may qualify for account credit wallet adjustments upon opening a support ticket within 7 days of the incident.</p>
        </section>
      </article>
    </main>
  );
}
