UPDATE events
SET currency = 'EUR'
WHERE currency <> 'EUR' OR currency IS NULL;

ALTER TABLE events
  ALTER COLUMN currency SET DEFAULT 'EUR';

