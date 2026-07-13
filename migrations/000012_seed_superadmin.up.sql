-- =============================================================================
-- Seed: Super Admin Account (local bootstrap only)
-- Login: email admin@localhost.local / password LocalDev@ChangeMe1
-- (API login uses email, not username.) NEVER commit real prod credentials.
-- =============================================================================

INSERT IGNORE INTO `users` (
    `username`, `email`, `password_hash`, `full_name`, `phone`,
    `date_of_birth`, `status`, `email_verified`
) VALUES (
    'admin_local',
    'admin@localhost.local',
    '$2a$10$tIgJSlVfv1hNKvcNZFpjK.HaE.YAhR3nFT0RUO.PoXWD1wJ9HBuFe',
    'Local Super Admin',
    '0000000000',
    '2000-01-01',
    'active',
    1
);

INSERT IGNORE INTO `user_roles` (`user_id`, `role_id`)
SELECT `id`, 1 FROM `users` WHERE `username` = 'admin_local';
