// internal/ui/grid.go
// Tela principal da planilha SC2 MSX - fiel ao original
//
// Layout exato do SuperCalc 2 MSX (80 colunas x 24 linhas):
//
//  Linha  0: [coord:valor ] [ conteudo da celula           ] [mem:NNNNN]
//  Linha  1: [   ][  A  ][  B  ][  C  ]...
//  Linhas 2-21: [NNN][dado ][dado ][dado ]...   (20 linhas de dados)
//  Linha 22: barra de entrada / status
//  Linha 23: barra de modo / ajuda rapida
//
// Comportamento SC2 original:
//  - Cursor invertido na celula atual
//  - Linha de status mostra: coordenada, tipo (V/L/F), conteudo bruto
//  - Entrada: qualquer tecla alfanumerica inicia edicao
//  - "/" abre menu de comandos (linha 22 mostra opcoes)
//  - Formulas mostram valor na celula, formula na linha de status
//  - Overflow de numero: "**********" (asteriscos)
//  - Texto mais largo que coluna: truncado (sem overflow para proxima)

package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/rivo/tview"

	"github.com/sc2msx/internal/spreadsheet"
)

// Layout fixo SC2 MSX
const (
	SC2_TOTAL_ROWS    = 24                                              // Altura total da tela MSX
	SC2_TOTAL_COLS    = 80                                              // Largura total da tela MSX
	SC2_HEADER_ROWS   = 2                                               // Linha de status + linha de colunas
	SC2_CMD_ROWS      = 2                                               // Duas linhas de barra de comandos
	SC2_DATA_ROWS     = SC2_TOTAL_ROWS - SC2_HEADER_ROWS - SC2_CMD_ROWS // 20
	SC2_ROW_NUM_W     = 3                                               // Largura do numero de linha (ex: "  1")
	SC2_SEP_W         = 1                                               // Separador "|"
	SC2_DEFAULT_COL_W = 9                                               // Largura padrao de coluna no SC2 original
)

// Modo de operacao da planilha
type InputMode int

const (
	ModeNormal  InputMode = iota // Navegacao
	ModeInput                    // Digitando conteudo
	ModeCommand                  // Menu "/" aberto
	ModeGoto                     // Goto celula (Ctrl+G ou F5)
)

// GridView e a view principal da planilha
type GridView struct {
	*tview.Box

	sheet       *spreadsheet.Spreadsheet
	mode        InputMode
	inputBuffer string // Buffer do que o usuario esta digitando
	statusMsg   string // Mensagem temporaria de status
	onCommand   func(string)
}

// NewGridView cria a view principal
func NewGridView(sheet *spreadsheet.Spreadsheet) *GridView {
	g := &GridView{
		Box:   tview.NewBox(),
		sheet: sheet,
		mode:  ModeNormal,
	}
	g.SetBorder(false)
	g.SetInputCapture(g.handleInput)
	return g
}

func (g *GridView) SetCommandHandler(fn func(string)) { g.onCommand = fn }
func (g *GridView) SetStatus(msg string)              { g.statusMsg = msg }

// ─── Renderizacao ─────────────────────────────────────────────────────────────

func (g *GridView) Draw(screen tcell.Screen) {
	g.Box.DrawForSubclass(screen, g)
	x, y, width, _ := g.GetInnerRect()

	// ── Paleta de cores fiel ao SC2 MSX original ──
	// SC2 usava: fundo preto, texto branco, cursor invertido (preto no branco)
	// Cabecalho de linha/coluna: azul escuro
	// Linha de status: azul com texto amarelo/branco
	// Barra de comandos: azul escuro

	sNormal := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorSilver)
	sHeader := tcell.StyleDefault.Background(tcell.ColorNavy).Foreground(tcell.ColorWhite).Bold(true)
	sColHL := tcell.StyleDefault.Background(tcell.ColorTeal).Foreground(tcell.ColorWhite).Bold(true)
	sCursor := tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack).Bold(true)
	sSep := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorNavy)
	sCmd := tcell.StyleDefault.Background(tcell.ColorNavy).Foreground(tcell.ColorWhite)
	sInput := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorYellow)
	sError := tcell.StyleDefault.Background(tcell.ColorMaroon).Foreground(tcell.ColorWhite)
	sFormula := tcell.StyleDefault.Background(tcell.ColorNavy).Foreground(tcell.ColorYellow)

	// ── Linha 0: Status (coord + tipo + conteudo + mem) ──
	g.drawStatusLine(screen, x, y, width, sHeader, sFormula)

	// ── Linha 1: Cabecalho de colunas ──
	g.drawColHeader(screen, x, y+1, width, sHeader, sColHL, sSep)

	// ── Linhas 2-21: Dados ──
	visCols := g.calcVisCols(width)
	for row := 0; row < SC2_DATA_ROWS; row++ {
		sheetRow := g.sheet.ViewRow + row
		sy := y + SC2_HEADER_ROWS + row
		g.drawDataRow(screen, x, sy, width, sheetRow, visCols, sNormal, sHeader, sSep, sCursor)
	}

	// ── Linha 22: Barra principal (entrada / comando / status) ──
	g.drawCmdBar1(screen, x, y+SC2_HEADER_ROWS+SC2_DATA_ROWS, width, sCmd, sInput, sError)

	// ── Linha 23: Barra secundaria (modo / ajuda / comandos) ──
	g.drawCmdBar2(screen, x, y+SC2_HEADER_ROWS+SC2_DATA_ROWS+1, width, sCmd, sInput)
}

// drawStatusLine - Linha 0
// Formato SC2: " A1: (V)  1500        " + espacos + "Mem:62459"
// Tipos: (V)=valor  (L)=label/texto  (F)=formula  ( )=vazio
func (g *GridView) drawStatusLine(screen tcell.Screen, x, y, width int, st, formulaSt tcell.Style) {
	cur := g.sheet.Cursor
	cell := g.sheet.GetCell(cur)

	// Tipo da celula: V=valor numerico, L=label/texto, F=formula, espaco=vazio
	typeChar := " "
	var content string
	switch cell.Type {
	case spreadsheet.CellEmpty:
		typeChar = " "
		content = ""
	case spreadsheet.CellText:
		typeChar = "L"
		content = `"` + cell.TextValue
	case spreadsheet.CellNumber:
		typeChar = "V"
		content = fmtFloat(cell.NumericValue)
	case spreadsheet.CellFormula:
		typeChar = "F"
		content = cell.Formula // Mostra a formula, nao o resultado
		st = formulaSt
	}

	// Parte esquerda: direcao do cursor + coordenada + tipo
	// SC2 mostra '>' '<' '^' 'v' para indicar direcao do Enter
	// Simplificamos: sempre mostra '>' (para baixo, mais comum)
	left := fmt.Sprintf(">%-5s(%s) ", cur.String(), typeChar)

	// Memoria disponivel (simulada - SC2 mostrava isso)
	mem := "Mem:62459"

	// Monta linha completa
	available := width - len(left) - len(mem) - 1
	if available < 0 {
		available = 0
	}
	if runewidth.StringWidth(content) > available {
		content = runewidthTrunc(content, available)
	}
	mid := runewidthPad(content, available)
	line := left + mid + " " + mem

	// Desenha
	col := x
	for i, ch := range line {
		s := st
		// Coord e tipo em estilo normal (nao formula)
		if i < len(left) {
			s = tcell.StyleDefault.Background(tcell.ColorNavy).Foreground(tcell.ColorWhite).Bold(true)
		}
		screen.SetContent(col, y, ch, nil, s)
		col += runewidth.RuneWidth(ch)
		if col >= x+width {
			break
		}
	}
	// Preenche resto
	for col < x+width {
		screen.SetContent(col, y, ' ', nil, st)
		col++
	}
}

// drawColHeader - Linha 1
// Formato SC2: "   |    A    |    B    |    C    |..."
func (g *GridView) drawColHeader(screen tcell.Screen, x, y, width int, st, hlSt, sepSt tcell.Style) {
	// Espaco do numero de linha
	for i := 0; i < SC2_ROW_NUM_W; i++ {
		screen.SetContent(x+i, y, ' ', nil, st)
	}
	screen.SetContent(x+SC2_ROW_NUM_W, y, '|', nil, sepSt)

	curX := x + SC2_ROW_NUM_W + SC2_SEP_W
	for ci, col := range g.calcVisCols(width) {
		colW := g.sheet.GetColWidth(col)
		name := spreadsheet.ColName(col)

		pad := colW - len(name)
		pLeft := pad / 2
		pRight := pad - pLeft
		if pLeft < 0 {
			pLeft = 0
		}
		if pRight < 0 {
			pRight = 0
		}

		hdr := strings.Repeat(" ", pLeft) + name + strings.Repeat(" ", pRight)

		// Destaca coluna do cursor
		s := st
		if col == g.sheet.Cursor.Col {
			s = hlSt
		}

		gridDrawAt(screen, curX, y, hdr, s)
		curX += colW

		if ci < len(g.calcVisCols(width))-1 && curX < x+width-1 {
			screen.SetContent(curX, y, '|', nil, sepSt)
			curX++
		}
		if curX >= x+width {
			break
		}
	}
	for curX < x+width {
		screen.SetContent(curX, y, ' ', nil, st)
		curX++
	}
}

// drawDataRow - Uma linha de dados
func (g *GridView) drawDataRow(screen tcell.Screen, x, y, width, sheetRow int,
	visCols []int, sNorm, sRowNum, sSep, sCursor tcell.Style) {

	// Numero da linha (3 digitos, alinhado a direita)
	rowStr := fmt.Sprintf("%3d", sheetRow)
	// Destaca linha do cursor
	rsty := sRowNum
	if sheetRow == g.sheet.Cursor.Row {
		rsty = sCursor
	}
	gridDrawAt(screen, x, y, rowStr, rsty)
	screen.SetContent(x+SC2_ROW_NUM_W, y, '|', nil, sSep)

	cellX := x + SC2_ROW_NUM_W + SC2_SEP_W
	for ci, col := range visCols {
		colW := g.sheet.GetColWidth(col)
		coord := spreadsheet.Coord{Row: sheetRow, Col: col}
		cell := g.sheet.GetCell(coord)
		isCursor := (coord == g.sheet.Cursor)

		// Se e a celula sendo editada, mostra o buffer
		var display string
		if isCursor && g.mode == ModeInput {
			buf := g.inputBuffer + "_"
			display = runewidthPad(runewidthTrunc(buf, colW), colW)
		} else {
			display = g.sheet.FormatCellValue(cell, colW)
		}

		s := sNorm
		if isCursor {
			s = sCursor
		}

		gridDrawAt(screen, cellX, y, display, s)
		cellX += colW

		// Separador entre colunas
		if ci < len(visCols)-1 && cellX < x+width-1 {
			screen.SetContent(cellX, y, '|', nil, sSep)
			cellX++
		}
		if cellX >= x+width {
			break
		}
	}
	// Preenche resto da linha
	for cellX < x+width {
		screen.SetContent(cellX, y, ' ', nil, sNorm)
		cellX++
	}
}

// drawCmdBar1 - Linha 22: entrada de dados / comando / status
// No SC2 original esta linha mostra:
//
//	Modo normal: mensagem de status OU vazia
//	Modo input:  o que o usuario esta digitando
//	Modo /cmd:   "A1 COMMAND: " + opcoes do menu
func (g *GridView) drawCmdBar1(screen tcell.Screen, x, y, width int, st, inputSt, errorSt tcell.Style) {
	var line string

	switch g.mode {
	case ModeNormal:
		if g.statusMsg != "" {
			line = " " + g.statusMsg
		} else {
			// SC2 mostrava o valor calculado da formula aqui quando em modo normal
			line = ""
		}

	case ModeInput:
		// SC2: mostra "Enter value/label:" + o que foi digitado
		prefix := " Enter: "
		line = prefix + g.inputBuffer

	case ModeCommand:
		// SC2: ">" indica modo de comando
		line = " > /" + g.inputBuffer

	case ModeGoto:
		line = " Goto celula: " + g.inputBuffer
	}

	var s tcell.Style
	if g.statusMsg != "" && strings.HasPrefix(g.statusMsg, "ERRO") {
		s = errorSt
	} else if g.mode != ModeNormal {
		s = inputSt
	} else {
		s = st
	}

	line = runewidthPad(runewidthTrunc(line, width), width)
	gridDrawAt(screen, x, y, line, s)
}

// drawCmdBar2 - Linha 23: ajuda contextual / lista de comandos
// No SC2 original esta linha mostra os comandos do menu quando "/" e pressionado
// ou informacao da celula atual no modo normal
func (g *GridView) drawCmdBar2(screen tcell.Screen, x, y, width int, st, inputSt tcell.Style) {
	var line string

	switch g.mode {
	case ModeNormal:
		cur := g.sheet.Cursor
		cell := g.sheet.GetCell(cur)
		switch cell.Type {
		case spreadsheet.CellEmpty:
			line = fmt.Sprintf(" %s: vazia  [/:Comandos] [?:Ajuda] [Setas:Mover] [Del:Apagar]", cur)
		case spreadsheet.CellText:
			line = fmt.Sprintf(" %s: (L) \"%s\"", cur, cell.TextValue)
		case spreadsheet.CellNumber:
			line = fmt.Sprintf(" %s: (V) %s", cur, fmtFloat(cell.NumericValue))
		case spreadsheet.CellFormula:
			if cell.TextValue != "" && strings.HasPrefix(cell.TextValue, "#") {
				line = fmt.Sprintf(" %s: (F) %s = %s", cur, cell.Formula, cell.TextValue)
			} else {
				line = fmt.Sprintf(" %s: (F) %s = %s", cur, cell.Formula, fmtFloat(cell.NumericValue))
			}
		}

	case ModeInput:
		// SC2: mostra dica do tipo de entrada
		line = " [Enter]:Confirma  [Esc]:Cancela  [Setas]:Confirma e move  [\"]texto  [+@(]:formula"

	case ModeCommand:
		// SC2: lista de comandos do menu principal
		line = " B:Blank C:Copy D:Delete F:Format G:Global I:Insert L:Load M:Move P:Print Q:Quit R:Rep S:Save T:Title W:Width X:Xfer Z:Zap"

	case ModeGoto:
		line = " [Enter]:Vai  [Esc]:Cancela  Ex: A1  B12  AA3"
	}

	s := st
	if g.mode == ModeCommand || g.mode == ModeInput {
		s = inputSt
	}

	line = runewidthPad(runewidthTrunc(line, width), width)
	gridDrawAt(screen, x, y, line, s)
}

// ─── Calculo de colunas visiveis ─────────────────────────────────────────────

func (g *GridView) calcVisCols(width int) []int {
	available := width - SC2_ROW_NUM_W - SC2_SEP_W
	var cols []int
	used := 0
	for col := g.sheet.ViewCol; col <= 63; col++ {
		colW := g.sheet.GetColWidth(col)
		if used+colW > available {
			break
		}
		cols = append(cols, col)
		used += colW + 1 // +1 para o separador
	}
	return cols
}

// ─── Processamento de entrada ─────────────────────────────────────────────────

func (g *GridView) handleInput(event *tcell.EventKey) *tcell.EventKey {
	switch g.mode {
	case ModeNormal:
		return g.handleNormal(event)
	case ModeInput:
		return g.handleInputMode(event)
	case ModeCommand:
		return g.handleCommand(event)
	case ModeGoto:
		return g.handleGoto(event)
	}
	return event
}

func (g *GridView) handleNormal(event *tcell.EventKey) *tcell.EventKey {
	// Limpa status ao navegar
	g.statusMsg = ""

	_, _, width, _ := g.GetInnerRect()
	visRows := SC2_DATA_ROWS
	visCols := len(g.calcVisCols(width))

	switch event.Key() {
	case tcell.KeyUp:
		g.sheet.MoveCursor(-1, 0)
	case tcell.KeyDown:
		g.sheet.MoveCursor(1, 0)
	case tcell.KeyLeft:
		g.sheet.MoveCursor(0, -1)
	case tcell.KeyRight:
		g.sheet.MoveCursor(0, 1)
	case tcell.KeyTab:
		// Tab: move para direita (SC2 original)
		g.sheet.MoveCursor(0, 1)
	case tcell.KeyBacktab:
		g.sheet.MoveCursor(0, -1)
	case tcell.KeyEnter:
		// Enter no SC2: desce uma linha (confirma posicao)
		g.sheet.MoveCursor(1, 0)
	case tcell.KeyPgDn:
		g.sheet.MoveCursor(visRows, 0)
	case tcell.KeyPgUp:
		g.sheet.MoveCursor(-visRows, 0)
	case tcell.KeyHome:
		// Home: vai para coluna A da linha atual (SC2 original)
		g.sheet.Cursor.Col = 1
	case tcell.KeyEnd:
		// End: vai para ultima coluna usada (por hora col 63)
		g.sheet.Cursor.Col = 63
	case tcell.KeyCtrlF:
		// Ctrl+F: page right (colunas)
		g.sheet.MoveCursor(0, visCols)
	case tcell.KeyCtrlB:
		// Ctrl+B: page left (colunas)
		g.sheet.MoveCursor(0, -visCols)
	case tcell.KeyCtrlG:
		// Ctrl+G: Goto celula
		g.mode = ModeGoto
		g.inputBuffer = ""
	case tcell.KeyDelete, tcell.KeyBackspace, tcell.KeyBackspace2:
		// Del: apaga conteudo da celula (SC2: /Blank ou Del)
		g.sheet.SetCell(g.sheet.Cursor, nil)
		g.sheet.Recalc()
		g.statusMsg = fmt.Sprintf("Celula %s apagada", g.sheet.Cursor)
	case tcell.KeyF5:
		// F5: Goto (alternativa)
		g.mode = ModeGoto
		g.inputBuffer = ""
	case tcell.KeyRune:
		switch event.Rune() {
		case '/':
			g.mode = ModeCommand
			g.inputBuffer = ""
		case '?':
			g.statusMsg = "SC2MSX v2 | /=Comandos Del=Apaga Tab=Direita =GoTo PgUp/Dn=Pagina !Recalc"
		case '=':
			// '=' ativa GoTo (comportamento exato do SC2 original)
			g.mode = ModeGoto
			g.inputBuffer = ""
		case '!':
			// '!' forca recalculo manual (SC2 original)
			g.sheet.Recalc()
			g.statusMsg = "Recalculo concluido"
		default:
			// Qualquer tecla alfanumerica inicia edicao (comportamento SC2)
			g.mode = ModeInput
			g.inputBuffer = string(event.Rune())
		}
	}

	g.sheet.AdjustView(visRows, visCols)
	return nil
}

func (g *GridView) handleInputMode(event *tcell.EventKey) *tcell.EventKey {
	_, _, width, _ := g.GetInnerRect()
	visRows := SC2_DATA_ROWS
	visCols := len(g.calcVisCols(width))

	switch event.Key() {
	case tcell.KeyEscape:
		// ESC: cancela sem gravar (SC2 original)
		g.mode = ModeNormal
		g.inputBuffer = ""

	case tcell.KeyEnter:
		// Enter: confirma e move para baixo (SC2 original)
		g.commitInput()
		g.mode = ModeNormal
		g.sheet.MoveCursor(1, 0)

	case tcell.KeyTab:
		// Tab: confirma e move para direita
		g.commitInput()
		g.mode = ModeNormal
		g.sheet.MoveCursor(0, 1)

	case tcell.KeyBacktab:
		g.commitInput()
		g.mode = ModeNormal
		g.sheet.MoveCursor(0, -1)

	// Setas confirmam e movem (comportamento SC2)
	case tcell.KeyUp:
		g.commitInput()
		g.mode = ModeNormal
		g.sheet.MoveCursor(-1, 0)
	case tcell.KeyDown:
		g.commitInput()
		g.mode = ModeNormal
		g.sheet.MoveCursor(1, 0)
	case tcell.KeyLeft:
		// Left no SC2 durante edicao: apaga ultimo char (nao move)
		if len(g.inputBuffer) > 0 {
			runes := []rune(g.inputBuffer)
			g.inputBuffer = string(runes[:len(runes)-1])
		}
	case tcell.KeyRight:
		// Right: confirma e move para direita
		g.commitInput()
		g.mode = ModeNormal
		g.sheet.MoveCursor(0, 1)

	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(g.inputBuffer) > 0 {
			runes := []rune(g.inputBuffer)
			g.inputBuffer = string(runes[:len(runes)-1])
		}

	case tcell.KeyRune:
		g.inputBuffer += string(event.Rune())
	}

	g.sheet.AdjustView(visRows, visCols)
	return nil
}

func (g *GridView) handleCommand(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		g.mode = ModeNormal
		g.inputBuffer = ""
		return nil
	case tcell.KeyRune:
		cmd := event.Rune()
		g.mode = ModeNormal
		g.inputBuffer = ""
		switch cmd {
		case 'B', 'b':
			// Blank: apaga celula atual (mesmo que Del)
			g.sheet.SetCell(g.sheet.Cursor, nil)
			g.sheet.Recalc()
			g.statusMsg = fmt.Sprintf("Celula %s apagada", g.sheet.Cursor)
		case 'Q', 'q':
			if g.onCommand != nil {
				g.onCommand("QUIT")
			}
		case 'S', 's':
			if g.onCommand != nil {
				g.onCommand("SAVE")
			}
		case 'L', 'l':
			if g.onCommand != nil {
				g.onCommand("LOAD")
			}
		case 'Z', 'z':
			if g.onCommand != nil {
				g.onCommand("ZAP")
			}
		case 'W', 'w':
			if g.onCommand != nil {
				g.onCommand("WIDTH")
			}
		case 'F', 'f':
			if g.onCommand != nil {
				g.onCommand("FORMAT")
			}
		case 'G', 'g':
			if g.onCommand != nil {
				g.onCommand("GLOBAL")
			}
		case 'I', 'i':
			if g.onCommand != nil {
				g.onCommand("INSERT")
			}
		case 'D', 'd':
			if g.onCommand != nil {
				g.onCommand("DELETE")
			}
		case 'C', 'c':
			if g.onCommand != nil {
				g.onCommand("COPY")
			}
		case 'M', 'm':
			if g.onCommand != nil {
				g.onCommand("MOVE")
			}
		case 'R', 'r':
			if g.onCommand != nil {
				g.onCommand("REPLICATE")
			}
		case 'T', 't':
			if g.onCommand != nil {
				g.onCommand("TITLE")
			}
		case 'P', 'p':
			if g.onCommand != nil {
				g.onCommand("PRINT")
			}
		case 'X', 'x':
			if g.onCommand != nil {
				g.onCommand("XFER")
			}
		case '?':
			g.statusMsg = "Comandos: B=Apaga C=Copia D=Del F=Formato G=Global I=Insere L=Carrega M=Move P=Imprime Q=Sai R=Replica S=Salva T=Titulo W=Largura X=Transf Z=Zera"
		default:
			g.statusMsg = fmt.Sprintf("Comando '/%c' nao reconhecido", cmd)
		}
	}
	return nil
}

func (g *GridView) handleGoto(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		g.mode = ModeNormal
		g.inputBuffer = ""
	case tcell.KeyEnter:
		coord, err := spreadsheet.ParseCoord(strings.ToUpper(strings.TrimSpace(g.inputBuffer)))
		if err != nil {
			g.statusMsg = fmt.Sprintf("Coordenada invalida: %s", g.inputBuffer)
		} else {
			g.sheet.Cursor = coord
			_, _, width, _ := g.GetInnerRect()
			g.sheet.AdjustView(SC2_DATA_ROWS, len(g.calcVisCols(width)))
		}
		g.mode = ModeNormal
		g.inputBuffer = ""
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(g.inputBuffer) > 0 {
			runes := []rune(g.inputBuffer)
			g.inputBuffer = string(runes[:len(runes)-1])
		}
	case tcell.KeyRune:
		g.inputBuffer += strings.ToUpper(string(event.Rune()))
	}
	return nil
}

// ─── Confirmacao de entrada ───────────────────────────────────────────────────

// commitInput grava o conteudo do buffer na celula atual e recalcula
func (g *GridView) commitInput() {
	if g.inputBuffer == "" {
		return
	}
	cell := parseInput(g.inputBuffer)
	g.sheet.SetCell(g.sheet.Cursor, cell)
	g.sheet.Recalc()
	g.inputBuffer = ""
}

// parseInput interpreta entrada do usuario - regras exatas do SC2 MSX:
//
//	SC2 determina o tipo pelo PRIMEIRO CARACTER:
//	" (aspas)        → texto (label) - remove a aspas
//	Letra A-Z        → texto (label) - sem aspas necessarias
//	Digito 0-9       → numero (tenta parsear como float)
//	+ - ( @          → formula
//	Qualquer outro   → texto
//
//	Numero com ponto: float. Sem ponto: inteiro.
//	Formula: avaliada pelo Evaluator apos SetCell+Recalc
func parseInput(input string) *spreadsheet.Cell {
	if input == "" {
		return &spreadsheet.Cell{Type: spreadsheet.CellEmpty}
	}

	first := rune(input[0])

	// Aspas: texto forcado (SC2: "texto - a aspas e o indicador, nao faz parte do valor)
	if first == '"' {
		text := ""
		if len(input) > 1 {
			text = input[1:]
		}
		return &spreadsheet.Cell{
			Type: spreadsheet.CellText, TextValue: text, RawInput: input,
		}
	}

	// Apostrofo: texto repetitivo (SC2: '---  preenche a celula com o padrao)
	if first == '\'' {
		return &spreadsheet.Cell{
			Type: spreadsheet.CellText, TextValue: input, RawInput: input,
		}
	}

	// Letra: texto (label) - SC2 nao precisa de aspas para textos que comecam com letra
	if (first >= 'A' && first <= 'Z') || (first >= 'a' && first <= 'z') {
		return &spreadsheet.Cell{
			Type: spreadsheet.CellText, TextValue: input, RawInput: input,
		}
	}

	// Numero puro OU expressao que comeca com digito (ex: 4+5, 3*2, 1.5/2)
	// SC2: se começa com digito e nao e numero puro, trata como formula
	if first >= '0' && first <= '9' || first == '.' {
		if isExactFloat(input) {
			// Numero puro: sem operadores, e um float valido completo
			f, _ := parseFloat(input)
			return &spreadsheet.Cell{
				Type: spreadsheet.CellNumber, NumericValue: f, RawInput: input,
			}
		}
		// Contem operadores (+, -, *, /, ^, =, <, >) apos o numero: e formula
		// O SC2 trata "4+5" igual a "+4+5" - expressao numerica
		return &spreadsheet.Cell{
			Type: spreadsheet.CellFormula, Formula: input, RawInput: input,
		}
	}

	// Formula: +  -  (  @
	if first == '+' || first == '-' || first == '(' || first == '@' {
		return &spreadsheet.Cell{
			Type: spreadsheet.CellFormula, Formula: input, RawInput: input,
		}
	}

	// Default: texto
	return &spreadsheet.Cell{
		Type: spreadsheet.CellText, TextValue: input, RawInput: input,
	}
}

// isExactFloat retorna true somente se a string inteira e um numero valido,
// sem nenhum caractere extra apos o valor numerico.
// Diferente de parseFloat (que usa Sscanf e para no primeiro char invalido),
// esta funcao exige que TODO o input seja consumido.
func isExactFloat(s string) bool {
	if s == "" {
		return false
	}
	// Tenta strconv.ParseFloat - ele exige o numero completo
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// ─── Helpers de renderizacao ──────────────────────────────────────────────────

// gridDrawAt desenha texto avancando X pela largura visual real de cada rune
func gridDrawAt(screen tcell.Screen, x, y int, text string, style tcell.Style) {
	col := x
	for _, ch := range text {
		screen.SetContent(col, y, ch, nil, style)
		col += runewidth.RuneWidth(ch)
	}
}

// runewidthTrunc trunca string para que largura visual nao exceda maxW
func runewidthTrunc(s string, maxW int) string {
	w := 0
	for i, ch := range s {
		cw := runewidth.RuneWidth(ch)
		if w+cw > maxW {
			return s[:i]
		}
		w += cw
	}
	return s
}

// runewidthPad completa string com espacos ate largura visual targetW
func runewidthPad(s string, targetW int) string {
	w := runewidth.StringWidth(s)
	if w >= targetW {
		return s
	}
	return s + strings.Repeat(" ", targetW-w)
}

// fmtFloat formata float para exibicao compacta (como SC2 original)
// Numeros inteiros sem casas decimais, floats com precisao necessaria
func fmtFloat(v float64) string {
	// Se e inteiro exato, exibe sem ponto
	if v == float64(int64(v)) && v >= -1e12 && v <= 1e12 {
		return fmt.Sprintf("%d", int64(v))
	}
	// Senao: 6 digitos significativos (padrao SC2)
	s := fmt.Sprintf("%.6g", v)
	return s
}
