import { AppPackage } from './api';

export interface AppLaunchTarget {
  type: 'route' | 'external' | 'terminal';
  url?: string;
  terminalCmd?: string;
  label: string;
  badge?: string;
  hasWebUi: boolean;
}

export function getAppLaunchTarget(app: AppPackage, hostname?: string): AppLaunchTarget {
  const host = hostname || (typeof window !== 'undefined' ? window.location.hostname : 'localhost');

  switch (app.id) {
    case 'docker':
      return {
        type: 'route',
        url: '/docker',
        terminalCmd: 'docker ps -a',
        label: 'Docker Control Panel',
        badge: 'UI Dashboard',
        hasWebUi: true,
      };

    case 'php83':
    case 'php82':
    case 'php81':
      return {
        type: 'route',
        url: '/php',
        terminalCmd: `${app.binary_path || 'php'} -v`,
        label: 'PHP Versions & Extensions',
        badge: 'PHP Manager',
        hasWebUi: true,
      };

    case 'mariadb':
    case 'postgresql':
    case 'mongodb':
    case 'redis':
    case 'memcached':
    case 'sqlite3':
      return {
        type: 'route',
        url: '/databases',
        terminalCmd: app.id === 'redis' ? 'redis-cli ping' : `${app.id} --version`,
        label: 'Database Management',
        badge: 'DB Dashboard',
        hasWebUi: true,
      };

    case 'ufw':
    case 'fail2ban':
      return {
        type: 'route',
        url: '/firewall',
        terminalCmd: app.id === 'ufw' ? 'ufw status verbose' : 'fail2ban-client status',
        label: 'Firewall & Security Rules',
        badge: 'Firewall',
        hasWebUi: true,
      };

    case 'certbot':
      return {
        type: 'route',
        url: '/ssl',
        terminalCmd: 'certbot certificates',
        label: 'SSL Certificates Hub',
        badge: 'SSL Manager',
        hasWebUi: true,
      };

    case 'pureftpd':
      return {
        type: 'route',
        url: '/files',
        terminalCmd: 'systemctl status pure-ftpd',
        label: 'File & FTP Manager',
        badge: 'File Manager',
        hasWebUi: true,
      };

    case 'netdata':
      return {
        type: 'external',
        url: `http://${host}:19999`,
        terminalCmd: 'systemctl status netdata',
        label: 'Netdata Real-time Web GUI',
        badge: 'Port 19999',
        hasWebUi: true,
      };

    case 'rabbitmq':
      return {
        type: 'external',
        url: `http://${host}:15672`,
        terminalCmd: 'rabbitmqctl status',
        label: 'RabbitMQ Web Management',
        badge: 'Port 15672',
        hasWebUi: true,
      };

    case 'openlitespeed':
      return {
        type: 'external',
        url: `https://${host}:7080`,
        terminalCmd: 'systemctl status lsws',
        label: 'OpenLiteSpeed WebAdmin',
        badge: 'Port 7080',
        hasWebUi: true,
      };

    case 'caddy':
    case 'nginx':
    case 'apache':
      return {
        type: 'external',
        url: `http://${host}`,
        terminalCmd: `systemctl status ${app.service_name || app.id}`,
        label: 'View Web Server HTTP',
        badge: 'Port 80',
        hasWebUi: true,
      };

    case 'supervisor':
      return {
        type: 'terminal',
        terminalCmd: 'supervisorctl status',
        label: 'Supervisor Process Console',
        badge: 'supervisorctl',
        hasWebUi: false,
      };

    case 'pm2':
      return {
        type: 'terminal',
        terminalCmd: 'pm2 list',
        label: 'PM2 Process Status',
        badge: 'pm2 list',
        hasWebUi: false,
      };

    case 'git':
      return {
        type: 'terminal',
        terminalCmd: 'git --version && git status',
        label: 'Git Terminal',
        badge: 'git CLI',
        hasWebUi: false,
      };

    case 'golang':
      return {
        type: 'terminal',
        terminalCmd: 'go version && go env',
        label: 'Go Toolchain CLI',
        badge: 'go CLI',
        hasWebUi: false,
      };

    case 'rust':
      return {
        type: 'terminal',
        terminalCmd: 'cargo --version && rustc --version',
        label: 'Rust & Cargo CLI',
        badge: 'cargo CLI',
        hasWebUi: false,
      };

    case 'java':
      return {
        type: 'terminal',
        terminalCmd: 'java -version',
        label: 'Java Runtime Environment',
        badge: 'java CLI',
        hasWebUi: false,
      };

    case 'nodejs':
      return {
        type: 'terminal',
        terminalCmd: 'node -v && npm -v',
        label: 'Node & NPM Runtime',
        badge: 'node CLI',
        hasWebUi: false,
      };

    case 'python-tools':
      return {
        type: 'terminal',
        terminalCmd: 'python3 --version && pip --version',
        label: 'Python Environment',
        badge: 'python3 CLI',
        hasWebUi: false,
      };

    default:
      return {
        type: 'terminal',
        terminalCmd: app.service_name ? `systemctl status ${app.service_name}` : `${app.binary_path || app.id} --help`,
        label: `Launch ${app.name}`,
        hasWebUi: false,
      };
  }
}

const PINNED_APPS_STORAGE_KEY = 'hostvra_pinned_apps';

export function getPinnedAppIds(): string[] {
  if (typeof window === 'undefined') return [];
  try {
    const raw = localStorage.getItem(PINNED_APPS_STORAGE_KEY);
    if (!raw) return [];
    return JSON.parse(raw);
  } catch {
    return [];
  }
}

export function isAppPinned(appId: string): boolean {
  return getPinnedAppIds().includes(appId);
}

export function togglePinApp(appId: string): boolean {
  const current = getPinnedAppIds();
  let updated: string[];
  let isNowPinned: boolean;

  if (current.includes(appId)) {
    updated = current.filter((id) => id !== appId);
    isNowPinned = false;
  } else {
    updated = [...current, appId];
    isNowPinned = true;
  }

  if (typeof window !== 'undefined') {
    localStorage.setItem(PINNED_APPS_STORAGE_KEY, JSON.stringify(updated));
    window.dispatchEvent(new Event('hostvra_pinned_apps_changed'));
  }

  return isNowPinned;
}
