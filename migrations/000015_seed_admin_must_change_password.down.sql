UPDATE `users`
SET `must_change_password` = TRUE
WHERE `username` = 'admin_local';
