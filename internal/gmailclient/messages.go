// internal/gmailclient/messages.go
package gmailclient

import (
	"encoding/base64"
	"fmt"

	"google.golang.org/api/gmail/v1"
)

const user = "me"

type ListMessagesResponse struct {
	Messages      []*gmail.Message
	NextPageToken string
}

// ListMessages busca y retorna una lista de IDs de mensajes que coinciden con la query.
func ListMessagesWithPagination(srv *gmail.Service, query string, pageToken string) (*ListMessagesResponse, error) {
	req := srv.Users.Messages.List(user).Q(query).MaxResults(500)
	// Si hay un pageToken, lo aplica
	if pageToken != "" {
		req = req.PageToken(pageToken)
	}
	res, err := req.Do()
	if err != nil {
		return nil, fmt.Errorf("no se pudieron obtener los mensajes: %w", err)
	}
	return &ListMessagesResponse{
		Messages:      res.Messages,
		NextPageToken: res.NextPageToken,
	}, nil
}

// GetMessageFull obtiene el contenido completo de un mensaje.
func GetMessageFull(srv *gmail.Service, messageId string) (*gmail.Message, error) {
	return srv.Users.Messages.Get(user, messageId).Format("full").Do()
}

// GetMessageMetadata obtiene solo los metadatos de un mensaje (más rápido si no necesitas el cuerpo).
func GetMessageMetadata(srv *gmail.Service, messageId string) (*gmail.Message, error) {
	return srv.Users.Messages.Get(user, messageId).Format("metadata").Do()
}

// GetAttachment descarga y decodifica un archivo adjunto.
func GetAttachment(srv *gmail.Service, messageId string, attachmentId string) ([]byte, error) {
	attachment, err := srv.Users.Messages.Attachments.Get(user, messageId, attachmentId).Do()
	if err != nil {
		return nil, fmt.Errorf("error al descargar el adjunto: %w", err)
	}

	decodedData, err := base64.URLEncoding.DecodeString(attachment.Data)
	if err != nil {
		return nil, fmt.Errorf("error al decodificar el adjunto: %w", err)
	}
	return decodedData, nil
}
