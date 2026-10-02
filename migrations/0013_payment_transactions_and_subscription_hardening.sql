-- ============================================================================
-- Migration 0013: Payment Transactions, Audit Trail & Subscription Hardening
-- Production PostgreSQL DDL for Payment Verification, State Machine & Transactions
-- ============================================================================

-- 1. Create payment_transactions table for immutable accounting audit records
CREATE TABLE IF NOT EXISTS payment_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    package_id UUID NOT NULL REFERENCES hosting_plans(id) ON DELETE RESTRICT,
    subscription_id UUID REFERENCES subscriptions(id) ON DELETE SET NULL,
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    gateway VARCHAR(50) NOT NULL, -- stripe, bkash, nagad
    transaction_id VARCHAR(255) NOT NULL,
    order_id VARCHAR(100) NOT NULL,
    amount NUMERIC(12, 2) NOT NULL,
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, initiated, completed, failed, cancelled, refunded
    gateway_reference VARCHAR(255),
    payment_method VARCHAR(50),
    raw_response TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    confirmed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_payment_transactions_org ON payment_transactions(organization_id);
CREATE INDEX IF NOT EXISTS idx_payment_transactions_user ON payment_transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_payment_transactions_txn ON payment_transactions(gateway, transaction_id);
CREATE INDEX IF NOT EXISTS idx_payment_transactions_invoice ON payment_transactions(invoice_id);
CREATE INDEX IF NOT EXISTS idx_payment_transactions_status ON payment_transactions(status);

-- 2. Add unique constraint on gateway transaction id to prevent duplicate activations
CREATE UNIQUE INDEX IF NOT EXISTS uq_payment_transactions_gateway_txn
    ON payment_transactions(gateway, transaction_id)
    WHERE status = 'completed' AND transaction_id != '';

-- 3. Ensure invoices have payment_gateway column
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS payment_gateway VARCHAR(50);
