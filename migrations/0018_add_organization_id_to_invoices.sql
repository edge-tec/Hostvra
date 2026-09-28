-- Migration 0018: Add organization_id to invoices for multi-tenant isolation
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_invoices_organization_id ON invoices(organization_id);

-- Backfill organization_id for existing invoices from their parent subscriptions or user's default organization
UPDATE invoices i
SET organization_id = s.organization_id
FROM subscriptions s
WHERE i.subscription_id = s.id AND i.organization_id IS NULL;

UPDATE invoices i
SET organization_id = u.default_org_id
FROM users u
WHERE i.user_id = u.id AND i.organization_id IS NULL;
