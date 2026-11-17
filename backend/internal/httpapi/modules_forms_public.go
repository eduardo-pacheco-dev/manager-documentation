package httpapi

import (
	"context"
	"fmt"
	"html/template"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/modules"
)

const (
	formMaxImages        = 5
	formMaxImageBytes    = 15 << 20
	formMaxFiles         = 5
	formMaxFileBytes     = 30 << 20
	formDraftCookieAge   = 30 * 24 * 60 * 60
	formPublicBodyLimit  = 200 << 20
	formPublicUploadHint = "Até 5 imagens por campo, com no máximo 15 MB por imagem."
)

type formPublicField struct {
	modules.FormField
	ScaleValues   []int
	Selected      map[string]bool
	NotApplicable bool
	Error         string
}

type formPublicStep struct {
	ID     string
	Title  string
	Fields []formPublicField
}

type formPublicData struct {
	Title          string
	Form           modules.Form
	Steps          []formPublicStep
	TotalQuestions int
	AnsweredCount  int
	Percent        int
	Submitted      bool
	Message        string
	ErrorMessage   string
	FieldErrors    map[string]string
	SubmitURL      string
	AutosaveURL    string
	ImageDeleteURL string
	FileDeleteURL  string
	PublicLink     string
	HasStructure   bool
	MaxImageHint   string
	MaxFileHint    string
}

var formPublicTemplate = template.Must(template.New("form-public").Funcs(template.FuncMap{
	"has": func(values map[string]bool, value string) bool {
		return values[value]
	},
	"base": func(value string) string {
		parsed, err := url.Parse(value)
		if err != nil {
			return value
		}
		return path.Base(parsed.Path)
	},
	"inc": func(value int) int { return value + 1 },
}).Parse(`<!doctype html>
<html lang="pt-BR">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="theme-color" content="#161616">
<meta name="robots" content="noindex">
<title>{{.Title}}</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=IBM+Plex+Sans:wght@400;500;600&display=swap" rel="stylesheet">
<style>
:root{--bg:#f4f4f4;--layer:#fff;--text:#161616;--muted:#525252;--helper:#6f6f6f;--border:#e0e0e0;--blue:#0f62fe;--blue-hover:#0043ce;--danger:#da1e28;--danger-bg:#fff1f1;--success:#0e6027;--success-bg:#defbe6}
*{box-sizing:border-box}
body{margin:0;background:var(--bg);color:var(--text);font-family:"IBM Plex Sans","Segoe UI",system-ui,sans-serif;line-height:1.5;-webkit-font-smoothing:antialiased}
.header{display:flex;align-items:center;justify-content:space-between;height:3rem;padding:0 1rem;background:#161616;color:#fff}
.brand{display:flex;align-items:center;gap:.75rem;font-size:.875rem;font-weight:600}
.mark{display:flex;align-items:center;justify-content:center;width:1.5rem;height:1.5rem;background:#fff;color:#161616;font-size:.75rem;font-weight:600}
.header span.meta{color:#c6c6c6;font-size:.75rem}
.shell{max-width:58rem;margin:0 auto;padding:2rem 1rem 4rem}
.eyebrow{margin:0 0 .35rem;color:var(--muted);font-size:.75rem;font-weight:500;letter-spacing:.08em;text-transform:uppercase}
h1{margin:0 0 .5rem;font-size:clamp(1.5rem,3.4vw,2.2rem);font-weight:400;line-height:1.2}
h2{margin:0;font-size:1.05rem;font-weight:500}
.lead{max-width:52rem;margin:0 0 1rem;color:var(--muted);font-size:.9rem}
.tags{display:flex;flex-wrap:wrap;gap:.5rem;margin:0 0 1rem}
.tag{display:inline-flex;align-items:center;min-height:1.5rem;padding:.15rem .55rem;border-radius:999px;background:#e0e0e0;color:#161616;font-size:.75rem}
.tag--blue{background:#d0e2ff;color:#002d9c}
.progress{display:flex;align-items:center;gap:.75rem;margin:0 0 1.5rem;color:var(--helper);font-size:.75rem}
.bar{flex:1;height:.35rem;background:#e0e0e0;overflow:hidden}
.bar span{display:block;height:100%;background:var(--blue)}
.panel{background:var(--layer);border:1px solid var(--border);padding:1.5rem;margin-bottom:1rem}
.section-head{display:flex;align-items:center;justify-content:space-between;gap:1rem;border-bottom:1px solid var(--border);padding-bottom:.85rem;margin-bottom:1rem}
.count{color:var(--helper);font-size:.75rem}
.alert{padding:1rem;margin-bottom:1rem;border-left:3px solid var(--success);background:var(--success-bg);color:var(--success);font-size:.85rem}
.alert--error{border-color:var(--danger);background:var(--danger-bg);color:#a2191f}
.field{padding:1rem 0;border-bottom:1px solid #f4f4f4}
.field:last-child{border-bottom:0;padding-bottom:0}
.label{display:block;margin:0 0 .1rem;font-size:.87rem;font-weight:500;color:var(--text)}
.required{color:var(--danger)}
.help{margin:0 0 .6rem;color:var(--helper);font-size:.75rem}
input[type=text],input[type=number],input[type=date],input[type=time],textarea,select{width:100%;padding:.7rem;border:0;border-bottom:1px solid #8d8d8d;background:#f4f4f4;color:var(--text);font-family:inherit;font-size:.87rem}
textarea{min-height:6rem;resize:vertical}
input:focus,textarea:focus,select:focus{outline:2px solid var(--blue);outline-offset:-2px}
.options{display:flex;flex-direction:column;gap:.45rem}
.options label{display:flex;align-items:flex-start;gap:.55rem;font-size:.85rem;color:var(--text);cursor:pointer}
.options input{margin-top:.2rem}
.scale{display:flex;flex-wrap:wrap;gap:.35rem}
.scale label{flex:1;min-width:2.6rem;display:flex;flex-direction:column;align-items:center;gap:.25rem;padding:.5rem .25rem;border:1px solid var(--border);font-size:.72rem;color:var(--muted);cursor:pointer;text-align:center}
.scale input{margin:0}
.upload-hint{color:var(--helper);font-size:.72rem;margin:.35rem 0 .5rem}
.images{display:flex;flex-wrap:wrap;gap:.6rem;margin:.5rem 0}
.image-card{position:relative;border:1px solid var(--border);background:#fff}
.image-card img{display:block;width:8rem;height:6rem;object-fit:cover}
.image-card button{position:absolute;top:.25rem;right:.25rem;border:0;background:#161616cc;color:#fff;font-size:.7rem;padding:.25rem .45rem;cursor:pointer}
.files{display:flex;flex-direction:column;gap:.4rem;margin:.5rem 0}
.file-row{display:flex;align-items:center;gap:.6rem;padding:.55rem .7rem;border:1px solid var(--border);font-size:.8rem}
.file-row a{color:var(--blue);text-decoration:none;flex:1;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.file-row button{border:0;background:none;color:var(--danger);font-size:.72rem;cursor:pointer}
.na{display:inline-flex;align-items:center;gap:.4rem;margin-top:.5rem;font-size:.78rem;color:var(--muted)}
.actions{display:flex;align-items:center;justify-content:space-between;gap:1rem;margin-top:1.5rem;flex-wrap:wrap}
.wizard-actions{display:flex;align-items:center;gap:.5rem;flex-wrap:wrap}
.btn--ghost{background:transparent;color:var(--blue);border-color:var(--blue)}
.btn--ghost:hover{background:#edf5ff;color:var(--blue-hover)}
.step-label{color:var(--helper);font-size:.75rem}
.form-step[hidden]{display:none}
.btn{display:inline-flex;align-items:center;justify-content:center;min-height:3rem;padding:0 1.25rem;border:1px solid transparent;background:var(--blue);color:#fff;font-family:inherit;font-size:.875rem;cursor:pointer;text-decoration:none}
.btn:hover{background:var(--blue-hover)}
.save-state{color:var(--helper);font-size:.75rem}
.field-error{margin:.35rem 0 0;color:var(--danger);font-size:.75rem}
.empty{color:var(--muted);font-size:.85rem;padding:1rem;border:1px dashed #c6c6c6;text-align:center}
.footer{max-width:58rem;margin:0 auto;padding:0 1rem 2rem;color:var(--helper);font-size:.72rem;display:flex;flex-wrap:wrap;gap:.5rem;justify-content:space-between}
@media (max-width:42rem){.shell{padding-top:1rem}.panel{padding:1rem}.header span.meta{display:none}.actions .btn{width:100%}}
</style>
</head>
<body>
<header class="header">
  <div class="brand"><span class="mark">DC</span> Documentation Center</div>
  <span class="meta">Formulário externo</span>
</header>
<main class="shell">
  <p class="eyebrow">Formulário externo</p>
  <h1>{{.Form.Code}} · {{.Form.Name}}</h1>
  <p class="lead">{{if .Form.ProjectName}}[{{.Form.ProjectCode}}] {{.Form.ProjectName}} · {{end}}{{.Form.Type}}{{if .Form.AllowsMultipleResponses}} · Múltiplas respostas{{else}} · Resposta única{{end}}</p>
  <div class="tags">
    <span class="tag tag--blue">Status: {{.Form.Status}}</span>
    <span class="tag">{{.TotalQuestions}} pergunta(s)</span>
    <span class="tag">{{.AnsweredCount}} respondida(s)</span>
  </div>
  <div class="progress"><span>{{.Percent}}% preenchido</span><div class="bar"><span style="width: {{.Percent}}%"></span></div></div>

  {{if .Message}}<div class="alert">{{.Message}}</div>{{end}}
  {{if .ErrorMessage}}<div class="alert alert--error">{{.ErrorMessage}}</div>{{end}}

  {{if not .HasStructure}}
    <div class="empty">Este formulário ainda não possui perguntas.</div>
  {{else}}
  <form method="post" enctype="multipart/form-data" action="{{.SubmitURL}}" id="public-form">
    {{range $si, $step := .Steps}}
    <section class="panel form-step" data-form-section data-index="{{$si}}">
      <div class="section-head">
        <h2>{{if $step.Title}}{{$step.Title}}{{else}}Seção {{inc $si}}{{end}}</h2>
        <span class="count">{{len $step.Fields}} pergunta(s)</span>
      </div>
      {{range $step.Fields}}
      {{$field := .}}
      <div class="field">
        <label class="label" id="field-{{$field.ID}}-label" for="field-{{$field.ID}}">{{$field.Label}}{{if $field.Required}} <span class="required">*</span>{{end}}</label>
        {{if $field.Help}}<p class="help">{{$field.Help}}</p>{{end}}

        {{if eq $field.Type "textarea"}}
          <textarea id="field-{{$field.ID}}" name="answers[{{$field.ID}}]" {{if $field.Required}}required{{end}}>{{$field.Value}}</textarea>
        {{else if eq $field.Type "number"}}
          <input type="number" step="any" id="field-{{$field.ID}}" name="answers[{{$field.ID}}]" value="{{$field.Value}}" {{if $field.Required}}required{{end}}>
        {{else if eq $field.Type "date"}}
          <input type="date" id="field-{{$field.ID}}" name="answers[{{$field.ID}}]" value="{{$field.Value}}" {{if $field.Required}}required{{end}}>
        {{else if eq $field.Type "time"}}
          <input type="time" id="field-{{$field.ID}}" name="answers[{{$field.ID}}]" value="{{$field.Value}}" {{if $field.Required}}required{{end}}>
        {{else if eq $field.Type "radio"}}
          <div class="options" role="radiogroup" aria-labelledby="field-{{$field.ID}}-label">
            {{range $field.Options}}
            <label><input type="radio" name="answers[{{$field.ID}}]" value="{{.}}" {{if has $field.Selected .}}checked{{end}} {{if $field.Required}}required{{end}}> <span>{{.}}</span></label>
            {{end}}
          </div>
        {{else if eq $field.Type "checkbox"}}
          <div class="options">
            {{range $field.Options}}
            <label><input type="checkbox" name="answers[{{$field.ID}}][]" value="{{.}}" {{if has $field.Selected .}}checked{{end}}> <span>{{.}}</span></label>
            {{end}}
          </div>
        {{else if eq $field.Type "dropdown"}}
          <select id="field-{{$field.ID}}" name="answers[{{$field.ID}}]" {{if $field.Required}}required{{end}}>
            <option value="">Selecione…</option>
            {{range $field.Options}}<option value="{{.}}" {{if has $field.Selected .}}selected{{end}}>{{.}}</option>{{end}}
          </select>
        {{else if eq $field.Type "scale"}}
          <div class="scale">
            {{range $field.ScaleValues}}
            <label><input type="radio" name="answers[{{$field.ID}}]" value="{{.}}" {{if eq $field.Value (printf "%d" .)}}checked{{end}} {{if $field.Required}}required{{end}}> <span>{{.}}</span></label>
            {{end}}
          </div>
          {{if or $field.ScaleMinLabel $field.ScaleMaxLabel}}<p class="upload-hint">{{$field.ScaleMinLabel}}{{if and $field.ScaleMinLabel $field.ScaleMaxLabel}} · {{end}}{{$field.ScaleMaxLabel}}</p>{{end}}
        {{else if eq $field.Type "image"}}
          <p class="upload-hint">{{$.MaxImageHint}}</p>
          {{if $field.Images}}
          <div class="images">
            {{range $index, $image := $field.Images}}
            <div class="image-card">
              <a href="{{$image}}" target="_blank" rel="noopener"><img src="{{$image}}" alt="{{$field.Label}}"></a>
              <button type="button" data-remove-image data-field="{{$field.ID}}" data-index="{{$index}}">Excluir</button>
            </div>
            {{end}}
          </div>
          {{end}}
          <input type="file" id="field-{{$field.ID}}" name="answers[{{$field.ID}}][]" accept="image/*" capture="environment" multiple>
          <label class="na"><input type="checkbox" name="na_answers[{{$field.ID}}]" value="1" {{if $field.NotApplicable}}checked{{end}}> Não aplicável</label>
        {{else if eq $field.Type "file"}}
          <p class="upload-hint">{{$.MaxFileHint}}</p>
          {{if $field.Files}}
          <div class="files">
            {{range $index, $file := $field.Files}}
            <div class="file-row"><a href="{{$file}}" target="_blank" rel="noopener">{{index $field.FileNames $index}}</a><button type="button" data-remove-file data-field="{{$field.ID}}" data-index="{{$index}}">Excluir</button></div>
            {{end}}
          </div>
          {{end}}
          <input type="file" id="field-{{$field.ID}}" name="answers[{{$field.ID}}][]" multiple>
        {{else}}
          <input type="text" id="field-{{$field.ID}}" name="answers[{{$field.ID}}]" value="{{$field.Value}}" {{if $field.Required}}required{{end}}>
        {{end}}

        {{if $field.Error}}<p class="field-error">{{$field.Error}}</p>{{end}}
      </div>
      {{end}}
    </section>
    {{end}}

    <div class="actions">
      <span class="save-state" id="save-state"></span>
      <div class="wizard-actions">
        <button class="btn btn--ghost" type="button" id="wizard-prev">Anterior</button>
        <span class="step-label" id="wizard-label"></span>
        <button class="btn" type="button" id="wizard-next">Próximo</button>
        <button class="btn" type="submit" id="wizard-submit">Enviar respostas</button>
      </div>
    </div>
  </form>
  {{end}}
</main>
<footer class="footer"><span>Documentation Center · Formulários</span><span>Powered by IBM Carbon Design System</span></footer>
<script>
(function () {
  const form = document.getElementById('public-form');
  if (!form) return;
  const state = document.getElementById('save-state');
  const autosaveUrl = {{.AutosaveURL}};
  const imageDeleteUrl = {{.ImageDeleteURL}};
  const fileDeleteUrl = {{.FileDeleteURL}};

  const sections = [...form.querySelectorAll('[data-form-section]')];
  const prevButton = document.getElementById('wizard-prev');
  const nextButton = document.getElementById('wizard-next');
  const submitButton = document.getElementById('wizard-submit');
  const wizardLabel = document.getElementById('wizard-label');
  const storageKey = 'public-form-step-' + window.location.pathname;
  let currentSection = 0;

  const showSection = (index) => {
    if (!sections.length) return;
    currentSection = Math.max(0, Math.min(sections.length - 1, index));
    sections.forEach((section, position) => {
      section.hidden = position !== currentSection;
    });
    if (prevButton) prevButton.disabled = currentSection === 0;
    if (nextButton) nextButton.hidden = currentSection === sections.length - 1;
    if (submitButton) submitButton.hidden = currentSection !== sections.length - 1;
    if (wizardLabel) wizardLabel.textContent = 'Etapa ' + (currentSection + 1) + ' de ' + sections.length;
    try {
      sessionStorage.setItem(storageKey, String(currentSection));
    } catch (error) {}
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  if (sections.length > 1) {
    let initial = 0;
    try {
      initial = Number(sessionStorage.getItem(storageKey) || '0') || 0;
    } catch (error) {}
    const errorField = form.querySelector('.field-error');
    if (errorField) {
      const section = errorField.closest('[data-form-section]');
      if (section) initial = sections.indexOf(section);
    }
    showSection(initial);
    prevButton.addEventListener('click', () => showSection(currentSection - 1));
    nextButton.addEventListener('click', () => showSection(currentSection + 1));
    submitButton.addEventListener('click', (event) => {
      sections.forEach((section) => { section.hidden = false; });
      if (!form.checkValidity()) {
        event.preventDefault();
        const invalid = form.querySelector(':invalid');
        if (invalid) {
          const section = invalid.closest('[data-form-section]');
          if (section) showSection(sections.indexOf(section));
          invalid.reportValidity();
        }
      }
    });
    form.addEventListener('keydown', (event) => {
      if (event.key === 'Enter' && event.target.tagName !== 'TEXTAREA' && currentSection < sections.length - 1) {
        event.preventDefault();
        showSection(currentSection + 1);
      }
    });
  } else {
    if (prevButton) prevButton.hidden = true;
    if (nextButton) nextButton.hidden = true;
    if (wizardLabel) wizardLabel.textContent = '';
  }

  const autosave = async () => {
    if (!autosaveUrl) return;
    state.textContent = 'Salvando…';
    try {
      const response = await fetch(autosaveUrl, { method: 'POST', body: new FormData(form), credentials: 'same-origin' });
      if (!response.ok) throw new Error('save failed');
      const data = await response.json();
      state.textContent = 'Salvo automaticamente.';
      if ((data.images_saved || 0) > 0 || (data.files_saved || 0) > 0 || data.reload) {
        window.location.reload();
      }
    } catch (error) {
      state.textContent = 'Não foi possível salvar automaticamente.';
    }
  };

  form.querySelectorAll('input[type=file], input[name^="na_answers"]').forEach((input) => {
    input.addEventListener('change', autosave);
  });

  const remove = async (url, button, type) => {
    const field = button.dataset.field;
    const index = Number(button.dataset.index || '0');
    if (!field) return;
    button.disabled = true;
    try {
      const response = await fetch(url, {
        method: 'POST',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ field: field, index: index }),
      });
      if (!response.ok) throw new Error('remove failed');
      const card = button.closest(type === 'image' ? '.image-card' : '.file-row');
      if (card) card.remove();
      state.textContent = type === 'image' ? 'Imagem excluída.' : 'Arquivo excluído.';
    } catch (error) {
      button.disabled = false;
      state.textContent = 'Não foi possível excluir o arquivo.';
    }
  };

  form.querySelectorAll('[data-remove-image]').forEach((button) => {
    button.addEventListener('click', () => remove(imageDeleteUrl, button, 'image'));
  });
  form.querySelectorAll('[data-remove-file]').forEach((button) => {
    button.addEventListener('click', () => remove(fileDeleteUrl, button, 'file'));
  });
})();
</script>
</body>
</html>`))

func formUnavailablePage(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusGone)
	_, _ = w.Write([]byte(`<!doctype html><html lang="pt-BR"><meta charset="utf-8"><title>Formulário indisponível</title>
<body style="font-family:system-ui,sans-serif;background:#f4f4f4;color:#161616;padding:3rem 1rem">
<main style="max-width:640px;margin:0 auto;background:#fff;border:1px solid #e0e0e0;padding:2rem">
<h1 style="font-weight:400">Formulário indisponível</h1><p style="color:#525252">Este link expirou, foi desativado ou não existe.</p></main></body></html>`))
}

func (s *Server) formPublicShow(w http.ResponseWriter, r *http.Request) {
	form, err := s.Modules.FindFormByToken(chi.URLParam(r, "token"))
	if err != nil {
		formUnavailablePage(w)
		return
	}
	response := s.formPublicResponse(form, r)
	steps := s.formPublicSteps(form, response)
	data := s.formPublicData(form, steps)
	data.Submitted = r.URL.Query().Get("submitted") == "1"
	if data.Submitted {
		data.Message = "Resposta enviada com sucesso."
	}
	if message := r.URL.Query().Get("error"); message != "" {
		data.ErrorMessage = message
	}
	s.renderFormPublic(w, http.StatusOK, data)
}

func (s *Server) formPublicSubmit(w http.ResponseWriter, r *http.Request) {
	form, err := s.Modules.FindFormByToken(chi.URLParam(r, "token"))
	if err != nil {
		formUnavailablePage(w)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, formPublicBodyLimit)
	if err := parseFormPublicRequest(r); err != nil {
		http.Redirect(w, r, "/f/"+form.PublicToken+"?error="+url.QueryEscape("Arquivo inválido ou maior que o permitido."), http.StatusSeeOther)
		return
	}
	response := s.formPublicResponse(form, r)
	steps := s.formPublicSteps(form, response)
	steps, captured, uploadErrors := s.captureFormUploads(r.Context(), form, steps, r)
	answers := formAnswersFromRequest(r)
	notApplicable := formNotApplicableFromRequest(r)
	steps = modules.ApplyAnswers(steps, answers, notApplicable)

	fieldErrors := uploadErrors
	for _, step := range steps {
		for _, field := range step.Fields {
			if _, hasUploadError := fieldErrors[field.ID]; hasUploadError {
				continue
			}
			if err := validatePublicFormField(field, notApplicable[field.ID]); err != "" {
				fieldErrors[field.ID] = err
			}
		}
	}

	if len(fieldErrors) > 0 {
		if captured > 0 {
			s.persistFormDraft(w, form, response, steps, r)
		}
		data := s.formPublicData(form, steps)
		data.FieldErrors = fieldErrors
		data.ErrorMessage = "Não foi possível enviar as respostas. Corrija os itens destacados e tente novamente. As imagens e os arquivos já enviados foram mantidos."
		s.renderFormPublic(w, http.StatusUnprocessableEntity, data)
		return
	}

	status := modules.FormResultStatus(form.Status)
	now := time.Now().UTC()
	var target *modules.FormResponse
	if response != nil {
		target = response
	}
	if target != nil {
		if err := s.Modules.UpdateFormResponse(form.ID, target.ID, status, steps, &now); err != nil {
			writeError(w, http.StatusInternalServerError, "Não foi possível salvar a resposta.", nil)
			return
		}
	} else {
		var userID *int64
		user := currentUser(r)
		if user.ID > 0 {
			userID = &user.ID
		}
		if _, err := s.Modules.CreateFormResponse(form.ID, modules.FormResponseInput{
			UserID: userID, Status: status, Steps: steps, SubmittedAt: &now,
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "Não foi possível salvar a resposta.", nil)
			return
		}
	}
	_ = s.Modules.UpdateFormStatusAndSteps(form.ID, status, steps)
	s.clearFormDraftCookie(w, r, form)
	http.Redirect(w, r, "/f/"+form.PublicToken+"?submitted=1", http.StatusSeeOther)
}

func (s *Server) formPublicAutosave(w http.ResponseWriter, r *http.Request) {
	form, err := s.Modules.FindFormByToken(chi.URLParam(r, "token"))
	if err != nil {
		writeError(w, http.StatusNotFound, "Formulário não encontrado.", nil)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, formPublicBodyLimit)
	if err := parseFormPublicRequest(r); err != nil {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"message": "Arquivo inválido ou maior que o permitido."})
		return
	}
	response := s.formPublicResponse(form, r)
	steps := s.formPublicSteps(form, response)
	steps, captured, uploadErrors := s.captureFormUploads(r.Context(), form, steps, r)
	if len(uploadErrors) > 0 {
		writeValidation(w, uploadErrors)
		return
	}
	answers := formAnswersFromRequest(r)
	steps = modules.ApplyAnswers(steps, answers, formNotApplicableFromRequest(r))
	draft := s.persistFormDraft(w, form, response, steps, r)
	responseID := int64(0)
	if draft != nil {
		responseID = draft.ID
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message":      "Rascunho salvo automaticamente.",
		"images_saved": captured,
		"response_id":  responseID,
		"saved_at":     time.Now().Format(time.RFC3339),
		"reload":       captured > 0,
	})
}

func (s *Server) formPublicDeleteImage(w http.ResponseWriter, r *http.Request) {
	s.formPublicDeleteUpload(w, r, true)
}

func (s *Server) formPublicDeleteFile(w http.ResponseWriter, r *http.Request) {
	s.formPublicDeleteUpload(w, r, false)
}

func (s *Server) formPublicDeleteUpload(w http.ResponseWriter, r *http.Request, image bool) {
	form, err := s.Modules.FindFormByToken(chi.URLParam(r, "token"))
	if err != nil {
		writeError(w, http.StatusNotFound, "Formulário não encontrado.", nil)
		return
	}
	var payload struct {
		Field string `json:"field"`
		Index int    `json:"index"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	response := s.formPublicResponse(form, r)
	if response == nil {
		writeError(w, http.StatusNotFound, "Nenhum arquivo encontrado para excluir.", nil)
		return
	}
	steps := modules.DecodeSteps(response.Steps)
	removed := ""
	found := false
	for stepIndex, step := range steps {
		for fieldIndex, field := range step.Fields {
			if field.ID != payload.Field || field.Type != map[bool]string{true: "image", false: "file"}[image] {
				continue
			}
			if image {
				if payload.Index < 0 || payload.Index >= len(field.Images) {
					continue
				}
				removed = field.Images[payload.Index]
				field.Images = append(append([]string{}, field.Images[:payload.Index]...), field.Images[payload.Index+1:]...)
				if payload.Index < len(field.ImageThumbnails) {
					field.ImageThumbnails = append(append([]string{}, field.ImageThumbnails[:payload.Index]...), field.ImageThumbnails[payload.Index+1:]...)
				}
				if field.Status != "nao_aplicavel" {
					if len(field.Images) == 0 {
						field.Status = "pendente"
					} else {
						field.Status = "enviado"
					}
				}
			} else {
				if payload.Index < 0 || payload.Index >= len(field.Files) {
					continue
				}
				removed = field.Files[payload.Index]
				field.Files = append(append([]string{}, field.Files[:payload.Index]...), field.Files[payload.Index+1:]...)
				if payload.Index < len(field.FileNames) {
					field.FileNames = append(append([]string{}, field.FileNames[:payload.Index]...), field.FileNames[payload.Index+1:]...)
				}
				if len(field.Files) == 0 {
					field.Status = "pendente"
				} else {
					field.Status = "enviado"
				}
			}
			steps[stepIndex].Fields[fieldIndex] = field
			found = true
			break
		}
		if found {
			break
		}
	}
	if !found {
		writeError(w, http.StatusNotFound, "Arquivo não encontrado.", nil)
		return
	}
	if err := s.Modules.UpdateFormResponse(form.ID, response.ID, response.Status, steps, response.SubmittedAt); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível excluir o arquivo.", nil)
		return
	}
	if removed != "" {
		if parsed, err := url.Parse(removed); err == nil {
			cleaned := strings.TrimPrefix(parsed.Path, "/")
			if strings.HasPrefix(cleaned, "uploads/") {
				s.Uploads.DeleteLocal(cleaned)
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Arquivo excluído."})
}

func (s *Server) formPublicResponse(form modules.Form, r *http.Request) *modules.FormResponse {
	if cookie, err := r.Cookie(s.formDraftCookieName(form)); err == nil {
		if id, parseErr := strconv.ParseInt(cookie.Value, 10, 64); parseErr == nil && id > 0 {
			if response, findErr := s.Modules.FindFormResponseByID(form.ID, id); findErr == nil && response != nil && response.SubmittedAt == nil {
				return response
			}
		}
	}
	if form.AllowsMultipleResponses {
		return nil
	}
	response, _ := s.Modules.LatestFormResponse(form.ID)
	return response
}

func (s *Server) formPublicSteps(form modules.Form, response *modules.FormResponse) []modules.FormStep {
	steps := modules.BlankSteps(modules.DecodeSteps(form.Steps))
	if response != nil {
		steps = modules.MergeSteps(steps, modules.DecodeSteps(response.Steps))
	}
	return steps
}

func (s *Server) formPublicData(form modules.Form, steps []modules.FormStep) formPublicData {
	data := formPublicData{
		Title:          form.Code + " · " + form.Name,
		Form:           form,
		SubmitURL:      "/f/" + form.PublicToken,
		AutosaveURL:    "/f/" + form.PublicToken + "/autosave",
		ImageDeleteURL: "/f/" + form.PublicToken + "/images/delete",
		FileDeleteURL:  "/f/" + form.PublicToken + "/files/delete",
		PublicLink:     "/f/" + form.PublicToken,
		MaxImageHint:   formPublicUploadHint,
		MaxFileHint:    "Até 5 arquivos por campo, com no máximo 30 MB por arquivo.",
		FieldErrors:    map[string]string{},
	}
	total := 0
	answered := 0
	for _, step := range steps {
		publicStep := formPublicStep{ID: step.ID, Title: step.Title}
		for _, field := range step.Fields {
			total++
			publicField := formPublicField{
				FormField:     field,
				Selected:      map[string]bool{},
				NotApplicable: field.Status == "nao_aplicavel",
			}
			for _, option := range field.Options {
				if option == field.Value || fmt.Sprint(option) == field.Value {
					publicField.Selected[option] = true
				}
			}
			for _, value := range strings.Split(field.Value, ", ") {
				if value != "" {
					publicField.Selected[value] = true
				}
			}
			if field.Type == "scale" {
				minimum := field.ScaleMin
				if minimum == 0 {
					minimum = 1
				}
				maximum := field.ScaleMax
				if maximum == 0 {
					maximum = 5
				}
				for value := minimum; value <= maximum; value++ {
					publicField.ScaleValues = append(publicField.ScaleValues, value)
				}
			}
			if field.Value != "" || len(field.Images) > 0 || len(field.Files) > 0 || field.Status == "nao_aplicavel" {
				answered++
			}
			publicStep.Fields = append(publicStep.Fields, publicField)
		}
		data.Steps = append(data.Steps, publicStep)
	}
	data.TotalQuestions = total
	data.AnsweredCount = answered
	if total > 0 {
		data.Percent = int(float64(answered)/float64(total)*100 + 0.5)
	}
	data.HasStructure = total > 0
	return data
}

func (s *Server) renderFormPublic(w http.ResponseWriter, status int, data formPublicData) {
	if data.FieldErrors == nil {
		data.FieldErrors = map[string]string{}
	}
	for stepIndex, step := range data.Steps {
		for fieldIndex, field := range step.Fields {
			if message, ok := data.FieldErrors[field.ID]; ok {
				data.Steps[stepIndex].Fields[fieldIndex].Error = message
			}
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = formPublicTemplate.Execute(w, data)
}

func (s *Server) captureFormUploads(ctx context.Context, form modules.Form, steps []modules.FormStep, r *http.Request) ([]modules.FormStep, int, map[string]string) {
	errors := map[string]string{}
	captured := 0
	for stepIndex, step := range steps {
		for fieldIndex, field := range step.Fields {
			if field.Type != "image" && field.Type != "file" {
				continue
			}
			uploads := formUploadedFieldFiles(r.MultipartForm, field.ID)
			if len(uploads) == 0 {
				continue
			}
			if field.Type == "image" {
				if len(field.Images)+len(uploads) > formMaxImages {
					errors[field.ID] = fmt.Sprintf("O campo %q aceita no máximo %d imagens.", field.Label, formMaxImages)
					continue
				}
				for _, header := range uploads {
					if header.Size > formMaxImageBytes {
						errors[field.ID] = fmt.Sprintf("A imagem de %q deve ter no máximo 15 MB.", field.Label)
						break
					}
					file, err := header.Open()
					if err != nil {
						errors[field.ID] = "Não foi possível ler a imagem enviada."
						break
					}
					result, err := s.Uploads.Save(ctx, file, header)
					_ = file.Close()
					if err != nil {
						errors[field.ID] = fmt.Sprintf("O campo %q deve conter uma imagem válida.", field.Label)
						break
					}
					field.Images = append(field.Images, result.URL)
					field.ImageThumbnails = append(field.ImageThumbnails, result.URL)
					captured++
				}
				if len(field.Images) > 0 {
					field.Status = "enviado"
				}
			} else {
				if len(field.Files)+len(uploads) > formMaxFiles {
					errors[field.ID] = fmt.Sprintf("O campo %q aceita no máximo %d arquivos.", field.Label, formMaxFiles)
					continue
				}
				for _, header := range uploads {
					if header.Size > formMaxFileBytes {
						errors[field.ID] = fmt.Sprintf("O arquivo de %q deve ter no máximo 30 MB.", field.Label)
						break
					}
					file, err := header.Open()
					if err != nil {
						errors[field.ID] = "Não foi possível ler o arquivo enviado."
						break
					}
					result, err := s.Uploads.SaveDocument(ctx, file, header, "uploads/forms")
					_ = file.Close()
					if err != nil {
						errors[field.ID] = fmt.Sprintf("O campo %q deve conter um arquivo válido.", field.Label)
						break
					}
					field.Files = append(field.Files, result.URL)
					field.FileNames = append(field.FileNames, header.Filename)
					captured++
				}
				if len(field.Files) > 0 {
					field.Status = "enviado"
				}
			}
			steps[stepIndex].Fields[fieldIndex] = field
		}
	}
	return steps, captured, errors
}

func parseFormPublicRequest(r *http.Request) error {
	if strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "multipart/form-data") {
		return r.ParseMultipartForm(60 << 20)
	}
	return r.ParseForm()
}

func formUploadedFiles(form *multipart.Form, key string) []*multipart.FileHeader {
	if form == nil {
		return nil
	}
	return form.File[key]
}

func formUploadedFieldFiles(form *multipart.Form, fieldID string) []*multipart.FileHeader {
	if form == nil {
		return nil
	}
	uploads := append([]*multipart.FileHeader{}, form.File["answers["+fieldID+"]"]...)
	uploads = append(uploads, form.File["answers["+fieldID+"][]"]...)
	return uploads
}

func formAnswersFromRequest(r *http.Request) map[string]string {
	answers := map[string]string{}
	if err := r.ParseForm(); err != nil {
		return answers
	}
	for key, values := range r.PostForm {
		if !strings.HasPrefix(key, "answers[") || !strings.HasSuffix(key, "]") {
			continue
		}
		fieldID := strings.TrimSuffix(strings.TrimPrefix(key, "answers["), "]")
		if strings.HasSuffix(fieldID, "[]") {
			fieldID = strings.TrimSuffix(fieldID, "[]")
		}
		if len(values) > 1 {
			answers[fieldID] = strings.Join(values, ", ")
		} else if len(values) == 1 {
			answers[fieldID] = values[0]
		}
	}
	return answers
}

func formNotApplicableFromRequest(r *http.Request) map[string]bool {
	result := map[string]bool{}
	for key, values := range r.PostForm {
		if !strings.HasPrefix(key, "na_answers[") || len(values) == 0 {
			continue
		}
		fieldID := strings.TrimSuffix(strings.TrimPrefix(key, "na_answers["), "]")
		result[fieldID] = true
	}
	return result
}

func validatePublicFormField(field modules.FormField, notApplicable bool) string {
	if field.Type == "image" {
		if notApplicable || len(field.Images) > 0 {
			return ""
		}
		if field.Required {
			return fmt.Sprintf("O campo %q é obrigatório.", field.Label)
		}
		return ""
	}
	if field.Type == "file" {
		if field.Required && len(field.Files) == 0 {
			return fmt.Sprintf("O campo %q é obrigatório.", field.Label)
		}
		return ""
	}
	value := strings.TrimSpace(field.Value)
	if value == "" {
		if field.Required {
			return fmt.Sprintf("O campo %q é obrigatório.", field.Label)
		}
		return ""
	}
	switch field.Type {
	case "number":
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return fmt.Sprintf("O campo %q deve conter um número.", field.Label)
		}
	case "date":
		if _, err := time.Parse("2006-01-02", value); err != nil {
			return fmt.Sprintf("O campo %q deve conter uma data válida.", field.Label)
		}
	case "time":
		if _, err := time.Parse("15:04", value); err != nil {
			return fmt.Sprintf("O campo %q deve conter um horário válido.", field.Label)
		}
	case "scale":
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Sprintf("O campo %q deve conter um número inteiro.", field.Label)
		}
		minimum := field.ScaleMin
		if minimum == 0 {
			minimum = 1
		}
		maximum := field.ScaleMax
		if maximum == 0 {
			maximum = 5
		}
		if parsed < minimum || parsed > maximum {
			return fmt.Sprintf("O valor de %q deve ficar entre %d e %d.", field.Label, minimum, maximum)
		}
	case "radio", "dropdown":
		if len(field.Options) > 0 {
			valid := false
			for _, option := range field.Options {
				if option == value {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Sprintf("O valor selecionado em %q é inválido.", field.Label)
			}
		}
	}
	if len([]rune(value)) > 5000 {
		return fmt.Sprintf("O campo %q excede o limite de caracteres.", field.Label)
	}
	return ""
}

func (s *Server) persistFormDraft(w http.ResponseWriter, form modules.Form, response *modules.FormResponse, steps []modules.FormStep, r *http.Request) *modules.FormResponse {
	if response != nil {
		_ = s.Modules.UpdateFormResponse(form.ID, response.ID, response.Status, steps, response.SubmittedAt)
		return response
	}
	var userID *int64
	user := currentUser(r)
	if user.ID > 0 {
		userID = &user.ID
	}
	draft, err := s.Modules.CreateFormResponse(form.ID, modules.FormResponseInput{
		UserID: userID, Status: "Em preenchimento", Steps: steps,
	})
	if err != nil {
		return nil
	}
	s.setFormDraftCookie(w, r, form, draft.ID)
	return &draft
}

func (s *Server) formDraftCookieName(form modules.Form) string {
	return fmt.Sprintf("dc_form_draft_%d", form.ID)
}

func (s *Server) formDraftCookiePath(form modules.Form) string {
	return "/f/" + form.PublicToken
}

func (s *Server) setFormDraftCookie(w http.ResponseWriter, r *http.Request, form modules.Form, responseID int64) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.formDraftCookieName(form),
		Value:    strconv.FormatInt(responseID, 10),
		Path:     s.formDraftCookiePath(form),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   formDraftCookieAge,
		Secure:   secureCookie(s.Config, r),
	})
}

func (s *Server) clearFormDraftCookie(w http.ResponseWriter, r *http.Request, form modules.Form) {
	http.SetCookie(w, &http.Cookie{
		Name:    s.formDraftCookieName(form),
		Value:   "",
		Path:    s.formDraftCookiePath(form),
		MaxAge:  -1,
		Expires: time.Unix(0, 0),
		Secure:  secureCookie(s.Config, r),
	})
}
