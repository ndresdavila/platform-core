-- Parity with nails product (multi-tenant generic names)
ALTER TABLE customers ADD COLUMN IF NOT EXISTS external_id TEXT;
ALTER TABLE customers ADD COLUMN IF NOT EXISTS national_id TEXT;
ALTER TABLE customers ADD COLUMN IF NOT EXISTS first_name TEXT NOT NULL DEFAULT '';
ALTER TABLE customers ADD COLUMN IF NOT EXISTS last_name TEXT NOT NULL DEFAULT '';
ALTER TABLE customers ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'CLIENTE';
ALTER TABLE customers ADD COLUMN IF NOT EXISTS active BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE customers ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE UNIQUE INDEX IF NOT EXISTS customers_tenant_external_uidx
  ON customers (tenant_id, external_id) WHERE external_id IS NOT NULL AND external_id <> '';
CREATE UNIQUE INDEX IF NOT EXISTS customers_tenant_national_uidx
  ON customers (tenant_id, national_id) WHERE national_id IS NOT NULL AND national_id <> '';

ALTER TABLE bookings ADD COLUMN IF NOT EXISTS design_id UUID;
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS operative_stage TEXT NOT NULL DEFAULT 'BOOKEADA';

ALTER TABLE payments ADD COLUMN IF NOT EXISTS bank_account_id UUID;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS bank_code TEXT NOT NULL DEFAULT '';
ALTER TABLE payments ADD COLUMN IF NOT EXISTS admin_notes TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS employees (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  external_id TEXT NOT NULL,
  email TEXT NOT NULL,
  first_name TEXT NOT NULL DEFAULT '',
  last_name TEXT NOT NULL DEFAULT '',
  role TEXT NOT NULL DEFAULT 'RECEPCIONISTA',
  active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, external_id),
  UNIQUE (tenant_id, email)
);

CREATE TABLE IF NOT EXISTS bank_accounts (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  bank_code TEXT NOT NULL,
  holder_name TEXT NOT NULL,
  account_number TEXT NOT NULL,
  account_type TEXT NOT NULL DEFAULT 'ahorros',
  tax_id TEXT NOT NULL DEFAULT '',
  notify_email TEXT NOT NULL DEFAULT '',
  active BOOLEAN NOT NULL DEFAULT TRUE,
  sort_order INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS designs (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  customer_id UUID NOT NULL REFERENCES customers(id),
  prompt TEXT NOT NULL,
  photo_url TEXT NOT NULL,
  result_url TEXT NOT NULL DEFAULT '',
  model_used TEXT NOT NULL DEFAULT 'mock',
  status TEXT NOT NULL DEFAULT 'PENDIENTE',
  error_message TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  processed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS designs_tenant_customer_idx ON designs (tenant_id, customer_id, created_at DESC);

CREATE TABLE IF NOT EXISTS tenant_settings (
  tenant_id UUID PRIMARY KEY REFERENCES tenants(id),
  feature_design_ai BOOLEAN NOT NULL DEFAULT FALSE,
  design_ai_daily_limit INT NOT NULL DEFAULT 3,
  design_ai_max_total INT NOT NULL DEFAULT 5,
  settings_json JSONB NOT NULL DEFAULT '{}'::jsonb
);
