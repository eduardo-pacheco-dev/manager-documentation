# Autenticação e autorização

## Sessões web

As sessões são tokens opacos armazenados com hash SHA-256 no banco. O navegador recebe apenas o token no cookie HttpOnly. Cada sessão possui um token CSRF associado, exigido em métodos POST, PUT, PATCH e DELETE.

## Tokens de API

**POST /api/v1/auth/login** e **/auth/signup** aceitam **device_name**. Quando informado, a resposta inclui um token Bearer. O token só é exibido uma vez e é armazenado como hash.

## Contas inativas

Usuários com **is_active = false** não conseguem autenticar. Sessões existentes são invalidadas no próximo acesso.

## Empresas e login

O login e o cadastro usam **empresa + e-mail**. Envie `company` (slug) ou `company_id` no corpo da requisição; quando omitido, a empresa `padrao` é assumida para manter as credenciais existentes funcionando. O mesmo e-mail pode existir em empresas diferentes.

A resposta de autenticação inclui o usuário e a empresa ativa. Sessões e tokens de API também são vinculados à empresa e não podem ser reutilizados em outra.

## Permissões

- Todas as rotas autenticadas exigem sessão ou token válido.
- Cada usuário acessa somente os dados da sua empresa.
- /users e /dev-docs exigem **is_admin = true** ou **is_super_admin = true** na empresa ativa.
- Configurações de sistema e armazenamento são exclusivas de administradores da empresa.
- Backups e /companies são exclusivos de **super-administradores**.
- Um super-administrador pode selecionar outra empresa ativa enviando o header **X-Company-ID** (slug, nome ou id). Nesse modo, todas as consultas passam a usar a empresa escolhida, enquanto a conta continua sendo a do super-administrador.
- Administradores de empresa não podem criar ou promover super-administradores.
- Um administrador não pode desativar nem excluir a própria conta.
