ALTER TABLE checkout_orders
    ADD COLUMN stripe_checkout_session_id VARCHAR(255) NULL UNIQUE,
    ADD COLUMN stripe_payment_intent_id VARCHAR(255) NULL UNIQUE,
    ADD COLUMN stripe_customer_id VARCHAR(255) NULL,
    ADD COLUMN paid_at TIMESTAMP NULL;

