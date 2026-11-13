# Colaboradores

O módulo **Colaboradores** é a ficha de RH da equipe e também o cadastro central
de pessoas que podem ser alocadas aos projetos. Em vez de repetir nome, função e
custo em cada projeto, o recurso do projeto é vinculado ao colaborador e herda
esses dados.

## Ficha do colaborador

### Dados pessoais

- Nome completo e nome social.
- CPF, data de nascimento, gênero, estado civil, nacionalidade e naturalidade.
- Nome da mãe e do pai.

### Documentos

- RG e órgão emissor.
- CTPS e série.
- PIS/PASEP, título de eleitor e certificado de reservista.
- Registro profissional (CREA, CRM, OAB etc.).
- CNH, categoria e validade.

### Contato e endereço

- E-mail corporativo e e-mail pessoal.
- Telefone e celular.
- CEP, logradouro, número, complemento, bairro, cidade, UF e país.
- Contato de emergência e telefone.

### Contrato e remuneração

- Cargo, departamento e gestor imediato.
- Tipo de contrato (CLT, PJ, estágio, temporário ou aprendiz).
- Regime de trabalho (presencial, híbrido ou remoto) e local.
- Data de admissão e data de desligamento.
- Jornada semanal, salário mensal, custo/hora e capacidade para projetos.
- Cor de identificação e status ativo/inativo.

### Dados bancários

- Banco, agência, conta e tipo de conta.
- Chave Pix.

## Integração com Projetos

Na aba **Recursos** de um projeto, selecione um colaborador no campo
**Colaborador**. O recurso é criado com nome, função, e-mail, custo/hora,
capacidade e cor copiados do cadastro. O vínculo é gravado em
`project_resources.collaborator_id`.

- O mesmo colaborador não pode ser vinculado duas vezes ao mesmo projeto.
- Custo/hora e capacidade podem ser ajustados por projeto.
- Ao excluir um colaborador, os recursos já criados são preservados como
  histórico; apenas o vínculo é removido (`ON DELETE SET NULL`).
- A listagem de colaboradores mostra em quantos projetos cada pessoa é usada.

## API

Todos os endpoints exigem sessão ou token:

| Método | Rota | Descrição |
| --- | --- | --- |
| GET | `/collaborators` | Lista com busca, status, ordenação e paginação |
| POST | `/collaborators` | Cria colaborador com a ficha completa |
| GET | `/collaborators/{id}` | Detalhe com todos os dados de RH |
| PUT | `/collaborators/{id}` | Atualiza a ficha |
| PATCH | `/collaborators/{id}/toggle` | Ativa/desativa |
| DELETE | `/collaborators/{id}` | Exclui e desvincula dos projetos |

Filtros da listagem: `search` (nome, CPF, e-mail, cargo, cidade etc.), `status`
(`active`/`inactive`), `sort`, `direction`, `page` e `per_page`.

## Boas práticas

- Mantenha CPF, documentos e dados bancários atualizados; CPF e e-mail
  corporativo são únicos por colaborador.
- Registre a data de desligamento e desative o colaborador para retirá-lo de
  novas alocações sem perder o histórico.
- Use o custo/hora e a capacidade como padrão dos recursos de projeto.
- Dados pessoais e bancários são sensíveis: restrinja o acesso conforme a
  política de privacidade da empresa.
