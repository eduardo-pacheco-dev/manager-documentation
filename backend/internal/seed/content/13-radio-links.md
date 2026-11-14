# Rádio Links

O módulo **Rádio Links** registra enlaces ponto a ponto entre duas estações do inventário: a **ponta A** e a **ponta B**. A página de detalhes mostra os dados técnicos do enlace e as informações das duas estações, com atalho para o cadastro de cada uma, além de mapa com a linha do link, **anexos** e **comentários**.

## Campos

| Campo | Descrição |
| --- | --- |
| Nome | Identificação do enlace. |
| Código | Código único opcional. |
| Ponta A / Ponta B | Estações de origem e destino. |
| Status | `planejado`, `ativo`, `inativo` ou `manutencao`. |
| Tecnologia | Ex.: MW, UHF, 5 GHz. |
| Frequência | Valor em MHz. |
| Capacidade | Valor em Mbps. |
| Distância | Valor em km. |
| Observações | Anotações livres. |

## Regras

- As pontas A e B precisam ser estações **diferentes** e já cadastradas em **Estações**.
- As buscas de estação usam o mesmo catálogo do módulo Estações.
- Ao excluir uma estação, os rádio links vinculados são removidos em cascata.
- O módulo pode ser habilitado, ordenado e movido entre a sidebar e o menu Módulos em **Configurações → Módulos**.

## API

- `GET /api/v1/radio-links` — lista com busca, filtros por status/tecnologia e paginação.
- `POST /api/v1/radio-links` — cria um enlace.
- `GET /api/v1/radio-links/{id}` — detalha um enlace.
- `PUT /api/v1/radio-links/{id}` — atualiza um enlace.
- `DELETE /api/v1/radio-links/{id}` — remove um enlace.
- `GET /api/v1/radio-links/stations` — busca estações para as pontas A e B.
