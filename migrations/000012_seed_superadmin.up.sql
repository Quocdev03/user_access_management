-- =============================================================================
-- Seed: Super Admin Account
-- =============================================================================

INSERT INTO `users` (
    `username`, `email`, `password_hash`, `full_name`, `phone`, 
    `date_of_birth`, `status`, `email_verified`
) VALUES (
    'admin_quocdev', 
    'admin@quocdev.com', 
    '$2b$10$UnRP6.d73ZTsALvnkBotj.ugbfuzQlAQp2wrUVXCBcZlEIyS9EVdW', 
    'Quoc Dev Administrator', 
    '0901234567', 
    '1995-01-01', 
    'active', 
    1
);

-- Gán quyền Admin (role_id = 1) cho user vừa tạo
INSERT INTO `user_roles` (`user_id`, `role_id`)
SELECT `id`, 1 FROM `users` WHERE `username` = 'admin_quocdev';
