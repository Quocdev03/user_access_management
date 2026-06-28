-- =============================================================================
-- User Access Management (UAM) — Database Schema
-- MySQL 8 | InnoDB | utf8mb4_unicode_ci
-- =============================================================================
-- File này chứa toàn bộ schema và seed data cho hệ thống UAM.
-- Thứ tự tạo bảng đảm bảo FK dependency:
--   users → roles → permissions → user_roles → role_permissions
--   → devices → sessions → otp_codes → password_reset_tokens → audit_logs
--   → seed data
-- =============================================================================

-- -----------------------------------------------------------------------------
-- 1. users — Người dùng
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `users` (
    `id`                    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `username`              VARCHAR(50)     NOT NULL,
    `email`                 VARCHAR(255)    NOT NULL,
    `password_hash`         VARCHAR(255)    NOT NULL,
    `full_name`             VARCHAR(100)    NULL DEFAULT NULL,
    `phone`                 VARCHAR(20)     NULL DEFAULT NULL,
    `avatar_url`            VARCHAR(500)    NULL DEFAULT NULL,
    `date_of_birth`         DATE            NULL DEFAULT NULL,
    `status`                ENUM('active', 'inactive', 'locked') NOT NULL DEFAULT 'inactive',
    `email_verified`        TINYINT(1)      NOT NULL DEFAULT 0,
    `failed_login_attempts` TINYINT UNSIGNED NOT NULL DEFAULT 0,
    `locked_until`          TIMESTAMP       NULL DEFAULT NULL,
    `last_login_at`         TIMESTAMP       NULL DEFAULT NULL,
    `created_at`            TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`            TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_users_username` (`username`),
    UNIQUE KEY `uk_users_email` (`email`),
    INDEX `idx_users_status_created` (`status`, `created_at`),
    INDEX `idx_users_email_verified_status` (`email_verified`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- -----------------------------------------------------------------------------
-- 2. roles — Vai trò
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `roles` (
    `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `name`        VARCHAR(50)     NOT NULL,
    `description` VARCHAR(255)    NULL DEFAULT NULL,
    `created_at`  TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`  TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_roles_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- -----------------------------------------------------------------------------
-- 3. permissions — Quyền hạn
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `permissions` (
    `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `name`        VARCHAR(100)    NOT NULL,
    `description` VARCHAR(255)    NULL DEFAULT NULL,
    `resource`    VARCHAR(50)     NOT NULL,
    `action`      VARCHAR(50)     NOT NULL,
    `created_at`  TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_permissions_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- -----------------------------------------------------------------------------
-- 4. user_roles — Gán vai trò cho người dùng
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `user_roles` (
    `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `user_id`     BIGINT UNSIGNED NOT NULL,
    `role_id`     BIGINT UNSIGNED NOT NULL,
    `assigned_at` TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_user_roles_user_role` (`user_id`, `role_id`),
    INDEX `idx_user_roles_role_id` (`role_id`),

    CONSTRAINT `fk_user_roles_user_id`
        FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
        ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT `fk_user_roles_role_id`
        FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`)
        ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- -----------------------------------------------------------------------------
-- 5. role_permissions — Gán quyền cho vai trò
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `role_permissions` (
    `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `role_id`       BIGINT UNSIGNED NOT NULL,
    `permission_id` BIGINT UNSIGNED NOT NULL,
    `assigned_at`   TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_role_permissions_role_perm` (`role_id`, `permission_id`),

    CONSTRAINT `fk_role_permissions_role_id`
        FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`)
        ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT `fk_role_permissions_permission_id`
        FOREIGN KEY (`permission_id`) REFERENCES `permissions` (`id`)
        ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- -----------------------------------------------------------------------------
-- 6. devices — Thiết bị đăng nhập
-- (Tạo trước sessions do sessions.device_id reference devices.id)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `devices` (
    `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `user_id`        BIGINT UNSIGNED NOT NULL,
    `device_name`    VARCHAR(100)    NULL DEFAULT NULL,
    `device_type`    VARCHAR(50)     NULL DEFAULT NULL,
    `os`             VARCHAR(50)     NULL DEFAULT NULL,
    `browser`        VARCHAR(50)     NULL DEFAULT NULL,
    `ip_address`     VARCHAR(45)     NULL DEFAULT NULL,
    `last_active_at` TIMESTAMP       NULL DEFAULT NULL,
    `created_at`     TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (`id`),
    INDEX `idx_devices_user_id` (`user_id`),

    CONSTRAINT `fk_devices_user_id`
        FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
        ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- -----------------------------------------------------------------------------
-- 7. sessions — Phiên đăng nhập
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `sessions` (
    `id`                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `user_id`            BIGINT UNSIGNED NOT NULL,
    `token_hash`         VARCHAR(255)    NOT NULL,
    `refresh_token_hash` VARCHAR(255)    NOT NULL,
    `ip_address`         VARCHAR(45)     NULL DEFAULT NULL,
    `user_agent`         VARCHAR(500)    NULL DEFAULT NULL,
    `device_id`          BIGINT UNSIGNED NULL DEFAULT NULL,
    `expires_at`         TIMESTAMP       NOT NULL,
    `created_at`         TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_sessions_token_hash` (`token_hash`),
    UNIQUE KEY `uk_sessions_refresh_token_hash` (`refresh_token_hash`),
    INDEX `idx_sessions_user_id` (`user_id`),
    INDEX `idx_sessions_expires_at` (`expires_at`),

    CONSTRAINT `fk_sessions_user_id`
        FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
        ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT `fk_sessions_device_id`
        FOREIGN KEY (`device_id`) REFERENCES `devices` (`id`)
        ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- -----------------------------------------------------------------------------
-- 8. otp_codes — Mã OTP
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `otp_codes` (
    `id`         BIGINT UNSIGNED                                               NOT NULL AUTO_INCREMENT,
    `user_id`    BIGINT UNSIGNED                                               NOT NULL,
    `code`       VARCHAR(10)                                                   NOT NULL,
    `type`       ENUM('email_verification', 'forgot_password', 'change_email') NOT NULL,
    `attempts`   TINYINT UNSIGNED                                              NOT NULL DEFAULT 0,
    `is_used`    TINYINT(1)                                                    NOT NULL DEFAULT 0,
    `expires_at` TIMESTAMP                                                     NOT NULL,
    `created_at` TIMESTAMP                                                     NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (`id`),
    INDEX `idx_otp_codes_user_id_type` (`user_id`, `type`),
    INDEX `idx_otp_codes_expires_at` (`expires_at`),

    CONSTRAINT `fk_otp_codes_user_id`
        FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
        ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- -----------------------------------------------------------------------------
-- 9. password_reset_tokens — Token đặt lại mật khẩu
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `password_reset_tokens` (
    `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `user_id`    BIGINT UNSIGNED NOT NULL,
    `token_hash` VARCHAR(255)    NOT NULL,
    `is_used`    TINYINT(1)      NOT NULL DEFAULT 0,
    `expires_at` TIMESTAMP       NOT NULL,
    `created_at` TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_password_reset_tokens_token_hash` (`token_hash`),
    INDEX `idx_password_reset_tokens_user_id` (`user_id`),
    INDEX `idx_password_reset_tokens_expires_at` (`expires_at`),

    CONSTRAINT `fk_password_reset_tokens_user_id`
        FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
        ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- -----------------------------------------------------------------------------
-- 10. audit_logs — Nhật ký kiểm toán
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `audit_logs` (
    `id`          BIGINT UNSIGNED            NOT NULL AUTO_INCREMENT,
    `user_id`     BIGINT UNSIGNED            NULL DEFAULT NULL,
    `action`      VARCHAR(50)                NOT NULL,
    `resource`    VARCHAR(50)                NULL DEFAULT NULL,
    `resource_id` VARCHAR(50)                NULL DEFAULT NULL,
    `ip_address`  VARCHAR(45)                NULL DEFAULT NULL,
    `user_agent`  VARCHAR(500)               NULL DEFAULT NULL,
    `old_values`  JSON                       NULL DEFAULT NULL,
    `new_values`  JSON                       NULL DEFAULT NULL,
    `status`      ENUM('success', 'failure') NOT NULL,
    `created_at`  TIMESTAMP                  NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (`id`),
    INDEX `idx_audit_logs_user_id` (`user_id`),
    INDEX `idx_audit_logs_action` (`action`),
    INDEX `idx_audit_logs_created_at` (`created_at`),
    INDEX `idx_audit_logs_resource` (`resource`, `resource_id`),

    CONSTRAINT `fk_audit_logs_user_id`
        FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
        ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 11. Seed Data — Roles, Permissions, Role-Permission Assignments
-- =============================================================================

-- Roles
INSERT INTO `roles` (`name`, `description`) VALUES
    ('admin',     'Quản trị viên hệ thống — toàn quyền'),
    ('moderator', 'Người kiểm duyệt — quản lý user và nội dung'),
    ('user',      'Người dùng thông thường');

-- Permissions
INSERT INTO `permissions` (`name`, `description`, `resource`, `action`) VALUES
    -- Users
    ('users.create',         'Tạo người dùng mới',               'users', 'create'),
    ('users.read',           'Xem thông tin người dùng',          'users', 'read'),
    ('users.update',         'Cập nhật thông tin người dùng',     'users', 'update'),
    ('users.delete',         'Xóa người dùng',                   'users', 'delete'),
    ('users.lock',           'Khóa / mở khóa tài khoản',         'users', 'lock'),
    ('users.reset_password', 'Reset mật khẩu người dùng',        'users', 'reset_password'),
    -- Roles
    ('roles.create',         'Tạo vai trò mới',                  'roles', 'create'),
    ('roles.read',           'Xem danh sách vai trò',            'roles', 'read'),
    ('roles.update',         'Cập nhật vai trò',                  'roles', 'update'),
    ('roles.delete',         'Xóa vai trò',                      'roles', 'delete'),
    ('roles.assign',         'Gán vai trò cho người dùng',        'roles', 'assign'),
    -- Permissions
    ('permissions.read',     'Xem danh sách quyền',              'permissions', 'read'),
    ('permissions.assign',   'Gán quyền cho vai trò',             'permissions', 'assign'),
    -- Audit Logs
    ('audit_logs.read',      'Xem nhật ký kiểm toán',            'audit_logs', 'read'),
    ('audit_logs.export',    'Xuất nhật ký kiểm toán',           'audit_logs', 'export');

-- Admin gets ALL permissions
INSERT INTO `role_permissions` (`role_id`, `permission_id`)
SELECT r.id, p.id
FROM `roles` r
CROSS JOIN `permissions` p
WHERE r.name = 'admin';

-- Moderator gets user management permissions
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
