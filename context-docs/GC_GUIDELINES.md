# Diretrizes GC e Complexidade

Objetivo: minimizar pressão no GC e manter operações padrão em O(n).

Regras:

- Prefira streaming (Read/Write) a buffers inteiros.
- Para listagens, exponha `MaxItems` para que o chamador saiba o tamanho máximo da resposta.
- Use tipos primitivos e structs compactas para dados retornados em massa.
- Evite alocações temporárias em loops quentes; reuso de slices via `[:0]` quando seguro.
- Documente custos de memória estimados nos métodos públicos.
