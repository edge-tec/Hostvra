module.exports = {
  apps: [
    {
      name: 'hostvra-api',
      script: './bin/hostvra-api',
      cwd: './',
      instances: 1,
      autorestart: true,
      max_memory_restart: '500M',
      env: {
        PORT: 8080,
      },
    },
    {
      name: 'hostvra-web',
      script: 'npm',
      args: 'run start',
      cwd: './apps/web',
      instances: 1,
      autorestart: true,
      max_memory_restart: '1G',
      env: {
        NODE_ENV: 'production',
        PORT: 3000,
      },
    },
  ],
};

