package httpapi

import (
	"errors"
	"net/http"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/upload"
)

func (s *Server) upload(w http.ResponseWriter, r *http.Request) {
	maxBytes := s.Uploads.MaxUploadMB() << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes+(1<<20))
	if err := r.ParseMultipartForm(maxBytes); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "O arquivo excede o limite de tamanho permitido.", nil)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeValidation(w, validationErrors{"file": "Selecione uma imagem para enviar."})
		return
	}
	defer file.Close()

	result, err := s.Uploads.Save(r.Context(), file, header)
	if err != nil {
		if errors.Is(err, upload.ErrInvalidImage) {
			writeValidation(w, validationErrors{"file": "Envie uma imagem JPG, PNG, GIF, WEBP ou BMP."})
			return
		}
		writeError(w, http.StatusInternalServerError, "Não foi possível armazenar a imagem.", nil)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}
