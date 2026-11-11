package httpapi

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/widgets"
)

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	available, err := widgets.Available(s.Store, user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar o dashboard.", nil)
		return
	}
	ordered := widgets.ApplyOrder(available, user.DashboardOrder.OrEmpty())
	spans := user.DashboardSpans.OrEmpty()

	now := time.Now()
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"greeting":      widgets.Greeting(user),
			"date":          now.Format(time.RFC3339),
			"widgets":       ordered,
			"hidden":        user.DashboardHidden.OrEmpty(),
			"order":         widgetIDs(ordered),
			"spans":         spans,
			"editing":       false,
			"server_time":   now.Format(time.RFC3339),
			"default_order": widgetIDs(ordered),
		},
	})
}

func widgetIDs(items []widgets.Widget) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.ID)
	}
	return result
}

func (s *Server) updateWidgetVisibility(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Hidden *bool `json:"hidden"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if payload.Hidden == nil {
		writeValidation(w, validationErrors{"hidden": "Informe se o widget deve ser ocultado."})
		return
	}
	user := currentUser(r)
	available, err := widgets.Available(s.Store, user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar os widgets.", nil)
		return
	}
	widget, ok := widgets.Find(chi.URLParam(r, "id"), available)
	if !ok {
		writeError(w, http.StatusNotFound, "Widget não encontrado.", nil)
		return
	}

	hidden := user.DashboardHidden.OrEmpty()
	result := make([]string, 0, len(hidden)+1)
	for _, id := range hidden {
		if id != widget.ID {
			result = append(result, id)
		}
	}
	if *payload.Hidden {
		result = append(result, widget.ID)
	}
	user.DashboardHidden = modelsStringList(result)
	if err := s.Store.UpdateUserPreferences(&user); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível atualizar o widget.", nil)
		return
	}

	message := "Widget adicionado ao painel."
	if *payload.Hidden {
		message = "Widget removido do painel."
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": message,
		"data":    map[string]any{"hidden": result},
	})
}

func (s *Server) updateWidgetOrder(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Order []string `json:"order"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	user := currentUser(r)
	available, err := widgets.Available(s.Store, user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar os widgets.", nil)
		return
	}
	order := widgets.NormalizeOrder(available, payload.Order)
	user.DashboardOrder = modelsStringList(order)
	if err := s.Store.UpdateUserPreferences(&user); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível salvar a ordem dos widgets.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Ordem dos widgets atualizada.",
		"data":    map[string]any{"order": order},
	})
}

func (s *Server) updateWidgetLayout(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Span *int `json:"span"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if payload.Span != nil && (*payload.Span < 1 || *payload.Span > 3) {
		writeValidation(w, validationErrors{"span": "A largura deve ser um número entre 1 e 3 colunas."})
		return
	}
	user := currentUser(r)
	available, err := widgets.Available(s.Store, user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar os widgets.", nil)
		return
	}
	widget, ok := widgets.Find(chi.URLParam(r, "id"), available)
	if !ok {
		writeError(w, http.StatusNotFound, "Widget não encontrado.", nil)
		return
	}

	spans := user.DashboardSpans.OrEmpty()
	message := "Widget atualizado."
	if payload.Span != nil {
		if *payload.Span == widget.DefaultSpan {
			delete(spans, widget.ID)
		} else {
			spans[widget.ID] = *payload.Span
		}
		message = "Largura do widget atualizada."
	}
	user.DashboardSpans = modelsIntMap(spans)
	if err := s.Store.UpdateUserPreferences(&user); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível atualizar o widget.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": message,
		"data":    map[string]any{"spans": spans},
	})
}
