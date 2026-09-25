---
name: frontend
description: |
  Frontend analysis guide for Hostvra Next.js 15 dashboard.
  Covers component patterns, API integration, TypeScript, and TailwindCSS.
---

# Hostvra Frontend Reference

## Stack
- Next.js 15.1.7 (App Router)
- React 19
- TypeScript 5.7.3
- TailwindCSS 3.4.17
- lucide-react 0.475.0 (icons)
- clsx + tailwind-merge (class utilities)

## Directory Structure

```
apps/web/
├── src/
│   ├── app/                 — Next.js App Router pages
│   │   ├── layout.tsx       — Root layout
│   │   ├── page.tsx         — Root page (redirects to dashboard)
│   │   ├── globals.css      — Global styles (42KB — comprehensive)
│   │   ├── error.tsx        — Error boundary
│   │   ├── global-error.tsx — Global error handler
│   │   ├── [page]/          — 34 page directories
│   │   └── api/             — Next.js API routes (proxy)
│   ├── components/          — Shared React components
│   └── lib/                 — Utility functions
├── middleware.ts             — Next.js middleware (auth redirects)
├── next.config.js           — Next.js config
├── tailwind.config.js       — Tailwind customization (1838 bytes)
└── tsconfig.json            — TypeScript config
```

## API Integration Pattern

The frontend communicates with the Go API at port 8080.
Next.js API routes in `src/app/api/` act as proxies.

Authentication: JWT stored in cookie/localStorage, sent as Authorization header.
All API calls go through a shared fetch utility.

## Key Component Patterns

### Page Components
Each page in apps/web/src/app/{section}/page.tsx:
- Fetches data from API
- Renders table/list/detail views
- Uses TailwindCSS for styling
- Error and loading states handled

### Authentication Flow
- Login page: POST /api/v1/auth/login → JWT stored
- Middleware redirects unauthenticated users to /login
- middleware.ts handles route protection

## Confirmed Active Development

These files are CURRENTLY BEING MODIFIED — do not touch:
- apps/web/src/app/email/page.tsx
- apps/web/src/app/websites/page.tsx
- apps/web/src/components/WebmailClient.tsx

## Build & Development

```bash
# Development
cd apps/web && npm run dev   # port 3000

# Build (type-check + bundle)
cd apps/web && npm run build

# Lint
cd apps/web && npm run lint
```

## No Frontend Tests Exist

The test suite for the frontend is ABSENT.
`npm test` in apps/web is not configured.
The CI only runs `npm run build` as a type-check.

If adding tests: Use Jest + React Testing Library.
Add `jest.config.js` and `@testing-library/react` dependency.
