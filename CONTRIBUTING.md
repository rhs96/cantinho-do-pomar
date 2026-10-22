# Contribuindo

Que bom que você quer ajudar a cuidar do pomar. Este guia explica como o projeto funciona no dia a dia.

## Antes de começar

- Procure uma issue aberta ou abra uma nova descrevendo o que pretende fazer. Para mudanças grandes, espere um retorno antes de investir tempo.
- Issues com a label `good first issue` são um bom ponto de partida.
- Ao participar, você concorda com o [Código de Conduta](CODE_OF_CONDUCT.md).

## Ambiente local

Requisitos: Docker, Node.js 24 e, opcionalmente, Go 1.27 para rodar a API fora do container.

```sh
cp deploy/.env.example deploy/.env
make up            # postgres, api (:8080) e web (:3000)
make web-dev       # front com hot reload, fora do docker
make api-test
make web-test
```

## Fluxo

1. Crie uma branch a partir da `main`: `feat/…`, `fix/…`, `docs/…`, `chore/…`.
2. Faça commits pequenos seguindo [Conventional Commits](https://www.conventionalcommits.org/pt-br/):
   `feat(api): …`, `fix(web): …`, `test(api): …`, `docs: …`.
3. Abra um PR para a `main` descrevendo o que mudou e como testar. A CI precisa estar verde.
4. Pelo menos uma aprovação de mantenedor antes do merge.

## Padrões

**API (Go)**
- Respeite as camadas descritas no [ADR 0001](docs/adr/0001-clean-architecture-and-authoritative-server.md): `domain` não conhece HTTP nem banco.
- Regra de jogo nova vem acompanhada de teste em `domain` ou `app`.
- `golangci-lint run` sem avisos.

**Web (Next.js)**
- A cena do jogo fica em `src/game` (TypeScript puro); componentes React ficam fora dela.
- `npm run lint` e `npm run typecheck` sem erros.

**Assets**
- Arte e som vão em `apps/web/public/assets/` e são licenciados sob CC BY-SA 4.0.
- Envie apenas material autoral ou com licença compatível, e cite a origem no PR.
- Não use arte, nomes ou personagens de outros jogos.

## Decisões de arquitetura

Mudanças estruturais (nova dependência importante, troca de biblioteca, novo serviço) devem vir com um ADR em `docs/adr/`, numerado em sequência.

## Licença

Ao contribuir, você concorda que o código será distribuído sob a [AGPL-3.0](LICENSE) e os assets sob a [CC BY-SA 4.0](LICENSE-ASSETS).
