-- =============================================================================
-- Revert: Seed Super Admin Account
-- =============================================================================

-- Xóa quyền Admin của user
DELETE FROM `user_roles` 
WHERE `user_id` = (SELECT `id` FROM `users` WHERE `username` = 'admin_quocdev');

-- Xóa user Admin
DELETE FROM `users` 
WHERE `username` = 'admin_quocdev';
