-- Hostvra Database Migration 0010: Enterprise Mail Server Subsystem Schema
-- Production PostgreSQL DDL for Provisioning and Managing Independent Email Server Nodes

CREATE TABLE IF NOT EXISTS mail_servers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    node_server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    hostname VARCHAR(255) NOT NULL,
    primary_domain VARCHAR(255) NOT NULL,
    additional_domains TEXT[] DEFAULT '{}',
    ipv4_address VARCHAR(45) NOT NULL,
    ipv6_address VARCHAR(45) DEFAULT '',
    timezone VARCHAR(50) DEFAULT 'UTC',
    storage_location VARCHAR(255) NOT NULL DEFAULT '/var/mail/vhosts',
    mailbox_storage_limit_bytes BIGINT NOT NULL DEFAULT 53687091200, -- 50GB default, 0 = unlimited
    max_mailbox_size_bytes BIGINT NOT NULL DEFAULT 10737418240,     -- 10GB default, 0 = unlimited
    max_attachment_size_bytes BIGINT NOT NULL DEFAULT 52428800,     -- 50MB default
    smtp_port INT NOT NULL DEFAULT 25,
    smtp_submission_port INT NOT NULL DEFAULT 587,
    smtps_port INT NOT NULL DEFAULT 465,
    imap_port INT NOT NULL DEFAULT 143,
    imaps_port INT NOT NULL DEFAULT 993,
    pop3_port INT NOT NULL DEFAULT 110,
    pop3s_port INT NOT NULL DEFAULT 995,
    tls_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    tls_cert_path VARCHAR(255) DEFAULT '',
    tls_key_path VARCHAR(255) DEFAULT '',
    spam_filter_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    antivirus_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    dkim_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    spf_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    dmarc_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    webmail_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    auto_ssl_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    backup_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, provisioning, stopped, error, maintenance
    provisioning_logs TEXT DEFAULT '',
    last_health_check_at TIMESTAMPTZ,
    health_status VARCHAR(50) DEFAULT 'healthy',
    rate_limit_per_mailbox_hr INT DEFAULT 500,
    rate_limit_per_domain_hr INT DEFAULT 5000,
    rate_limit_per_ip_hr INT DEFAULT 10000,
    auth_failure_threshold INT DEFAULT 5,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(node_server_id, hostname)
);

CREATE INDEX IF NOT EXISTS idx_mail_servers_org ON mail_servers(organization_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_mail_servers_node ON mail_servers(node_server_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_mail_servers_domain ON mail_servers(primary_domain) WHERE deleted_at IS NULL;

-- Link email_domains to dedicated mail_servers if not already existing
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'email_domains' AND column_name = 'mail_server_id') THEN
        ALTER TABLE email_domains ADD COLUMN mail_server_id UUID REFERENCES mail_servers(id) ON DELETE SET NULL;
        CREATE INDEX IF NOT EXISTS idx_email_domains_mail_server ON email_domains(mail_server_id);
    END IF;
END $$;
