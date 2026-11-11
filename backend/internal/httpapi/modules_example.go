package httpapi

import "net/http"

func (s *Server) exampleIndex(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"title":   "Módulo de exemplo",
			"message": "O plugin Example foi convertido em um módulo nativo do sistema.",
			"version": "1.0.0",
			"features": []string{
				"Rota autenticada /example",
				"Widget de status no dashboard",
				"Item de navegação no menu Módulos",
			},
		},
	})
}
