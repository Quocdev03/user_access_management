-- Xóa dữ liệu seed theo thứ tự ngược (do FK constraints)
DELETE FROM `role_permissions`;
DELETE FROM `permissions`;
DELETE FROM `roles`;
