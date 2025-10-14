CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    description TEXT,
    code TEXT NOT NULL,
    price NUMERIC(10, 2) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    stock_quantity INTEGER NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NULL
);

CREATE INDEX idx_products_code ON products(code);
CREATE INDEX idx_products_created_at ON products(created_at);
