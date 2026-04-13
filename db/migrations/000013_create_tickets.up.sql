CREATE TABLE tickets (
    id CHAR(36) NOT NULL PRIMARY KEY,
    code CHAR(36) NOT NULL UNIQUE,
    order_id CHAR(36) NOT NULL,
    order_item_id CHAR(36) NOT NULL,
    customer_name VARCHAR(255) NOT NULL,
    customer_email VARCHAR(255) NOT NULL,
    ticket_type_id CHAR(36) NOT NULL,
    ticket_type_name VARCHAR(255) NOT NULL,
    event_id CHAR(36) NOT NULL,
    event_title VARCHAR(255) NOT NULL,
    session_id CHAR(36) NOT NULL,
    session_starts_at TIMESTAMP NOT NULL,
    session_ends_at TIMESTAMP NOT NULL,
    checked_in_at TIMESTAMP NULL,
    checked_in_by_user_id CHAR(36) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_tickets_order
        FOREIGN KEY (order_id) REFERENCES checkout_orders(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_tickets_order_item
        FOREIGN KEY (order_item_id) REFERENCES checkout_order_items(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_tickets_customer_email ON tickets (customer_email);
CREATE INDEX idx_tickets_event_id ON tickets (event_id);
CREATE INDEX idx_tickets_session_id ON tickets (session_id);
CREATE INDEX idx_tickets_order_id ON tickets (order_id);

