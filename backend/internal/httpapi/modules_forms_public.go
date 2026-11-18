package httpapi

import (
	"context"
	"fmt"
	"html/template"
	"log"
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

// ImageStatus returns the evaluation status of a single image. Individual
// image decisions take precedence over the field status.
func (f formPublicField) ImageStatus(index int) string {
	if value := f.ImageDecisions[strconv.Itoa(index)]; value != "" {
		return value
	}
	if f.Status != "" {
		return f.Status
	}
	return "pendente"
}

func formImageStatusLabel(status string) string {
	switch status {
	case "aprovado":
		return "Aprovado"
	case "reprovado":
		return "Reprovado"
	case "enviado":
		return "Enviado"
	case "nao_aplicavel":
		return "Não aplicável"
	default:
		return "Pendente"
	}
}

type formPublicStep struct {
	ID     string
	Title  string
	Status string
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
	MaxImages      int
}

var formPublicTemplate = template.Must(template.New("form-public").Funcs(template.FuncMap{
	"has": func(values map[string]bool, value string) bool {
		return values[value]
	},
	"statusLabel": formImageStatusLabel,
	"base": func(value string) string {
		parsed, err := url.Parse(value)
		if err != nil {
			return value
		}
		return path.Base(parsed.Path)
	},
	"inc": func(value int) int { return value + 1 },
	"thumb": func(field formPublicField, index int) string {
		if index < 0 {
			return ""
		}
		candidate := ""
		if index < len(field.ImageThumbnails) && field.ImageThumbnails[index] != "" {
			candidate = field.ImageThumbnails[index]
		} else if index < len(field.Images) {
			candidate = field.Images[index]
		}
		if index < len(field.Images) && candidate == field.Images[index] && strings.HasPrefix(candidate, "https://picsum.photos/seed/") {
			rest := strings.TrimPrefix(candidate, "https://picsum.photos/seed/")
			if parts := strings.Split(rest, "/"); len(parts) == 3 {
				return "https://picsum.photos/seed/" + parts[0] + "/320/240"
			}
		}
		return candidate
	},
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
.stepper{display:grid;grid-template-columns:repeat(auto-fit,minmax(8.5rem,1fr));gap:.4rem;margin:0 0 1rem;padding:0;list-style:none}
.stepper li{display:flex}
.stepper button{display:flex;flex-direction:column;align-items:center;justify-content:center;gap:.4rem;width:100%;height:100%;padding:.75rem .5rem;border:1px solid var(--border);background:var(--layer);color:var(--muted);font-family:inherit;font-size:.8rem;line-height:1.25;text-align:center;cursor:pointer}
.stepper button:hover{border-color:var(--blue);color:var(--blue)}
.stepper button[aria-current="step"]{border-color:var(--blue);background:#edf5ff;color:#002d9c;font-weight:500}
.stepper .num{display:inline-flex;align-items:center;justify-content:center;width:1.4rem;height:1.4rem;flex:none;border-radius:50%;background:#e0e0e0;color:#161616;font-size:.7rem;font-weight:600}
.stepper button[data-step-status="andamento"] .num{background:#f1c21b;color:#161616}
.stepper button[data-step-status="aprovado"] .num{background:var(--success);color:#fff}
.stepper button[data-step-status="vazio"]{border-bottom:3px solid #c6c6c6}
.stepper button[data-step-status="andamento"]{border-bottom:3px solid #f1c21b}
.stepper button[data-step-status="aprovado"]{border-bottom:3px solid var(--success)}
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
.options{display:flex;flex-direction:column;gap:.5rem;margin-top:.35rem}
.options label{display:flex;align-items:center;gap:.75rem;padding:.8rem 1rem;border:1px solid #8d8d8d;background:var(--layer);font-size:.87rem;color:var(--text);cursor:pointer;transition:border-color .15s,background .15s,color .15s}
.options label:hover{border-color:var(--blue)}
.options label:has(input:checked){border-color:var(--blue);background:#edf5ff;color:#002d9c}
.options input[type=radio],.options input[type=checkbox]{appearance:none;-webkit-appearance:none;width:1.15rem;height:1.15rem;flex:none;margin:0;border:1px solid #8d8d8d;background:#fff;cursor:pointer}
.options input[type=radio]{border-radius:50%}
.options input[type=radio]:checked{border:5px solid var(--blue)}
.options input[type=checkbox]{border-radius:2px;position:relative}
.options input[type=checkbox]:checked{background:var(--blue);border-color:var(--blue)}
.options input[type=checkbox]:checked::after{content:"";position:absolute;left:.33rem;top:.1rem;width:.32rem;height:.62rem;border:solid #fff;border-width:0 2px 2px 0;transform:rotate(45deg)}
.scale{display:flex;flex-wrap:wrap;gap:.35rem}
.scale label{flex:1;min-width:2.6rem;display:flex;flex-direction:column;align-items:center;gap:.25rem;padding:.5rem .25rem;border:1px solid var(--border);font-size:.72rem;color:var(--muted);cursor:pointer;text-align:center}
.scale input{margin:0}
.upload-hint{color:var(--helper);font-size:.72rem;margin:.35rem 0 .5rem}
.upload-actions{display:flex;gap:.5rem;flex-wrap:wrap;margin:.35rem 0 .6rem}
.upload-action{display:inline-flex;align-items:center;justify-content:center;gap:.45rem;flex:1 1 7rem;min-height:2.75rem;padding:.5rem .75rem;border:1px solid #8d8d8d;background:var(--layer);color:var(--text);font-family:inherit;font-size:.82rem;cursor:pointer;transition:border-color .15s,background .15s,color .15s}
.upload-action:hover:not(:disabled){border-color:var(--blue);color:var(--blue)}
.upload-action.is-active{border-color:var(--blue);background:#edf5ff;color:#002d9c}
.upload-action:disabled{opacity:.45;cursor:not-allowed}
.upload-action svg{width:1rem;height:1rem;flex:none}
.upload-count{display:inline-flex;align-items:center;margin:.1rem 0 .55rem;padding:.2rem .55rem;background:#e8f4ff;color:#0043ce;font-size:.72rem}
.images{display:grid;grid-template-columns:repeat(auto-fill,minmax(6.5rem,1fr));gap:.5rem;margin:.5rem 0}
.image-card{position:relative;border:1px solid var(--border);background:#fff;aspect-ratio:4/3;overflow:hidden}
.image-card img{display:block;width:100%;height:100%;object-fit:cover}
.image-card button{position:absolute;top:.3rem;right:.3rem;display:flex;align-items:center;justify-content:center;width:1.5rem;height:1.5rem;border:0;background:#161616cc;color:#fff;font-size:.85rem;line-height:1;cursor:pointer}
.image-card button:hover{background:#161616}
.image-status{position:absolute;right:.3rem;bottom:.3rem;display:flex;align-items:center;justify-content:center;width:1.4rem;height:1.4rem;border-radius:50%;background:#8d8d8d;color:#fff;pointer-events:none}
.image-status svg{width:.9rem;height:.9rem}
.image-status.is-aprovado{background:var(--success)}
.image-status.is-enviado{background:var(--blue)}
.image-status.is-reprovado{background:var(--danger)}
.image-status.is-nao_aplicavel{background:#6f6f6f}
.image-status.is-pendente{background:#8d8d8d}
.image-card a{cursor:zoom-in}
.lightbox{position:fixed;inset:0;z-index:100;display:flex;align-items:center;justify-content:center;padding:3rem 3.5rem;background:#161616f2}
.lightbox[hidden]{display:none}
.lightbox img{max-width:min(1100px,100%);max-height:calc(100vh - 8rem);object-fit:contain;background:#262626}
.lightbox__close{position:absolute;top:.75rem;right:.75rem;display:flex;align-items:center;justify-content:center;width:2.5rem;height:2.5rem;border:0;background:#393939;color:#fff;font-size:1.5rem;line-height:1;cursor:pointer}
.lightbox__close:hover{background:#525252}
.lightbox__nav{position:absolute;top:50%;transform:translateY(-50%);display:flex;align-items:center;justify-content:center;width:2.75rem;height:3.5rem;border:0;background:#393939cc;color:#fff;font-size:1.75rem;line-height:1;cursor:pointer}
.lightbox__nav:hover{background:#525252}
.lightbox__nav--prev{left:.5rem}
.lightbox__nav--next{right:.5rem}
.lightbox__bar{position:absolute;left:0;right:0;bottom:0;display:flex;align-items:center;justify-content:space-between;gap:1rem;padding:.85rem 1rem;background:#161616;color:#f4f4f4;font-size:.8rem}
.lightbox__bar a{color:#78a9ff;text-decoration:none}
.lightbox__bar a:hover{text-decoration:underline}
.files{display:flex;flex-direction:column;gap:.4rem;margin:.5rem 0}
.file-row{display:flex;align-items:center;gap:.6rem;padding:.55rem .7rem;border:1px solid var(--border);font-size:.8rem}
.file-row a{color:var(--blue);text-decoration:none;flex:1;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.file-row button{border:0;background:none;color:var(--danger);font-size:.72rem;cursor:pointer}
.na{display:inline-flex;align-items:center;gap:.4rem;margin-top:.5rem;font-size:.78rem;color:var(--muted)}
.actions{position:sticky;bottom:0;z-index:10;display:flex;align-items:center;justify-content:space-between;gap:1rem;margin:1.5rem 0 0;padding:.85rem 0;background:var(--bg);border-top:1px solid var(--border);flex-wrap:wrap}
.wizard-actions{display:flex;align-items:center;gap:.5rem;flex-wrap:nowrap}
.wizard-actions .btn{flex:0 0 auto}
.btn--ghost{background:transparent;color:var(--blue);border-color:var(--blue)}
.btn--ghost:hover{background:#edf5ff;color:var(--blue-hover)}
.step-label{color:var(--helper);font-size:.75rem}
.sync-state{display:flex;align-items:center;gap:.75rem;flex-wrap:wrap}
.btn--sync{display:inline-flex;align-items:center;gap:.4rem;min-height:2.5rem;padding:0 .9rem}
.btn--sync svg{width:1rem;height:1rem}
.btn--sync[disabled] svg{animation:spin 1s linear infinite}
@keyframes spin{to{transform:rotate(360deg)}}
.form-step[hidden]{display:none}
.btn{display:inline-flex;align-items:center;justify-content:center;min-height:3rem;padding:0 1.25rem;border:1px solid transparent;background:var(--blue);color:#fff;font-family:inherit;font-size:.875rem;cursor:pointer;text-decoration:none}
.btn:hover{background:var(--blue-hover)}
.save-state{color:var(--helper);font-size:.75rem}
.field-error{margin:.35rem 0 0;color:var(--danger);font-size:.75rem}
.empty{color:var(--muted);font-size:.85rem;padding:1rem;border:1px dashed #c6c6c6;text-align:center}
.footer{max-width:58rem;margin:0 auto;padding:0 1rem 2rem;color:var(--helper);font-size:.72rem;display:flex;flex-wrap:wrap;gap:.5rem;justify-content:space-between}
    @media (max-width:42rem){.shell{padding-top:1rem}.panel{padding:1rem}.header span.meta{display:none}.actions{gap:.5rem}.wizard-actions{flex:1;justify-content:flex-end}.wizard-actions .btn{flex:1 1 0;min-width:0;padding:0 .75rem}.lightbox{padding:3.5rem .5rem 4.5rem}.lightbox__nav{width:2.25rem;height:2.75rem}}
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
    <ol class="stepper" id="wizard-steps" aria-label="Etapas do formulário">
      {{range $si, $step := .Steps}}
      <li>
        <button type="button" data-step-target="{{$si}}" data-step-status="{{$step.Status}}">
          <span class="num">{{inc $si}}</span>
          <span>{{if $step.Title}}{{$step.Title}}{{else}}Seção {{inc $si}}{{end}}</span>
        </button>
      </li>
      {{end}}
    </ol>
    {{range $si, $step := .Steps}}
    <section class="panel form-step" data-form-section data-index="{{$si}}">
      <div class="section-head">
        <h2>{{if $step.Title}}{{$step.Title}}{{else}}Seção {{inc $si}}{{end}}</h2>
        <span class="count">{{len $step.Fields}} pergunta(s)</span>
      </div>
      {{range $step.Fields}}
      {{$field := .}}
      <div class="field">
        <label class="label" id="field-{{$field.ID}}-label" for="field-{{$field.ID}}">{{$field.Label}}{{if $field.Required}} <span class="required">(obrigatório)</span>{{end}}</label>
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
          <div class="upload-actions">
            <button type="button" class="upload-action{{if $field.NotApplicable}} is-active{{end}}" data-na-toggle aria-pressed="{{if $field.NotApplicable}}true{{else}}false{{end}}">
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.3" aria-hidden="true"><rect x="2.5" y="2.5" width="11" height="11" rx="1"></rect><path d="M4 12L12 4"></path></svg>
              N/A
            </button>
            <button type="button" class="upload-action" data-upload-trigger="field-{{$field.ID}}-gallery" {{if or $field.NotApplicable (ge (len $field.Images) $.MaxImages)}}disabled{{end}}>
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.3" aria-hidden="true"><rect x="2" y="3" width="12" height="10" rx="1"></rect><circle cx="5.5" cy="6.5" r="1.2"></circle><path d="M2.5 12l3.5-3.5L9 11l2-2 2.5 2.5"></path></svg>
              Galeria
            </button>
            <button type="button" class="upload-action" data-upload-trigger="field-{{$field.ID}}-camera" {{if or $field.NotApplicable (ge (len $field.Images) $.MaxImages)}}disabled{{end}}>
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.3" aria-hidden="true"><path d="M2 5.5h2.6L5.6 4h4.8l1 1.5H14v8H2z"></path><circle cx="8" cy="9" r="2.3"></circle></svg>
              Câmera
            </button>
          </div>
          <input type="checkbox" name="na_answers[{{$field.ID}}]" value="1" class="na-input" {{if $field.NotApplicable}}checked{{end}} hidden>
          <input type="file" id="field-{{$field.ID}}-gallery" name="answers[{{$field.ID}}][]" accept="image/*" multiple hidden>
          <input type="file" id="field-{{$field.ID}}-camera" name="answers[{{$field.ID}}][]" accept="image/*" capture="environment" multiple hidden>
          <span class="upload-count">{{len $field.Images}}/{{$.MaxImages}} imagens</span>
          {{if $field.Images}}
          <div class="images">
            {{range $index, $image := $field.Images}}
            {{$status := $field.ImageStatus $index}}
            <div class="image-card">
              <a href="{{$image}}" target="_blank" rel="noopener" data-image-preview data-field="{{$field.ID}}" data-index="{{$index}}" data-src="{{$image}}" data-label="{{$field.Label}}"><img src="{{thumb $field $index}}" alt="{{$field.Label}}" loading="lazy"></a>
              <span class="image-status is-{{$status}}" title="Status: {{statusLabel $status}}" aria-label="Status: {{statusLabel $status}}">
                {{if or (eq $status "aprovado") (eq $status "enviado")}}
                <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M3 8.5l3.2 3L13 5"></path></svg>
                {{else if eq $status "reprovado"}}
                <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M4 4l8 8M12 4l-8 8"></path></svg>
                {{else if eq $status "nao_aplicavel"}}
                <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M4 8h8"></path></svg>
                {{else}}
                <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true"><circle cx="8" cy="8" r="5.5"></circle><path d="M8 4.8V8l2.2 1.4"></path></svg>
                {{end}}
              </span>
              <button type="button" data-remove-image data-field="{{$field.ID}}" data-index="{{$index}}" aria-label="Excluir imagem">&times;</button>
            </div>
            {{end}}
          </div>
          {{end}}
          <p class="upload-hint">{{$.MaxImageHint}}</p>
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
      <div class="sync-state">
        <button class="btn btn--ghost btn--sync" type="button" id="sync-button">
          <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true"><path d="M13 8a5 5 0 1 1-1.46-3.54"></path><path d="M13 2.5V5h-2.5"></path></svg>
          Sincronizar
        </button>
        <span class="save-state" id="save-state"></span>
      </div>
      <div class="wizard-actions">
        <button class="btn btn--ghost" type="button" id="wizard-prev">Anterior</button>
        <span class="step-label" id="wizard-label"></span>
        <button class="btn" type="button" id="wizard-next">Próximo</button>
      </div>
    </div>
  </form>
  {{end}}
</main>
<div class="lightbox" id="image-lightbox" hidden role="dialog" aria-modal="true" aria-label="Pré-visualização da imagem" tabindex="-1">
  <button type="button" class="lightbox__close" id="lightbox-close" aria-label="Fechar">&times;</button>
  <button type="button" class="lightbox__nav lightbox__nav--prev" id="lightbox-prev" aria-label="Imagem anterior">&lsaquo;</button>
  <img id="lightbox-image" src="" alt="">
  <button type="button" class="lightbox__nav lightbox__nav--next" id="lightbox-next" aria-label="Próxima imagem">&rsaquo;</button>
  <div class="lightbox__bar">
    <span id="lightbox-caption"></span>
    <a id="lightbox-open" href="#" target="_blank" rel="noopener">Abrir original</a>
  </div>
</div>
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
  const wizardLabel = document.getElementById('wizard-label');
  const stepper = document.getElementById('wizard-steps');
  const stepButtons = [...form.querySelectorAll('[data-step-target]')];
  const storageKey = 'public-form-step-' + window.location.pathname;
  let currentSection = 0;

  const showSection = (index) => {
    if (!sections.length) return;
    currentSection = Math.max(0, Math.min(sections.length - 1, index));
    sections.forEach((section, position) => {
      section.hidden = position !== currentSection;
    });
    stepButtons.forEach((button, position) => {
      if (position === currentSection) {
        button.setAttribute('aria-current', 'step');
      } else {
        button.removeAttribute('aria-current');
      }
    });
    if (prevButton) prevButton.disabled = currentSection === 0;
    if (nextButton) nextButton.hidden = currentSection === sections.length - 1;
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
    stepButtons.forEach((button) => {
      button.addEventListener('click', () => {
        showSection(Number(button.dataset.stepTarget || '0'));
      });
    });
    const markInProgress = (section) => {
      if (!section) return;
      const button = stepButtons[sections.indexOf(section)];
      if (button && button.dataset.stepStatus === 'vazio') {
        button.dataset.stepStatus = 'andamento';
      }
    };
    form.addEventListener('input', (event) => markInProgress(event.target.closest('[data-form-section]')));
    form.addEventListener('change', (event) => markInProgress(event.target.closest('[data-form-section]')));
    prevButton.addEventListener('click', () => showSection(currentSection - 1));
    nextButton.addEventListener('click', () => showSection(currentSection + 1));
    form.addEventListener('keydown', (event) => {
      if (event.key === 'Enter' && event.target.tagName !== 'TEXTAREA' && currentSection < sections.length - 1) {
        event.preventDefault();
        showSection(currentSection + 1);
      }
    });
  } else {
    if (stepper) stepper.hidden = true;
    if (prevButton) prevButton.hidden = true;
    if (nextButton) nextButton.hidden = true;
    if (wizardLabel) wizardLabel.textContent = '';
  }

  const autosave = async () => {
    if (!autosaveUrl) return true;
    state.textContent = 'Salvando…';
    try {
      const response = await fetch(autosaveUrl, { method: 'POST', body: new FormData(form), credentials: 'same-origin' });
      if (!response.ok) throw new Error('save failed');
      const data = await response.json();
      state.textContent = 'Salvo automaticamente.';
      if ((data.images_saved || 0) > 0 || (data.files_saved || 0) > 0 || data.reload) {
        window.location.reload();
      }
      return true;
    } catch (error) {
      state.textContent = 'Não foi possível salvar automaticamente.';
      return false;
    }
  };

  let saveTimer = null;
  const scheduleAutosave = (delay) => {
    if (saveTimer) window.clearTimeout(saveTimer);
    saveTimer = window.setTimeout(() => {
      saveTimer = null;
      void autosave();
    }, delay);
  };
  const flushAutosave = () => {
    if (saveTimer) {
      window.clearTimeout(saveTimer);
      saveTimer = null;
    }
    void autosave();
  };

  form.addEventListener('submit', (event) => {
    event.preventDefault();
    flushAutosave();
  });
  form.addEventListener('input', (event) => {
    if (event.target.type === 'file') return;
    scheduleAutosave(700);
  });
  form.addEventListener('change', (event) => {
    if (event.target.type === 'file') {
      flushAutosave();
      return;
    }
    scheduleAutosave(250);
  });
  window.addEventListener('pagehide', () => {
    if (!autosaveUrl) return;
    if (saveTimer) {
      window.clearTimeout(saveTimer);
      saveTimer = null;
    }
    try {
      navigator.sendBeacon(autosaveUrl, new FormData(form));
    } catch (error) {}
  });

  const syncButton = document.getElementById('sync-button');
  if (syncButton) {
    syncButton.addEventListener('click', async () => {
      if (saveTimer) {
        window.clearTimeout(saveTimer);
        saveTimer = null;
      }
      syncButton.disabled = true;
      syncButton.setAttribute('aria-busy', 'true');
      state.textContent = 'Sincronizando…';
      const saved = await autosave();
      if (saved) {
        window.location.reload();
        return;
      }
      syncButton.disabled = false;
      syncButton.removeAttribute('aria-busy');
    });
  }

  form.querySelectorAll('[data-upload-trigger]').forEach((button) => {
    button.addEventListener('click', () => {
      const input = document.getElementById(button.dataset.uploadTrigger);
      if (input) input.click();
    });
  });
  form.querySelectorAll('[data-na-toggle]').forEach((button) => {
    button.addEventListener('click', () => {
      const field = button.closest('.field');
      if (!field) return;
      const checkbox = field.querySelector('input[name^="na_answers"]');
      if (!checkbox) return;
      checkbox.checked = !checkbox.checked;
      const active = checkbox.checked;
      button.classList.toggle('is-active', active);
      button.setAttribute('aria-pressed', active ? 'true' : 'false');
      field.querySelectorAll('[data-upload-trigger]').forEach((trigger) => {
        trigger.disabled = active;
      });
      checkbox.dispatchEvent(new Event('change', { bubbles: true }));
    });
  });

  const lightbox = document.getElementById('image-lightbox');
  const lightboxImage = document.getElementById('lightbox-image');
  const lightboxCaption = document.getElementById('lightbox-caption');
  const lightboxOpen = document.getElementById('lightbox-open');
  const lightboxPrev = document.getElementById('lightbox-prev');
  const lightboxNext = document.getElementById('lightbox-next');
  let gallery = [];
  let galleryIndex = 0;

  const renderLightbox = () => {
    if (!lightbox || !gallery.length) return;
    galleryIndex = Math.max(0, Math.min(gallery.length - 1, galleryIndex));
    const item = gallery[galleryIndex];
    lightboxImage.src = item.src;
    lightboxImage.alt = item.label;
    lightboxCaption.textContent = item.label + ' · ' + (galleryIndex + 1) + ' / ' + gallery.length;
    lightboxOpen.href = item.src;
    if (lightboxPrev) lightboxPrev.hidden = galleryIndex === 0;
    if (lightboxNext) lightboxNext.hidden = galleryIndex === gallery.length - 1;
  };

  const closeLightbox = () => {
    if (!lightbox) return;
    lightbox.hidden = true;
    lightboxImage.src = '';
    document.body.style.overflow = '';
  };

  const openLightbox = (links, index) => {
    if (!lightbox) return;
    gallery = links.map((link) => ({
      src: link.dataset.src,
      label: link.dataset.label || 'Imagem',
    }));
    galleryIndex = index;
    renderLightbox();
    lightbox.hidden = false;
    document.body.style.overflow = 'hidden';
    lightbox.focus();
  };

  form.querySelectorAll('[data-image-preview]').forEach((link) => {
    link.addEventListener('click', (event) => {
      event.preventDefault();
      const field = link.dataset.field || '';
      const links = [...form.querySelectorAll('[data-image-preview][data-field="' + field + '"]')];
      openLightbox(links, Number(link.dataset.index || '0'));
    });
  });

  if (lightbox) {
    lightbox.addEventListener('click', (event) => {
      if (event.target === lightbox) closeLightbox();
    });
    const closeButton = document.getElementById('lightbox-close');
    if (closeButton) closeButton.addEventListener('click', closeLightbox);
    if (lightboxPrev) {
      lightboxPrev.addEventListener('click', () => {
        galleryIndex -= 1;
        renderLightbox();
      });
    }
    if (lightboxNext) {
      lightboxNext.addEventListener('click', () => {
        galleryIndex += 1;
        renderLightbox();
      });
    }
    window.addEventListener('keydown', (event) => {
      if (lightbox.hidden) return;
      if (event.key === 'Escape') {
        closeLightbox();
        return;
      }
      if (event.key === 'ArrowLeft' && galleryIndex > 0) {
        galleryIndex -= 1;
        renderLightbox();
      }
      if (event.key === 'ArrowRight' && galleryIndex < gallery.length - 1) {
        galleryIndex += 1;
        renderLightbox();
      }
    });
  }

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
		MaxImages:      formMaxImages,
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
		publicStep.Status = formPublicStepStatus(publicStep.Fields)
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

// formPublicStepStatus summarizes a step for the wizard indicator:
// "aprovado" when every field is approved (or not applicable), "andamento"
// when at least one answer was submitted, and "vazio" when nothing was filled.
func formPublicStepStatus(fields []formPublicField) string {
	if len(fields) == 0 {
		return "vazio"
	}
	concluded := 0
	touched := false
	for _, field := range fields {
		switch field.Status {
		case "aprovado", "nao_aplicavel":
			concluded++
		case "enviado", "reprovado":
			touched = true
		default:
			if field.Value != "" || len(field.Images) > 0 || len(field.Files) > 0 {
				touched = true
			}
		}
	}
	if concluded == len(fields) {
		return "aprovado"
	}
	if touched {
		return "andamento"
	}
	return "vazio"
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
	if err := formPublicTemplate.Execute(w, data); err != nil {
		log.Printf("forms public template: %v", err)
	}
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
					thumbnail := result.URL
					if generated := s.storeFormThumbnail(ctx, header); generated != "" {
						thumbnail = generated
					}
					field.Images = append(field.Images, result.URL)
					field.ImageThumbnails = append(field.ImageThumbnails, thumbnail)
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
