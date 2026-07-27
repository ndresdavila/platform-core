-- Facturación electrónica MVP (Masterview / Contifico Mock)
CREATE TABLE IF NOT EXISTS persons (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  kind TEXT NOT NULL DEFAULT 'CLIENTE', -- CLIENTE | PROVEEDOR | AMBOS
  identification_type TEXT NOT NULL DEFAULT '04', -- SRI: 04 RUC, 05 cédula, 06 pasaporte, 07 consumidor final
  identification TEXT NOT NULL,
  name TEXT NOT NULL,
  email TEXT NOT NULL DEFAULT '',
  phone TEXT NOT NULL DEFAULT '',
  address TEXT NOT NULL DEFAULT '',
  active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, identification)
);

CREATE INDEX IF NOT EXISTS persons_tenant_name_idx ON persons (tenant_id, name);

CREATE TABLE IF NOT EXISTS products (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  code TEXT NOT NULL DEFAULT '',
  name TEXT NOT NULL,
  unit TEXT NOT NULL DEFAULT 'UND',
  price_cents INT NOT NULL DEFAULT 0,
  iva_rate NUMERIC(5,2) NOT NULL DEFAULT 15.00,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, code)
);

CREATE INDEX IF NOT EXISTS products_tenant_name_idx ON products (tenant_id, name);

CREATE TABLE IF NOT EXISTS document_sequences (
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  establishment TEXT NOT NULL DEFAULT '001',
  emission_point TEXT NOT NULL DEFAULT '001',
  doc_type TEXT NOT NULL DEFAULT 'FACTURA',
  next_number BIGINT NOT NULL DEFAULT 1,
  PRIMARY KEY (tenant_id, establishment, emission_point, doc_type)
);

CREATE TABLE IF NOT EXISTS electronic_documents (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  doc_type TEXT NOT NULL DEFAULT 'FACTURA',
  party_kind TEXT NOT NULL DEFAULT 'CLIENTE', -- CLIENTE | PROVEEDOR
  person_id UUID REFERENCES persons(id),
  person_name TEXT NOT NULL DEFAULT '',
  person_identification TEXT NOT NULL DEFAULT '',
  establishment TEXT NOT NULL DEFAULT '001',
  emission_point TEXT NOT NULL DEFAULT '001',
  document_number TEXT NOT NULL,
  access_key TEXT NOT NULL DEFAULT '',
  issue_date DATE NOT NULL,
  due_days INT NOT NULL DEFAULT 0,
  reference TEXT NOT NULL DEFAULT '',
  seller TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  is_export BOOLEAN NOT NULL DEFAULT FALSE,
  status TEXT NOT NULL DEFAULT 'DRAFT', -- DRAFT | SAVED | SENT | AUTHORIZED | REJECTED
  sri_message TEXT NOT NULL DEFAULT '',
  subtotal_15_cents INT NOT NULL DEFAULT 0,
  subtotal_5_cents INT NOT NULL DEFAULT 0,
  subtotal_0_cents INT NOT NULL DEFAULT 0,
  discount_cents INT NOT NULL DEFAULT 0,
  iva_15_cents INT NOT NULL DEFAULT 0,
  iva_5_cents INT NOT NULL DEFAULT 0,
  ice_cents INT NOT NULL DEFAULT 0,
  total_cents INT NOT NULL DEFAULT 0,
  created_by_external_id TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, document_number)
);

CREATE INDEX IF NOT EXISTS electronic_documents_tenant_date_idx
  ON electronic_documents (tenant_id, issue_date DESC);

CREATE TABLE IF NOT EXISTS electronic_document_lines (
  id UUID PRIMARY KEY,
  document_id UUID NOT NULL REFERENCES electronic_documents(id) ON DELETE CASCADE,
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  line_no INT NOT NULL,
  product_id UUID REFERENCES products(id),
  product_name TEXT NOT NULL,
  unit TEXT NOT NULL DEFAULT 'UND',
  quantity NUMERIC(14,4) NOT NULL DEFAULT 1,
  unit_price_cents INT NOT NULL DEFAULT 0,
  iva_rate NUMERIC(5,2) NOT NULL DEFAULT 15.00,
  discount_percent NUMERIC(7,4) NOT NULL DEFAULT 0,
  discount_cents INT NOT NULL DEFAULT 0,
  subtotal_cents INT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS electronic_document_lines_doc_idx
  ON electronic_document_lines (document_id, line_no);
