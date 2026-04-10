# SC2MSX — Manual do Usuário

**SuperCalc 2 para MSX/PC · Versão Go/TUI**

> Este manual descreve o SC2MSX, uma reimplementação fiel do SuperCalc 2 MSX
> para computadores modernos com interface de terminal. O comportamento é
> intencionalmente idêntico ao SC2 original para MSX.

---

## Índice

1. [Início de Operação](#1-início-de-operação)
2. [A Tela da Planilha](#2-a-tela-da-planilha)
3. [Movimentando o Cursor](#3-movimentando-o-cursor)
4. [Entrada de Dados](#4-entrada-de-dados)
5. [Fórmulas e Funções](#5-fórmulas-e-funções)
6. [Os Comandos de Barra `/`](#6-os-comandos-de-barra-)
7. [Arquivos — Salvar e Carregar](#7-arquivos--salvar-e-carregar)
8. [Formatação de Células](#8-formatação-de-células)
9. [Mensagens de Erro](#9-mensagens-de-erro)
10. [Referência Rápida de Teclas](#10-referência-rápida-de-teclas)
11. [Referência Completa de Funções](#11-referência-completa-de-funções)

---

## 1. Início de Operação

### Executando o SC2MSX

```bash
./sc2msx
```

A tela de apresentação aparecerá. Pressione `Enter`, `Esc` ou `?` para entrar
na planilha.

### A Tela de Apresentação

```
+----------------------------------+
|                                  |
|    S U P E R C A L C  2  M S X  |
|                                  |
|   Planilha de Calculo Eletron.   |
|         para  M S X              |
|                                  |
+----------------------------------+

      Versao 2.0 - Compativel com SC2 MSX

      Pressione  ENTER  para continuar
```

---

## 2. A Tela da Planilha

A planilha simula exatamente o monitor MSX de **80 colunas × 24 linhas**:

```
>A1   (V) 1500                                              Mem:62459  ← Linha 0: Status
    |  A  |  B  |  C  |  D  |  E  |  F  |  G  |  H  |  I  ← Linha 1: Colunas
  1 |VENDA| 1500| 1300| 1800| 3500| 3200|11200|24500|     ← Linha 2
  2 |CUSTO|  756|  650|  900| 1750| 2600| 3600|10256|     ← Linha 3
  3 |     |     |     |     |     |     |     |     |     ← ...
  ...                                                      (20 linhas de dados)
 Enter: /=Comandos  =GoTo  !Recalc  ?=Ajuda  Del=Apaga     ← Linha 22
 A1: (V) 1500                                              ← Linha 23
```

### Linha de Status (Linha 0)

A linha de status no topo contém:

| Campo | Significado |
|---|---|
| `>` | Indicador de direção do cursor (`>` = para baixo ao confirmar) |
| `A1` | Coordenada da célula ativa |
| `(V)` | Tipo: `V`=valor · `L`=label/texto · `F`=fórmula · ` `=vazio |
| Conteúdo | O conteúdo bruto da célula (fórmulas mostram a expressão, não o resultado) |
| `Mem:NNNNN` | Memória disponível (simulada) |

**Exemplo:** `>B5   (F) @SUM(B2:B4)` indica que a célula B5 contém uma fórmula
de soma. O valor calculado (`6506`) aparece na própria célula na grade.

### Cabeçalho de Colunas (Linha 1)

As colunas são nomeadas de `A` a `BK` (63 colunas no total):
- Colunas simples: `A` a `Z` (1–26)
- Colunas duplas: `AA` a `AZ` (27–52), `BA` a `BK` (53–63)

A coluna onde o cursor está aparece destacada em vídeo invertido.

### Área de Dados (Linhas 2–21)

Exibe **20 linhas** da planilha de cada vez. O número da linha aparece à
esquerda (1–254). A linha do cursor aparece com o número destacado.

### Barra de Comandos (Linhas 22–23)

Muda conforme o modo de operação:

**Modo Normal:**
```
 Enter: /=Comandos  =GoTo  !Recalc  ?=Ajuda  Del=Apaga
 A1: (V) 1500
```

**Modo de Entrada:**
```
 Enter: VENDAS_
 [Enter]:Confirma  [Esc]:Cancela  [Setas]:Confirma e move
```

**Modo de Comando (após `/`):**
```
 > /S
 B:Blank C:Copy D:Delete F:Format G:Global I:Insert L:Load M:Move ...
```

---

## 3. Movimentando o Cursor

### Teclas de Movimento

| Tecla | Movimento |
|---|---|
| `↑` `↓` `←` `→` | Uma célula na direção da seta |
| `Enter` | Uma linha para baixo (confirma direção) |
| `Tab` | Uma coluna para a direita |
| `Shift+Tab` | Uma coluna para a esquerda |
| `PgDn` | 20 linhas para baixo |
| `PgUp` | 20 linhas para cima |
| `Home` | Vai para a coluna A da linha atual |
| `Ctrl+F` | Uma "página" de colunas para a direita |
| `Ctrl+B` | Uma "página" de colunas para a esquerda |

### O Comando GoTo (`=`)

Para ir diretamente a uma célula, pressione `=` e digite a coordenada:

```
=B12    → vai para B12
=AA3    → vai para AA3
=BK254  → vai para a última célula possível
```

Pressione `Enter` para confirmar ou `Esc` para cancelar. Se a célula destino
estiver fora da janela atual, a planilha se reposiciona automaticamente com a
célula no canto superior esquerdo.

**Nota:** O SC2 original usa a tecla `=` para GoTo — o mesmo comportamento
foi mantido no SC2MSX.

### Scrolling

Ao mover o cursor além da borda visível da tela, a planilha faz "scroll"
automaticamente para mostrar a nova posição. A janela sempre se ajusta para
manter o cursor visível.

---

## 4. Entrada de Dados

### Como Entrar Dados

Posicione o cursor na célula desejada e comece a digitar. O SC2 determina
o tipo de dado pelo **primeiro caractere digitado**:

| Primeiro caractere | Tipo resultante | Exemplo |
|---|---|---|
| `"` | **Texto** (aspas não aparecem na célula) | `"Total do mês` |
| `'` | **Texto repetido** (preenche a célula) | `'---` ou `'=` |
| Letra `A`–`Z` | **Texto** (label) | `VENDAS` |
| Nome de função | **Fórmula** (automático) | `SUM(A1:G1)` |
| Referência de célula | **Fórmula** (automático) | `A1` |
| Dígito `0`–`9` ou `.` | **Número** (se puro) | `1500` ou `3.14` |
| Dígito + operador | **Fórmula** (automático) | `4+5` ou `100/4` |
| `+` `-` `(` `@` | **Fórmula** | `+A1+B1` ou `@SUM(A1:A10)` |

### Confirmando a Entrada

| Tecla | Ação |
|---|---|
| `Enter` | Confirma e move o cursor **para baixo** |
| `Tab` | Confirma e move o cursor **para a direita** |
| `↑` `↓` `←` `→` | Confirma e move na direção da seta |
| `Esc` | **Cancela** a entrada (célula não é alterada) |
| `Ctrl+Z` ou `F2` | Cancela e limpa a linha de entrada |

### Apagando uma Célula

Com o cursor na célula, pressione `Delete` ou `Backspace`. O conteúdo é
removido e a célula fica vazia. O equivalente em comandos é `/B` (Blank).

### Corrigindo Durante a Entrada

Enquanto digita, `Backspace` apaga o último caractere. Use `Esc` para
cancelar tudo e voltar ao modo normal sem alterar a célula.

---

## 5. Fórmulas e Funções

### Tipos de Fórmula

Uma fórmula é uma expressão matemática que calcula um valor. No SC2, fórmulas
podem começar com `+`, `-`, `(`, `@`, ou com um número seguido de operador.

```
+A1+B1          soma A1 e B1
-A1             negativo de A1
(A1+B1)/2       média de dois valores
4+5             calcula 9
@SUM(A1:A10)    soma do intervalo
SUM(A1:A10)     equivalente (SC2MSX aceita sem @)
```

### Referências de Células

Uma referência identifica uma célula pelo nome da coluna e número da linha:

```
A1          coluna A, linha 1
B12         coluna B, linha 12
AA3         coluna AA, linha 3 (colunas duplas: AA a BK)
BK254       última célula possível
```

### Intervalos

Um intervalo especifica um bloco retangular de células, separando os cantos
por dois-pontos:

```
A1:A10      coluna A, linhas 1 a 10
B2:G2       linha 2, colunas B a G
B2:G5       bloco 6×4 colunas
```

Intervalos são usados principalmente dentro de funções:
`@SUM(B2:G2)`, `@AVG(A1:A20)`, `@MAX(B1:B12)`.

### Operadores Aritméticos

| Operador | Operação | Exemplo |
|---|---|---|
| `+` | Adição | `A1+B1` |
| `-` | Subtração | `A1-B1` |
| `*` | Multiplicação | `A1*2` |
| `/` | Divisão | `A1/B1` |
| `^` ou `**` | Potência | `A1^2` |

### Operadores Relacionais

Retornam **1** (verdadeiro) ou **0** (falso):

| Operador | Significado | Exemplo |
|---|---|---|
| `=` | Igual | `A1=100` |
| `<>` | Diferente | `A1<>0` |
| `<` | Menor que | `A1<B1` |
| `>` | Maior que | `A1>B1` |
| `<=` | Menor ou igual | `A1<=100` |
| `>=` | Maior ou igual | `A1>=B1` |

Usados principalmente dentro de `@IF`:
```
@IF(A1>0,B1,C1)           se A1 > 0, usa B1; senão usa C1
@IF(AND(A1>0,A1<100),1,0) verdadeiro se A1 entre 0 e 100
```

### Recálculo

Toda vez que uma célula é alterada, **todas as fórmulas** da planilha são
recalculadas automaticamente. Para forçar um recálculo manual, pressione `!`.

---

## 6. Os Comandos de Barra `/`

Pressione `/` para entrar no modo de comandos. A linha de aviso mostra
todas as opções disponíveis. Pressione a letra do comando desejado ou
`Esc` para cancelar.

```
ENTER: B,C,D,F,G,I,L,M,O,P,Q,R,S,T,U,W,X,Z,?
```

### `/B` — Blank (Apagar célula)

Apaga o conteúdo da célula ativa. Equivalente a pressionar `Delete`.

```
/B    → apaga a célula atual
```

### `/D` — Delete (Apagar linhas ou colunas)

Apaga uma linha inteira ou uma coluna inteira. As linhas ou colunas
adjacentes se movem para preencher o espaço. Fórmulas que referenciavam
células deletadas passam a exibir `#REF!`.

```
/D → escolha: Linha | Coluna
```

### `/F` — Format (Formatar)

Define o formato de exibição da célula ativa. O formato afeta apenas como
o valor é mostrado na tela — não altera o valor em si.

| Código | Formato | Exemplo |
|---|---|---|
| `G` | General — inteiro se possível, senão float | `1500` ou `3.14` |
| `I` | Integer — sem decimais | `1500` |
| `F` | Fixed — ponto fixo com N casas | `1500.00` |
| `E` | Exponential — notação científica | `1.5e3` |
| `$` | Dollar — 2 casas decimais | `1500.00` |
| `%` | Percent — multiplica por 100 | `75.00%` |
| `*` | Bar graph — asteriscos proporcionais | `***` |
| `R` | Right — alinha à direita |  |
| `L` | Left — alinha à esquerda |  |
| `TR` | Text Right — texto à direita |  |
| `TL` | Text Left — texto à esquerda |  |

### `/G` — Global (Configurações globais)

Acessa configurações que afetam toda a planilha. Atualmente disponível:
recálculo manual forçado.

### `/I` — Insert (Inserir)

Insere uma linha ou coluna em branco na posição do cursor. Todo o conteúdo
após a inserção é deslocado.

```
/I → escolha: Linha | Coluna
```

### `/L` — Load (Carregar arquivo)

Carrega uma planilha de um arquivo no formato SDI (`.SDI` ou `.CAL`).

```
/L → digite o nome do arquivo → Enter
```

Exemplo: `PLANILHA.SDI` ou `VENDAS.CAL`

### `/Q` — Quit (Sair)

Encerra o SC2MSX. Uma confirmação é pedida para evitar sair por engano.

```
/Q → "Deseja sair?" → Sim | Não
```

### `/S` — Save (Salvar)

Salva a planilha atual em um arquivo no formato SDI.

```
/S → digite o nome do arquivo → Enter
```

Se o arquivo já tem um nome (foi carregado ou salvo antes), o nome atual
aparece como sugestão. Pressione `Enter` para confirmar ou digite outro nome.

Extensões aceitas: `.SDI` (padrão) ou `.CAL`.

### `/W` — Width (Largura de coluna)

Define a largura em caracteres da coluna onde o cursor está.

```
/W → digite a largura (1–72) → Enter
```

A largura padrão é **9** caracteres, igual ao SC2 original.

### `/Z` — Zap (Limpar tudo)

Apaga **toda** a planilha, voltando ao estado inicial. Uma confirmação
é pedida pois a operação não pode ser desfeita.

```
/Z → "Apagar TODA a planilha?" → Sim | Não
```

---

## 7. Arquivos — Salvar e Carregar

### Formato SDI

O SC2MSX usa o formato **SDI** (SuperData Interchange), o formato de texto
nativo do SuperCalc 2. É o mesmo formato produzido pelo utilitário `SDI.COM`
do MSX ao converter arquivos `.CAL`.

O arquivo SDI é texto ASCII puro e pode ser inspecionado em qualquer editor.
Exemplo de um arquivo SDI simples:

```
TABLE
0,1
""
GDISP-FORMAT
9,0
GTL
DATA
0,0
""
-1,0
BOT
1,0
VENDAS
0,1500
V
0,1300
V
-4,0
@SUM(B2:C2)
-1,0
EOD
```

### Salvando

```
/S → Nome do arquivo [PLANILHA.SDI]: VENDAS → Enter
```

Após salvar, a barra de status confirma:
`Salvo: VENDAS.SDI (42 celulas)`

### Carregando

```
/L → Nome do arquivo (.SDI ou .CAL): VENDAS → Enter
```

O arquivo é carregado, todas as fórmulas são recalculadas, e a planilha
é exibida a partir da célula A1.

### Compatibilidade com o MSX

Para usar um arquivo gerado no SC2MSX no MSX real:
1. Salve como `.SDI` no SC2MSX
2. Copie o arquivo para um disco MSX
3. No MSX, execute `SDI` e converta de `.SDI` para `.CAL`
4. Abra o `.CAL` normalmente no SuperCalc 2 MSX

Para o caminho inverso (do MSX para o PC):
1. No MSX, execute `SDI` e converta o `.CAL` para `.SDI`
2. Copie o `.SDI` para o PC
3. Abra com `/L` no SC2MSX

---

## 8. Formatação de Células

### Prioridade de Formatação

O SC2 aplica formatos em quatro níveis, do menor para o maior:

```
4. Global  (menor prioridade)
3. Coluna
2. Linha
1. Entrada  (maior prioridade — sobrepõe todos)
```

Ao formatar com `/F`, você está definindo o formato de **Entrada** da célula,
que tem a maior prioridade.

### Largura de Coluna

A largura padrão é **9 caracteres**. Use `/W` para alterar. Se um número
não couber na largura definida, a célula exibe `*********` (asteriscos).

Para texto, o conteúdo é simplesmente truncado na borda — sem overflow para
a célula vizinha (comportamento idêntico ao SC2 original).

---

## 9. Mensagens de Erro

Quando uma fórmula não pode ser calculada, a célula exibe um código de erro:

| Código | Causa |
|---|---|
| `#DIV/0!` | Divisão por zero |
| `#REF!` | Referência inválida (célula inexistente) |
| `#VALUE!` | Tipo de dado incompatível com a operação |
| `N/A` | Valor não disponível (`@NA`) |
| `#CIRC!` | Referência circular detectada |
| `#ERROR!` | Erro geral de avaliação |
| `*********` | Número não cabe na largura da coluna |

Células com erro propagam o erro: uma fórmula que referencia uma célula
com `#DIV/0!` também exibirá `#DIV/0!`.

---

## 10. Referência Rápida de Teclas

### Modo Normal (Navegação)

| Tecla | Ação |
|---|---|
| `↑ ↓ ← →` | Mover cursor uma célula |
| `Enter` | Mover para baixo |
| `Tab` | Mover para a direita |
| `Shift+Tab` | Mover para a esquerda |
| `PgUp` / `PgDn` | Mover 20 linhas |
| `Home` | Ir para coluna A |
| `Ctrl+F` | Avançar página de colunas |
| `Ctrl+B` | Recuar página de colunas |
| `=` | GoTo — ir para célula específica |
| `/` | Abrir menu de comandos |
| `!` | Forçar recálculo manual |
| `?` | Exibir ajuda rápida |
| `Del` | Apagar célula atual |

### Modo de Entrada (Digitando)

| Tecla | Ação |
|---|---|
| `Enter` | Confirmar e mover para baixo |
| `Tab` | Confirmar e mover para a direita |
| `↑ ↓ ← →` | Confirmar e mover na direção |
| `Esc` | Cancelar (não altera a célula) |
| `Backspace` | Apagar último caractere |
| `Ctrl+Z` ou `F2` | Cancelar e limpar linha |

### Modo de Comando (após `/`)

| Tecla | Comando |
|---|---|
| `B` | Blank — apagar célula |
| `D` | Delete — apagar linha ou coluna |
| `F` | Format — formatar célula |
| `G` | Global — configurações globais |
| `I` | Insert — inserir linha ou coluna |
| `L` | Load — carregar arquivo |
| `Q` | Quit — sair |
| `S` | Save — salvar arquivo |
| `W` | Width — largura da coluna |
| `Z` | Zap — limpar tudo |
| `Esc` | Cancelar — voltar ao modo normal |
| `?` | Listar todos os comandos |

### Modo GoTo (após `=`)

| Tecla | Ação |
|---|---|
| Letras e dígitos | Digitar coordenada (ex: `B12`) |
| `Enter` | Confirmar e ir para a célula |
| `Esc` | Cancelar |

---

## 11. Referência Completa de Funções

Todas as funções podem ser usadas com ou sem o prefixo `@`.
`SUM(A1:A10)` e `@SUM(A1:A10)` são equivalentes.

### Funções Aritméticas

| Função | Descrição | Exemplo |
|---|---|---|
| `@ABS(valor)` | Valor absoluto | `@ABS(-237)` = 237 |
| `@INT(valor)` | Parte inteira (trunca) | `@INT(2.9)` = 2 |
| `@SQRT(valor)` | Raiz quadrada | `@SQRT(4)` = 2 |
| `@LOG(valor)` | Logaritmo base 10 | `@LOG(100)` = 2 |
| `@LOG10(valor)` | Logaritmo base 10 (alias) | `@LOG10(1000)` = 3 |
| `@LN(valor)` | Logaritmo natural | `@LN(1)` = 0 |
| `@EXP(valor)` | e elevado ao valor | `@EXP(1)` ≈ 2.718 |
| `@MOD(v1,v2)` | Resto da divisão | `@MOD(10,3)` = 1 |
| `@ROUND(v,casas)` | Arredonda | `@ROUND(3.567,2)` = 3.57 |
| `@PI` | Valor de π | `@PI` = 3.14159... |

### Funções Trigonométricas

Ângulos em **radianos**.

| Função | Descrição | Exemplo |
|---|---|---|
| `@SIN(valor)` | Seno | `@SIN(@PI/2)` = 1 |
| `@COS(valor)` | Coseno | `@COS(@PI)` = -1 |
| `@TAN(valor)` | Tangente | `@TAN(0)` = 0 |
| `@ASIN(valor)` | Arco-seno | `@ASIN(1)` ≈ 1.5708 |
| `@ACOS(valor)` | Arco-coseno | `@ACOS(1)` = 0 |
| `@ATAN(valor)` | Arco-tangente | `@ATAN(1)` ≈ 0.7854 |
| `@ATAN2(v1,v2)` | Arco-tangente de v1/v2 | `@ATAN2(1,1)` ≈ 0.7854 |

### Funções de Intervalo e Lista

Aceitam intervalos (`A1:A10`), listas (`A1,B3,C5`) ou combinações.

| Função | Descrição | Exemplo |
|---|---|---|
| `@SUM(lista)` | Soma | `@SUM(B2:G2)` |
| `@AVG(lista)` | Média aritmética | `@AVG(A1:A12)` |
| `@AVERAGE(lista)` | Média (alias de AVG) | `@AVERAGE(A1:A12)` |
| `@MIN(lista)` | Menor valor | `@MIN(B2:B13)` |
| `@MAX(lista)` | Maior valor | `@MAX(B2:B13)` |
| `@COUNT(lista)` | Conta células não vazias | `@COUNT(A1:A20)` |
| `@STD(lista)` | Desvio padrão | `@STD(B2:B13)` |
| `@VAR(lista)` | Variância | `@VAR(B2:B13)` |
| `@NPV(taxa,lista)` | Valor presente líquido | `@NPV(0.1,B1:F1)` |

### Funções Lógicas

| Função | Descrição | Exemplo |
|---|---|---|
| `@IF(cond,v_sim,v_nao)` | Condicional | `@IF(A1>0,B1,0)` |
| `@AND(v1,v2)` | E lógico | `@AND(A1>0,A1<100)` |
| `@OR(v1,v2)` | OU lógico | `@OR(A1=1,A1=2)` |
| `@NOT(valor)` | NÃO lógico | `@NOT(A1=0)` |
| `@TRUE` | Verdadeiro (1) | `@TRUE` = 1 |
| `@FALSE` | Falso (0) | `@FALSE` = 0 |

**Combinações úteis:**
```
@IF(AND(A1>500,A1<1000),5,0)   → 5 se A1 entre 500 e 1000
@IF(OR(A1>5000,B1<100),5,0)   → 5 se A1>5000 OU B1<100
@IF(NOT(A1=0),B1/A1,0)        → evita divisão por zero
```

### Funções Especiais

| Função | Descrição |
|---|---|
| `@NA` | Marca a célula como "Não Disponível" (exibe `N/A`) |
| `@ERROR` | Força exibição de `#ERROR!` |
| `@ISERROR(valor)` | 1 se valor for ERROR, 0 caso contrário |
| `@ISNA(valor)` | 1 se valor for N/A, 0 caso contrário |
| `@LOOKUP(chave,tabela)` | Busca valor em tabela ordenada |

**Exemplo de @IF com @ISERROR:**
```
@IF(ISERROR(A1/B1),0,A1/B1)   → divide A1 por B1, retorna 0 se der erro
```

### Funções de Calendário

| Função | Descrição | Exemplo |
|---|---|---|
| `@DATE(MM,DD,YY)` | Cria valor de data | `@DATE(12,31,99)` |
| `@DATE(MM,DD,YYYY)` | Com ano de 4 dígitos | `@DATE(1,1,2026)` |
| `@MONTH(data)` | Mês da data | `@MONTH(A1)` |
| `@DAY(data)` | Dia da data | `@DAY(A1)` |
| `@YEAR(data)` | Ano da data | `@YEAR(A1)` |
| `@WDAY(data)` | Dia da semana (1=Dom) | `@WDAY(A1)` |
| `@JDATE(data)` | Data Juliana Modificada | `@JDATE(A1)` |

---

## Exemplos Práticos

### Planilha de Vendas Mensal

```
     A           B      C      D      E      F      G      H
1    -           JAN    FEV    MAR    ABR    MAI    JUN    TOTAL
2    VENDA BRUTA 1500   1300   1800   3500   3200   11200  @SUM(B2:G2)
3    CUSTO1      756    650    900    1750   2600   3600   @SUM(B3:G3)
4    CUSTO2      255    221    306    595    884    1704   @SUM(B4:G4)
5    VENDA LIQ.  +B2-B3-B4  +C2-C3-C4  ...        @SUM(B5:G5)
6    ------------
7    MEDIA/MES   @AVG(B2:G2)
8    MAX MES     @MAX(B2:G2)
9    MIN MES     @MIN(B2:G2)
10   DESVIO      @STD(B2:G2)
```

### Cálculo de Financiamento

```
A1: Taxa mensal:  0.015
A2: Parcelas:     24
A3: Valor total:  @NPV(A1,B1:B24)
```

### Classificação com @IF

```
A1: Nota do aluno
B1: @IF(A1>=7,"APROVADO",@IF(A1>=5,"RECUPERACAO","REPROVADO"))
```

---

*Manual do SC2MSX · Trabalho em progresso · Consulte o README.md para novidades.*
