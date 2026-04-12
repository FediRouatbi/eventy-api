CREATE TABLE ticket_reservations (
    id CHAR(36) NOT NULL PRIMARY KEY,
    token CHAR(36) NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE INDEX idx_ticket_reservations_expires_at ON ticket_reservations (expires_at);

CREATE TABLE ticket_reservation_items (
    id CHAR(36) NOT NULL PRIMARY KEY,
    reservation_id CHAR(36) NOT NULL,
    ticket_type_id CHAR(36) NOT NULL,
    quantity INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_ticket_reservation_items_reservation
        FOREIGN KEY (reservation_id) REFERENCES ticket_reservations(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_ticket_reservation_items_ticket_type
        FOREIGN KEY (ticket_type_id) REFERENCES ticket_types(id)
        ON DELETE CASCADE,
    CONSTRAINT uq_ticket_reservation_items_reservation_ticket
        UNIQUE (reservation_id, ticket_type_id)
);

CREATE INDEX idx_ticket_reservation_items_reservation_id ON ticket_reservation_items (reservation_id);
CREATE INDEX idx_ticket_reservation_items_ticket_type_id ON ticket_reservation_items (ticket_type_id);
