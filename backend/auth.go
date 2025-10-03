package backend

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image/png"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (d *Db) GenerateJWT(vendedor Vendedor) (string, error) {
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

func (d *Db) GenerarMFA(email string) (MFASetupResponse, error) {
	// --- LÓGICA MODIFICADA ---
	var vendedor Vendedor
	if err := d.LocalDB.Where("email = ?", email).First(&vendedor).Error; err != nil {
		return MFASetupResponse{}, errors.New("vendedor no encontrado")
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "LUNA_POS",
		AccountName: email,
	})
	if err != nil {
		return MFASetupResponse{}, errors.New("no se pudo generar la clave MFA")
	}

	vendedor.MFASecret = key.Secret()
	if err := d.LocalDB.Save(&vendedor).Error; err != nil {
		return MFASetupResponse{}, errors.New("no se pudo guardar la clave MFA")
	}

	if d.isRemoteDBAvailable() {
		go d.syncVendedorToRemote(vendedor.ID)
	}

	var buf bytes.Buffer
	img, err := key.Image(200, 200)
	if err != nil {
		return MFASetupResponse{}, errors.New("no se pudo generar la imagen QR")
	}
	if err := png.Encode(&buf, img); err != nil {
		return MFASetupResponse{}, errors.New("no se pudo codificar la imagen QR")
	}
	imgBase64Str := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())

	return MFASetupResponse{
		Secret:   key.Secret(),
		ImageURL: imgBase64Str,
	}, nil
}

type contextKey string

func (d *Db) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &Claims{}

		tkn, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return d.jwtKey, nil
		})

		if err != nil || !tkn.Valid {
			http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
			return
		}
		if claims.MFAStep == "pending" {
			http.Error(w, "Unauthorized: MFA verification required", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), contextKey("userEmail"), claims.Email)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (d *Db) HabilitarMFA(email string, code string) (bool, error) {
	var vendedor Vendedor
	if err := d.LocalDB.Where("email = ?", email).First(&vendedor).Error; err != nil {
		return false, errors.New("usuario no encontrado")
	}

	if vendedor.MFASecret == "" {
		return false, errors.New("el secreto MFA no ha sido generado aún")
	}

	valid := totp.Validate(code, vendedor.MFASecret)
	if !valid {
		return false, errors.New("el código de verificación es incorrecto")
	}

	// Actualizamos el estado de MFA en el objeto
	vendedor.MFAEnabled = true
	if err := d.LocalDB.Save(&vendedor).Error; err != nil {
		return false, errors.New("no se pudo habilitar MFA")
	}

	if d.isRemoteDBAvailable() {
		go d.syncVendedorToRemote(vendedor.ID)
	}

	return true, nil
}

func (d *Db) VerificarLoginMFA(tempToken string, code string) (LoginResponse, error) {
	var response LoginResponse
	claims := &Claims{}

	tkn, err := jwt.ParseWithClaims(tempToken, claims, func(token *jwt.Token) (interface{}, error) {
		return d.jwtKey, nil
	})
	if err != nil || !tkn.Valid || claims.MFAStep != "pending" {
		return response, errors.New("token temporal inválido o expirado")
	}

	var vendedor Vendedor
	if err := d.LocalDB.Where("email = ?", claims.Email).First(&vendedor).Error; err != nil {
		return response, errors.New("usuario no encontrado")
	}

	if !vendedor.MFAEnabled || vendedor.MFASecret == "" {
		return response, errors.New("MFA no está habilitado para este usuario")
	}
	valid := totp.Validate(code, vendedor.MFASecret)
	if !valid {
		return response, errors.New("código MFA incorrecto")
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	finalClaims := &Claims{
		UserID: vendedor.ID,
		Email:  vendedor.Email,
		Nombre: vendedor.Nombre,
		Cedula: vendedor.Cedula,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	finalToken := jwt.NewWithClaims(jwt.SigningMethodHS256, finalClaims)
	tokenString, err := finalToken.SignedString(d.jwtKey)
	if err != nil {
		return response, err
	}

	vendedor.Contrasena = ""
	response.Token = tokenString
	response.Vendedor = vendedor
	response.MFARequired = false

	return response, nil
}
