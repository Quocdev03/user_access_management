CREATE TABLE IF NOT EXISTS `devices` (
    `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `user_id`        BIGINT UNSIGNED NOT NULL,
    `device_name`    VARCHAR(255) NULL DEFAULT NULL,
    `device_type`    VARCHAR(50)  NULL DEFAULT NULL,
    `os`             VARCHAR(50)  NULL DEFAULT NULL,
    `browser`        VARCHAR(50)  NULL DEFAULT NULL,
    `ip_address`     VARCHAR(45)  NULL DEFAULT NULL,
    `last_active_at` DATETIME     NULL DEFAULT NULL,
    `created_at`     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (`id`),
    INDEX `idx_devices_user_id` (`user_id`),

    CONSTRAINT `fk_devices_user_id`
        FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
        ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Thêm FK constraint cho sessions.device_id → devices.id (deferred từ migration 000006)
ALTER TABLE `sessions`
    ADD CONSTRAINT `fk_sessions_device_id`
        FOREIGN KEY (`device_id`) REFERENCES `devices` (`id`)
        ON DELETE SET NULL ON UPDATE CASCADE;
