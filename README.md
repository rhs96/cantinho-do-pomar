# Cantinho do Pomar

Um jogo de fazenda para o navegador, feito pela comunidade. Plante, regue, colha, cuide dos animais e visite o cantinho dos amigos — inspirado nos jogos de fazenda que marcaram as redes sociais no fim dos anos 2000.

> Projeto em estágio inicial. Ainda não há nada jogável.

## Stack

| Parte | Tecnologia |
|---|---|
| API | Go, chi, pgx |
| Banco | PostgreSQL |
| Web | Next.js, TypeScript, PixiJS |
| Infra | Docker, Docker Compose, GitHub Actions |

## Estrutura

```
apps/web/        cliente web (Next.js)
services/api/    API do jogo (Go)
deploy/          docker compose e configuração de ambiente
docs/adr/        decisões de arquitetura
```

## Licença

- Código: [AGPL-3.0](LICENSE)
- Arte, sons e demais assets (`apps/web/public/assets/`): [CC BY-SA 4.0](LICENSE-ASSETS)
