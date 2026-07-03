CREATE TABLE IF NOT EXISTS `otp_codes` (
    `id`         BIGINT                                                     NOT NULL AUTO_INCREMENT,
    `user_id`    BIGINT                                                     NOT NULL,
    `code`       VARCHAR(10)                                                NOT NULL,
    `type`       ENUM('email_verification', 'forgot_password', 'change_email', 'change_email_old', 'change_email_new') NOT NULL,
    `attempts`   INT                                                        NOT NULL DEFAULT 0,
    `is_used`    TINYINT(1)                                                 NOT NULL DEFAULT 0,
    `expires_at` DATETIME                                                   NOT NULL,
    `created_at` DATETIME                                                   NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (`id`),
    INDEX `idx_otp_codes_user_id_type` (`user_id`, `type`),
    INDEX `idx_otp_codes_expires_at` (`expires_at`),

    CONSTRAINT `fk_otp_codes_user_id`
        FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
        ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
