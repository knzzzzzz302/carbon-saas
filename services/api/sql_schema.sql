-- Schéma minimal CarbonV2 (MVP)

CREATE TABLE IF NOT EXISTS tenants (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    siret       TEXT,
    plan        TEXT NOT NULL DEFAULT 'free',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    tenant_id     BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email         TEXT NOT NULL UNIQUE,
    role          TEXT NOT NULL DEFAULT 'user',
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS entries (
    id          BIGSERIAL PRIMARY KEY,
    tenant_id   BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    type        TEXT NOT NULL, -- ex: expense, energy, fuel
    amount      NUMERIC(18,2) NOT NULL,
    currency    TEXT NOT NULL DEFAULT 'EUR',
    date        DATE NOT NULL,
    category    TEXT,
    source      TEXT,
    metadata    JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS emissions (
    id                  BIGSERIAL PRIMARY KEY,
    entry_id            BIGINT NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    scope               TEXT NOT NULL, -- '1', '2', '3'
    tco2e               NUMERIC(18,6) NOT NULL,
    methodology_version TEXT NOT NULL DEFAULT 'v1',
    computed_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS factors (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT NOT NULL,
    value      NUMERIC(18,6) NOT NULL,
    unit       TEXT NOT NULL, -- ex: "kgCO2e/EUR"
    source     TEXT,
    valid_from DATE,
    valid_to   DATE,
    version    TEXT NOT NULL DEFAULT 'v1'
);

-- Documents (factures, contrats énergie, etc.) liés au tenant.
CREATE TABLE IF NOT EXISTS documents (
    id           BIGSERIAL PRIMARY KEY,
    tenant_id    BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    original_name TEXT NOT NULL,
    mime_type    TEXT NOT NULL,
    size_bytes   BIGINT NOT NULL,
    storage_path TEXT NOT NULL,
    source       TEXT,               -- ex: "EDF", "GRDF", "Fournisseur X"
    kind         TEXT,               -- ex: "facture_energie", "facture_transport"
    analysis     JSONB,              -- future analyse IA (facteurs, kWh, postes, etc.)
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS billing (
    sub_id             TEXT PRIMARY KEY,
    tenant_id          BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    stripe_customer_id TEXT,
    plan               TEXT NOT NULL,
    status             TEXT NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);


