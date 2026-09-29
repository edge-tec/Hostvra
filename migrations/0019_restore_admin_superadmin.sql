-- Migration 0019: Restore administrator and superadmin permissions
-- 1. Ensure 'admin' role exists for all organizations
INSERT INTO roles (organization_id, name, description, is_system, created_at, updated_at)
SELECT id, 'admin', 'Full system administrator', true, NOW(), NOW()
FROM organizations
ON CONFLICT (organization_id, name) DO NOTHING;

-- 2. Promote the primary admin users to superadmin
UPDATE users
SET is_superadmin = TRUE, updated_at = NOW()
WHERE email LIKE 'admin@%' OR email LIKE '%@hostvra.com' OR email LIKE '%@hostvra.local';

-- If no users matched, promote the first created user
UPDATE users
SET is_superadmin = TRUE, updated_at = NOW()
WHERE id = (SELECT id FROM users ORDER BY created_at ASC LIMIT 1);

-- 3. Link superadmin users to admin role in organization_members
UPDATE organization_members om
SET role_id = r.id, updated_at = NOW()
FROM users u
JOIN roles r ON r.organization_id = om.organization_id AND r.name = 'admin'
WHERE om.user_id = u.id AND u.is_superadmin = TRUE;
