module.exports = {
  apps: [
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
