// Hostvra Core API Client

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
    details?: any;
    request_id?: string;
  };
  meta?: {
    page?: number;
    limit?: number;
    total?: number;
  };
}

export interface User {
  id: string;
  email: string;
  full_name: string;
  is_active: boolean;
  is_superadmin: boolean;
  default_org_id: string;
  role: string;
}

export interface Organization {
  id: string;
  name: string;
  slug: string;
  plan_tier: string;
  max_servers: number;
  max_websites: number;
}

export interface Server {
  id: string;
  organization_id: string;
  name: string;
  hostname: string;
  ip_address: string;
  os_name: string;
  os_version: string;
  architecture: string;
  kernel_version?: string;
  agent_version: string;
  status: 'online' | 'offline' | 'connecting' | 'maintenance' | 'error';
  cpu_cores: number;
  cpu_model?: string;
  ram_total_mb: number;
  disk_total_gb: number;
  last_heartbeat_at?: string;
  uptime_seconds?: number;
  created_at: string;
}

export interface EnrollmentTokenResponse {
  token_id: string;
  raw_token: string;
  label: string;
  expires_at: string;
  install_command: string;
}

export interface AuditLog {
  id: string;
  action: string;
  resource_type: string;
  resource_id?: string;
  ip_address?: string;
  status: 'success' | 'failure';
  error_message?: string;
  metadata?: Record<string, any>;
  created_at: string;
}

export function getStoredToken(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem('hostvra_access_token');
}

export function setStoredToken(token: string) {
  if (typeof window !== 'undefined') {
    localStorage.setItem('hostvra_access_token', token);
  }
}

export function clearStoredAuth() {
  if (typeof window !== 'undefined') {
    localStorage.removeItem('hostvra_access_token');
    localStorage.removeItem('hostvra_user');
  }
}

export async function apiFetch<T>(endpoint: string, options: RequestInit = {}): Promise<ApiResponse<T>> {
  const token = getStoredToken();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string> || {}),
  };

  if (token && !headers['Authorization']) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  try {
    const res = await fetch(`${API_BASE_URL}${endpoint}`, {
      ...options,
      headers,
    });

    const data: ApiResponse<T> = await res.json();
    if (!res.ok && !data.error) {
      return {
        success: false,
        error: {
          code: 'HTTP_ERROR',
          message: `Request failed with status ${res.status}`,
        },
      };
    }
    return data;
  } catch (err: any) {
    return {
      success: false,
      error: {
        code: 'NETWORK_ERROR',
        message: err.message || 'Unable to connect to Hostvra API server',
      },
    };
  }
}
