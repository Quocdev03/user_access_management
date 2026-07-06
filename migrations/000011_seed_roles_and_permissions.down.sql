-- Xóa dữ liệu seed theo thứ tự ngược (do FK constraints)
DELETE FROM `role_permissions`;
DELETE FROM `user_roles`;
DELETE FROM `permissions`;
DELETE FROM `roles`;
