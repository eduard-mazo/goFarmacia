-- Add NIT identifier to proveedors (used to link with DIAN electronic invoices)
ALTER TABLE proveedors ADD COLUMN IF NOT EXISTS nit text NOT NULL DEFAULT '';

-- Unique index on nit (non-empty only) so we can upsert by NIT
CREATE UNIQUE INDEX IF NOT EXISTS idx_proveedors_nit ON proveedors(nit) WHERE nit != '';
