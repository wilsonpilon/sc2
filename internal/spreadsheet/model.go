// internal/spreadsheet/model.go
// Modelo de dados da planilha - compativel com SuperCalc 2 MSX
//
// Comportamento fiel ao SC2 MSX original:
//   - Planilha esparsa: 254 linhas x 63 colunas (A..BK)
//   - Celulas: vazio, texto (L), numero (V), formula (F)
//   - Texto: alinhado a esquerda, truncado na borda da celula (SEM overflow)
//   - Numero: alinhado a direita, "**...*" se nao couber
//   - Largura de coluna padrao: 9 caracteres
//   - Coordenadas: A1..BK254
package spreadsheet

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// CellType - tipo do conteudo da celula
type CellType int

const (
	CellEmpty   CellType = iota
	CellText             // (L) Label - texto
	CellNumber           // (V) Value - numero
	CellFormula          // (F) Formula - expressao
)

// Alignment - alinhamento na celula
type Alignment int

const (
	AlignDefault Alignment = iota // texto=esq, numero=dir
	AlignLeft
	AlignRight
	AlignCenter
)

// FormatType - tipo de formatacao numerica (comando /Format do SC2)
type FormatType int

const (
	FormatGeneral    FormatType = iota // G - inteiro se possivel, senao float
	FormatDefault                      // D - padrao (2 casas decimais)
	FormatInteger                      // I - inteiro (sem decimais)
	FormatFixed                        // F - ponto fixo (N casas)
	FormatScientific                   // S - notacao cientifica
	FormatDollar                       // $ - moeda
	FormatPercent                      // % - percentual
	FormatBar                          // * - barra grafica (SC2 especial)
)

// CellFormat - formatacao da celula (definida por /Format no SC2)
type CellFormat struct {
	Type     FormatType
	Decimals int  // 0-9 casas decimais
	Commas   bool // separador de milhares
}

// Cell - uma celula da planilha
type Cell struct {
	RawInput     string // Conteudo bruto digitado
	Type         CellType
	NumericValue float64 // Valor numerico (CellNumber ou resultado de CellFormula)
	TextValue    string  // Valor texto (CellText) ou codigo de erro de formula
	Formula      string  // Expressao original (CellFormula)
	Format       CellFormat
	Align        Alignment
	Protected    bool
}

// Coord - coordenada de celula (1-based)
type Coord struct {
	Row int // 1..254
	Col int // 1..63 (A=1 .. BK=63)
}

func (c Coord) String() string {
	return ColName(c.Col) + strconv.Itoa(c.Row)
}

// ColName converte numero de coluna para nome SC2
// 1=A, 2=B ... 26=Z, 27=AA, 28=AB ... 52=AZ, 53=BA ... 63=BK
func ColName(col int) string {
	if col < 1 {
		return "?"
	}
	if col <= 26 {
		return string(rune('A' + col - 1))
	}
	// Dupla letra: col 27 = AA
	col -= 27
	major := col / 26
	minor := col % 26
	return string(rune('A'+major)) + string(rune('A'+minor))
}

// ParseCoord converte string de coordenada para Coord
// Aceita: A1, B12, AA3, BK254 (case insensitive)
func ParseCoord(s string) (Coord, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if len(s) < 2 {
		return Coord{}, fmt.Errorf("coordenada invalida: %s", s)
	}

	i := 0
	for i < len(s) && s[i] >= 'A' && s[i] <= 'Z' {
		i++
	}
	if i == 0 || i == len(s) {
		return Coord{}, fmt.Errorf("coordenada invalida: %s", s)
	}

	colStr := s[:i]
	rowStr := s[i:]

	row, err := strconv.Atoi(rowStr)
	if err != nil || row < 1 || row > 254 {
		return Coord{}, fmt.Errorf("linha invalida: %s (1-254)", rowStr)
	}

	var col int
	switch len(colStr) {
	case 1:
		col = int(colStr[0]-'A') + 1
	case 2:
		col = (int(colStr[0]-'A')+1)*26 + int(colStr[1]-'A') + 1
	default:
		return Coord{}, fmt.Errorf("coluna invalida: %s", colStr)
	}

	if col < 1 || col > 63 {
		return Coord{}, fmt.Errorf("coluna invalida: %s (A-BK)", colStr)
	}

	return Coord{Row: row, Col: col}, nil
}

// Spreadsheet - a planilha
type Spreadsheet struct {
	Cells         map[Coord]*Cell
	ColWidths     map[int]int // largura por coluna (default: 9)
	Cursor        Coord
	ViewRow       int // primeira linha visivel
	ViewCol       int // primeira coluna visivel
	Filename      string
	Modified      bool
	DefaultFormat CellFormat
	DefaultWidth  int
	Title         string // titulo global (/Title no SC2)

	// Titulos fixos (freeze) - /Title Horizontal e /Title Vertical do SC2
	TitleRow int // linha de titulo fixo (0 = nenhum)
	TitleCol int // coluna de titulo fixo (0 = nenhum)
}

// NewSpreadsheet cria planilha vazia com defaults do SC2
func NewSpreadsheet() *Spreadsheet {
	return &Spreadsheet{
		Cells:         make(map[Coord]*Cell),
		ColWidths:     make(map[int]int),
		Cursor:        Coord{Row: 1, Col: 1},
		ViewRow:       1,
		ViewCol:       1,
		DefaultFormat: CellFormat{Type: FormatGeneral, Decimals: 2},
		DefaultWidth:  9,
	}
}

// GetCell retorna celula (nunca nil - retorna celula vazia se nao existir)
func (s *Spreadsheet) GetCell(c Coord) *Cell {
	if cell, ok := s.Cells[c]; ok {
		return cell
	}
	return &Cell{Type: CellEmpty}
}

// SetCell grava celula (nil ou CellEmpty remove do map)
func (s *Spreadsheet) SetCell(c Coord, cell *Cell) {
	if cell == nil || cell.Type == CellEmpty {
		delete(s.Cells, c)
	} else {
		s.Cells[c] = cell
	}
	s.Modified = true
}

// GetColWidth retorna largura da coluna (default SC2 = 9)
func (s *Spreadsheet) GetColWidth(col int) int {
	if w, ok := s.ColWidths[col]; ok && w > 0 {
		return w
	}
	return s.DefaultWidth
}

// SetColWidth define largura de coluna (comando /Width do SC2)
// SC2 aceita 1-72
func (s *Spreadsheet) SetColWidth(col, w int) {
	if w < 1 {
		w = 1
	}
	if w > 72 {
		w = 72
	}
	s.ColWidths[col] = w
}

// ─── Formatacao de celula para exibicao ──────────────────────────────────────

// FormatCellValue formata o valor de uma celula para caber em 'width' caracteres
// Comportamento fiel ao SC2 MSX:
//
//	Texto:  alinhado a esquerda, truncado na borda (SEM overflow para proxima celula)
//	Numero: alinhado a direita, "****" se nao couber
//	Erro:   codigo de erro centralizado (ex: #DIV/0!)
//	Vazio:  espacos
func (s *Spreadsheet) FormatCellValue(cell *Cell, width int) string {
	if cell == nil || cell.Type == CellEmpty {
		return strings.Repeat(" ", width)
	}

	switch cell.Type {
	case CellText:
		return formatText(cell.TextValue, cell.Align, width)

	case CellNumber, CellFormula:
		// Formula com erro
		if cell.TextValue != "" && strings.HasPrefix(cell.TextValue, "#") {
			return formatError(cell.TextValue, width)
		}
		return formatNumeric(cell.NumericValue, cell.Format, width)
	}

	return strings.Repeat(" ", width)
}

// formatText - texto alinhado a esquerda, truncado (SC2 nao faz overflow)
func formatText(text string, align Alignment, width int) string {
	runes := []rune(text)
	if len(runes) > width {
		runes = runes[:width]
	}
	s := string(runes)
	switch align {
	case AlignRight:
		return fmt.Sprintf("%*s", width, s)
	case AlignCenter:
		pad := width - len([]rune(s))
		pL := pad / 2
		pR := pad - pL
		return strings.Repeat(" ", pL) + s + strings.Repeat(" ", pR)
	default: // AlignLeft, AlignDefault
		return fmt.Sprintf("%-*s", width, s)
	}
}

// formatError - codigo de erro (ex: #DIV/0!) centralizado ou truncado
func formatError(code string, width int) string {
	if len(code) > width {
		return strings.Repeat("*", width)
	}
	// Centraliza o codigo de erro
	pad := width - len(code)
	pL := pad / 2
	pR := pad - pL
	return strings.Repeat(" ", pL) + code + strings.Repeat(" ", pR)
}

// formatNumeric - numero alinhado a direita, asteriscos se nao couber
func formatNumeric(v float64, f CellFormat, width int) string {
	s := formatNumber(v, f)
	if len(s) > width {
		return strings.Repeat("*", width) // Overflow: SC2 exibe ****
	}
	return fmt.Sprintf("%*s", width, s)
}

// formatNumber converte float64 para string no formato especificado
func formatNumber(v float64, f CellFormat) string {
	dec := f.Decimals
	if dec < 0 {
		dec = 2
	}

	switch f.Type {
	case FormatGeneral:
		// SC2 General: se inteiro exato, sem decimais; senao ate 6 sig digits
		if v == math.Trunc(v) && math.Abs(v) < 1e12 {
			s := fmt.Sprintf("%.0f", v)
			if f.Commas {
				s = addCommas(s)
			}
			return s
		}
		s := fmt.Sprintf("%.6g", v)
		return s

	case FormatDefault:
		// D: 2 casas decimais (padrao SC2)
		s := fmt.Sprintf("%.2f", v)
		if f.Commas {
			s = addCommasFloat(s)
		}
		return s

	case FormatInteger:
		// I: inteiro (trunca)
		s := fmt.Sprintf("%.0f", math.Trunc(v))
		if f.Commas {
			s = addCommas(s)
		}
		return s

	case FormatFixed:
		// F: ponto fixo com N casas
		s := fmt.Sprintf("%."+strconv.Itoa(dec)+"f", v)
		if f.Commas {
			s = addCommasFloat(s)
		}
		return s

	case FormatScientific:
		// S: notacao cientifica
		return fmt.Sprintf("%."+strconv.Itoa(dec)+"E", v)

	case FormatDollar:
		// $: moeda
		s := fmt.Sprintf("%."+strconv.Itoa(dec)+"f", math.Abs(v))
		if f.Commas {
			s = addCommasFloat(s)
		}
		if v < 0 {
			return "-$" + s
		}
		return "$" + s

	case FormatPercent:
		// %: percentual
		return fmt.Sprintf("%."+strconv.Itoa(dec)+"f%%", v*100)

	case FormatBar:
		// *: barra grafica SC2 - exibe '*' repetido proporcional ao valor
		// (usado em graficos de texto)
		n := int(math.Abs(v))
		if n > 50 {
			n = 50
		}
		return strings.Repeat("*", n)
	}

	return fmt.Sprintf("%v", v)
}

// addCommas insere separadores de milhar em string inteira
func addCommas(s string) string {
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	n := len(s)
	if n <= 3 {
		if neg {
			return "-" + s
		}
		return s
	}
	var b strings.Builder
	for i, ch := range s {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteRune('.')
		}
		b.WriteRune(ch)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// addCommasFloat insere separadores de milhar em string float (antes do ponto)
func addCommasFloat(s string) string {
	dot := strings.Index(s, ".")
	if dot < 0 {
		return addCommas(s)
	}
	return addCommas(s[:dot]) + s[dot:]
}

// ─── Movimentacao e scroll ────────────────────────────────────────────────────

// MoveCursor move o cursor respeitando limites SC2 (254 linhas, 63 colunas)
func (s *Spreadsheet) MoveCursor(dRow, dCol int) {
	r := s.Cursor.Row + dRow
	c := s.Cursor.Col + dCol
	if r < 1 {
		r = 1
	}
	if r > 254 {
		r = 254
	}
	if c < 1 {
		c = 1
	}
	if c > 63 {
		c = 63
	}
	s.Cursor = Coord{Row: r, Col: c}
}

// AdjustView ajusta a janela de visualizacao para manter cursor visivel
func (s *Spreadsheet) AdjustView(visRows, visCols int) {
	// Vertical
	if s.Cursor.Row < s.ViewRow {
		s.ViewRow = s.Cursor.Row
	} else if s.Cursor.Row >= s.ViewRow+visRows {
		s.ViewRow = s.Cursor.Row - visRows + 1
	}
	// Horizontal
	if s.Cursor.Col < s.ViewCol {
		s.ViewCol = s.Cursor.Col
	} else if s.Cursor.Col >= s.ViewCol+visCols {
		s.ViewCol = s.Cursor.Col - visCols + 1
	}
	if s.ViewRow < 1 {
		s.ViewRow = 1
	}
	if s.ViewCol < 1 {
		s.ViewCol = 1
	}
}

// ─── Recalculo ────────────────────────────────────────────────────────────────

// Recalc recalcula todas as formulas da planilha
func (s *Spreadsheet) Recalc() {
	NewEvaluator(s).Recalc()
}

// EvalCellFormula avalia formula de uma celula especifica
func (s *Spreadsheet) EvalCellFormula(coord Coord) {
	cell := s.GetCell(coord)
	if cell.Type != CellFormula {
		return
	}
	e := NewEvaluator(s)
	val, err := e.EvalFormula(coord, cell.Formula)
	if err != nil {
		cell.NumericValue = 0
		cell.TextValue = err.Error()
	} else {
		cell.NumericValue = val
		cell.TextValue = ""
	}
	s.Cells[coord] = cell
}

// ─── Operacoes em bloco (base para /Copy, /Move, /Delete, /Insert) ───────────

// ClearRange apaga todas as celulas em um retangulo
func (s *Spreadsheet) ClearRange(from, to Coord) {
	minR, maxR := from.Row, to.Row
	minC, maxC := from.Col, to.Col
	if minR > maxR {
		minR, maxR = maxR, minR
	}
	if minC > maxC {
		minC, maxC = maxC, minC
	}
	for r := minR; r <= maxR; r++ {
		for c := minC; c <= maxC; c++ {
			delete(s.Cells, Coord{Row: r, Col: c})
		}
	}
	s.Modified = true
}

// InsertRow insere linha vazia em rowNum, empurra linhas abaixo para baixo
func (s *Spreadsheet) InsertRow(rowNum int) {
	newCells := make(map[Coord]*Cell)
	for coord, cell := range s.Cells {
		if coord.Row >= rowNum {
			newCells[Coord{Row: coord.Row + 1, Col: coord.Col}] = cell
		} else {
			newCells[coord] = cell
		}
	}
	s.Cells = newCells
	s.Modified = true
}

// DeleteRow remove a linha rowNum e sobe as linhas abaixo
func (s *Spreadsheet) DeleteRow(rowNum int) {
	newCells := make(map[Coord]*Cell)
	for coord, cell := range s.Cells {
		if coord.Row == rowNum {
			continue // remove
		}
		if coord.Row > rowNum {
			newCells[Coord{Row: coord.Row - 1, Col: coord.Col}] = cell
		} else {
			newCells[coord] = cell
		}
	}
	s.Cells = newCells
	s.Modified = true
}

// InsertCol insere coluna vazia em colNum, empurra colunas a direita
func (s *Spreadsheet) InsertCol(colNum int) {
	newCells := make(map[Coord]*Cell)
	for coord, cell := range s.Cells {
		if coord.Col >= colNum {
			newCells[Coord{Row: coord.Row, Col: coord.Col + 1}] = cell
		} else {
			newCells[coord] = cell
		}
	}
	s.Cells = newCells
	s.Modified = true
}

// DeleteCol remove a coluna colNum e move as demais para a esquerda
func (s *Spreadsheet) DeleteCol(colNum int) {
	newCells := make(map[Coord]*Cell)
	for coord, cell := range s.Cells {
		if coord.Col == colNum {
			continue
		}
		if coord.Col > colNum {
			newCells[Coord{Row: coord.Row, Col: coord.Col - 1}] = cell
		} else {
			newCells[coord] = cell
		}
	}
	s.Cells = newCells
	s.Modified = true
}
