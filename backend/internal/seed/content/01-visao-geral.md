# Visão geral do sistema

O **Documentation Center** é uma central administrativa e wiki técnica construída com **Go** no backend e **React + IBM Carbon** no frontend.

## Principais recursos

- Autenticação web com sessão segura e API com tokens.
- Dashboard configurável com widgets, reordenação, redimensionamento e ocultação.
- Gestão de usuários com busca, filtros, ordenação, paginação e três modos de visualização.
- Wiki de desenvolvimento em Markdown com categorias, busca, rascunhos e publicação.
- Configurações de tema, módulos, notificações, fuso horário, armazenamento e conta.
- Notificações internas e por e-mail.
- API v1 documentada com OpenAPI.

## Módulos nativos

Os antigos plugins foram convertidos em módulos nativos:

| Módulo | Recursos |
| --- | --- |
| Agenda | Calendários, eventos, participantes, recorrência, lembretes e ICS |
| Tarefas | Listas, tarefas, etapas, Meu Dia, importância e recorrência |
| Follow-ups | Status, prioridade, prazos, responsáveis, histórico e resumo |
| Estações | Inventário, busca, anexos, comentários e importação CSV |
| Projetos Nokia | Sites, checklist FAM, tarefas, datas, rejeições e interrupções |


## Stack

| Camada | Tecnologia |
| --- | --- |
| Backend | Go 1.26+, chi, SQLite |
| Frontend | React 19, TypeScript, Vite |
| Design system | IBM Carbon Design System |
| Tipografia | IBM Plex Sans / Mono |
| Autenticação | Sessão HttpOnly + tokens Bearer |
| Documentação | OpenAPI + Swagger UI |

## URLs

| URL | Descrição |
| --- | --- |
| /dashboard | Painel principal |
| /users | Gestão de usuários (admin) |
| /dev-docs | Wiki de desenvolvimento (admin) |
| /settings | Configurações |
| /notifications | Central de notificações |
| /api/documentation | Swagger UI |
| /docs | Especificação OpenAPI em JSON |
