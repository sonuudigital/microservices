DROP TRIGGER IF EXISTS trigger_set_default_payment_status ON payments;
DROP FUNCTION IF EXISTS set_default_payment_status();
DROP TABLE IF EXISTS payments;
DROP INDEX IF EXISTS idx_payment_status_name;
DROP TABLE IF EXISTS payment_statuses;
DROP EXTENSION IF EXISTS "uuid-ossp";