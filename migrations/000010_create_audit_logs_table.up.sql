CREATE TABLE IF NOT EXISTS `audit_logs` (
    `id`          BIGINT UNSIGNED              NOT NULL AUTO_INCREMENT,
    `user_id`     BIGINT UNSIGNED              NULL DEFAULT NULL,
    `action`      VARCHAR(50)                  NOT NULL,
    `resource`    VARCHAR(50)                  NULL DEFAULT NULL,
    `resource_id` VARCHAR(50)                  NULL DEFAULT NULL,
    `ip_address`  VARCHAR(45)                  NULL DEFAULT NULL,
    `user_agent`  TEXT                         NULL DEFAULT NULL,
    `old_values`  JSON                         NULL DEFAULT NULL,
    `new_values`  JSON                         NULL DEFAULT NULL,
    `status`      VARCHAR(50)                  NOT NULL,
    `created_at`  DATETIME                     NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (`id`),
    INDEX `idx_audit_logs_user_id` (`user_id`),
    INDEX `idx_audit_logs_action` (`action`),
    INDEX `idx_audit_logs_created_at` (`created_at`),
    INDEX `idx_audit_logs_resource` (`resource`, `resource_id`),

    CONSTRAINT `fk_audit_logs_user_id`
        FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
        ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
