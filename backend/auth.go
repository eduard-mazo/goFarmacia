package backend

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Claims define la estructura de los datos que contendrá nuestro JWT.
type Claims struct {
	UserID uint   `json:"user_id"`
	Nombre string `json:"nombre"`
	Cedula string `json:"cedula"`
	jwt.RegisteredClaims
}

// HashPassword toma una contraseña en texto plano y devuelve su hash bcrypt.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14) // El costo 14 es un buen balance de seguridad/velocidad.
	return string(bytes), err
}

// CheckPasswordHash compara una contraseña en texto plano con un hash para ver si coinciden.
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateJWT crea un nuevo token JWT para un vendedor.
func (d *Db) GenerateJWT(vendedor Vendedor) (string, error) {
	// El token expira en 24 horas.
	// En la función initDB garantizamos la captura del secreto para crear el jwt
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID: vendedor.ID,
		Nombre: vendedor.Nombre,
		Cedula: vendedor.Cedula,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(d.jwtKey)
}
