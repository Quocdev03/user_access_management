-- Local bootstrap: allow login with seed password without force-change friction.
-- (Production: still change LocalDev@ChangeMe1 immediately after first deploy.)
UPDATE `users`
SET `must_change_password` = FALSE
WHERE `username` = 'admin_local';
