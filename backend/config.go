package backend

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

// DBConfig holds persistent configuration stored in db_config.json next to the executable.
type DBConfig struct {
	DSN       string `json:"dsn"`
	JWTSecret string `json:"jwt_secret,omitempty"`
}

func dbConfigPath() string {
	return filepath.Join(baseDir(), "db_config.json")
}

// LoadDBConfig reads db_config.json. Returns (config, true) if the file exists and has a DSN.
func LoadDBConfig() (DBConfig, bool) {
	data, err := os.ReadFile(dbConfigPath())
	if err != nil {
		return DBConfig{}, false
	}
	var cfg DBConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DBConfig{}, false
	}
	return cfg, cfg.DSN != ""
}

// SaveDBConfig writes db_config.json to the exe directory.
func SaveDBConfig(cfg DBConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(dbConfigPath(), data, 0600)
}

// GenerateSecureKey creates a cryptographically random 32-byte hex string.
func GenerateSecureKey() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "goFarmacia-fallback-key-please-set-JWT_SECRET_KEY-in-env"
	}
	return hex.EncodeToString(b)
}
