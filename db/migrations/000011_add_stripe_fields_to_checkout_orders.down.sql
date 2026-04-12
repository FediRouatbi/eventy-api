ALTER TABLE checkout_orders
    DROP COLUMN paid_at,
    DROP COLUMN stripe_customer_id,
    DROP COLUMN stripe_payment_intent_id,
    DROP COLUMN stripe_checkout_session_id;

