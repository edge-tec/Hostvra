-- Migration 0016: Add organization_id to databases for multi-tenant database isolation
ALTER TABLE databases ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_databases_organization_id ON databases(organization_id);

-- Backfill organization_id for existing production database records from their parent server
UPDATE databases d
SET organization_id = s.organization_id
FROM servers s
WHERE d.server_id = s.id AND d.organization_id IS NULL;
