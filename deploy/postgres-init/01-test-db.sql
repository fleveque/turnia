-- Runs once, when the volume is first created. Tests get their own database
-- so `make test` never touches development data.
CREATE DATABASE turnia_test OWNER turnia;
