# 0001 — Arquitetura limpa na API e servidor autoritativo

- Status: aceita
- Data: 2026-10-22

## Contexto

O jogo terá interação entre jogadores (visitar fazendas, ajudar, roubar colheita) e economia própria (moedas, XP, loja). Qualquer regra executada só no cliente pode ser burlada editando o JavaScript ou chamando a API diretamente.

O projeto também é aberto para a comunidade, então o código precisa ser fácil de entender e de testar por quem chega sem contexto.

## Decisão

1. **O servidor é a fonte da verdade.** Crescimento das plantas, colheita, compras e roubo são calculados na API com o relógio do servidor. O cliente envia intenções (`plantar semente X no canteiro Y`) e renderiza o estado devolvido. Ele pode prever o resultado para a UI ficar responsiva, mas sempre reconcilia com a resposta.

2. **A API em Go segue arquitetura limpa**, com dependências apontando para dentro:

   ```
   internal/
     domain/     entidades e regras puras (Plot, Crop, Farm, Player). Sem I/O.
     app/        casos de uso. Dependem de interfaces (repositórios, relógio).
     adapters/   HTTP, Postgres e outros detalhes que implementam essas interfaces.
     platform/   infraestrutura compartilhada (conexão com banco, logger).
   ```

   `domain` não importa nada de `app` ou `adapters`. O tempo é injetado (`Clock`), para que as regras de crescimento sejam testáveis sem `time.Sleep`.

3. **No front**, a cena do jogo (PixiJS) fica em `src/game` como TypeScript puro. React cuida de páginas, HUD e menus, e apenas monta/desmonta a cena.

## Consequências

- Toda ação de jogo custa uma requisição. Para manter a sensação de resposta imediata, o cliente faz atualização otimista e desfaz em caso de erro.
- Regras de jogo ficam concentradas em `domain`, cobertas por testes unitários rápidos.
- Há mais arquivos e interfaces do que um CRUD simples precisaria. Aceitamos isso em troca de regras isoladas e trocas de infraestrutura baratas.
