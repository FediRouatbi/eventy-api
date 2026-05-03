DROP INDEX idx_users_firebase_uid ON users;

ALTER TABLE users
    DROP COLUMN firebase_uid;
