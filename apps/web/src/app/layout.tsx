import type { Metadata } from 'next';
import './globals.css';

export const metadata: Metadata = {
  title: 'Hostvra — Modern Self-Hosted Cloud & Server Control Panel',
  description: 'Production-grade self-hosted VPS and dedicated server management platform.',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className="dark">
      <body className="bg-[#090d16] text-slate-100 min-h-screen antialiased">
        {children}
      </body>
    </html>
  );
}
