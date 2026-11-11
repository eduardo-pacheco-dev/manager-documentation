# Testes e qualidade

## Backend

```bash
cd backend
go test ./...
go vet ./...
```

Os testes usam um banco SQLite temporário e cobrem autenticação, usuários, wiki, widgets, notificações e uploads.

## Frontend

```bash
cd frontend
npm run typecheck
npm run build
```

## Checklist manual

1. Login e logout.
2. Cadastro e validação de e-mail duplicado.
3. Criação, edição e exclusão de usuário.
4. Criação de documento, rascunho e busca.
5. Ocultar, reordenar e redimensionar widgets.
6. Alternar tema claro/escuro/sistema.
7. Marcar e limpar notificações.
8. Upload de imagem via API ou editor.
