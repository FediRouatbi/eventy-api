CREATE TABLE checkout_orders (
    id CHAR(36) PRIMARY KEY,
    token CHAR(36) NOT NULL UNIQUE,
    reservation_id CHAR(36) NOT NULL UNIQUE,
    order_number VARCHAR(32) NOT NULL UNIQUE,
    status VARCHAR(32) NOT NULL DEFAULT 'pending_payment',
    customer_name VARCHAR(255) NOT NULL,
    customer_email VARCHAR(255) NOT NULL,
    currency CHAR(3) NOT NULL,
    subtotal DECIMAL(12,2) NOT NULL DEFAULT 0,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_checkout_orders_reservation
        FOREIGN KEY (reservation_id) REFERENCES ticket_reservations (id)
        ON DELETE CASCADE
);

CREATE INDEX idx_checkout_orders_expires_at ON checkout_orders (expires_at);
CREATE INDEX idx_checkout_orders_customer_email ON checkout_orders (customer_email);

CREATE TABLE checkout_order_items (
    id CHAR(36) PRIMARY KEY,
    order_id CHAR(36) NOT NULL,
    ticket_type_id CHAR(36) NOT NULL,
    ticket_type_name VARCHAR(255) NOT NULL,
    quantity INT NOT NULL,
    unit_price DECIMAL(12,2) NOT NULL,
    currency CHAR(3) NOT NULL,
    event_id CHAR(36) NOT NULL,
    event_title VARCHAR(255) NOT NULL,
    session_id CHAR(36) NOT NULL,
    session_starts_at TIMESTAMP NOT NULL,
    session_ends_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_checkout_order_items_order
        FOREIGN KEY (order_id) REFERENCES checkout_orders (id)
        ON DELETE CASCADE,
    CONSTRAINT fk_checkout_order_items_ticket_type
        FOREIGN KEY (ticket_type_id) REFERENCES ticket_types (id)
        ON DELETE RESTRICT
);

CREATE INDEX idx_checkout_order_items_order_id ON checkout_order_items (order_id);
