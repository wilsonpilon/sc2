// internal/spreadsheet/sdi.go
// Leitor e gravador do formato SDI (SuperData Interchange) do SuperCalc 2 MSX
//
// O formato SDI e o unico formato de arquivo do SC2 MSX.
// O arquivo .CAL e armazenado internamente no disco MSX como SDI.
// O utilitario SDI.COM converte entre .CAL (binario MSX) e .SDI (texto ASCII).
// Nos gravamos/lemos diretamente no formato SDI texto, que e 100% compativel.
//
// Estrutura do arquivo SDI:
//
//   SECAO DE CABECALHO (opcional):
//     TABLE
//     0,1              <- linha de inicio (sempre 0,1 para SC2)
//     ""               <- vazio (inicio do cabecalho)
//     COL-FORMAT       <- formato de coluna especifica
//     N,W              <- coluna N, largura W
//     CODIGO_FORMATO   <- ex: $ I G TL TR *
//     GDISP-FORMAT     <- formato global de display
//     9,0              <- largura global, 0 = sem decimais especiais
//     GTL              <- formato padrao (G=geral, TL=texto esquerda)
//     DATA             <- inicio da secao de dados
//     0,0              <- coordenada inicial (col=0 linha=0 = A1)
//     ""               <- vazio
//
//   SECAO DE DADOS (uma entrada por celula, percorrendo linha por linha):
//     -1,0             <- BOT (Begin Of Track = inicio de nova linha SC2)
//     BOT
//     tipo,valor_num   <- campo1=tipo, campo2=valor numerico
//     valor_str        <- campo3=valor string
//
//   Tipos de campo1:
//      0   Dado numerico
//      1   Sequencia de texto (campo2=0:texto normal, 1:texto repetido)
//     -1   Definicao de dados (BOT ou EOD)
//     -2   Especificador de origem (pulo de celula)
//     -3   Formato a nivel de entrada
//     -4   Formula
//     -5   Contador de repeticao
//
//   Indicadores de valor (campo3 para tipo=0):
//     V       valor numerico valido
//     NA      nao disponivel
//     NULL    vazio
//     ERROR   erro
//
//   FIM DO ARQUIVO:
//     -1,0
//     EOD
//
// Exemplo completo de arquivo SDI:
//   TABLE
//   0,1
//   ""
//   GDISP-FORMAT
//   9,0
//   GTL
//   DATA
//   0,0
//   ""
//   -1,0
//   BOT
//   0,1500
//   V
//   -4,0
//   +B2-B3
//   -1,0
//   EOD

package spreadsheet

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

// ─── GRAVACAO SDI ─────────────────────────────────────────────────────────────

// SaveSDI grava a planilha no formato SDI para o arquivo especificado
func (s *Spreadsheet) SaveSDI(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("erro ao criar arquivo: %w", err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)

	// ── Secao de cabecalho ──
	fmt.Fprintln(w, "TABLE")
	fmt.Fprintln(w, "0,1")
	fmt.Fprintln(w, `""`)

	// Larguras de colunas nao-padrao
	for col := 1; col <= 63; col++ {
		if ww, ok := s.ColWidths[col]; ok && ww != s.DefaultWidth {
			fmt.Fprintln(w, "COL-FORMAT")
			fmt.Fprintf(w, "%d,%d\n", col, ww)
			fmt.Fprintln(w, "G") // formato geral para a coluna
		}
	}

	// Formato global de display
	fmt.Fprintln(w, "GDISP-FORMAT")
	fmt.Fprintf(w, "%d,0\n", s.DefaultWidth)
	fmt.Fprintln(w, "GTL") // G=geral, TL=texto esquerda (padrao SC2)

	// ── Inicio da secao de dados ──
	fmt.Fprintln(w, "DATA")
	fmt.Fprintln(w, "0,0")
	fmt.Fprintln(w, `""`)

	// Percorre as linhas em ordem
	// Descobre range de linhas e colunas utilizadas
	maxRow, maxCol := 0, 0
	for coord := range s.Cells {
		if coord.Row > maxRow {
			maxRow = coord.Row
		}
		if coord.Col > maxCol {
			maxCol = coord.Col
		}
	}

	if maxRow == 0 {
		// Planilha vazia
		fmt.Fprintln(w, "-1,0")
		fmt.Fprintln(w, "EOD")
		return w.Flush()
	}

	for row := 1; row <= maxRow; row++ {
		// BOT = Begin Of Track (inicio de linha no SC2)
		fmt.Fprintln(w, "-1,0")
		fmt.Fprintln(w, "BOT")

		for col := 1; col <= maxCol; col++ {
			coord := Coord{Row: row, Col: col}
			cell, exists := s.Cells[coord]
			if !exists || cell == nil {
				continue
			}
			writeSDICell(w, col, cell)
		}
	}

	// EOD = End Of Data
	fmt.Fprintln(w, "-1,0")
	fmt.Fprintln(w, "EOD")

	return w.Flush()
}

// writeSDICell grava uma celula no formato SDI
func writeSDICell(w *bufio.Writer, col int, cell *Cell) {
	// Especificador de origem: pula para a coluna correta
	// (col-1 porque SDI e 0-based internamente via notacao col:row)
	// Usamos -2 para indicar a coluna exata quando necessario
	// Na pratica, o SC2 grava celulas consecutivas dentro da linha
	// e usa -2 para pular celulas vazias
	// Simplificacao: para cada celula nao vazia, gravamos diretamente

	switch cell.Type {
	case CellNumber:
		// Tipo 0: dado numerico
		fmt.Fprintf(w, "0,%s\n", formatSDINumber(cell.NumericValue))
		fmt.Fprintln(w, "V")

	case CellText:
		// Tipo 1: sequencia de texto
		fmt.Fprintln(w, "1,0")
		// Texto entre aspas se contiver espacos ou for vazio
		text := cell.TextValue
		if text == "" || strings.Contains(text, " ") {
			fmt.Fprintf(w, "%q\n", text)
		} else {
			fmt.Fprintln(w, text)
		}

	case CellFormula:
		// Tipo -4: formula
		fmt.Fprintln(w, "-4,0")
		fmt.Fprintln(w, cell.Formula)
	}
}

// formatSDINumber formata numero para SDI (sem trailing zeros desnecessarios)
func formatSDINumber(v float64) string {
	if v == math.Trunc(v) && math.Abs(v) < 1e15 {
		return fmt.Sprintf("%.0f", v)
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// ─── LEITURA SDI ──────────────────────────────────────────────────────────────

// LoadSDI carrega uma planilha de um arquivo SDI
func LoadSDI(path string) (*Spreadsheet, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir arquivo: %w", err)
	}
	defer f.Close()

	s := NewSpreadsheet()
	s.Filename = path

	scanner := bufio.NewScanner(f)
	lines := []string{}
	for scanner.Scan() {
		lines = append(lines, strings.TrimRight(scanner.Text(), "\r"))
	}

	p := &sdiParser{lines: lines, sheet: s}
	if err := p.parse(); err != nil {
		return nil, err
	}

	s.Recalc()
	return s, nil
}

// sdiParser faz o parsing do arquivo SDI linha a linha
type sdiParser struct {
	lines  []string
	pos    int
	sheet  *Spreadsheet
	curRow int // linha atual (1-based)
	curCol int // coluna atual dentro da linha (1-based)
}

func (p *sdiParser) peek() string {
	if p.pos >= len(p.lines) {
		return ""
	}
	return p.lines[p.pos]
}

func (p *sdiParser) next() string {
	if p.pos >= len(p.lines) {
		return ""
	}
	l := p.lines[p.pos]
	p.pos++
	return l
}

func (p *sdiParser) done() bool { return p.pos >= len(p.lines) }

func (p *sdiParser) parse() error {
	// Pula ate TABLE ou DATA
	for !p.done() {
		line := p.peek()
		if line == "TABLE" || line == "DATA" {
			break
		}
		p.next()
	}

	if p.done() {
		return nil
	}

	// Processa secao TABLE (cabecalho)
	if p.peek() == "TABLE" {
		p.next()
		if err := p.parseHeader(); err != nil {
			return err
		}
	}

	// Processa secao DATA
	if p.peek() == "DATA" {
		p.next()
		if err := p.parseData(); err != nil {
			return err
		}
	}

	return nil
}

func (p *sdiParser) parseHeader() error {
	for !p.done() {
		line := p.next()

		switch line {
		case "DATA":
			p.pos-- // volta para DATA ser processado pelo parse()
			return nil

		case "COL-FORMAT":
			// Proximo: "col,largura"
			spec := p.next()
			parts := strings.SplitN(spec, ",", 2)
			if len(parts) == 2 {
				col, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
				ww, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
				if col >= 1 && col <= 63 && ww >= 1 {
					p.sheet.ColWidths[col] = ww
				}
			}
			p.next() // consome o codigo de formato

		case "GDISP-FORMAT":
			// Proximo: "largura,decimais"
			spec := p.next()
			parts := strings.SplitN(spec, ",", 2)
			if len(parts) >= 1 {
				ww, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
				if ww >= 1 {
					p.sheet.DefaultWidth = ww
				}
			}
			p.next() // consome o codigo de formato (GTL, G, etc)
		}
		// Outras linhas do cabecalho sao ignoradas
	}
	return nil
}

func (p *sdiParser) parseData() error {
	// Consome linha de inicio "0,0" e string vazia
	for !p.done() {
		line := p.peek()
		if line == "-1,0" || line == "BOT" {
			break
		}
		p.next()
	}

	p.curRow = 0
	p.curCol = 1

	for !p.done() {
		// Le campo1,campo2
		field12 := p.next()
		if field12 == "" {
			continue
		}

		parts := strings.SplitN(field12, ",", 2)
		if len(parts) != 2 {
			// Pode ser BOT ou EOD orfao
			continue
		}

		tipo, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			continue
		}

		val2Str := strings.TrimSpace(parts[1])

		switch tipo {
		case -1: // Definicao de dados
			field3 := p.next()
			switch strings.TrimSpace(field3) {
			case "BOT":
				p.curRow++
				p.curCol = 1
			case "EOD":
				return nil
			}

		case -2: // Especificador de origem: pula para coluna/linha
			field3 := p.next() // "col:row"
			parts3 := strings.SplitN(field3, ":", 2)
			if len(parts3) == 2 {
				col, _ := strconv.Atoi(strings.TrimSpace(parts3[0]))
				row, _ := strconv.Atoi(strings.TrimSpace(parts3[1]))
				if col >= 1 {
					p.curCol = col
				}
				if row >= 1 {
					p.curRow = row
				}
			}

		case -3: // Formato a nivel de entrada
			field3 := p.next()
			// Aplica formato a celula anterior
			coord := Coord{Row: p.curRow, Col: p.curCol - 1}
			if cell, ok := p.sheet.Cells[coord]; ok {
				applySDIFormat(cell, field3)
			}

		case -4: // Formula
			formula := p.next()
			if p.curRow >= 1 && p.curCol >= 1 {
				coord := Coord{Row: p.curRow, Col: p.curCol}
				p.sheet.SetCell(coord, &Cell{
					Type:     CellFormula,
					Formula:  strings.TrimSpace(formula),
					RawInput: strings.TrimSpace(formula),
				})
				p.curCol++
			}

		case -5: // Contador de repeticao
			count, _ := strconv.Atoi(val2Str)
			field3 := p.next() // deve ser "R"
			_ = field3
			// Repete a ultima celula N vezes
			if p.curCol >= 2 {
				prev := Coord{Row: p.curRow, Col: p.curCol - 1}
				if prevCell, ok := p.sheet.Cells[prev]; ok {
					for i := 0; i < count && p.curCol <= 63; i++ {
						coord := Coord{Row: p.curRow, Col: p.curCol}
						// Copia a celula anterior
						clone := *prevCell
						p.sheet.Cells[coord] = &clone
						p.curCol++
					}
				}
			}

		case 0: // Dado numerico
			field3 := p.next() // V, NA, NULL, ERROR
			if p.curRow >= 1 && p.curCol >= 1 {
				coord := Coord{Row: p.curRow, Col: p.curCol}
				indicator := strings.TrimSpace(field3)
				switch indicator {
				case "V":
					numVal, _ := strconv.ParseFloat(strings.TrimSpace(val2Str), 64)
					p.sheet.SetCell(coord, &Cell{
						Type:         CellNumber,
						NumericValue: numVal,
						RawInput:     val2Str,
					})
				case "NA":
					p.sheet.SetCell(coord, &Cell{
						Type:      CellFormula,
						Formula:   "NA",
						TextValue: "N/A",
					})
				case "ERROR":
					p.sheet.SetCell(coord, &Cell{
						Type:      CellFormula,
						Formula:   "ERROR",
						TextValue: "#ERROR!",
					})
					// NULL: celula vazia, nao grava
				}
				p.curCol++
			}

		case 1: // Sequencia de texto
			textType, _ := strconv.Atoi(val2Str) // 0=normal, 1=repetido
			field3 := p.next()                   // o texto em si
			text := parseSDIString(field3)

			if p.curRow >= 1 && p.curCol >= 1 {
				coord := Coord{Row: p.curRow, Col: p.curCol}
				if textType == 1 {
					// Texto repetido (comeca com ')
					p.sheet.SetCell(coord, &Cell{
						Type:      CellText,
						TextValue: "'" + text,
						RawInput:  "'" + text,
					})
				} else {
					p.sheet.SetCell(coord, &Cell{
						Type:      CellText,
						TextValue: text,
						RawInput:  text,
					})
				}
				p.curCol++
			}
		}
	}
	return nil
}

// parseSDIString remove aspas de uma string SDI se presentes
func parseSDIString(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// applySDIFormat aplica codigo de formato SDI a uma celula
func applySDIFormat(cell *Cell, code string) {
	code = strings.TrimSpace(strings.ToUpper(code))
	switch code {
	case "I":
		cell.Format.Type = FormatInteger
	case "G":
		cell.Format.Type = FormatGeneral
	case "E":
		cell.Format.Type = FormatScientific
	case "$":
		cell.Format.Type = FormatDollar
	case "R":
		cell.Align = AlignRight
	case "L":
		cell.Align = AlignLeft
	case "TR":
		cell.Align = AlignRight
	case "TL":
		cell.Align = AlignLeft
	case "*":
		cell.Format.Type = FormatBar
	case "H":
		// Hide - por hora ignora
	}
}
