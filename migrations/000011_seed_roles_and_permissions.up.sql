-- =============================================================================
-- Seed: Roles
-- =============================================================================
INSERT INTO `roles` (`name`, `description`) VALUES
    ('admin',     'Quản trị viên hệ thống — toàn quyền'),
    ('moderator', 'Người kiểm duyệt — quản lý user và nội dung'),
    ('user',      'Người dùng thông thường');

-- =============================================================================
-- Seed: Permissions
-- =============================================================================
INSERT INTO `permissions` (`name`, `description`, `resource`, `action`) VALUES
    -- Users
    ('users.create',  'Tạo người dùng mới',                'users', 'create'),
    ('users.read',    'Xem thông tin người dùng',           'users', 'read'),
    ('users.update',  'Cập nhật thông tin người dùng',      'users', 'update'),
    ('users.delete',  'Xóa người dùng',                    'users', 'delete'),
    ('users.lock',    'Khóa / mở khóa tài khoản',          'users', 'lock'),
    ('users.reset_password', 'Reset mật khẩu người dùng',  'users', 'reset_password'),

    -- Roles
    ('roles.create',  'Tạo vai trò mới',                   'roles', 'create'),
    ('roles.read',    'Xem danh sách vai trò',              'roles', 'read'),
    ('roles.update',  'Cập nhật vai trò',                   'roles', 'update'),
    ('roles.delete',  'Xóa vai trò',                        'roles', 'delete'),
    ('roles.assign',  'Gán vai trò cho người dùng',         'roles', 'assign'),

    -- Permissions
    ('permissions.read',   'Xem danh sách quyền',           'permissions', 'read'),
    ('permissions.assign', 'Gán quyền cho vai trò',          'permissions', 'assign'),

    -- Audit Logs
    ('audit_logs.read',   'Xem nhật ký kiểm toán',          'audit_logs', 'read'),
    ('audit_logs.export', 'Xuất nhật ký kiểm toán',         'audit_logs', 'export');

-- =============================================================================
-- Seed: Role-Permission assignments
-- =============================================================================

-- Admin gets ALL permissions
INSERT INTO `role_permissions` (`role_id`, `permission_id`)
SELECT r.id, p.id
FROM `roles` r
CROSS JOIN `permissions` p
WHERE r.name = 'admin';

-- Moderator gets user management permissions (read, update, lock, reset_password)
INSERT INTO `role_permissions` (`role_id`, `permission_id`)
SELECT r.id, p.id
FROM `roles` r
CROSS JOIN `permissions` p
WHERE r.name = 'moderator'
  AND p.name IN (
      'users.read',
      'users.update',
      'users.lock',
      'users.reset_password',
      'roles.read',
      'permissions.read',
      'audit_logs.read'
  );

-- User gets basic self-service permissions
INSERT INTO `role_permissions` (`role_id`, `permission_id`)
SELECT r.id, p.id
FROM `roles` r
CROSS JOIN `permissions` p
WHERE r.name = 'user'
  AND p.name IN (
      'users.read'
  );
