CREATE TABLE IF NOT EXISTS `password_reset_tokens` (
    `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `user_id`    BIGINT UNSIGNED NOT NULL,
    `token_hash` VARCHAR(255) NOT NULL,
    `is_used`    TINYINT(1)   NOT NULL DEFAULT 0,
    `expires_at` DATETIME     NOT NULL,
    `created_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_password_reset_tokens_token_hash` (`token_hash`),
    INDEX `idx_password_reset_tokens_user_id` (`user_id`),
    INDEX `idx_password_reset_tokens_expires_at` (`expires_at`),

    CONSTRAINT `fk_password_reset_tokens_user_id`
        FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
        ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
