# SC2MSX - SuperCalc 2 MSX para Go/TUI

Implementação do SuperCalc 2 MSX em Go, usando tview como framework TUI
e SQLite como backend de armazenamento.

## Compatibilidade

- 100% compatível com o formato de arquivos SuperCalc 2 MSX (.CAL / .SDI)
- Simula tela de 80 colunas x 24 linhas (monitor padrão MSX)
- Leitura e gravação de arquivos no formato MSX original

## Estrutura do Projeto

```
sc2msx/
├── cmd/
│   └── sc2msx/
│       └── main.go          # Ponto de entrada
├── internal/
│   ├── spreadsheet/
│   │   └── model.go         # Modelo de dados, células, coordenadas
│   ├── ui/
│   │   ├── splash.go        # Tela de apresentação
│   │   └── grid.go          # Tela da planilha (80x24)
│   └── storage/             # (próximo passo) leitura/gravação .CAL
├── go.mod
└── README.md
```

## Como Compilar e Executar

### Requisitos

- Go 1.21 ou superior
- CGO habilitado (para o sqlite3)
- GCC instalado (para compilar o driver sqlite3)

### Instalação das dependências

```bash
cd sc2msx
go mod tidy
```

### Compilar

```bash
go build -o sc2msx ./cmd/sc2msx/
```

### Executar

```bash
./sc2msx
```

### Compilar e executar direto

```bash
go run ./cmd/sc2msx/
```

## Teclas de Controle

### Modo Normal (Navegação)
| Tecla | Ação |
|-------|------|
| ↑↓←→ | Mover cursor |
| Enter | Confirma / desce uma linha |
| Tab | Avança uma coluna |
| PgUp/PgDn | Sobe/Desce 20 linhas |
| Home | Vai para coluna A |
| / | Abre menu de comandos |
| ? | Ajuda |
| Del | Apaga célula atual |

### Modo de Entrada (Digitando)
| Tecla | Ação |
|-------|------|
| Enter | Confirma e desce |
| Esc | Cancela |
| ↑↓←→ | Confirma e move |
| Backspace | Apaga último caractere |

### Regras de Entrada (compatível SC2)
- Letra ou `"texto` → Texto (Label)
- Número ou `+num` → Valor numérico  
- `+expr`, `@FUNC()` → Fórmula (avaliação em breve)

### Menu de Comandos (após `/`)
| Tecla | Comando |
|-------|---------|
| Q | Quit (sair) |
| S | Save (salvar) |
| L | Load (carregar) |
| Z | Zap (limpar tudo) |
| ? | Lista todos os comandos |

## Próximos Passos (Roadmap)

### Passo 2: Avaliador de Fórmulas
- Funções SC2: @SUM, @AVG, @MIN, @MAX, @IF, @COUNT, etc.
- Referências de células e intervalos (A1:B10)
- Recálculo automático

### Passo 3: Formato de Arquivo .CAL
- Leitura de arquivos .CAL do MSX
- Gravação de arquivos .CAL
- Conversão via formato SDI

### Passo 4: Comandos Completos
- /F Format (formatação de células)
- /W Width (largura de colunas)
- /C Copy, /M Move, /R Replicate
- /I Insert, /D Delete (linhas e colunas)

### Passo 5: SQLite e Persistência
- Armazenamento interno em SQLite
- Histórico de versões
- Auto-save

### Passo 6: BarGraph
- Leitura de arquivos .BGS
- Renderização de gráficos de barras em TUI

## Formato de Arquivo SC2 MSX (.CAL)

O formato .CAL do SuperCalc 2 MSX é um arquivo de texto estruturado:
- Cabeçalho com dimensões e configurações
- Dados de células em formato: `coluna,linha,tipo,valor`
- Largura das colunas
- Fórmulas em notação SC2

A implementação seguirá o formato exato para máxima compatibilidade.
