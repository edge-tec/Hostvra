-- Migration 0016: Add organization_id to databases for multi-tenant database isolation
ALTER TABLE databases ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_databases_organization_id ON databases(organization_id);
