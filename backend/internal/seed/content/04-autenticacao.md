# Autenticação e autorização

## Sessões web

As sessões são tokens opacos armazenados com hash SHA-256 no banco. O navegador recebe apenas o token no cookie HttpOnly. Cada sessão possui um token CSRF associado, exigido em métodos POST, PUT, PATCH e DELETE.

## Tokens de API

**POST /api/v1/auth/login** e **/auth/signup** aceitam **device_name**. Quando informado, a resposta inclui um token Bearer. O token só é exibido uma vez e é armazenado como hash.

## Contas inativas

Usuários com **is_active = false** não conseguem autenticar. Sessões existentes são invalidadas no próximo acesso.

## Permissões

- Todas as rotas autenticadas exigem sessão ou token válido.
- /users e /dev-docs exigem **is_admin = true**.
- Configurações de sistema e armazenamento também são exclusivas de administradores.
- Um administrador não pode desativar nem excluir a própria conta.
