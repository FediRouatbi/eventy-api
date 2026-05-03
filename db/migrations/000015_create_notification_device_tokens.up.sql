CREATE TABLE notification_device_tokens (
    id CHAR(36) NOT NULL PRIMARY KEY,
    user_id CHAR(36) NOT NULL,
    token VARCHAR(512) NOT NULL,
    provider VARCHAR(20) NOT NULL,
    platform VARCHAR(20) NOT NULL,
    device_name VARCHAR(255) NULL,
    revoked_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_notification_device_tokens_user
        FOREIGN KEY (user_id) REFERENCES users(id)
        ON DELETE CASCADE,
    CONSTRAINT uq_notification_device_tokens_token UNIQUE (token)
);

CREATE INDEX idx_notification_device_tokens_user_id ON notification_device_tokens (user_id);
CREATE INDEX idx_notification_device_tokens_revoked_at ON notification_device_tokens (revoked_at);
