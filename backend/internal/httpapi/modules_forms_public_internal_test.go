package httpapi

import (
	"testing"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/modules"
)

func publicField(status, value string) formPublicField {
	return formPublicField{FormField: modules.FormField{Status: status, Value: value}}
}

func TestFormPublicStepStatus(t *testing.T) {
	cases := []struct {
		name   string
		fields []formPublicField
		want   string
	}{
		{
			name:   "sem campos",
			fields: nil,
			want:   "vazio",
		},
		{
			name:   "nada preenchido",
			fields: []formPublicField{publicField("pendente", ""), publicField("", "")},
			want:   "vazio",
		},
		{
			name:   "em andamento com valor enviado",
			fields: []formPublicField{publicField("aprovado", "ok"), publicField("enviado", "texto")},
			want:   "andamento",
		},
		{
			name:   "em andamento com conteudo sem status",
			fields: []formPublicField{publicField("", ""), {FormField: modules.FormField{Images: []string{"foto.png"}}}},
			want:   "andamento",
		},
		{
			name:   "reprovado continua em andamento",
			fields: []formPublicField{publicField("aprovado", "ok"), publicField("reprovado", "nao")},
			want:   "andamento",
		},
		{
			name:   "todas aprovadas",
			fields: []formPublicField{publicField("aprovado", "ok"), publicField("aprovado", "ok")},
			want:   "aprovado",
		},
		{
			name:   "aprovado com nao aplicavel",
			fields: []formPublicField{publicField("aprovado", "ok"), publicField("nao_aplicavel", "")},
			want:   "aprovado",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := formPublicStepStatus(tc.fields); got != tc.want {
				t.Fatalf("formPublicStepStatus() = %q, want %q", got, tc.want)
			}
		})
	}
}
