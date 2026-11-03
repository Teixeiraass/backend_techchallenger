# Análise de vulnerabilidades

Data da análise: 24/10/2026. Os relatórios brutos estão nesta pasta e são gerados de novo pelos comandos indicados. O CI (`.github/workflows/ci.yml`, job `security`) roda as mesmas ferramentas a cada push e PR.

## Resumo

| Ferramenta | O que analisa | Comando | Resultado |
|---|---|---|---|
| [govulncheck](https://go.dev/blog/govulncheck) | Dependências Go e biblioteca padrão, considerando o que o código realmente chama | `make vulncheck` | **0 vulnerabilidades alcançáveis** ([govulncheck.txt](govulncheck.txt)) |
| [gosec](https://github.com/securego/gosec) | Análise estática do código (SAST) | `make gosec` | **0 problemas** ([gosec.txt](gosec.txt)) |
| [Trivy](https://trivy.dev) | Imagem Docker: pacotes do Alpine e binários Go | `make trivy` | Alpine: **0**. Binário `migrate`: **0**. Binário da API: **1 aviso sem correção, não alcançável** ([trivy-image.txt](trivy-image.txt)) |

## O que foi encontrado e corrigido

1. **Dependências com vulnerabilidades alcançáveis** (govulncheck): `golang.org/x/net` v0.55.0 (GO-2026-6603, 6611, 6612, 6617, HTTP/2) e `github.com/quic-go/quic-go` v0.59.0 (GO-2026-5676), ambas trazidas pelo Gin. Atualizadas para `x/net` v0.60.0 e `quic-go` v0.59.1.
2. **Biblioteca padrão do Go**: a versão 1.26.0 tinha 32 vulnerabilidades corrigidas em patches posteriores. O `go.mod` fixa `toolchain go1.26.10` e o Dockerfile usa `golang:1.26.10-alpine3.23`.
3. **Binário `migrate` da imagem** (Trivy): o binário pré-compilado do golang-migrate trazia 114 vulnerabilidades (4 críticas e 51 altas), vindas de Go e drivers antigos. Agora o Dockerfile compila o `migrate` v4.20.1 com o mesmo Go 1.26.10 e só com o driver postgres. Resultado: 0.
4. **gosec**:
   - G104: erros do `viper.BindEnv` eram ignorados. Agora são tratados.
   - G404/G115: o gerador de dados de teste (`util/random.go`) usa `math/rand`. Esse código não é usado em segurança e está marcado com `#nosec` e justificativa. A conversão `int -> byte` foi trocada por `strconv`.

## Risco aceito

| Achado | Motivo |
|---|---|
| `golang.org/x/crypto` GO-2026-5932: o pacote `openpgp` está descontinuado | Não existe versão corrigida. A API não importa `openpgp`, e o govulncheck confirma que o código não chama o pacote afetado. Usamos só `bcrypt` e `chacha20poly1305` do mesmo módulo. |

## Controles de segurança da aplicação

- **Autenticação**: JWT HS256 por padrão (`TOKEN_TYPE=jwt`), com opção de PASETO v2 (`TOKEN_TYPE=paseto`). O algoritmo é fixado na verificação, então `alg=none` e outros algoritmos são recusados. O access token expira em 15 min. O refresh token fica salvo em sessão e pode ser revogado no logout. O JWT é assinado, mas não é cifrado: quem tiver o token consegue ler o payload (e-mail, papel e IDs do usuário). Por isso o token não leva senha nem dado do cliente e dura pouco. Quem precisar do payload cifrado usa `TOKEN_TYPE=paseto`. Trocar o `TOKEN_TYPE` invalida os tokens já emitidos: os usuários precisam fazer login de novo.
- **Autorização por papel**: ADMINISTRADOR, ATENDENTE e MECANICO, aplicada por middleware em cada rota administrativa.
- **Usuário desativado** perde o acesso na hora: o middleware consulta o usuário a cada requisição.
- **Senhas** guardadas com bcrypt. O login responde com a mesma mensagem para e-mail inexistente e para senha errada.
- **Rotas públicas do cliente** exigem o UUID da OS e o CPF/CNPJ do dono. Documento errado recebe 404, igual a OS inexistente, para não revelar quais OS existem.
- **Validação de entrada**: CPF/CNPJ com dígitos verificadores, placa nos padrões antigo e Mercosul, e binding com limites de tamanho.
- **Erros 500** voltam com mensagem genérica. O detalhe vai só para o log.
- **SQL** só pelo sqlc, com queries parametrizadas.
- **Container** roda como usuário sem privilégios (`app`).
- **Segredos** ficam fora do repositório: `app.env` está no `.gitignore`. O `app.env.example` e o `docker-compose.yaml` têm só valores de desenvolvimento local, identificados como tal.
