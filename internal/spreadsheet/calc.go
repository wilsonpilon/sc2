// internal/spreadsheet/calc.go
// Recalculo ordenado de formulas - compativel com SC2 MSX
//
// O SC2 recalcula na ordem: linha por linha, coluna por coluna (Row order).
// Pode ser alternado para Column order via /G.
// Duas passagens garantem resolucao de dependencias simples.
// Referencia circular e detectada e reportada como #CIRC!
//
// Ordenacao (/A Arrange):
// O SC2 MSX suporta ordenacao por linha ou coluna, ascendente ou descendente.

package spreadsheet

import (
	"sort"
	"strings"
)

// CalcOrder define a ordem de recalculo
type CalcOrder int

const (
	CalcRowOrder    CalcOrder = iota // Linha por linha (padrao SC2)
	CalcColumnOrder                  // Coluna por coluna
)

// RecalcOrdered recalcula todas as formulas na ordem definida.
// Substitui o Recalc() simples com suporte a CalcOrder e duas passagens.
func (s *Spreadsheet) RecalcOrdered(order CalcOrder) {
	e := NewEvaluator(s)

	// Coleta coordenadas de celulas com formula
	var formulaCells []Coord
	for coord, cell := range s.Cells {
		if cell.Type == CellFormula {
			formulaCells = append(formulaCells, coord)
		}
	}

	// Ordena conforme a ordem de calculo
	if order == CalcRowOrder {
		// Linha por linha: row ASC, col ASC
		sort.Slice(formulaCells, func(i, j int) bool {
			if formulaCells[i].Row != formulaCells[j].Row {
				return formulaCells[i].Row < formulaCells[j].Row
			}
			return formulaCells[i].Col < formulaCells[j].Col
		})
	} else {
		// Coluna por coluna: col ASC, row ASC
		sort.Slice(formulaCells, func(i, j int) bool {
			if formulaCells[i].Col != formulaCells[j].Col {
				return formulaCells[i].Col < formulaCells[j].Col
			}
			return formulaCells[i].Row < formulaCells[j].Row
		})
	}

	// Duas passagens para resolver dependencias encadeadas
	for pass := 0; pass < 2; pass++ {
		for _, coord := range formulaCells {
			cell, ok := s.Cells[coord]
			if !ok || cell.Type != CellFormula {
				continue
			}
			e.visiting = make(map[Coord]bool)
			val, err := e.EvalFormula(coord, cell.Formula)
			if err != nil {
				cell.NumericValue = 0
				if isCircularError(err) {
					cell.TextValue = "#CIRC!"
				} else {
					cell.TextValue = err.Error()
				}
			} else {
				cell.NumericValue = val
				cell.TextValue = ""
			}
		}
	}
}

// isCircularError verifica se o erro e de referencia circular
func isCircularError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "CIRC")
}

// ─── Arrange (/A) ─────────────────────────────────────────────────────────────

// ArrangeDirection define a direcao da ordenacao
type ArrangeDirection int

const (
	ArrangeAscending ArrangeDirection = iota
	ArrangeDescending
)

// ArrangeRows ordena um intervalo de linhas baseado nos valores de uma coluna.
// Equivalente ao /A Row do SC2 MSX.
//
// Parametros:
//
//	fromRow, toRow: intervalo de linhas a ordenar
//	keyCol:         coluna usada como chave de ordenacao
//	dir:            Ascending ou Descending
//	adjustFormulas: se true, ajusta referencias de formulas apos ordenar
func (s *Spreadsheet) ArrangeRows(fromRow, toRow, keyCol int, dir ArrangeDirection, adjustFormulas bool) {
	if fromRow > toRow {
		fromRow, toRow = toRow, fromRow
	}
	if fromRow < 1 {
		fromRow = 1
	}
	if toRow > 254 {
		toRow = 254
	}

	// Descobre o range de colunas usadas
	maxCol := 0
	for coord := range s.Cells {
		if coord.Row >= fromRow && coord.Row <= toRow {
			if coord.Col > maxCol {
				maxCol = coord.Col
			}
		}
	}
	if maxCol == 0 {
		return
	}

	// Copia as linhas para slice de maps
	type rowData struct {
		row   int
		cells map[int]*Cell // col -> cell
	}
	rows := make([]rowData, toRow-fromRow+1)
	for i := range rows {
		rows[i].row = fromRow + i
		rows[i].cells = make(map[int]*Cell)
		for col := 1; col <= maxCol; col++ {
			coord := Coord{Row: fromRow + i, Col: col}
			if cell, ok := s.Cells[coord]; ok {
				clone := *cell
				rows[i].cells[col] = &clone
			}
		}
	}

	// Funcao de comparacao pela coluna chave
	keyVal := func(rd rowData) float64 {
		if cell, ok := rd.cells[keyCol]; ok {
			switch cell.Type {
			case CellNumber, CellFormula:
				return cell.NumericValue
			case CellText:
				// Texto: usa valor ASCII do primeiro caractere para comparacao
				if len(cell.TextValue) > 0 {
					return float64(cell.TextValue[0])
				}
			}
		}
		return 0
	}

	// Ordena
	sort.SliceStable(rows, func(i, j int) bool {
		vi := keyVal(rows[i])
		vj := keyVal(rows[j])
		if dir == ArrangeAscending {
			return vi < vj
		}
		return vi > vj
	})

	// Limpa o intervalo original
	for row := fromRow; row <= toRow; row++ {
		for col := 1; col <= maxCol; col++ {
			delete(s.Cells, Coord{Row: row, Col: col})
		}
	}

	// Reescreve na nova ordem
	for newIdx, rd := range rows {
		newRow := fromRow + newIdx
		rowDelta := newRow - rd.row // quanto esta linha se moveu
		for col, cell := range rd.cells {
			clone := *cell
			// Ajusta referencias de formulas se necessario
			if adjustFormulas && clone.Type == CellFormula && rowDelta != 0 {
				clone.Formula = adjustFormulaRow(clone.Formula, rowDelta)
			}
			s.Cells[Coord{Row: newRow, Col: col}] = &clone
		}
	}

	s.Modified = true
}

// ArrangeCols ordena um intervalo de colunas baseado nos valores de uma linha.
// Equivalente ao /A Column do SC2 MSX.
func (s *Spreadsheet) ArrangeCols(fromCol, toCol, keyRow int, dir ArrangeDirection, adjustFormulas bool) {
	if fromCol > toCol {
		fromCol, toCol = toCol, fromCol
	}
	if fromCol < 1 {
		fromCol = 1
	}
	if toCol > 63 {
		toCol = 63
	}

	// Descobre o range de linhas usadas
	maxRow := 0
	for coord := range s.Cells {
		if coord.Col >= fromCol && coord.Col <= toCol {
			if coord.Row > maxRow {
				maxRow = coord.Row
			}
		}
	}
	if maxRow == 0 {
		return
	}

	type colData struct {
		col   int
		cells map[int]*Cell // row -> cell
	}
	cols := make([]colData, toCol-fromCol+1)
	for i := range cols {
		cols[i].col = fromCol + i
		cols[i].cells = make(map[int]*Cell)
		for row := 1; row <= maxRow; row++ {
			coord := Coord{Row: row, Col: fromCol + i}
			if cell, ok := s.Cells[coord]; ok {
				clone := *cell
				cols[i].cells[row] = &clone
			}
		}
	}

	keyVal := func(cd colData) float64 {
		if cell, ok := cd.cells[keyRow]; ok {
			switch cell.Type {
			case CellNumber, CellFormula:
				return cell.NumericValue
			case CellText:
				if len(cell.TextValue) > 0 {
					return float64(cell.TextValue[0])
				}
			}
		}
		return 0
	}

	sort.SliceStable(cols, func(i, j int) bool {
		vi := keyVal(cols[i])
		vj := keyVal(cols[j])
		if dir == ArrangeAscending {
			return vi < vj
		}
		return vi > vj
	})

	// Limpa e reescreve
	for col := fromCol; col <= toCol; col++ {
		for row := 1; row <= maxRow; row++ {
			delete(s.Cells, Coord{Row: row, Col: col})
		}
	}
	for newIdx, cd := range cols {
		newCol := fromCol + newIdx
		colDelta := newCol - cd.col
		for row, cell := range cd.cells {
			clone := *cell
			if adjustFormulas && clone.Type == CellFormula && colDelta != 0 {
				clone.Formula = adjustFormulaCol(clone.Formula, colDelta)
			}
			s.Cells[Coord{Row: row, Col: newCol}] = &clone
		}
	}

	s.Modified = true
}

// adjustFormulaRow ajusta os numeros de linha em referencias de formulas
func adjustFormulaRow(formula string, delta int) string {
	if delta == 0 {
		return formula
	}
	result := strings.Builder{}
	i := 0
	f := strings.ToUpper(formula)
	for i < len(f) {
		ch := f[i]
		if ch >= 'A' && ch <= 'Z' {
			j := i
			for j < len(f) && f[j] >= 'A' && f[j] <= 'Z' {
				j++
			}
			k := j
			for k < len(f) && f[k] >= '0' && f[k] <= '9' {
				k++
			}
			if k > j && j > i && j-i <= 2 {
				ref := f[i:k]
				coord, err := ParseCoord(ref)
				if err == nil {
					newRow := coord.Row + delta
					if newRow >= 1 && newRow <= 254 {
						result.WriteString(Coord{Row: newRow, Col: coord.Col}.String())
					} else {
						result.WriteString(ref)
					}
					i = k
					continue
				}
			}
			result.WriteByte(formula[i])
			i++
			continue
		}
		result.WriteByte(formula[i])
		i++
	}
	return result.String()
}

// adjustFormulaCol ajusta os numeros de coluna em referencias de formulas
func adjustFormulaCol(formula string, delta int) string {
	if delta == 0 {
		return formula
	}
	result := strings.Builder{}
	i := 0
	f := strings.ToUpper(formula)
	for i < len(f) {
		ch := f[i]
		if ch >= 'A' && ch <= 'Z' {
			j := i
			for j < len(f) && f[j] >= 'A' && f[j] <= 'Z' {
				j++
			}
			k := j
			for k < len(f) && f[k] >= '0' && f[k] <= '9' {
				k++
			}
			if k > j && j > i && j-i <= 2 {
				ref := f[i:k]
				coord, err := ParseCoord(ref)
				if err == nil {
					newCol := coord.Col + delta
					if newCol >= 1 && newCol <= 63 {
						result.WriteString(Coord{Row: coord.Row, Col: newCol}.String())
					} else {
						result.WriteString(ref)
					}
					i = k
					continue
				}
			}
			result.WriteByte(formula[i])
			i++
			continue
		}
		result.WriteByte(formula[i])
		i++
	}
	return result.String()
}
