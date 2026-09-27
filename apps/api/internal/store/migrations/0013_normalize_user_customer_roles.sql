-- Migration 0013: Normalize legacy 'owner' roles to 'customer' for hosting users

-- 1. Ensure 'customer' role exists for any organization that has an 'owner' role
INSERT INTO roles (organization_id, name, description, is_system, created_at, updated_at)
SELECT organization_id, 'customer', 'Standard hosting customer with package allocation', false, NOW(), NOW()
FROM roles
WHERE name = 'owner'
ON CONFLICT (organization_id, name) DO NOTHING;

-- 2. Migrate organization members who had the 'owner' role to the 'customer' role
UPDATE organization_members om
SET role_id = cust_role.id, updated_at = NOW()
FROM roles own_role
JOIN roles cust_role ON own_role.organization_id = cust_role.organization_id AND cust_role.name = 'customer'
WHERE own_role.name = 'owner' AND om.role_id = own_role.id;

-- 3. Delete leftover 'owner' roles so they no longer appear
DELETE FROM roles WHERE name = 'owner';
