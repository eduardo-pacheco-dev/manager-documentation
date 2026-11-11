package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) listNotifications(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	perPage := parsePositiveInt(r.URL.Query().Get("per_page"), 20)
	if perPage > 50 {
		perPage = 50
	}
	items, total, unread, err := s.Store.ListNotifications(user.ID, page, perPage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar as notificações.", nil)
		return
	}
	latest, _, _ := s.Store.LatestNotifications(user.ID, 6)
	writeJSON(w, http.StatusOK, map[string]any{
		"data":   items,
		"latest": latest,
		"unread": unread,
		"meta":   paginationMeta(page, perPage, total),
	})
}

func (s *Server) markNotificationRead(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id := chi.URLParam(r, "id")
	owned, err := s.Store.NotificationOwnedBy(user.ID, id)
	if err != nil || !owned {
		writeError(w, http.StatusNotFound, "Notificação não encontrada.", nil)
		return
	}
	if _, err := s.Store.MarkNotificationRead(user.ID, id); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível atualizar a notificação.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Notificação marcada como lida."})
}

func (s *Server) markAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if err := s.Store.MarkAllNotificationsRead(user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível atualizar as notificações.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Todas as notificações foram marcadas como lidas."})
}

func (s *Server) deleteNotification(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id := chi.URLParam(r, "id")
	owned, err := s.Store.NotificationOwnedBy(user.ID, id)
	if err != nil || !owned {
		writeError(w, http.StatusNotFound, "Notificação não encontrada.", nil)
		return
	}
	if err := s.Store.DeleteNotification(user.ID, id); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível remover a notificação.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Notificação removida."})
}

func (s *Server) clearNotifications(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if err := s.Store.ClearNotifications(user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível limpar as notificações.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Todas as notificações foram removidas."})
}

// notificationLevels is used by the frontend to validate icon choices. It also
// documents the accepted values in OpenAPI.
var notificationLevels = []string{"info", "success", "warning", "danger"}
