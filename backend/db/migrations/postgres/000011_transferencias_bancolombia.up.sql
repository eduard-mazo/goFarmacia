-- Notificaciones de transferencia recibidas por Bancolombia (PSE / Nequi / Corresponsal)
CREATE TABLE IF NOT EXISTS transferencias_bancolombia (
    uuid           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email_id       text NOT NULL,
    fecha          timestamp with time zone,
    monto          numeric(15,2) NOT NULL DEFAULT 0,
    remitente      text NOT NULL DEFAULT '',
    referencia     text NOT NULL DEFAULT '',
    cuenta_destino text NOT NULL DEFAULT '',
    concepto       text NOT NULL DEFAULT '',
    raw_subject    text NOT NULL DEFAULT '',
    leido          boolean NOT NULL DEFAULT false,
    created_at     timestamp with time zone DEFAULT NOW(),
    CONSTRAINT transferencias_bancolombia_email_unique UNIQUE (email_id)
);

CREATE INDEX IF NOT EXISTS idx_tb_fecha  ON transferencias_bancolombia(fecha DESC);
CREATE INDEX IF NOT EXISTS idx_tb_monto  ON transferencias_bancolombia(monto DESC);
CREATE INDEX IF NOT EXISTS idx_tb_leido  ON transferencias_bancolombia(leido);
