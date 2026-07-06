-- =============================================================================
-- Revert: Seed Super Admin Account
-- =============================================================================

-- Lấy user_id lưu vào biến
SET @admin_id = (SELECT `id` FROM `users` WHERE `username` = 'admin_quocdev');

-- Xóa các bảng phụ thuộc trước
DELETE FROM `audit_logs` WHERE `user_id` = @admin_id;
DELETE FROM `sessions` WHERE `user_id` = @admin_id;
DELETE FROM `devices` WHERE `user_id` = @admin_id;
DELETE FROM `otp_codes` WHERE `user_id` = @admin_id;
DELETE FROM `password_reset_tokens` WHERE `user_id` = @admin_id;
DELETE FROM `user_roles` WHERE `user_id` = @admin_id;

-- Xóa user Admin
DELETE FROM `users` WHERE `id` = @admin_id;
