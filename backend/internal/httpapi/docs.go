package httpapi

import (
	"net/http"
	"strings"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/markdown"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/models"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/store"
)

type docPayload struct {
	Title       string `json:"title"`
	Category    string `json:"category"`
	Summary     string `json:"summary"`
	Content     string `json:"content"`
	IsPublished bool   `json:"is_published"`
}

type navigationGroup struct {
	Category string                 `json:"category"`
	Label    string                 `json:"label"`
	Items    []models.DevDocSummary `json:"items"`
}

func (s *Server) listDocs(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	category := query.Get("category")
	if category != "" && !models.ValidCategory(category) {
		category = ""
	}
	page := parsePositiveInt(query.Get("page"), 1)
	filter := store.DocFilter{
		Search:   strings.TrimSpace(query.Get("search")),
		Category: category,
		Page:     page,
	}
	docs, total, err := s.Store.ListDocs(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar a documentação.", nil)
		return
	}
	stats, err := s.Store.DocStats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar as estatísticas.", nil)
		return
	}
	navigation, err := s.docNavigation()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar a navegação.", nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":       docs,
		"categories": models.DevDocCategories,
		"navigation": navigation,
		"stats":      stats,
		"meta":       paginationMeta(page, 12, total),
		"filters":    map[string]any{"search": filter.Search, "category": category},
		"can_manage": currentUser(r).IsAdmin,
	})
}

func (s *Server) showDoc(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Documento não encontrado.", nil)
		return
	}
	doc, err := s.Store.FindDoc(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Documento não encontrado.", nil)
		return
	}
	navigation, _ := s.docNavigation()
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"document":   doc,
			"html":       renderDoc(doc.Content, doc.Title),
			"category":   map[string]string{"key": doc.Category, "label": models.CategoryLabel(doc.Category)},
			"navigation": navigation,
			"can_manage": currentUser(r).IsAdmin,
		},
	})
}

func (s *Server) createDoc(w http.ResponseWriter, r *http.Request) {
	var payload docPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	fields := validateDoc(payload)
	if !fields.empty() {
		writeValidation(w, fields)
		return
	}
	slug, err := s.Store.UniqueSlug(slugifyText(payload.Title))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível gerar o identificador do documento.", nil)
		return
	}
	user := currentUser(r)
	doc := models.DevDoc{
		UserID:      user.ID,
		Title:       strings.TrimSpace(payload.Title),
		Slug:        slug,
		Category:    payload.Category,
		Summary:     ptr(payload.Summary),
		Content:     payload.Content,
		IsPublished: payload.IsPublished,
	}
	if err := s.Store.CreateDoc(&doc); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível criar o documento.", nil)
		return
	}
	created, _ := s.Store.FindDoc(doc.ID)
	writeJSON(w, http.StatusCreated, map[string]any{
		"message": "Documento criado com sucesso.",
		"data":    map[string]any{"document": created},
	})
}

func (s *Server) updateDoc(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Documento não encontrado.", nil)
		return
	}
	doc, err := s.Store.FindDoc(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Documento não encontrado.", nil)
		return
	}
	var payload docPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	fields := validateDoc(payload)
	if !fields.empty() {
		writeValidation(w, fields)
		return
	}
	doc.Title = strings.TrimSpace(payload.Title)
	doc.Category = payload.Category
	doc.Summary = ptr(payload.Summary)
	doc.Content = payload.Content
	doc.IsPublished = payload.IsPublished
	if err := s.Store.UpdateDoc(&doc); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível atualizar o documento.", nil)
		return
	}
	updated, _ := s.Store.FindDoc(doc.ID)
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Documento atualizado com sucesso.",
		"data":    map[string]any{"document": updated},
	})
}

func (s *Server) deleteDoc(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Documento não encontrado.", nil)
		return
	}
	if _, err := s.Store.FindDoc(id); err != nil {
		writeError(w, http.StatusNotFound, "Documento não encontrado.", nil)
		return
	}
	if err := s.Store.DeleteDoc(id); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível excluir o documento.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Documento excluído com sucesso."})
}

func (s *Server) docNavigation() ([]navigationGroup, error) {
	docs, err := s.Store.DocNavigation()
	if err != nil {
		return nil, err
	}
	groups := make([]navigationGroup, 0, len(models.DevDocCategories))
	for _, category := range models.DevDocCategories {
		group := navigationGroup{Category: category.Key, Label: category.Label, Items: []models.DevDocSummary{}}
		for _, doc := range docs {
			if doc.Category == category.Key {
				group.Items = append(group.Items, doc)
			}
		}
		if len(group.Items) > 0 {
			groups = append(groups, group)
		}
	}
	return groups, nil
}

func renderDoc(content, title string) string {
	lines := strings.SplitN(content, "\n", 2)
	if len(lines) == 2 {
		first := strings.TrimSpace(lines[0])
		heading := strings.TrimSpace(strings.TrimPrefix(first, "#"))
		if strings.HasPrefix(first, "#") && strings.EqualFold(heading, strings.TrimSpace(title)) {
			content = lines[1]
		}
	}
	return markdown.Render(content)
}

func validateDoc(payload docPayload) validationErrors {
	fields := validationErrors{}
	if strings.TrimSpace(payload.Title) == "" {
		fields.add("title", "O título é obrigatório.")
	} else if len([]rune(payload.Title)) > 180 {
		fields.add("title", "O título deve ter no máximo 180 caracteres.")
	}
	if payload.Category == "" {
		fields.add("category", "Selecione uma categoria.")
	} else if !models.ValidCategory(payload.Category) {
		fields.add("category", "Categoria inválida.")
	}
	if len([]rune(payload.Summary)) > 300 {
		fields.add("summary", "O resumo deve ter no máximo 300 caracteres.")
	}
	if strings.TrimSpace(payload.Content) == "" {
		fields.add("content", "O conteúdo é obrigatório.")
	}
	return fields
}

func slugifyText(value string) string {
	replacer := strings.NewReplacer(
		"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
		"é", "e", "è", "e", "ê", "e", "ë", "e",
		"í", "i", "ì", "i", "î", "i", "ï", "i",
		"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
		"ú", "u", "ù", "u", "û", "u", "ü", "u",
		"ç", "c", "ñ", "n",
		"Á", "a", "À", "a", "Â", "a", "Ã", "a", "Ä", "a",
		"É", "e", "È", "e", "Ê", "e", "Ë", "e",
		"Í", "i", "Ì", "i", "Î", "i", "Ï", "i",
		"Ó", "o", "Ò", "o", "Ô", "o", "Õ", "o", "Ö", "o",
		"Ú", "u", "Ù", "u", "Û", "u", "Ü", "u",
		"Ç", "c", "Ñ", "n",
	)
	value = strings.ToLower(replacer.Replace(value))
	var builder strings.Builder
	previousDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
			previousDash = false
		default:
			if !previousDash && builder.Len() > 0 {
				builder.WriteByte('-')
				previousDash = true
			}
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "documento"
	}
	return result
}
