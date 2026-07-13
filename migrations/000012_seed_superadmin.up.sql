-- =============================================================================
-- Seed: ONE bootstrap admin (dev / first deploy only)
--
--   email:    admin@localhost.local   ← POST /auth/login uses email
--   password: LocalDev@ChangeMe1
--   username: admin_local             ← not used for login
--   role:     admin (role_id = 1)
--
-- NEVER commit production credentials. Change password after first use.
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
