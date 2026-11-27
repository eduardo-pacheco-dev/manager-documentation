# API de autenticação

Prefixo: **/api/v1**.

| Método | Rota | Descrição |
| --- | --- | --- |
| POST | /auth/signup | Cria conta e retorna token quando device_name é enviado. |
| POST | /auth/login | Autentica e retorna token. |
| POST | /auth/logout | Revoga o token atual. |
| GET | /auth/me | Retorna o usuário autenticado. |
| POST | /uploads | Envia uma imagem (multipart file). |

## Exemplo

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"password","device_name":"cli"}'
```

A resposta inclui **data.user** e **data.token**. Use o token em **Authorization: Bearer ...**.

## Uploads

O endpoint aceita imagens JPG, PNG, GIF, WEBP e BMP. O tipo real é validado pelos magic bytes. O limite padrão é 20 MB e pode ser alterado em Configurações > Armazenamento.

## Empresas

O corpo de `POST /api/v1/auth/login` e `POST /api/v1/auth/signup` aceita `company` (slug) ou `company_id`. Se ausente, `padrao` é usado. A resposta inclui `data.company`.

Super-administradores podem enviar o header `X-Company-ID` para operar em outra empresa ativa; `GET /api/v1/auth/me` devolve a empresa ativa correspondente.

Rotas de gestão de empresas (`GET/POST /api/v1/companies` e `PUT /api/v1/companies/{id}`) exigem `is_super_admin = true`. A criação pode incluir `admin_name`, `admin_email` e `admin_password` para provisionar o primeiro administrador da nova empresa.

### Gestão de usuários por empresa

`POST/PUT /api/v1/users` aceitam `company_id` e `is_super_admin` **somente** quando o chamador é super-administrador. Administradores de empresa não podem promover usuários a super-administrador; o backend ignora e valida esses campos. Para gerenciar outra empresa, o super-administrador pode enviar `X-Company-ID` ou informar `company_id` no corpo da requisição ao criar um usuário. Não é possível mover um usuário existente entre empresas.

O super-administrador não pode remover o próprio acesso. A empresa padrão (`padrao`, id 1) não pode ser renomeada nem desativada.

`GET /api/v1/settings` expõe `is_super_admin` além de `is_admin`, permitindo que a interface habilite apenas os recursos globais para o super-administrador.
