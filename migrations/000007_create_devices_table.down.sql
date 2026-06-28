-- Xóa FK constraint trên sessions trước khi drop devices
ALTER TABLE `sessions` DROP FOREIGN KEY `fk_sessions_device_id`;

DROP TABLE IF EXISTS `devices`;
