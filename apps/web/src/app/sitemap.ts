import type { MetadataRoute } from 'next';

export default function sitemap(): MetadataRoute.Sitemap {
  const base = 'https://hostvra.com';
  const now = new Date();
  return [
    { url: `${base}/`, lastModified: now, changeFrequency: 'weekly', priority: 1.0 },
    { url: `${base}/order`, lastModified: now, changeFrequency: 'weekly', priority: 0.9 },
    { url: `${base}/status`, lastModified: now, changeFrequency: 'always', priority: 0.8 },
    { url: `${base}/login`, lastModified: now, changeFrequency: 'monthly', priority: 0.6 },
    { url: `${base}/register`, lastModified: now, changeFrequency: 'monthly', priority: 0.7 },
    { url: `${base}/terms`, lastModified: now, changeFrequency: 'monthly', priority: 0.4 },
    { url: `${base}/privacy`, lastModified: now, changeFrequency: 'monthly', priority: 0.4 },
    { url: `${base}/refund-policy`, lastModified: now, changeFrequency: 'monthly', priority: 0.4 },
    { url: `${base}/sla`, lastModified: now, changeFrequency: 'monthly', priority: 0.4 },
    { url: `${base}/acceptable-use-policy`, lastModified: now, changeFrequency: 'monthly', priority: 0.4 },
  ];
}
