# Projetos

O módulo **Projetos** traz para o Documentation Center o planejamento no estilo Microsoft Project: estrutura analítica do projeto (EAP), dependências, calendário, recursos, custos, linha de base, caminho crítico e gráfico de Gantt.

## EAP e tarefas

Cada projeto possui uma árvore de tarefas:

- **Tarefa**: atividade com duração em dias úteis e progresso de 0 a 100%.
- **Resumo**: tarefa pai calculada automaticamente a partir das filhas. Início, fim, duração, progresso e custo são consolidados.
- **Marco**: tarefa de duração zero, usada para aprovações e entregas.

Os códigos EAP (`1`, `1.1`, `1.2.3`) são gerados automaticamente. Ao mover uma tarefa para baixo de outra, a tarefa pai é promovida a resumo.

## Dependências e cronograma

Quatro tipos de dependência são suportados, com lag em dias úteis:

| Tipo | Significado |
| --- | --- |
| FS | Término → Início |
| SS | Início → Início |
| FF | Término → Término |
| SF | Início → Término |

O servidor calcula o cronograma por ordenação topológica: cada tarefa começa respeitando as predecessoras, o calendário de dias úteis e os feriados. Dependências circulares são rejeitadas na criação. A **data de início** da tarefa funciona como restrição “não iniciar antes de”.

## Caminho crítico

Uma tarefa folha é marcada como crítica quando o aumento de um dia em sua duração empurra a data de término do projeto. Tarefas resumo tornam-se críticas quando possuem filhas críticas. O cálculo cobre FS/SS/FF/SF e lag, sem depender de uma fórmula simplificada.

## Calendário

Cada projeto define:

- **Dias úteis** em formato CSV (`1,2,3,4,5` — domingo é `0`).
- **Horas por dia** (usadas como referência de jornada).
- **Feriados** com data e descrição, ignorados no agendamento.

## Recursos, alocações e custos

Recursos possuem função, e-mail, custo por hora, capacidade e cor. Cada alocação liga um recurso a uma tarefa com unidades percentuais e horas planejadas. O custo é `horas planejadas × custo/hora`; o custo da tarefa soma o custo fixo e as alocações; o projeto consolida tudo no orçamento e no saldo.

## Linha de base

Capturar uma linha de base grava início, fim, duração e custo de cada tarefa folha. A tela compara com o cronograma atual e mostra variações em dias úteis e em custo por tarefa.

## API

Todos os endpoints ficam em `/api/v1/projects` e exigem sessão ou token:

| Método | Rota | Descrição |
| --- | --- | --- |
| GET | `/projects` | Portfólio (`?archived=1` inclui arquivados) |
| POST | `/projects` | Cria projeto |
| GET | `/projects/{id}` | Projeto completo com cronograma calculado |
| PUT/DELETE | `/projects/{id}` | Atualiza/exclui |
| PATCH | `/projects/{id}/archive` | Arquiva/restaura |
| GET | `/projects/{id}/export` | Exporta CSV |
| POST | `/projects/{id}/tasks` | Cria tarefa, resumo ou marco |
| PUT/DELETE | `/projects/{id}/tasks/{taskId}` | Atualiza/exclui tarefa |
| PATCH | `/projects/{id}/tasks/{taskId}/move` | Move na EAP |
| POST/DELETE | `/projects/{id}/dependencies` | Cria/remove dependência |
| POST/PUT/DELETE | `/projects/{id}/resources` | Recursos |
| POST/DELETE | `/projects/{id}/assignments` | Alocações |
| POST/DELETE | `/projects/{id}/holidays` | Feriados |
| POST/DELETE | `/projects/{id}/baselines` | Linhas de base |

## Boas práticas

- Cadastre primeiro as tarefas folha e depois agrupe-as em tarefas resumo.
- Use marcos para aprovações formais do cliente.
- Capture uma linha de base após o aceite do cronograma e antes de iniciar a execução.
- Atualize o progresso das tarefas; o progresso e o custo do projeto são consolidados automaticamente.
