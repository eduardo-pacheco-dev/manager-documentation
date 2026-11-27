# Frontend e Carbon Design System

## Estrutura

- **src/pages**: telas da aplicação.
- **src/components**: layout, widgets, modais e componentes reutilizáveis.
- **src/contexts**: autenticação, tema e toasts.
- **src/lib/api.ts**: cliente HTTP com CSRF e tratamento de erros.

## Carbon

Os componentes vêm de **@carbon/react** e os ícones de **@carbon/icons-react**. A tipografia é IBM Plex Sans e IBM Plex Mono, carregada via **@fontsource**.

## Temas

O usuário pode escolher **Sistema**, **Claro** ou **Escuro**. O tema é aplicado com o componente **Theme** do Carbon (g10 para claro, g100 para escuro) e persistido na conta.

## Responsividade

- Sidebar vira gaveta com overlay em telas menores que 1056px.
- Grids de widgets e cards se reorganizam em uma coluna.
- Tabelas usam rolagem horizontal e há modos em cards/lista para usuários.
- Formulários passam para uma coluna e ações empilham.
