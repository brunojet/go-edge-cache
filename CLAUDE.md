# go-edge-cache

## Convenção de commits

Usar **Conventional Commits** (padrão SemVer, ecossistema Go):

```
<tipo>[escopo opcional]: <descrição em minúsculas, imperativo>

[corpo opcional]

[rodapé opcional]
```

**Tipos e impacto no SemVer:**

| Tipo | Uso | Bump |
|---|---|---|
| `feat` | nova funcionalidade | MINOR |
| `fix` | correção de bug | PATCH |
| `refactor` | mudança de código sem alterar comportamento | — |
| `perf` | melhoria de performance | PATCH |
| `docs` | só documentação | — |
| `test` | só testes | — |
| `build` | build, deps, toolchain | — |
| `ci` | pipeline CI | — |
| `chore` | manutenção geral | — |

**Breaking changes** → `feat!:` / `fix!:` ou rodapé `BREAKING CHANGE:` → bump MAJOR.

**Regras:**
- Descrição (subject) ≤ 50 chars, minúscula, modo imperativo ("add", não "added"/"adds")
- Sem ponto final no subject
- Corpo só quando o "porquê" não é óbvio; explica motivação, não o "como"
- Um commit = uma mudança lógica coesa
