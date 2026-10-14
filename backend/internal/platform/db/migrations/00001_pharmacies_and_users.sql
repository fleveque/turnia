-- +goose Up

-- Case-insensitive text, so Laia@Farmacia.es and laia@farmacia.es are the
-- same email.
CREATE EXTENSION IF NOT EXISTS citext;

-- A pharmacy is a tenant: everything else belongs to exactly one (design §3).
CREATE TABLE pharmacies (
    id             uuid        PRIMARY KEY DEFAULT uuidv7(),
    name           text        NOT NULL CHECK (name <> ''),
    timezone       text        NOT NULL DEFAULT 'Europe/Madrid',
    default_locale text        NOT NULL DEFAULT 'es' CHECK (default_locale IN ('es', 'ca')),
    plan           text        NOT NULL DEFAULT 'free',
    swaps_enabled  boolean     NOT NULL DEFAULT false,
    created_at     timestamptz NOT NULL DEFAULT now()
);

-- The people of a pharmacy. Turnia's own staff are a separate table (ADR 0008).
CREATE TABLE users (
    id          uuid        PRIMARY KEY DEFAULT uuidv7(),
    pharmacy_id uuid        NOT NULL REFERENCES pharmacies (id),
    email       citext      NOT NULL UNIQUE,
    name        text        NOT NULL CHECK (name <> ''),
    role        text        NOT NULL CHECK (role IN ('admin', 'employee')),
    locale      text        NOT NULL DEFAULT 'es' CHECK (locale IN ('es', 'ca')),
    active      boolean     NOT NULL DEFAULT true,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX users_pharmacy_id_idx ON users (pharmacy_id);

-- +goose Down

DROP TABLE users;
DROP TABLE pharmacies;
