// gmailclient/client.go
package gmailclient

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

const tokenFile = "token.json"

// NewService crea y retorna un nuevo servicio de Gmail autenticado.
func NewService(credentialsFile string) (*gmail.Service, error) {
	ctx := context.Background()

	b, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer el archivo de credenciales '%s': %w", credentialsFile, err)
	}

	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("no se pudo parsear el archivo de secretos del cliente: %w", err)
	}

	client := getClient(config)
	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("no se pudo crear el cliente de Gmail: %w", err)
	}

	log.Println("✅ Servicio de Gmail autenticado correctamente.")
	return srv, nil
}

// getClient obtiene un cliente, refrescando y guardando el token si es necesario.
func getClient(config *oauth2.Config) *http.Client {
	tok, err := tokenFromFile(tokenFile)
	if err != nil {
		log.Println("No se encontró un token local, obteniendo uno nuevo desde la web...")
		tok = getTokenFromWeb(config)
		saveToken(tokenFile, tok)
	}

	tokenSource := config.TokenSource(context.Background(), tok)

	// Envolvemos el TokenSource original para manejar la expiración y guardar el token al refrescar.
	savingTokenSource := &savingTokenSource{
		source:    tokenSource,
		tokenPath: tokenFile,
	}

	return oauth2.NewClient(context.Background(), savingTokenSource)
}

// getTokenFromWeb solicita un nuevo token al usuario a través de la web.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	fmt.Printf("Ve a la siguiente URL en tu navegador y autoriza la aplicación: \n%v\n", authURL)
	fmt.Printf("Luego, ingresa el código de autorización que recibas: ")

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("No se pudo leer el código de autorización: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("No se pudo obtener el token desde la web: %v", err)
	}
	return tok
}

// tokenFromFile lee un token desde un archivo.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// saveToken guarda un token en un archivo.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Guardando el token de credenciales en: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("No se pudo guardar el token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// savingTokenSource es un wrapper para oauth2.TokenSource que guarda el token
// cada vez que se actualiza y maneja la expiración del mismo.
type savingTokenSource struct {
	source    oauth2.TokenSource
	tokenPath string
	mu        sync.Mutex
}

// Token es parte de la interfaz oauth2.TokenSource.
func (s *savingTokenSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Obtiene el token de la fuente subyacente.
	t, err := s.source.Token()
	if err != nil {
		// Si el error indica que el token es inválido o ha sido revocado, lo borramos.
		if strings.Contains(err.Error(), "invalid_grant") {
			log.Println("El token de acceso ha expirado o ha sido revocado.")
			log.Printf("Eliminando '%s'...", s.tokenPath)
			if err := os.Remove(s.tokenPath); err != nil {
				log.Printf("ADVERTENCIA: No se pudo eliminar el archivo de token: %v", err)
			}
			log.Fatal("Por favor, ejecuta el comando de nuevo para obtener un nuevo token de autenticación.")
		}
		return nil, err
	}

	// Compara con el token existente en disco para ver si ha sido refrescado.
	existingToken, _ := tokenFromFile(s.tokenPath)
	if existingToken == nil || t.AccessToken != existingToken.AccessToken {
		saveToken(s.tokenPath, t)
	}

	return t, nil
}
