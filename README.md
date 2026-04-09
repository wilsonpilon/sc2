# SC2MSX — SuperCalc 2 para MSX, reimplementado em Go

> ⚠️ **Trabalho em progresso** — Este projeto está sendo construído aos poucos,
> com calma e cuidado, peça por peça. Não é um produto acabado.

![SC2MSX em execução](images/sc2-01.png)

---

## A história por trás do projeto

Tenho planilhas que uso **até hoje** no meu MSX. Sim, em 2025. Planilhas feitas
no **SuperCalc 2**, o software de spreadsheet da PRACTICA Informática Ltda.,
lançado no Brasil em 1989 para a plataforma MSX. Fórmulas, dados históricos,
cálculos que funcionam perfeitamente naquele ambiente — e que eu preciso continuar
usando, consultando e eventualmente expandindo.

O problema é que rodar um MSX real ou um emulador só para abrir uma planilha já
não é mais prático no dia a dia. E converter para Excel ou LibreOffice significa
perder a compatibilidade exata com o formato `.CAL` do SC2, que tem suas próprias
regras de fórmulas, formatação e estrutura de arquivo.

A solução? Reescrever o SuperCalc 2 MSX do zero, em Go, com interface de terminal
(TUI), compatível byte a byte com o formato original. Assim posso abrir, editar e
salvar minhas planilhas no computador moderno, e se precisar, levar de volta pro
MSX.

---

## O que é este projeto

**SC2MSX** é uma reimplementação do SuperCalc 2 MSX escrita em Go, usando
terminal como interface gráfica. O objetivo central é a **compatibilidade total**
com o SC2 original:

- Ler e escrever arquivos `.CAL` exatamente como o MSX gravaria
- Suportar todas as fórmulas e funções `@` do SC2 original
- Simular a tela de 80 colunas × 24 linhas do monitor MSX
- Manter o mesmo comportamento de navegação, entrada de dados e comandos

Não é uma planilha genérica. É especificamente o SuperCalc 2 MSX, para que
planilhas feitas nos anos 80 e 90 possam ser abertas sem nenhuma adaptação.

---

## Ferramentas utilizadas

### Linguagem: **Go 1.21+**

Escolhido pela simplicidade, desempenho e excelente suporte a terminal. O
compilador estático facilita a distribuição — um único binário, sem dependências
externas (exceto pelo SQLite).

```
https://go.dev
```

### Interface TUI: **tview**

Framework de interface de usuário para terminal, construído sobre o `tcell`.
Permite criar telas, caixas, modais e capturar eventos de teclado com precisão —
essencial para simular o comportamento de uma planilha em modo texto.

```
github.com/rivo/tview
```

### Suporte a terminal: **tcell v2**

Biblioteca de baixo nível para acesso ao terminal — controle de cores, células
individuais da tela, eventos de teclado e mouse. Toda a renderização da planilha
(células, cursor, cabeçalhos) é feita diretamente via `tcell.Screen`.

```
github.com/gdamore/tcell/v2
```

### Largura de caracteres: **go-runewidth**

Biblioteca que calcula a largura visual correta de cada caractere Unicode no
terminal. Fundamental para alinhar corretamente células com texto acentuado
(como "Venda Líquida" ou "Médias") sem deslocamento visual.

```
github.com/mattn/go-runewidth
```

### Banco de dados: **SQLite + go-sqlite3**

Planejado para armazenamento interno — histórico de versões de planilhas,
auto-save, metadados. O driver CGO do sqlite3 permite acesso direto e eficiente
sem servidor.

```
github.com/mattn/go-sqlite3
```

### Referências e documentação

- Manual original do **SuperCalc 2 MSX** — PRACTICA Informática Ltda., 1989
- Manual do **BarGraph** — complemento gráfico do SC2 MSX
- Manual do **SuperCalc 2** versão Amsoft/UK (inglês) — para referência técnica
- Arquivos `.BAT` originais do disco mestre do SC2 MSX (`PORT-40`, `ING-80`, etc.)

---

## Estado atual do projeto

### ✅ Implementado

**Tela de entrada (splash screen)**
- Visual fiel ao estilo MSX: fundo preto, bordas ASCII, paleta ciano/amarelo
- Entrada por `Enter`, `Esc` ou `?`

**Tela da planilha (80×24)**
- Simulação exata do monitor MSX de 80 colunas × 24 linhas
- Linha 0: status — coordenada atual, conteúdo da célula, memória simulada
- Linha 1: cabeçalho de colunas com destaque na coluna ativa
- Linhas 2–21: 20 linhas de dados com números de linha, separadores e cursor invertido
- Linhas 22–23: barra de comandos dupla (muda conforme o modo)

**Modelo de dados**
- Células esparsas (map) — eficiente para planilhas grandes com poucos dados
- Tipos: vazio, texto, número, fórmula
- Coordenadas SC2: A–Z (cols 1–26), AA–AZ (27–52), BA–BK (53–63), linhas 1–254
- Formatação: General, Integer, Fixed, Scientific, Dollar, Percent, Bar
- Scroll de viewport com ajuste automático ao cursor

**Avaliador de fórmulas (100% compatível SC2)**
- Parser recursivo completo: `+`, `-`, `*`, `/`, `^`, `( )`
- Referências de células: `A1`, `B12`, `AA3`
- Intervalos: `A1:G5` dentro de funções `@`
- **Funções `@` implementadas:**
  - Intervalo: `@SUM`, `@AVG`, `@MIN`, `@MAX`, `@COUNT`, `@STD`, `@VAR`
  - Matemática: `@ABS`, `@INT`, `@SQRT`, `@LOG`, `@LN`, `@EXP`, `@MOD`, `@ROUND`
  - Trigonometria: `@SIN`, `@COS`, `@TAN`, `@ASIN`, `@ACOS`, `@ATAN`, `@ATAN2`
  - Lógica: `@IF`, `@AND`, `@OR`, `@NOT`, `@TRUE`, `@FALSE`
  - Especiais: `@NA`, `@ISERROR`, `@PI`
- Erros nas células: `#DIV/0!`, `#REF!`, `#VALUE!`, `#NA!`, `#CIRC!`, `#ERROR!`
- Recálculo automático a cada edição
- Proteção contra referências circulares

**Navegação**
- Setas, `Tab`, `PgUp`, `PgDn`, `Home`
- Modo de entrada de dados com confirmação por `Enter` ou setas
- Modo de comandos com `/`

**Comandos disponíveis (parcial)**
- `/Q` Quit com confirmação
- `/Z` Zap (limpar planilha) com confirmação
- `/S` Save — estrutura pronta, formato `.CAL` em desenvolvimento
- `/L` Load — estrutura pronta, formato `.CAL` em desenvolvimento

---

### 🔧 Em desenvolvimento

**Formato de arquivo `.CAL`**
O próximo passo principal. Leitura e escrita dos arquivos do SC2 MSX para
poder abrir as planilhas reais do MSX.

**Comandos completos do SC2**
- `/F` Format — formatação de células (decimais, tipo, alinhamento)
- `/W` Width — largura de colunas
- `/C` Copy, `/M` Move, `/R` Replicate
- `/I` Insert, `/D` Delete (linhas e colunas)
- `/T` Title — títulos de linhas/colunas fixos
- `/G` Global — configurações globais

**Persistência SQLite**
- Auto-save
- Histórico de versões
- Metadados das planilhas

---

### 📋 Planejado

**BarGraph**
O complemento gráfico do SC2 MSX. Geração de gráficos de barras em terminal
(modo texto) a partir dos dados da planilha, compatível com o formato `.BGS`.

**Conversor SDI**
O formato intermediário `.SDI` que o SC2 usa para exportar dados. Necessário
para interoperabilidade com o MSX real.

**Impressão**
Suporte ao formato de impressão do SC2 (padrão Epson), para gerar saída
compatível com impressoras matriciais — ou exportar para PDF.

---

## Estrutura do projeto

```
sc2msx/
├── cmd/
│   └── sc2msx/
│       └── main.go              # Ponto de entrada, dados de exemplo
├── internal/
│   ├── spreadsheet/
│   │   ├── model.go             # Células, coordenadas, planilha, formatação
│   │   └── formula.go           # Parser e avaliador de fórmulas SC2
│   ├── ui/
│   │   ├── splash.go            # Tela de apresentação
│   │   └── grid.go              # Tela da planilha (80×24), modos, comandos
│   └── storage/                 # (em desenvolvimento) formato .CAL
├── go.mod
├── setup.sh                     # Script de instalação de dependências
└── README.md
```

**Tamanho atual:** ~2.100 linhas de Go em 5 arquivos.

---

## Como compilar e executar

### Requisitos

- Go 1.21 ou superior → https://go.dev/dl/
- GCC (para compilar o driver sqlite3 via CGO)
  - Linux: `sudo apt install gcc`
  - Windows: [MinGW-w64](https://www.mingw-w64.org/)
  - macOS: `xcode-select --install`

### Instalação

```bash
git clone <repo>
cd sc2msx
chmod +x setup.sh
./setup.sh
```

O script `setup.sh` busca automaticamente a versão mais recente do `tview` e
demais dependências, compila e gera o executável `./sc2msx`.

### Ou manualmente

```bash
cd sc2msx
go get github.com/rivo/tview@latest
go get github.com/gdamore/tcell/v2@latest
go get github.com/mattn/go-sqlite3@latest
go mod tidy
go build -o sc2msx ./cmd/sc2msx/
./sc2msx
```

---

## Teclas de controle

### Modo Normal (navegação)

| Tecla | Ação |
|-------|------|
| `↑ ↓ ← →` | Mover cursor |
| `Enter` | Confirma / desce uma linha |
| `Tab` | Avança uma coluna |
| `PgUp` / `PgDn` | Sobe/desce 20 linhas |
| `Home` | Vai para coluna A |
| `/` | Abre menu de comandos SC2 |
| `?` | Ajuda |
| `Del` | Apaga célula atual |

### Modo de Entrada (digitando)

| Tecla | Ação |
|-------|------|
| `Enter` | Confirma e desce |
| `Esc` | Cancela |
| `↑ ↓ ← →` | Confirma e move |
| `Backspace` | Apaga último caractere |

### Regras de entrada (compatível SC2 original)

| Início | Tipo | Exemplo |
|--------|------|---------|
| Letra ou `"` | Texto (label) | `VENDAS` ou `"Total` |
| Número | Valor numérico | `1500` ou `3.14` |
| `+`, `-`, `(`, `@` | Fórmula | `+A1+B1` ou `@SUM(A1:G1)` |

### Menu de comandos (após `/`)

| Tecla | Comando |
|-------|---------|
| `Q` | Quit — sair do programa |
| `S` | Save — salvar arquivo `.CAL` |
| `L` | Load — carregar arquivo `.CAL` |
| `Z` | Zap — limpar toda a planilha |
| `?` | Lista todos os comandos |

---

## Exemplos de fórmulas suportadas

```
+A1+B1                   soma duas celulas
+B2-B3-B4                venda liquida
@SUM(B2:G2)              soma de janeiro a junho
@AVG(B2:G2)              media mensal
@MAX(B2:G2)              maior valor do periodo
@IF(A1>0,B1,C1)          condicional
@ROUND(@AVG(B2:G2),2)    media arredondada a 2 casas
+@SUM(A1:A10)*1.1        soma com acrescimo de 10%
@SQRT(@SUM(B1:B10))      raiz da soma
```

---

## Sobre o SuperCalc 2 MSX original

O **SuperCalc 2** foi distribuído no Brasil pela **PRACTICA Informática Ltda.**
(Av. Açocê, 579 — São Paulo, SP), em 1989. Era vendido em disquete de 5¼" (face
dupla) ou 3½" (face simples), com versões para tela de 40 e 80 colunas, em
português e inglês.

O software incluía o complemento **BarGraph** para geração de gráficos de barras
diretamente da planilha, e o utilitário **SDI** para conversão de formatos.

O disco mestre continha os seguintes arquivos principais:
- `SC2.COM` — executável principal
- `SC2.OVL` — overlay (obrigatório no disco de trabalho)
- `SC2.HLP` — arquivo de ajuda
- `BARGRAPH.BAS` — programa de gráficos em BASIC
- `BG1.BIN`, `IMPGRA.BIN` — rotinas gráficas binárias

Este projeto preserva a memória desse software e garante que as planilhas criadas
nele possam continuar sendo usadas.

---

*"Copiar é crime."* — contracapa do manual original, 1989. ✦ Este projeto é uma
reimplementação independente, sem uso de código do software original.
