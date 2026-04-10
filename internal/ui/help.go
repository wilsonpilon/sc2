// internal/ui/help.go
// Sistema de Help contextual do SC2MSX
//
// Fiel ao SC2 original:
//   - ? ou F1 ativa o help a qualquer momento
//   - O conteudo muda conforme o contexto atual (modo normal, entrada, comando)
//   - Ocupa a tela inteira (sobrepoe a planilha)
//   - Qualquer tecla fecha e volta exatamente ao ponto anterior
//   - Layout 80x24, texto em branco sobre fundo azul (padrao SC2)
//
// O SC2.HLP original era um arquivo binario proprietario da Sorcim Corp.
// Esta implementacao reproduz o comportamento e conteudo equivalente.

package ui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// HelpContext define em qual contexto o help foi ativado
type HelpContext int

const (
	HelpNormal  HelpContext = iota // Modo planilha (navegacao)
	HelpInput                      // Modo de entrada de dados
	HelpCommand                    // Modo de comando (apos /)
	HelpGoto                       // Modo GoTo (apos =)
	HelpFormat                     // Apos /F
	HelpGlobal                     // Apos /G
)

// helpPage representa uma pagina de help com titulo e linhas de conteudo
type helpPage struct {
	title   string
	content []string
}

// helpPages contem todo o conteudo de help indexado por contexto
var helpPages = map[HelpContext]helpPage{

	HelpNormal: {
		title: " SUPERCALC 2 - AJUDA - MODO PLANILHA ",
		content: []string{
			"",
			" MOVIMENTACAO DO CURSOR",
			" ----------------------",
			" Seta Cima/Baixo/Esq/Dir  Mover uma celula",
			" Enter                    Mover na direcao estabelecida (padrao: baixo)",
			" Tab                      Mover para a direita",
			" Shift+Tab                Mover para a esquerda",
			" PgUp / PgDn              Mover 20 linhas para cima/baixo",
			" Home                     Ir para coluna A da linha atual",
			" Ctrl+F / Ctrl+B          Pagina de colunas direita/esquerda",
			"",
			" COMANDOS ESPECIAIS",
			" ------------------",
			" =    GoTo  - ir direto para uma celula (ex: =B12, =AA3)",
			" /    Comandos de barra - abre menu de comandos",
			" !    Recalculo manual forcado de toda a planilha",
			" ?    Esta tela de ajuda",
			" F1   Esta tela de ajuda (alternativa)",
			"",
			" ENTRADA DE DADOS",
			" ----------------",
			" Qualquer letra ou numero inicia entrada de dados",
			" \"texto   Entrada forcada como texto (label)",
			" 'padrao  Texto repetitivo (preenche a celula)",
			" +expr    Formula (ex: +A1+B1)",
			" @FUNC()  Funcao (ex: @SUM(A1:G1))",
			" Delete   Apaga o conteudo da celula atual",
			"",
			" LINHA DE STATUS (topo da tela)",
			" ------------------------------",
			" >A1 (V) 1500    Mem:62459",
			"  |   |   |        |",
			"  |   |   |        +-- Memoria disponivel",
			"  |   |   +---------- Conteudo bruto da celula",
			"  |   +--------------- Tipo: V=valor L=label F=formula",
			"  +-------------------- Coordenada da celula ativa",
			"",
			" Pressione qualquer tecla para voltar a planilha...",
		},
	},

	HelpInput: {
		title: " SUPERCALC 2 - AJUDA - ENTRADA DE DADOS ",
		content: []string{
			"",
			" TIPOS DE ENTRADA",
			" ----------------",
			" O PRIMEIRO caractere determina o tipo:",
			"",
			" \"        Texto (as aspas nao aparecem na celula)",
			"           Exemplo: \"Total do mes",
			"",
			" '        Texto repetitivo (preenche a celula com o padrao)",
			"           Exemplo: '--- ou '=",
			"",
			" Letra    Texto label (nao precisa de aspas)",
			"           Exemplo: VENDAS  JANEIRO  TOTAL",
			"",
			" Funcao   Formula automatica (sem precisar de @)",
			"           Exemplo: SUM(A1:G1)  AVG(B1:B12)  IF(A1>0,1,0)",
			"",
			" Digito   Numero puro ou formula se tiver operador",
			"           Exemplo: 1500  3.14  4+5  100/4",
			"",
			" + - ( @  Formula",
			"           Exemplo: +A1+B1  -A1  @SUM(A1:A10)  (A1+B1)/2",
			"",
			" CONFIRMACAO E CANCELAMENTO",
			" --------------------------",
			" Enter         Confirma e move para baixo",
			" Tab           Confirma e move para a direita",
			" Seta Cima     Confirma e move para cima",
			" Seta Baixo    Confirma e move para baixo",
			" Seta Esq      Apaga ultimo caractere (NAO move)",
			" Seta Dir      Confirma e move para a direita",
			" Esc           Cancela (celula nao e alterada)",
			" Ctrl+Z / F2   Cancela e limpa a linha",
			"",
			" LIMITES",
			" -------",
			" Texto:   ate 115 caracteres",
			" Formula: ate 116 caracteres",
			" Numero:  ate 16 digitos significativos",
			"          Maximo: 9.999999999999999e62",
			"          Minimo: -1.0e-64",
			"",
			" Pressione qualquer tecla para voltar...",
		},
	},

	HelpCommand: {
		title: " SUPERCALC 2 - AJUDA - COMANDOS DE BARRA (/) ",
		content: []string{
			"",
			" Pressione a letra do comando desejado. Esc cancela.",
			"",
			" /A  Arrange    Ordena linhas ou colunas",
			" /B  Blank      Apaga o conteudo de uma celula ou intervalo",
			" /C  Copy       Copia celulas para outro local",
			" /D  Delete     Apaga linhas, colunas ou arquivo",
			" /E  Edit       Edita o conteudo de uma celula",
			" /F  Format     Define formato de exibicao das celulas",
			"      I=Inteiro  G=Geral  E=Exponencial  $=Dinheiro",
			"      %=Percent  *=Barra  R=Dir  L=Esq  TR/TL=Texto",
			" /G  Global     Configuracoes globais da planilha",
			"      Formula display  Next move  Border  Tab lockout",
			"      Row/Col calc order  Manual/Auto recalculate",
			" /I  Insert     Insere linhas ou colunas em branco",
			" /L  Load       Carrega planilha do disco (.SDI ou .CAL)",
			" /M  Move       Move linhas ou colunas",
			" /O  Output     Configuracao de impressao",
			" /P  Protect    Protege celulas contra alteracao",
			" /Q  Quit       Sai do SC2MSX",
			" /R  Replicate  Replica celulas com ajuste de formulas",
			" /S  Save       Salva planilha no disco (.SDI)",
			" /T  Title      Trava linha/coluna de titulo (freeze)",
			" /U  Unprotect  Remove protecao de celulas",
			" /W  Width      Define largura da coluna atual (1-72)",
			"      Padrao: 9 caracteres",
			" /X  Execute    Executa arquivo de macro (.XQT)",
			" /Z  Zap        Apaga TODA a planilha",
			"",
			" Pressione qualquer tecla para voltar...",
		},
	},

	HelpGoto: {
		title: " SUPERCALC 2 - AJUDA - GOTO (=) ",
		content: []string{
			"",
			" O comando GoTo move o cursor diretamente para qualquer celula.",
			"",
			" USO",
			" ---",
			" Pressione = e digite a coordenada da celula destino:",
			"",
			"   =A1     vai para a celula A1",
			"   =B12    vai para a celula B12",
			"   =AA3    vai para a celula AA3 (coluna dupla)",
			"   =BK254  vai para a ultima celula possivel",
			"",
			" Pressione Enter para confirmar ou Esc para cancelar.",
			"",
			" COORDENADAS",
			" -----------",
			" Colunas: A a Z (1-26), AA a AZ (27-52), BA a BK (53-63)",
			" Linhas:  1 a 254",
			"",
			" Se a celula destino nao estiver visivel na tela atual,",
			" a planilha se reposiciona com a celula no canto",
			" superior esquerdo da area de dados.",
			"",
			" GoTo sem celula (apenas Enter) reposiciona a celula",
			" ativa no canto superior esquerdo da tela.",
			"",
			" Pressione qualquer tecla para voltar...",
		},
	},

	HelpFormat: {
		title: " SUPERCALC 2 - AJUDA - FORMATO (/F) ",
		content: []string{
			"",
			" CODIGOS DE FORMATO",
			" ------------------",
			" I   Integer    Inteiro sem decimais         1500",
			" G   General    Inteiro ou float automatico  1500 / 3.14",
			" E   Exponenc.  Notacao cientifica           1.5e3",
			" $   Dollar     Duas casas decimais          1500.00",
			" %   Percent    Multiplica por 100           75.00%",
			" *   Bar graph  Asteriscos proporcionais     ***",
			" R   Right      Alinha numeros a direita",
			" L   Left       Alinha numeros a esquerda",
			" TR  Text Right Alinha texto a direita",
			" TL  Text Left  Alinha texto a esquerda",
			" H   Hide       Esconde o valor (celula parece vazia)",
			" D   Default    Restaura formato padrao",
			"",
			" NIVEIS DE FORMATO (prioridade decrescente)",
			" -------------------------------------------",
			" 1. Entry  - formato especifico da celula (maior prioridade)",
			" 2. Row    - formato da linha inteira",
			" 3. Column - formato da coluna inteira",
			" 4. Global - formato padrao da planilha (menor prioridade)",
			"",
			" LARGURA DA COLUNA (/W)",
			" ----------------------",
			" Padrao: 9 caracteres",
			" Se o numero nao couber: ********* (asteriscos)",
			" Texto e truncado na borda (sem overflow)",
			"",
			" Pressione qualquer tecla para voltar...",
		},
	},

	HelpGlobal: {
		title: " SUPERCALC 2 - AJUDA - GLOBAL (/G) ",
		content: []string{
			"",
			" CONFIGURACOES GLOBAIS",
			" ----------------------",
			" Formula display  Mostra formulas ou valores nas celulas",
			" Next move        Direcao do cursor apos Enter",
			" Border display   Exibe ou oculta bordas da planilha",
			" Tab lockout      Trava o Tab na area definida",
			" Calc order       Ordem de calculo: por linha ou coluna",
			" Recalculate      Automatico (a cada edicao) ou Manual (!)",
			"",
			" RECALCULO",
			" ---------",
			" Auto:   toda edicao dispara recalculo completo",
			" Manual: use ! para recalcular quando quiser",
			"",
			" ORDEM DE CALCULO",
			" ----------------",
			" Row (Linha):   calcula da esquerda para a direita,",
			"                de cima para baixo",
			" Column (Col):  calcula de cima para baixo,",
			"                da esquerda para a direita",
			"",
			" Pressione qualquer tecla para voltar...",
		},
	},
}

// ─── HelpView ────────────────────────────────────────────────────────────────

// ShowHelp exibe a tela de help contextual sobre a planilha
// Retorna um *tview.Box que deve ser adicionado como pagina modal
func NewHelpView(ctx HelpContext, onClose func()) *tview.Box {
	page, ok := helpPages[ctx]
	if !ok {
		page = helpPages[HelpNormal]
	}

	box := tview.NewBox()
	box.SetDrawFunc(func(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
		// Fundo azul escuro (padrao SC2 para telas de help)
		bgStyle := tcell.StyleDefault.
			Background(tcell.ColorNavy).
			Foreground(tcell.ColorWhite)

		titleStyle := tcell.StyleDefault.
			Background(tcell.ColorTeal).
			Foreground(tcell.ColorWhite).
			Bold(true)

		contentStyle := tcell.StyleDefault.
			Background(tcell.ColorNavy).
			Foreground(tcell.ColorWhite)

		hlStyle := tcell.StyleDefault.
			Background(tcell.ColorNavy).
			Foreground(tcell.ColorYellow).
			Bold(true)

		// Preenche fundo
		for row := y; row < y+height; row++ {
			for col := x; col < x+width; col++ {
				screen.SetContent(col, row, ' ', nil, bgStyle)
			}
		}

		// Borda superior
		for col := x; col < x+width; col++ {
			screen.SetContent(col, y, ' ', nil, titleStyle)
		}

		// Titulo centralizado
		title := page.title
		if len(title) > width-2 {
			title = title[:width-2]
		}
		tx := x + (width-len(title))/2
		for i, ch := range title {
			screen.SetContent(tx+i, y, ch, nil, titleStyle)
		}

		// Borda inferior do titulo
		for col := x; col < x+width; col++ {
			screen.SetContent(col, y+1, '-', nil, titleStyle)
		}

		// Conteudo
		maxLines := height - 4 // reserva titulo (2) + rodape (2)
		for i, line := range page.content {
			if i >= maxLines {
				break
			}
			screenY := y + 2 + i

			// Detecta linhas de destaque (secoes com CAPS e "---")
			st := contentStyle
			trimmed := strings.TrimSpace(line)
			if len(trimmed) > 0 &&
				trimmed == strings.ToUpper(trimmed) &&
				!strings.HasPrefix(trimmed, "/") &&
				!strings.HasPrefix(trimmed, "=") &&
				!strings.HasPrefix(trimmed, "+") &&
				!strings.HasPrefix(trimmed, "\"") {
				st = hlStyle
			}

			// Desenha a linha
			col := x
			for _, ch := range line {
				if col >= x+width {
					break
				}
				screen.SetContent(col, screenY, ch, nil, st)
				col++
			}
			// Preenche resto da linha
			for col < x+width {
				screen.SetContent(col, screenY, ' ', nil, bgStyle)
				col++
			}
		}

		// Rodape
		footerY := y + height - 1
		footer := " SC2MSX Help | ? ou F1 = Help contextual "
		for col := x; col < x+width; col++ {
			screen.SetContent(col, footerY, ' ', nil, titleStyle)
		}
		for i, ch := range footer {
			if x+i >= x+width {
				break
			}
			screen.SetContent(x+i, footerY, ch, nil, titleStyle)
		}

		return x, y, width, height
	})

	// Qualquer tecla fecha o help (comportamento SC2)
	box.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		onClose()
		return nil
	})

	return box
}
