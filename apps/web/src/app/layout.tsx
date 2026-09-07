import type { Metadata } from 'next';
import './globals.css';
import { ThemeProvider } from '@/components/ThemeProvider';
import { ChunkLoadRecovery } from '@/components/ChunkLoadRecovery';

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
    <html lang="en" className="dark" suppressHydrationWarning>
      <body className="min-h-screen antialiased transition-colors duration-150" suppressHydrationWarning>
        <ChunkLoadRecovery />
        <ThemeProvider>
          {children}
        </ThemeProvider>
      </body>
    </html>
  );
}
