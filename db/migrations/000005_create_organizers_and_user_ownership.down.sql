ALTER TABLE users
DROP FOREIGN KEY fk_users_organizer,
DROP COLUMN organizer_id;

DROP TABLE IF EXISTS organizers;
