'use client';

import { useEffect } from 'react';

export function ChunkLoadRecovery() {
  useEffect(() => {
    const handleChunkError = (event: any) => {
      const msg = (
        event?.message ||
        event?.reason?.message ||
        event?.reason ||
        String(event || '')
      ).toLowerCase();

      if (
        msg.includes('loading chunk') ||
        msg.includes('chunkloaderror') ||
        msg.includes('failed to fetch dynamically imported module') ||
        msg.includes('load chunk failed')
      ) {
        console.warn('Chunk load error detected, reloading to fetch latest assets:', msg);
        const lastReload = sessionStorage.getItem('last_chunk_reload');
        const now = Date.now();
        if (!lastReload || now - parseInt(lastReload, 10) > 8000) {
          sessionStorage.setItem('last_chunk_reload', now.toString());
          const cleanUrl = window.location.pathname.split('?')[0];
          window.location.replace(`${cleanUrl}?_r=${now}`);
        }
      }
    };

    window.addEventListener('error', handleChunkError);
    window.addEventListener('unhandledrejection', handleChunkError);

    return () => {
      window.removeEventListener('error', handleChunkError);
      window.removeEventListener('unhandledrejection', handleChunkError);
    };
  }, []);

  return null;
}
