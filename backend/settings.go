package backend

import (
	"context"
	"database/sql"
	"errors"
)

// GetSetting returns the value for the given key, or "" if not set.
func (d *Db) GetSetting(key string) (string, error) {
	var value string
	err := d.QueryRow(context.Background(),
		`SELECT value FROM app_settings WHERE key = $1`, key,
	).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return value, err
}

// SetSetting upserts a key-value pair in app_settings.
func (d *Db) SetSetting(key, value string) error {
	_, err := d.Exec(context.Background(),
		`INSERT INTO app_settings (key, value) VALUES ($1, $2)
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`,
		key, value,
	)
	return err
}
