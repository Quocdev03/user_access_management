CREATE TABLE IF NOT EXISTS `users` (
    `id`                    BIGINT          NOT NULL AUTO_INCREMENT,
    `username`              VARCHAR(50)     NOT NULL,
    `email`                 VARCHAR(255)    NOT NULL,
    `password_hash`         VARCHAR(255)    NOT NULL,
    `full_name`             VARCHAR(100)    NULL DEFAULT NULL,
    `phone`                 VARCHAR(20)     NULL DEFAULT NULL,
    `avatar_url`            VARCHAR(500)    NULL DEFAULT NULL,
    `date_of_birth`         DATE            NULL DEFAULT NULL,
    `status`                ENUM('active', 'inactive', 'locked') NOT NULL DEFAULT 'inactive',
    `email_verified`        TINYINT(1)      NOT NULL DEFAULT 0,
    `failed_login_attempts` INT             NOT NULL DEFAULT 0,
    `locked_until`          DATETIME        NULL DEFAULT NULL,
    `last_login_at`         DATETIME        NULL DEFAULT NULL,
    `created_at`            DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`            DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_users_username` (`username`),
    UNIQUE KEY `uk_users_email` (`email`),
    INDEX `idx_users_status_created` (`status`, `created_at`),
    INDEX `idx_users_email_verified_status` (`email_verified`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
