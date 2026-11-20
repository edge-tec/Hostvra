import type { MetadataRoute } from 'next';

export default function robots(): MetadataRoute.Robots {
  return {
    rules: [
      {
        userAgent: '*',
        allow: ['/', '/order', '/status', '/terms', '/privacy', '/refund-policy', '/sla', '/acceptable-use-policy'],
        disallow: ['/admin/', '/api/', '/dashboard/', '/files/', '/terminal/', '/phpmyadmin/'],
      },
    ],
    sitemap: 'https://hostvra.com/sitemap.xml',
  };
}
