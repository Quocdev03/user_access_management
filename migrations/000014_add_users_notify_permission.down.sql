DELETE rp FROM role_permissions rp
JOIN permissions p ON rp.permission_id = p.id
WHERE p.name = 'users.notify';

DELETE FROM permissions WHERE name = 'users.notify';