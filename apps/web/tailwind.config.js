/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: 'class',
  content: [
    './src/pages/**/*.{js,ts,jsx,tsx,mdx}',
    './src/components/**/*.{js,ts,jsx,tsx,mdx}',
    './src/app/**/*.{js,ts,jsx,tsx,mdx}',
  ],
  theme: {
    extend: {
      colors: {
        brand: {
          50: '#eef2ff',
          100: '#e0e7ff',
          500: '#6366f1',
          600: '#4f46e5',
          700: '#4338ca',
          800: '#3730a3',
          900: '#312e81',
          950: '#1e1b4b',
        },
        surface: {
          50: '#f8fafc',
          100: '#f1f5f9',
          800: '#1e293b',
          850: '#172033',
          900: '#0f172a',
          950: '#020617',
        },
        // Hostvra Unified Design System Tokens
        design: {
          bg: '#F8FAFC',
          card: '#FFFFFF',
          sidebar: '#FFFFFF',
          primary: '#111827',
          secondary: '#4B5563',
          muted: '#6B7280',
          border: '#E5E7EB',
          borderHover: '#D1D5DB',
          success: '#16A34A',
          warning: '#EA580C',
          danger: '#DC2626',
          info: '#2563EB',
        },
      },
    },
  },
  plugins: [],
}
