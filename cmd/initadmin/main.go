// cmd/initadmin/main.go
// Creates the first admin user in the goFarmacia database.
// Run via: make db-create-admin NOMBRE=... APELLIDO=... EMAIL=... CEDULA=... PASSWORD=...
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	nombre := flag.String("nombre", "", "Nombre del administrador")
	apellido := flag.String("apellido", "", "Apellido del administrador")
	email := flag.String("email", "", "Email del administrador")
	cedula := flag.String("cedula", "", "Cédula del administrador")
	password := flag.String("password", "", "Contraseña del administrador")
	flag.Parse()

	// Load .env from current or parent directory
	if err := godotenv.Load(); err != nil {
		godotenv.Load("../../.env")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "Error: DATABASE_URL no está configurado en .env")
		os.Exit(1)
	}

	// Validate required flags
	var missing []string
	if *nombre == "" {
		missing = append(missing, "--nombre")
	}
	if *apellido == "" {
		missing = append(missing, "--apellido")
	}
	if *email == "" {
		missing = append(missing, "--email")
	}
	if *cedula == "" {
		missing = append(missing, "--cedula")
	}
	if *password == "" {
		missing = append(missing, "--password")
	}
	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "Faltan parámetros: %s\n\n", strings.Join(missing, ", "))
		fmt.Fprintln(os.Stderr, "Uso:")
		fmt.Fprintln(os.Stderr, "  make db-create-admin NOMBRE=Juan APELLIDO=Pérez EMAIL=admin@ejemplo.com CEDULA=12345678 PASSWORD=miClave123")
		os.Exit(1)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error al abrir la BD: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "Error al conectar a la BD: %v\n", err)
		os.Exit(1)
	}

	// Warn if users already exist
	var count int
	db.QueryRow("SELECT COUNT(*) FROM vendedors WHERE deleted_at IS NULL").Scan(&count)
	if count > 0 {
		fmt.Printf("Advertencia: ya existen %d usuario(s) en la BD. Se creará de todas formas con rol 'admin'.\n", count)
	}

	// Hash password with bcrypt cost 14 (matches backend)
	hash, err := bcrypt.GenerateFromPassword([]byte(*password), 14)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error al encriptar contraseña: %v\n", err)
		os.Exit(1)
	}

	now := time.Now()
	userUUID := uuid.New().String()
	normalizedEmail := strings.ToLower(strings.TrimSpace(*email))

	_, err = db.Exec(`
		INSERT INTO vendedors (uuid, nombre, apellido, cedula, email, contrasena, mfa_enabled, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, false, 'admin', $7, $8)
	`, userUUID, strings.TrimSpace(*nombre), strings.TrimSpace(*apellido),
		strings.TrimSpace(*cedula), normalizedEmail, string(hash), now, now)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error al crear usuario: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("✓ Administrador creado exitosamente")
	fmt.Printf("  Nombre:  %s %s\n", strings.TrimSpace(*nombre), strings.TrimSpace(*apellido))
	fmt.Printf("  Email:   %s\n", normalizedEmail)
	fmt.Printf("  Cédula:  %s\n", strings.TrimSpace(*cedula))
	fmt.Printf("  Rol:     admin\n")
	fmt.Printf("  UUID:    %s\n", userUUID)
	fmt.Println()
}
