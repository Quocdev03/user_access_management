-- Full User-Agent fit (Chrome/Edge/Firefox/Safari/mobile ~ 255)
ALTER TABLE `devices`
    MODIFY COLUMN `device_name` VARCHAR(255) NULL DEFAULT NULL;
