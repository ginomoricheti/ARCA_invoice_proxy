-- Migration: 001_initial_schema.sql
-- Description: Initial database schema for ARCA Invoice Proxy

-- Users table (customers of the SaaS)
CREATE TABLE users (
    id          VARCHAR(36) PRIMARY KEY,
    cuit        VARCHAR(11) NOT NULL UNIQUE,
    name        VARCHAR(255) NOT NULL,
    email       VARCHAR(255),
    address     TEXT NOT NULL,
    iva_condition VARCHAR(2) NOT NULL,
    country_code CHAR(2) NOT NULL DEFAULT 'AR',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_cuit ON users(cuit);
CREATE INDEX idx_users_email ON users(email);

-- API Keys table
CREATE TABLE api_keys (
    id           VARCHAR(36) PRIMARY KEY,
    user_id      VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         VARCHAR(255) NOT NULL,
    prefix       VARCHAR(20) NOT NULL,
    hash         VARCHAR(64) NOT NULL,
    last_four    VARCHAR(4) NOT NULL,
    scopes       TEXT[] NOT NULL DEFAULT '{}',
    expires_at   TIMESTAMPTZ,
    revoked_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_api_keys_prefix_hash ON api_keys(prefix, hash);
CREATE INDEX idx_api_keys_last_four ON api_keys(last_four);

-- ARCA Credentials table
CREATE TABLE arca_credentials (
    id           VARCHAR(36) PRIMARY KEY,
    user_id      VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    environment  VARCHAR(20) NOT NULL CHECK (environment IN ('homologation', 'production')),
    cuit         VARCHAR(11) NOT NULL,
    cert_pem     BYTEA NOT NULL,
    key_pem      BYTEA NOT NULL,
    expires_at   TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_arca_credentials_user_id ON arca_credentials(user_id);
CREATE INDEX idx_arca_credentials_cuit_env ON arca_credentials(cuit, environment);

-- Invoices table
CREATE TABLE invoices (
    id               VARCHAR(36) PRIMARY KEY,
    user_id          VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    invoice_type     CHAR(1) NOT NULL CHECK (invoice_type IN ('A', 'B', 'C')),
    point_of_sale    INTEGER NOT NULL CHECK (point_of_sale > 0),
    status           VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'issued', 'rejected', 'cancelled')),
    subtotal_cents   BIGINT NOT NULL DEFAULT 0,
    tax_cents        BIGINT NOT NULL DEFAULT 0,
    total_cents      BIGINT NOT NULL DEFAULT 0,
    cae              VARCHAR(14),
    cae_expires_at   DATE,
    voucher_number   BIGINT,
    voucher_type     INTEGER,
    arca_result      VARCHAR(20),
    arca_observations JSONB,
    idempotency_key  VARCHAR(64),
    issued_at        TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_invoices_user_id ON invoices(user_id);
CREATE INDEX idx_invoices_status ON invoices(status);
CREATE INDEX idx_invoices_idempotency_key ON invoices(idempotency_key);
CREATE INDEX idx_invoices_created_at ON invoices(created_at);
CREATE UNIQUE INDEX idx_invoices_user_idempotency ON invoices(user_id, idempotency_key) WHERE idempotency_key IS NOT NULL;

-- Invoice Items table
CREATE TABLE invoice_items (
    id              VARCHAR(36) PRIMARY KEY,
    invoice_id      VARCHAR(36) NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    description     TEXT NOT NULL,
    quantity        INTEGER NOT NULL CHECK (quantity > 0),
    unit_price_cents BIGINT NOT NULL CHECK (unit_price_cents >= 0),
    total_cents     BIGINT NOT NULL CHECK (total_cents >= 0),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_invoice_items_invoice_id ON invoice_items(invoice_id);

-- Idempotency Keys table
CREATE TABLE idempotency_keys (
    key            VARCHAR(64) PRIMARY KEY,
    user_id        VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    request_hash   VARCHAR(64) NOT NULL,
    status         VARCHAR(20) NOT NULL DEFAULT 'processing' CHECK (status IN ('processing', 'succeeded', 'failed')),
    response_body  BYTEA,
    error_message  TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at     TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_idempotency_keys_user_id ON idempotency_keys(user_id);
CREATE INDEX idx_idempotency_keys_expires_at ON idempotency_keys(expires_at);
CREATE INDEX idx_idempotency_keys_status ON idempotency_keys(status);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_api_keys_updated_at BEFORE UPDATE ON api_keys FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_arca_credentials_updated_at BEFORE UPDATE ON arca_credentials FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_invoices_updated_at BEFORE UPDATE ON invoices FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_idempotency_keys_updated_at BEFORE UPDATE ON idempotency_keys FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();