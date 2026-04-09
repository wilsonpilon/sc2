// internal/spreadsheet/model.go
// Modelo de dados da planilha - compatível com SuperCalc 2 MSX
package spreadsheet

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// CellType representa o tipo de conteúdo de uma célula
type CellType int

const (
	CellEmpty   CellType = iota
	CellText             // Label/String (começa com " ou letra)
	CellNumber           // Valor numérico
	CellFormula          // Fórmula (começa com +, -, (, @, ou número)
)

// Alinhamento da célula
type Alignment int

const (
	AlignDefault Alignment = iota // Default: texto=esquerda, número=direita
	AlignLeft
	AlignRight
	AlignCenter
)

// Cell representa uma célula da planilha
type Cell struct {
	// Conteúdo bruto digitado pelo usuário
	RawInput string

	// Tipo da célula
	Type CellType

	// Valor numérico (se CellNumber ou CellFormula com resultado numérico)
	NumericValue float64

	// Valor de texto (se CellText)
	TextValue string

	// Fórmula (se CellFormula)
	Formula string

	// Formato de exibição (número de casas decimais, etc)
	Format CellFormat

	// Alinhamento
	Align Alignment

	// Largura da coluna (não é da célula em si, mas usamos aqui como override)
	Protected bool
}

// CellFormat define como o valor é exibido
type CellFormat struct {
	Type      FormatType
	Decimals  int  // casas decimais (0-9)
	Commas    bool // separador de milhares
}

type FormatType int

const (
	FormatGeneral FormatType = iota // G - geral
	FormatDefault                   // D - default (integer)
	FormatInteger                   // I - inteiro
	FormatFixed                     // F - ponto fixo
	FormatScientific                // S - científico (E notation)
	FormatDollar                    // $ - dólar
	FormatPercent                   // % - percentual
	FormatBar                       // * - barra gráfica
)

// Coord representa uma coordenada de célula (linha, coluna)
type Coord struct {
	Row int // 1-based (1..254 no SC2)
	Col int // 1-based (1..63 no SC2, A..BK)
}

// String converte Coord para notação SC2 (ex: A1, B12, AA3)
func (c Coord) String() string {
	return ColName(c.Col) + strconv.Itoa(c.Row)
}

// ColName converte número de coluna (1-based) para letra(s)
// SC2 MSX suporta até 63 colunas: A-Z (1-26), AA-AZ (27-52), BA-BK (53-63)
func ColName(col int) string {
	if col <= 0 {
		return "?"
	}
	if col <= 26 {
		return string(rune('A' + col - 1))
	}
	// Dupla letra
	first := (col-1)/26 - 1
	second := (col-1)%26
	// Ajuste SC2: AA=27, AB=28... AZ=52, BA=53...
	major := (col - 27) / 26
	minor := (col - 27) % 26
	_ = first
	_ = second
	return string(rune('A'+major)) + string(rune('A'+minor))
}

// ParseCoord converte string de coordenada para Coord
// Ex: "A1" -> {1,1}, "B12" -> {12,2}, "AA3" -> {3,27}
func ParseCoord(s string) (Coord, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if len(s) < 2 {
		return Coord{}, fmt.Errorf("coordenada inválida: %s", s)
	}

	// Encontra onde terminam as letras
	i := 0
	for i < len(s) && s[i] >= 'A' && s[i] <= 'Z' {
		i++
	}
	if i == 0 || i == len(s) {
		return Coord{}, fmt.Errorf("coordenada inválida: %s", s)
	}

	colStr := s[:i]
	rowStr := s[i:]

	row, err := strconv.Atoi(rowStr)
	if err != nil || row < 1 {
		return Coord{}, fmt.Errorf("linha inválida: %s", rowStr)
	}

	var col int
	if len(colStr) == 1 {
		col = int(colStr[0]-'A') + 1
	} else if len(colStr) == 2 {
		col = (int(colStr[0]-'A')+1)*26 + int(colStr[1]-'A') + 1
	} else {
		return Coord{}, fmt.Errorf("coluna inválida: %s", colStr)
	}

	return Coord{Row: row, Col: col}, nil
}

// Spreadsheet é a planilha principal
type Spreadsheet struct {
	// Células armazenadas como map para eficiência (planilha esparsa)
	Cells map[Coord]*Cell

	// Largura das colunas (em caracteres). Default: 9
	ColWidths map[int]int

	// Altura das linhas (SC2 sempre 1, mas mantemos por extensibilidade)
	// RowHeights map[int]int

	// Cursor atual
	Cursor Coord

	// Janela de visualização (primeira linha/coluna visível)
	ViewRow int
	ViewCol int

	// Nome do arquivo atual
	Filename string

	// Flag de modificação
	Modified bool

	// Configurações globais
	DefaultFormat CellFormat
	DefaultWidth  int
	
	// Título da planilha (não existe no SC2 original, mas útil)
	Title string
}

// NewSpreadsheet cria uma planilha nova vazia
func NewSpreadsheet() *Spreadsheet {
	return &Spreadsheet{
		Cells:     make(map[Coord]*Cell),
		ColWidths: make(map[int]int),
		Cursor:    Coord{Row: 1, Col: 1},
		ViewRow:   1,
		ViewCol:   1,
		DefaultFormat: CellFormat{
			Type:     FormatGeneral,
			Decimals: 2,
		},
		DefaultWidth: 9,
	}
}

// GetCell retorna a célula na coordenada (nunca nil - cria se não existir)
func (s *Spreadsheet) GetCell(c Coord) *Cell {
	if cell, ok := s.Cells[c]; ok {
		return cell
	}
	return &Cell{Type: CellEmpty}
}

// SetCell define uma célula
func (s *Spreadsheet) SetCell(c Coord, cell *Cell) {
	if cell == nil || cell.Type == CellEmpty {
		delete(s.Cells, c)
		return
	}
	s.Cells[c] = cell
	s.Modified = true
}

// GetColWidth retorna a largura da coluna (usa default se não definida)
func (s *Spreadsheet) GetColWidth(col int) int {
	if w, ok := s.ColWidths[col]; ok {
		return w
	}
	return s.DefaultWidth
}

// FormatCellValue formata o valor de uma célula para exibição
func (s *Spreadsheet) FormatCellValue(cell *Cell, width int) string {
	if cell == nil || cell.Type == CellEmpty {
		return strings.Repeat(" ", width)
	}

	switch cell.Type {
	case CellText:
		// Texto: alinha à esquerda, trunca se necessário
		text := cell.TextValue
		if len(text) > width {
			text = text[:width]
		}
		return fmt.Sprintf("%-*s", width, text)

	case CellNumber, CellFormula:
		// Se formula com erro, exibe o codigo de erro alinhado a direita
		if cell.TextValue != "" && strings.HasPrefix(cell.TextValue, "#") {
			errStr := cell.TextValue
			if len(errStr) > width {
				return strings.Repeat("*", width)
			}
			return fmt.Sprintf("%*s", width, errStr)
		}
		// Numero: alinha a direita
		formatted := formatNumber(cell.NumericValue, cell.Format, width)
		if len(formatted) > width {
			// Overflow: exibe asteriscos (padrao SC2)
			return strings.Repeat("*", width)
		}
		return fmt.Sprintf("%*s", width, formatted)
	}

	return strings.Repeat(" ", width)
}

// formatNumber converte float64 para string conforme o formato
func formatNumber(v float64, f CellFormat, width int) string {
	switch f.Type {
	case FormatGeneral, FormatDefault:
		// Inteiro se não tiver decimal relevante
		if v == math.Trunc(v) && math.Abs(v) < 1e12 {
			return fmt.Sprintf("%.0f", v)
		}
		return strconv.FormatFloat(v, 'g', -1, 64)

	case FormatFixed:
		dec := f.Decimals
		if dec < 0 { dec = 2 }
		return fmt.Sprintf("%."+strconv.Itoa(dec)+"f", v)

	case FormatInteger:
		return fmt.Sprintf("%.0f", v)

	case FormatScientific:
		dec := f.Decimals
		if dec < 0 { dec = 2 }
		return fmt.Sprintf("%."+strconv.Itoa(dec)+"E", v)

	case FormatDollar:
		dec := f.Decimals
		if dec < 0 { dec = 2 }
		if v < 0 {
			return fmt.Sprintf("-$%."+strconv.Itoa(dec)+"f", -v)
		}
		return fmt.Sprintf("$%."+strconv.Itoa(dec)+"f", v)

	case FormatPercent:
		dec := f.Decimals
		if dec < 0 { dec = 2 }
		return fmt.Sprintf("%."+strconv.Itoa(dec)+"f%%", v*100)
	}

	return fmt.Sprintf("%v", v)
}

// MoveCursor move o cursor na planilha
func (s *Spreadsheet) MoveCursor(dRow, dCol int) {
	newRow := s.Cursor.Row + dRow
	newCol := s.Cursor.Col + dCol

	// Limites SC2 MSX: 254 linhas, 63 colunas
	if newRow < 1 { newRow = 1 }
	if newRow > 254 { newRow = 254 }
	if newCol < 1 { newCol = 1 }
	if newCol > 63 { newCol = 63 }

	s.Cursor = Coord{Row: newRow, Col: newCol}
}

// AdjustView ajusta a janela de visualização para manter o cursor visível
// visRows e visCols são o número de linhas/colunas visíveis na tela
func (s *Spreadsheet) AdjustView(visRows, visCols int) {
	// Scroll vertical
	if s.Cursor.Row < s.ViewRow {
		s.ViewRow = s.Cursor.Row
	} else if s.Cursor.Row >= s.ViewRow+visRows {
		s.ViewRow = s.Cursor.Row - visRows + 1
	}

	// Scroll horizontal
	if s.Cursor.Col < s.ViewCol {
		s.ViewCol = s.Cursor.Col
	} else if s.Cursor.Col >= s.ViewCol+visCols {
		s.ViewCol = s.Cursor.Col - visCols + 1
	}

	if s.ViewRow < 1 { s.ViewRow = 1 }
	if s.ViewCol < 1 { s.ViewCol = 1 }
}

// Recalc recalcula todas as formulas da planilha
func (s *Spreadsheet) Recalc() {
	e := NewEvaluator(s)
	e.Recalc()
}

// EvalCellFormula avalia a formula de uma celula e atualiza seu valor
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
