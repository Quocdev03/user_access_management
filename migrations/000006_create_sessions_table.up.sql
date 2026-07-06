CREATE TABLE IF NOT EXISTS `sessions` (
    `id`                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `user_id`            BIGINT UNSIGNED NOT NULL,
    `token_hash`         VARCHAR(255) NOT NULL,
    `refresh_token_hash` VARCHAR(255) NOT NULL,
    `ip_address`         VARCHAR(45)  NULL DEFAULT NULL,
    `user_agent`         TEXT         NULL DEFAULT NULL,
    `device_id`          BIGINT UNSIGNED NULL DEFAULT NULL,
    `expires_at`         DATETIME     NOT NULL,
    `created_at`         DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_sessions_token_hash` (`token_hash`),
    UNIQUE KEY `uk_sessions_refresh_token_hash` (`refresh_token_hash`),
    INDEX `idx_sessions_user_id` (`user_id`),
    INDEX `idx_sessions_expires_at` (`expires_at`),

    CONSTRAINT `fk_sessions_user_id`
        FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
        ON DELETE CASCADE ON UPDATE CASCADE
    -- FK cho device_id sẽ được thêm ở migration 000007 (sau khi tạo bảng devices)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
