DROP TRIGGER IF EXISTS trigger_set_default_order_status ON orders;
DROP FUNCTION IF EXISTS set_default_order_status();
DROP TABLE IF EXISTS orders;
DROP INDEX IF EXISTS idx_order_status_name;
DROP TABLE IF EXISTS order_statuses;
DROP EXTENSION IF EXISTS "uuid-ossp";