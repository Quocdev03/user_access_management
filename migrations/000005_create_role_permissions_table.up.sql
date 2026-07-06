CREATE TABLE IF NOT EXISTS `role_permissions` (
    `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `role_id`       BIGINT UNSIGNED NOT NULL,
    `permission_id` BIGINT UNSIGNED NOT NULL,
    `assigned_at`   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_role_permissions_role_perm` (`role_id`, `permission_id`),

    CONSTRAINT `fk_role_permissions_role_id`
        FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`)
        ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT `fk_role_permissions_permission_id`
        FOREIGN KEY (`permission_id`) REFERENCES `permissions` (`id`)
        ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
