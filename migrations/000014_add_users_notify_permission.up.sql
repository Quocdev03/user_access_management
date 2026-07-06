INSERT IGNORE INTO permissions (name, description, resource, action)
VALUES ('users.notify', 'Gửi thông báo bắt buộc cho người dùng', 'users', 'notify');

-- Gán lại TẤT CẢ permissions cho admin
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'admin';