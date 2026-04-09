// internal/ui/grid.go
// Tela principal da planilha - simula monitor 80x24 do MSX
// Layout fiel ao SuperCalc 2 MSX original
package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/rivo/tview"

	"github.com/sc2msx/internal/spreadsheet"
)

// Constantes de layout - tela 80x24 como no MSX
const (
	SC2_COLS        = 80  // Largura da tela
	SC2_ROWS        = 24  // Altura total da tela
	SC2_HEADER_ROWS = 2   // Linhas de cabeçalho (linha de status + linha de colunas)
	SC2_CMD_ROWS    = 2   // Linhas da barra de comandos (rodapé)
	SC2_DATA_ROWS   = SC2_ROWS - SC2_HEADER_ROWS - SC2_CMD_ROWS // 20 linhas de dados
	SC2_ROW_NUM_W   = 3   // Largura da coluna de números de linha (ex: "  1", " 12")
	SC2_SEP_W       = 1   // Separador "|"
)

// GridView é a tela principal da planilha
type GridView struct {
	*tview.Box

	// Planilha
	sheet *spreadsheet.Spreadsheet

	// Modo de entrada
	mode InputMode

	// Buffer de entrada do usuário
	inputBuffer string

	// Linha de status
	statusMsg string

	// Callbacks
	onCommand func(cmd string)
}

// InputMode define o modo atual da planilha
type InputMode int

const (
	ModeNormal  InputMode = iota // Navegação normal
	ModeInput                   // Digitando valor/fórmula
	ModeCommand                 // Após pressionar "/"
)

// NewGridView cria a view principal da planilha
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

// SetCommandHandler define callback para comandos
func (g *GridView) SetCommandHandler(fn func(cmd string)) {
	g.onCommand = fn
}

// SetStatus define mensagem de status
func (g *GridView) SetStatus(msg string) {
	g.statusMsg = msg
}

// Draw renderiza a planilha completa
func (g *GridView) Draw(screen tcell.Screen) {
	g.Box.DrawForSubclass(screen, g)

	x, y, width, height := g.GetInnerRect()
	_ = height

	// Estilos de cores (paleta MSX-like)
	styleHeader := tcell.StyleDefault.
		Background(tcell.ColorNavy).
		Foreground(tcell.ColorWhite).
		Bold(true)

	styleColHeader := tcell.StyleDefault.
		Background(tcell.ColorTeal).
		Foreground(tcell.ColorWhite).
		Bold(true)

	styleRowNum := tcell.StyleDefault.
		Background(tcell.ColorNavy).
		Foreground(tcell.ColorWhite)

	styleCursor := tcell.StyleDefault.
		Background(tcell.ColorWhite).
		Foreground(tcell.ColorBlack).
		Bold(true)

	styleCell := tcell.StyleDefault.
		Background(tcell.ColorBlack).
		Foreground(tcell.ColorWhite)

	styleFormula := tcell.StyleDefault.
		Background(tcell.ColorBlack).
		Foreground(tcell.ColorYellow)

	styleSep := tcell.StyleDefault.
		Background(tcell.ColorBlack).
		Foreground(tcell.ColorGray)

	styleCmdBar := tcell.StyleDefault.
		Background(tcell.ColorNavy).
		Foreground(tcell.ColorWhite)

	styleInput := tcell.StyleDefault.
		Background(tcell.ColorBlack).
		Foreground(tcell.ColorYellow)

	styleStatus := tcell.StyleDefault.
		Background(tcell.ColorMaroon).
		Foreground(tcell.ColorWhite)

	// ─────────────────────────────────────────────────────────────
	// LINHA 0: Linha de status superior (célula atual + conteúdo)
	// Formato SC2: "A1: 9999  <conteúdo>"
	// ─────────────────────────────────────────────────────────────
	g.drawStatusLine(screen, x, y, width, styleHeader, styleFormula)

	// ─────────────────────────────────────────────────────────────
	// LINHA 1: Cabeçalho de colunas
	// Formato SC2: "   |  A  |  B  |  C  | ..."
	// ─────────────────────────────────────────────────────────────
	g.drawColumnHeader(screen, x, y+1, width, styleColHeader, styleRowNum, styleSep)

	// ─────────────────────────────────────────────────────────────
	// LINHAS 2..21: Dados da planilha (20 linhas)
	// ─────────────────────────────────────────────────────────────
	visibleCols := g.calcVisibleCols(width)
	for row := 0; row < SC2_DATA_ROWS; row++ {
		sheetRow := g.sheet.ViewRow + row
		screenY := y + SC2_HEADER_ROWS + row

		// Número da linha (3 chars, alinhado à direita) - só dígitos ASCII, sem problema
		rowNumStr := fmt.Sprintf("%3d", sheetRow)
		gridDrawAt(screen, x, screenY, rowNumStr, styleRowNum)

		// Separador "|"
		screen.SetContent(x+SC2_ROW_NUM_W, screenY, '|', nil, styleSep)

		// Células das colunas visíveis
		cellX := x + SC2_ROW_NUM_W + SC2_SEP_W
		for ci, col := range visibleCols {
			colW := g.sheet.GetColWidth(col)
			coord := spreadsheet.Coord{Row: sheetRow, Col: col}
			cell := g.sheet.GetCell(coord)
			isCursor := coord == g.sheet.Cursor

			// Formata o valor para exibição
			display := g.sheet.FormatCellValue(cell, colW)
			if len([]rune(display)) > colW {
				display = display[:colW]
			}
			// Pad se necessário
			for len([]rune(display)) < colW {
				display += " "
			}

			// Escolhe estilo
			var cellStyle tcell.Style
			if isCursor {
				cellStyle = styleCursor
			} else {
				cellStyle = styleCell
			}

			// Desenha o conteúdo da célula usando largura visual correta
			gridDrawAt(screen, cellX, screenY, display, cellStyle)

			cellX += colW

			// Separador entre colunas ("|")
			if ci < len(visibleCols)-1 && cellX < x+width-1 {
				screen.SetContent(cellX, screenY, '|', nil, styleSep)
				cellX++
			}

			// Não ultrapassa a largura da tela
			if cellX >= x+width {
				break
			}
		}

		// Preenche o resto da linha com espaço
		for cx := cellX; cx < x+width; cx++ {
			screen.SetContent(cx, screenY, ' ', nil, styleCell)
		}
	}

	// ─────────────────────────────────────────────────────────────
	// LINHA 22: Barra de comandos - linha 1
	// ─────────────────────────────────────────────────────────────
	cmdY1 := y + SC2_HEADER_ROWS + SC2_DATA_ROWS
	g.drawCmdLine1(screen, x, cmdY1, width, styleCmdBar, styleInput, styleStatus)

	// ─────────────────────────────────────────────────────────────
	// LINHA 23: Barra de comandos - linha 2
	// ─────────────────────────────────────────────────────────────
	cmdY2 := cmdY1 + 1
	g.drawCmdLine2(screen, x, cmdY2, width, styleCmdBar, styleInput)
}

// drawStatusLine renderiza a linha de status (linha 0)
// Formato: "B5: 2500     Fórmula ou valor"
func (g *GridView) drawStatusLine(screen tcell.Screen, x, y, width int, style, formulaStyle tcell.Style) {
	cursor := g.sheet.Cursor
	cell := g.sheet.GetCell(cursor)

	// Parte esquerda: coordenada e tipo
	left := fmt.Sprintf(" %-4s", cursor.String())

	// Conteúdo da célula na linha de status
	var content string
	switch cell.Type {
	case spreadsheet.CellEmpty:
		content = ""
	case spreadsheet.CellText:
		content = `"` + cell.TextValue
	case spreadsheet.CellNumber:
		content = fmt.Sprintf("%v", cell.NumericValue)
	case spreadsheet.CellFormula:
		content = cell.Formula
	}

	// Memória disponível (simulada) e modo
	modeStr := ""
	switch g.mode {
	case ModeInput:
		modeStr = " [INPUT] "
	case ModeCommand:
		modeStr = " [COMMAND] "
	}

	// Monta linha de status
	memStr := "Mem: 62459"
	statusLine := left + " " + content
	statusW := runewidth.StringWidth(statusLine)
	modeW   := runewidth.StringWidth(modeStr)
	memW    := runewidth.StringWidth(memStr)
	rightPad := width - statusW - modeW - memW - 1
	if rightPad < 0 { rightPad = 0 }
	full := statusLine + strings.Repeat(" ", rightPad) + modeStr + " " + memStr

	// Trunca pela largura visual, nao pelo numero de runes
	full = runewidthTrunc(full, width)
	// Completa com espacos se necessario
	for runewidth.StringWidth(full) < width {
		full += " "
	}

	col := x
	for j, ch := range full {
		s := style
		if j >= len([]rune(left))+1 {
			s = formulaStyle
		}
		screen.SetContent(col, y, ch, nil, s)
		col += runewidth.RuneWidth(ch)
		if col >= x+width {
			break
		}
	}
}

// drawColumnHeader renderiza o cabeçalho de colunas (linha 1)
// Formato: "   |  A  |  B  |  C  |..."
func (g *GridView) drawColumnHeader(screen tcell.Screen, x, y, width int, colStyle, numStyle, sepStyle tcell.Style) {
	// Espaço para números de linha
	header := fmt.Sprintf("%3s", " ")
	gridDrawAt(screen, x, y, header, numStyle)

	// Separador "|" após row nums
	screen.SetContent(x+SC2_ROW_NUM_W, y, '|', nil, sepStyle)

	curX := x + SC2_ROW_NUM_W + SC2_SEP_W
	visibleCols := g.calcVisibleCols(width)

	for ci, col := range visibleCols {
		colW := g.sheet.GetColWidth(col)
		colName := spreadsheet.ColName(col)

		// Centraliza o nome da coluna dentro da largura (colName e ASCII puro)
		padTotal := colW - len(colName)
		padLeft := padTotal / 2
		padRight := padTotal - padLeft
		if padLeft < 0 { padLeft = 0 }
		if padRight < 0 { padRight = 0 }

		colHeader := strings.Repeat(" ", padLeft) + colName + strings.Repeat(" ", padRight)

		// Destaca a coluna do cursor
		st := colStyle
		if col == g.sheet.Cursor.Col {
			st = tcell.StyleDefault.
				Background(tcell.ColorWhite).
				Foreground(tcell.ColorBlack).
				Bold(true)
		}

		gridDrawAt(screen, curX, y, colHeader, st)
		curX += colW

		// Separador
		if ci < len(visibleCols)-1 && curX < x+width-1 {
			screen.SetContent(curX, y, '|', nil, sepStyle)
			curX++
		}

		if curX >= x+width {
			break
		}
	}

	// Preenche o resto
	for cx := curX; cx < x+width; cx++ {
		screen.SetContent(cx, y, ' ', nil, colStyle)
	}
}

// drawCmdLine1 renderiza a primeira linha da barra de comandos
func (g *GridView) drawCmdLine1(screen tcell.Screen, x, y, width int, barStyle, inputStyle, statusStyle tcell.Style) {
	var line string

	switch g.mode {
	case ModeNormal:
		if g.statusMsg != "" {
			line = " " + g.statusMsg
			line = runewidthPad(runewidthTrunc(line, width), width)
			gridDrawAt(screen, x, y, line, statusStyle)
			return
		}
		line = " Setas:Mover  Enter:Confirmar  /:Comandos  ?:Ajuda  Esc:Cancela"

	case ModeInput:
		line = " Entrada: " + g.inputBuffer + "_"

	case ModeCommand:
		line = " Comando: /" + g.inputBuffer
	}

	line = runewidthPad(runewidthTrunc(line, width), width)

	st := barStyle
	if g.mode != ModeNormal {
		st = inputStyle
	}

	gridDrawAt(screen, x, y, line, st)
}

// drawCmdLine2 renderiza a segunda linha da barra de comandos
func (g *GridView) drawCmdLine2(screen tcell.Screen, x, y, width int, barStyle, inputStyle tcell.Style) {
	var line string

	switch g.mode {
	case ModeNormal:
		cur := g.sheet.Cursor
		cell := g.sheet.GetCell(cur)
		switch cell.Type {
		case spreadsheet.CellEmpty:
			line = fmt.Sprintf(" Celula %s vazia - Digite para inserir", cur.String())
		case spreadsheet.CellText:
			line = fmt.Sprintf(" Texto: \"%s\"", cell.TextValue)
		case spreadsheet.CellNumber:
			line = fmt.Sprintf(" Numero: %v", cell.NumericValue)
		case spreadsheet.CellFormula:
			line = fmt.Sprintf(" Formula: %s = %v", cell.Formula, cell.NumericValue)
		}

	case ModeInput:
		line = " [Enter]:Confirmar  [Esc]:Cancelar"

	case ModeCommand:
		line = " /B:Blank /C:Copy /D:Delete /F:Format /G:Global /I:Insert /L:Load /M:Move /P:Print /Q:Quit /R:Replicate /S:Save /T:Title /W:Width /X:Xfer /Z:Zap"
	}

	line = runewidthPad(runewidthTrunc(line, width), width)

	st := barStyle
	if g.mode == ModeCommand {
		st = inputStyle
	}

	gridDrawAt(screen, x, y, line, st)
}

// calcVisibleCols calcula quais colunas são visíveis dado a largura disponível
func (g *GridView) calcVisibleCols(width int) []int {
	available := width - SC2_ROW_NUM_W - SC2_SEP_W
	var cols []int
	used := 0

	for col := g.sheet.ViewCol; col <= 63; col++ {
		colW := g.sheet.GetColWidth(col)
		if used+colW+1 > available {
			break
		}
		cols = append(cols, col)
		used += colW + 1 // +1 para o separador "|"
	}

	return cols
}

// handleInput processa as teclas de entrada
func (g *GridView) handleInput(event *tcell.EventKey) *tcell.EventKey {
	switch g.mode {
	case ModeNormal:
		return g.handleNormal(event)
	case ModeInput:
		return g.handleInputMode(event)
	case ModeCommand:
		return g.handleCommandMode(event)
	}
	return event
}

// handleNormal processa teclas no modo normal (navegação)
func (g *GridView) handleNormal(event *tcell.EventKey) *tcell.EventKey {
	sheet := g.sheet
	_, _, width, height := g.GetInnerRect()
	visRows := SC2_DATA_ROWS
	visCols := len(g.calcVisibleCols(width))
	_ = height

	switch event.Key() {
	case tcell.KeyUp:
		sheet.MoveCursor(-1, 0)
	case tcell.KeyDown:
		sheet.MoveCursor(1, 0)
	case tcell.KeyLeft:
		sheet.MoveCursor(0, -1)
	case tcell.KeyRight:
		sheet.MoveCursor(0, 1)
	case tcell.KeyTab:
		sheet.MoveCursor(0, 1)
	case tcell.KeyBacktab:
		sheet.MoveCursor(0, -1)

	// PgDn / PgUp
	case tcell.KeyPgDn:
		sheet.MoveCursor(visRows, 0)
	case tcell.KeyPgUp:
		sheet.MoveCursor(-visRows, 0)

	// Home: vai para coluna A da linha atual
	case tcell.KeyHome:
		sheet.Cursor.Col = 1

	// End: vai para última coluna com dado
	case tcell.KeyEnd:
		// Por hora, vai para coluna 63
		sheet.Cursor.Col = 63

	// Ctrl+Right/Left: pula uma "página" de colunas
	// tcell não tem KeyCtrlRight/Left — usa Ctrl+F (forward) e Ctrl+B (back)
	case tcell.KeyCtrlF:
		sheet.MoveCursor(0, visCols)
	case tcell.KeyCtrlB:
		sheet.MoveCursor(0, -visCols)

	// Enter: confirma (desce uma linha, como no SC2)
	case tcell.KeyEnter:
		sheet.MoveCursor(1, 0)

	// Rune keys
	case tcell.KeyRune:
		switch event.Rune() {
		case '/':
			// Abre barra de comandos
			g.mode = ModeCommand
			g.inputBuffer = ""

		case '?':
			// Ajuda (por ora mostra mensagem)
			g.statusMsg = "AJUDA: / = Comandos | Setas = Navegar | Esc = Cancela | Enter = Confirma"

		case 127, 8: // DEL / Backspace - apaga célula atual
			sheet.SetCell(sheet.Cursor, nil)

		default:
			// Começa a digitar na célula atual
			g.mode = ModeInput
			g.inputBuffer = string(event.Rune())
		}

	case tcell.KeyDelete, tcell.KeyBackspace, tcell.KeyBackspace2:
		sheet.SetCell(sheet.Cursor, nil)
		g.statusMsg = "Célula apagada"
	}

	// Ajusta a view para manter cursor visível
	sheet.AdjustView(visRows, visCols)

	return nil
}

// handleInputMode processa teclas no modo de entrada de dados
func (g *GridView) handleInputMode(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		// Cancela entrada
		g.mode = ModeNormal
		g.inputBuffer = ""

	case tcell.KeyEnter:
		// Confirma entrada
		g.commitInput()
		g.mode = ModeNormal
		g.sheet.MoveCursor(1, 0)
		_, _, width, _ := g.GetInnerRect()
		g.sheet.AdjustView(SC2_DATA_ROWS, len(g.calcVisibleCols(width)))

	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(g.inputBuffer) > 0 {
			runes := []rune(g.inputBuffer)
			g.inputBuffer = string(runes[:len(runes)-1])
		}

	case tcell.KeyRune:
		g.inputBuffer += string(event.Rune())

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
		g.commitInput()
		g.mode = ModeNormal
		g.sheet.MoveCursor(0, -1)
	case tcell.KeyRight:
		g.commitInput()
		g.mode = ModeNormal
		g.sheet.MoveCursor(0, 1)
	}

	return nil
}

// commitInput confirma o que foi digitado e armazena na celula
// Apos gravar, faz recalculo completo da planilha
func (g *GridView) commitInput() {
	if g.inputBuffer == "" {
		return
	}

	input := g.inputBuffer
	cell := parseInput(input)
	coord := g.sheet.Cursor
	g.sheet.SetCell(coord, cell)

	// Recalcula todas as formulas
	g.sheet.Recalc()
}

// handleCommandMode processa o menu de comandos (após "/")
func (g *GridView) handleCommandMode(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		g.mode = ModeNormal
		g.inputBuffer = ""

	case tcell.KeyRune:
		cmd := event.Rune()

		switch cmd {
		case '?':
			g.mode = ModeNormal
			g.inputBuffer = ""
			g.statusMsg = "COMANDOS SC2: B=Blank C=Copy D=Delete F=Format G=Global I=Insert L=Load M=Move P=Print Q=Quit R=Replicate S=Save T=Title W=Width X=Xfer Z=Zap"

		case 'Q', 'q':
			// Quit - será tratado pelo app principal
			g.mode = ModeNormal
			g.inputBuffer = ""
			if g.onCommand != nil {
				g.onCommand("QUIT")
			}

		case 'S', 's':
			// Save
			g.mode = ModeNormal
			g.inputBuffer = ""
			if g.onCommand != nil {
				g.onCommand("SAVE")
			}

		case 'L', 'l':
			// Load
			g.mode = ModeNormal
			g.inputBuffer = ""
			if g.onCommand != nil {
				g.onCommand("LOAD")
			}

		case 'Z', 'z':
			// Zap (limpa planilha)
			g.mode = ModeNormal
			g.inputBuffer = ""
			if g.onCommand != nil {
				g.onCommand("ZAP")
			}

		default:
			// Comando ainda não implementado
			g.mode = ModeNormal
			g.inputBuffer = ""
			g.statusMsg = fmt.Sprintf("Comando '/%c' ainda não implementado", cmd)
		}
	}

	return nil
}

// parseInput interpreta a entrada do usuário e cria uma Cell
// Regras SC2:
//   - Começa com " ou letra → texto (label)
//   - Começa com número, +, -, (, @ → número ou fórmula
//   - Começa com + ou número puro → número
func parseInput(input string) *spreadsheet.Cell {
	if input == "" {
		return &spreadsheet.Cell{Type: spreadsheet.CellEmpty}
	}

	first := rune(input[0])

	// Forçar texto com aspas (como no SC2: "texto)
	if first == '"' {
		return &spreadsheet.Cell{
			Type:      spreadsheet.CellText,
			TextValue: input[1:], // Remove a aspa inicial
			RawInput:  input,
		}
	}

	// Letra sem aspas → também é texto (label)
	if first >= 'A' && first <= 'Z' || first >= 'a' && first <= 'z' {
		return &spreadsheet.Cell{
			Type:      spreadsheet.CellText,
			TextValue: input,
			RawInput:  input,
		}
	}

	// Tenta número puro
	if f, err := parseFloat(input); err == nil {
		return &spreadsheet.Cell{
			Type:         spreadsheet.CellNumber,
			NumericValue: f,
			RawInput:     input,
		}
	}

	// Começa com +, -, (, @ → formula SC2
	// O valor sera calculado pelo Recalc() apos SetCell
	if first == '+' || first == '-' || first == '(' || first == '@' {
		return &spreadsheet.Cell{
			Type:         spreadsheet.CellFormula,
			Formula:      input,
			NumericValue: 0,
			RawInput:     input,
		}
	}

	// Default: texto
	return &spreadsheet.Cell{
		Type:      spreadsheet.CellText,
		TextValue: input,
		RawInput:  input,
	}
}

// parseFloat tenta converter string para float64
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// ─── Helpers de renderização com runewidth ───────────────────────────────────

// gridDrawAt desenha texto na tela avancando X pela largura visual de cada rune.
// Isso evita desalinhamento com caracteres acentuados ou de largura dupla.
func gridDrawAt(screen tcell.Screen, x, y int, text string, style tcell.Style) {
	col := x
	for _, ch := range text {
		screen.SetContent(col, y, ch, nil, style)
		col += runewidth.RuneWidth(ch)
	}
}

// runewidthTrunc trunca a string para que sua largura visual nao exceda maxW.
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

// runewidthPad completa a string com espacos ate atingir a largura visual targetW.
func runewidthPad(s string, targetW int) string {
	w := runewidth.StringWidth(s)
	if w >= targetW {
		return s
	}
	return s + strings.Repeat(" ", targetW-w)
}
