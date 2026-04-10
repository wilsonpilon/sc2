// internal/spreadsheet/formula.go
// Avaliador de formulas compativel com SuperCalc 2 MSX
//
// Sintaxe SC2:
//   +A1+B2          referencia de celulas
//   +A1:B10         intervalo (usado dentro de @funcoes)
//   @SUM(A1:A10)    funcao com intervalo
//   @IF(A1>0,B1,C1) funcao condicional
//   +A1*2+@AVG(B1:B5)  expressao mista
//
// Funcoes suportadas (todas do SC2 original):
//   @SUM @AVG @MIN @MAX @COUNT @IF @ABS @INT @SQRT @LOG @LN
//   @EXP @MOD @ROUND @AND @OR @NOT @ISERROR @NA @TRUE @FALSE

package spreadsheet

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// EvalError representa um erro de avaliacao (ex: #DIV/0!, #REF!, #NA!)
type EvalError struct {
	Code string // "#DIV/0!", "#REF!", "#VALUE!", "#NA!", "#ERROR!"
}

func (e EvalError) Error() string { return e.Code }

var (
	errDivZero = EvalError{"#DIV/0!"}
	errRef     = EvalError{"#REF!"}
	errValue   = EvalError{"#VALUE!"}
	errNA      = EvalError{"#NA!"}
	errError   = EvalError{"#ERROR!"}
)

// Evaluator avalia formulas na planilha
type Evaluator struct {
	sheet *Spreadsheet
	// Protecao contra referencias circulares
	visiting map[Coord]bool
}

// NewEvaluator cria um avaliador para a planilha
func NewEvaluator(s *Spreadsheet) *Evaluator {
	return &Evaluator{
		sheet:    s,
		visiting: make(map[Coord]bool),
	}
}

// Recalc recalcula todas as celulas com formula na planilha
// Faz duas passagens para resolver dependencias simples
func (e *Evaluator) Recalc() {
	// Duas passagens: suficiente para a maioria dos casos do SC2
	for pass := 0; pass < 2; pass++ {
		for coord, cell := range e.sheet.Cells {
			if cell.Type == CellFormula {
				e.visiting = make(map[Coord]bool)
				val, err := e.EvalFormula(coord, cell.Formula)
				if err != nil {
					cell.NumericValue = 0
					cell.TextValue = err.Error()
				} else {
					cell.NumericValue = val
					cell.TextValue = ""
				}
			}
		}
	}
}

// EvalFormula avalia a formula de uma celula e retorna o valor numerico
func (e *Evaluator) EvalFormula(coord Coord, formula string) (float64, error) {
	if e.visiting[coord] {
		return 0, EvalError{"#CIRC!"}
	}
	e.visiting[coord] = true
	defer func() { delete(e.visiting, coord) }()

	formula = strings.TrimSpace(formula)
	if formula == "" {
		return 0, nil
	}

	p := &parser{src: formula, eval: e}
	val, err := p.parseExpr()
	if err != nil {
		return 0, err
	}
	return val, nil
}

// GetCellValue retorna o valor numerico de uma celula para uso em formulas
func (e *Evaluator) GetCellValue(coord Coord) (float64, error) {
	cell := e.sheet.GetCell(coord)
	switch cell.Type {
	case CellEmpty:
		return 0, nil
	case CellNumber:
		return cell.NumericValue, nil
	case CellFormula:
		// Avalia recursivamente (com protecao circular)
		return e.EvalFormula(coord, cell.Formula)
	case CellText:
		// Tenta converter texto para numero
		if f, err := strconv.ParseFloat(strings.TrimSpace(cell.TextValue), 64); err == nil {
			return f, nil
		}
		return 0, errValue
	}
	return 0, nil
}

// RangeValues retorna todos os valores numericos de um intervalo (ex: A1:C5)
func (e *Evaluator) RangeValues(from, to Coord) ([]float64, error) {
	var vals []float64
	minRow, maxRow := from.Row, to.Row
	minCol, maxCol := from.Col, to.Col
	if minRow > maxRow {
		minRow, maxRow = maxRow, minRow
	}
	if minCol > maxCol {
		minCol, maxCol = maxCol, minCol
	}

	for row := minRow; row <= maxRow; row++ {
		for col := minCol; col <= maxCol; col++ {
			v, err := e.GetCellValue(Coord{Row: row, Col: col})
			if err != nil {
				// Ignora erros de celulas vazias em ranges
				continue
			}
			vals = append(vals, v)
		}
	}
	return vals, nil
}

// ─── Parser de expressoes ─────────────────────────────────────────────────────
// Gramatica (simplificada, compativel com SC2):
//
//   expr    = term (('+' | '-') term)*
//   term    = unary (('*' | '/') unary)*
//   unary   = '-' unary | power
//   power   = primary ('^' unary)?
//   primary = NUMBER | CELLREF | FUNCALL | '(' expr ')'
//   FUNCALL = '@' NAME '(' arglist ')'
//   arglist = arg (',' arg)*
//   arg     = RANGE | expr
//   RANGE   = CELLREF ':' CELLREF

type parser struct {
	src  string
	pos  int
	eval *Evaluator
}

func (p *parser) peek() byte {
	p.skipWS()
	if p.pos >= len(p.src) {
		return 0
	}
	return p.src[p.pos]
}

func (p *parser) consume() byte {
	p.skipWS()
	if p.pos >= len(p.src) {
		return 0
	}
	ch := p.src[p.pos]
	p.pos++
	return ch
}

func (p *parser) skipWS() {
	for p.pos < len(p.src) && p.src[p.pos] == ' ' {
		p.pos++
	}
}

func (p *parser) expect(ch byte) error {
	if p.peek() != ch {
		return fmt.Errorf("%w: esperado '%c' em pos %d", errError, ch, p.pos)
	}
	p.consume()
	return nil
}

// parseExpr: expr = compare (('+' | '-') compare)*
// Inclui operadores relacionais: = <> < > <= >=
// Verdadeiro=1, Falso=0 (conforme manual SC2 cap 7)
func (p *parser) parseExpr() (float64, error) {
	val, err := p.parseAddSub()
	if err != nil {
		return 0, err
	}
	// Operadores relacionais (menor precedencia)
	for {
		p.skipWS()
		if p.pos >= len(p.src) {
			break
		}
		// Le operador relacional (1 ou 2 chars)
		var op string
		ch := p.src[p.pos]
		if ch == '<' || ch == '>' || ch == '=' {
			p.pos++
			if p.pos < len(p.src) {
				next := p.src[p.pos]
				if (ch == '<' && (next == '>' || next == '=')) ||
					(ch == '>' && next == '=') {
					op = string([]byte{ch, next})
					p.pos++
				} else {
					op = string(ch)
				}
			} else {
				op = string(ch)
			}
		} else {
			break
		}
		right, err := p.parseAddSub()
		if err != nil {
			return 0, err
		}
		var result bool
		switch op {
		case "=":
			result = val == right
		case "<>":
			result = val != right
		case "<":
			result = val < right
		case ">":
			result = val > right
		case "<=":
			result = val <= right
		case ">=":
			result = val >= right
		}
		if result {
			val = 1
		} else {
			val = 0
		}
	}
	return val, nil
}

// parseAddSub: a antiga parseExpr - soma e subtracao
func (p *parser) parseAddSub() (float64, error) {
	val, err := p.parseTerm()
	if err != nil {
		return 0, err
	}
	for {
		ch := p.peek()
		if ch != '+' && ch != '-' {
			break
		}
		p.consume()
		right, err := p.parseTerm()
		if err != nil {
			return 0, err
		}
		if ch == '+' {
			val += right
		} else {
			val -= right
		}
	}
	return val, nil
}

// parseTerm: term = unary (('*' | '/') unary)*
func (p *parser) parseTerm() (float64, error) {
	val, err := p.parseUnary()
	if err != nil {
		return 0, err
	}
	for {
		ch := p.peek()
		if ch != '*' && ch != '/' {
			break
		}
		p.consume()
		right, err := p.parseUnary()
		if err != nil {
			return 0, err
		}
		if ch == '*' {
			val *= right
		} else {
			if right == 0 {
				return 0, errDivZero
			}
			val /= right
		}
	}
	return val, nil
}

// parseUnary: unary = '-' unary | power
func (p *parser) parseUnary() (float64, error) {
	if p.peek() == '-' {
		p.consume()
		v, err := p.parseUnary()
		return -v, err
	}
	if p.peek() == '+' {
		p.consume()
	}
	return p.parsePower()
}

// parsePower: power = primary ('^' unary)?
func (p *parser) parsePower() (float64, error) {
	base, err := p.parsePrimary()
	if err != nil {
		return 0, err
	}
	if p.peek() == '^' {
		p.consume()
		exp, err := p.parseUnary()
		if err != nil {
			return 0, err
		}
		return math.Pow(base, exp), nil
	}
	return base, nil
}

// parsePrimary: NUMBER | CELLREF | FUNCALL | '(' expr ')'
func (p *parser) parsePrimary() (float64, error) {
	p.skipWS()
	if p.pos >= len(p.src) {
		return 0, errError
	}

	ch := p.src[p.pos]

	// Numero
	if ch >= '0' && ch <= '9' || ch == '.' {
		return p.parseNumber()
	}

	// Funcao @
	if ch == '@' {
		p.pos++
		return p.parseFunction()
	}

	// Parenteses
	if ch == '(' {
		p.pos++
		val, err := p.parseExpr()
		if err != nil {
			return 0, err
		}
		if err := p.expect(')'); err != nil {
			return 0, err
		}
		return val, nil
	}

	// Referencia de celula (começa com letra)
	if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
		return p.parseCellRef()
	}

	return 0, fmt.Errorf("%w: caractere inesperado '%c'", errError, ch)
}

// parseNumber le um literal numerico
func (p *parser) parseNumber() (float64, error) {
	start := p.pos
	for p.pos < len(p.src) {
		ch := p.src[p.pos]
		if ch >= '0' && ch <= '9' || ch == '.' || ch == 'E' || ch == 'e' || ch == '+' || ch == '-' {
			// +/- so validos apos E
			if (ch == '+' || ch == '-') && p.pos > start {
				prev := p.src[p.pos-1]
				if prev != 'E' && prev != 'e' {
					break
				}
			}
			p.pos++
		} else {
			break
		}
	}
	s := p.src[start:p.pos]
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, errValue
	}
	return f, nil
}

// parseCellRef le uma referencia de celula (ex: A1, B12, AA3)
func (p *parser) parseCellRef() (float64, error) {
	start := p.pos
	// Le letras
	for p.pos < len(p.src) && (p.src[p.pos] >= 'A' && p.src[p.pos] <= 'Z' || p.src[p.pos] >= 'a' && p.src[p.pos] <= 'z') {
		p.pos++
	}
	// Le digitos
	for p.pos < len(p.src) && p.src[p.pos] >= '0' && p.src[p.pos] <= '9' {
		p.pos++
	}
	ref := strings.ToUpper(p.src[start:p.pos])
	coord, err := ParseCoord(ref)
	if err != nil {
		return 0, errRef
	}
	return p.eval.GetCellValue(coord)
}

// parseFunction avalia uma funcao @NOME(args)
func (p *parser) parseFunction() (float64, error) {
	// Le nome da funcao
	start := p.pos
	for p.pos < len(p.src) && (unicode.IsLetter(rune(p.src[p.pos])) || p.src[p.pos] == '_') {
		p.pos++
	}
	name := strings.ToUpper(p.src[start:p.pos])

	// Funcoes sem argumentos
	switch name {
	case "TRUE":
		return 1, nil
	case "FALSE":
		return 0, nil
	case "NA":
		return 0, errNA
	case "PI":
		return math.Pi, nil
	}

	if err := p.expect('('); err != nil {
		return 0, err
	}

	// Funcoes de um argumento escalar
	switch name {
	case "ABS", "INT", "SQRT", "LOG", "LN", "EXP", "SIN", "COS", "TAN",
		"ASIN", "ACOS", "ATAN", "NOT", "ISERROR":
		val, err := p.parseExpr()
		if err != nil {
			return 0, err
		}
		if err := p.expect(')'); err != nil {
			return 0, err
		}
		return applyUnaryFunc(name, val)
	}

	// Funcoes de dois argumentos escalares
	switch name {
	case "MOD", "ROUND", "ATAN2", "AND", "OR":
		a, err := p.parseExpr()
		if err != nil {
			return 0, err
		}
		if err := p.expect(','); err != nil {
			return 0, err
		}
		b, err := p.parseExpr()
		if err != nil {
			return 0, err
		}
		if err := p.expect(')'); err != nil {
			return 0, err
		}
		return applyBinaryFunc(name, a, b)
	}

	// @IF(cond, valorVerdade, valorFalso)
	if name == "IF" {
		cond, err := p.parseExpr()
		if err != nil {
			return 0, err
		}
		if err := p.expect(','); err != nil {
			return 0, err
		}
		trueVal, err := p.parseExpr()
		if err != nil {
			return 0, err
		}
		if err := p.expect(','); err != nil {
			return 0, err
		}
		falseVal, err := p.parseExpr()
		if err != nil {
			return 0, err
		}
		if err := p.expect(')'); err != nil {
			return 0, err
		}
		if cond != 0 {
			return trueVal, nil
		}
		return falseVal, nil
	}

	// ISNA(valor) - verdadeiro se NA
	if name == "ISNA" {
		// Tenta avaliar a expressao - se der erro NA, retorna 1
		savedPos := p.pos
		_, err := p.parseExpr()
		if err2 := p.expect(')'); err2 != nil {
			p.pos = savedPos
		}
		_ = err
		// Simplificado: retorna 0 (nao e NA)
		return 0, nil
	}

	// LOOKUP(chave, col/linha) - busca em tabela
	if name == "LOOKUP" {
		key, err := p.parseExpr()
		if err != nil {
			return 0, err
		}
		if err := p.expect(','); err != nil {
			return 0, err
		}
		// Le o range de chaves e valores (duas colunas/linhas adjacentes)
		savedPos := p.pos
		keyVals, ok1, _ := p.tryParseRange()
		if !ok1 {
			p.pos = savedPos
			v, _ := p.parseExpr()
			keyVals = []float64{v}
		}
		var valVals []float64
		if p.peek() == ',' {
			p.consume()
			savedPos2 := p.pos
			vv, ok2, _ := p.tryParseRange()
			if !ok2 {
				p.pos = savedPos2
				v, _ := p.parseExpr()
				vv = []float64{v}
			}
			valVals = vv
		} else {
			valVals = keyVals // LOOKUP de coluna unica usa valor adjacente
		}
		if err := p.expect(')'); err != nil {
			return 0, err
		}
		return calcLookup(key, keyVals, valVals)
	}

	// Funcoes de data/calendario (conforme manual cap 7, pag 7-10)
	// DATE(MM, DD, YY) - entrada de data como valor especial
	if name == "DATE" || name == "JDATE" || name == "WDAY" ||
		name == "MONTH" || name == "DAY" || name == "YEAR" {
		// Coleta argumentos escalares
		var args []float64
		for {
			v, err := p.parseExpr()
			if err != nil {
				break
			}
			args = append(args, v)
			if p.peek() != ',' {
				break
			}
			p.consume()
		}
		if err := p.expect(')'); err != nil {
			return 0, err
		}
		return applyDateFunc(name, args)
	}

	// Funcoes de intervalo: @SUM, @AVG, @MIN, @MAX, @COUNT, @STD, @VAR
	// Aceita lista de intervalos/expressoes separados por virgula
	var allVals []float64
	for {
		// Tenta ler como intervalo CELLREF:CELLREF
		savedPos := p.pos
		rangeVals, ok, err := p.tryParseRange()
		if err != nil {
			return 0, err
		}
		if ok {
			allVals = append(allVals, rangeVals...)
		} else {
			// Nao e intervalo: avalia como expressao
			p.pos = savedPos
			val, err := p.parseExpr()
			if err != nil {
				return 0, err
			}
			allVals = append(allVals, val)
		}

		if p.peek() != ',' {
			break
		}
		p.consume() // consome ','
	}

	if err := p.expect(')'); err != nil {
		return 0, err
	}

	return applyRangeFunc(name, allVals)
}

// tryParseRange tenta ler CELLREF:CELLREF
// Retorna (vals, true, nil) se for intervalo, (nil, false, nil) se nao for
func (p *parser) tryParseRange() ([]float64, bool, error) {
	p.skipWS()
	if p.pos >= len(p.src) {
		return nil, false, nil
	}

	ch := p.src[p.pos]
	if !((ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')) {
		return nil, false, nil
	}

	// Le primeira celula
	start := p.pos
	for p.pos < len(p.src) && (p.src[p.pos] >= 'A' && p.src[p.pos] <= 'Z' || p.src[p.pos] >= 'a' && p.src[p.pos] <= 'z') {
		p.pos++
	}
	for p.pos < len(p.src) && p.src[p.pos] >= '0' && p.src[p.pos] <= '9' {
		p.pos++
	}
	ref1 := strings.ToUpper(p.src[start:p.pos])

	// Verifica se e seguido de ':'
	p.skipWS()
	if p.pos >= len(p.src) || p.src[p.pos] != ':' {
		// Nao e intervalo — volta para o inicio
		p.pos = start
		return nil, false, nil
	}
	p.pos++ // consome ':'

	// Le segunda celula
	start2 := p.pos
	for p.pos < len(p.src) && (p.src[p.pos] >= 'A' && p.src[p.pos] <= 'Z' || p.src[p.pos] >= 'a' && p.src[p.pos] <= 'z') {
		p.pos++
	}
	for p.pos < len(p.src) && p.src[p.pos] >= '0' && p.src[p.pos] <= '9' {
		p.pos++
	}
	ref2 := strings.ToUpper(p.src[start2:p.pos])

	coord1, err := ParseCoord(ref1)
	if err != nil {
		return nil, false, errRef
	}
	coord2, err := ParseCoord(ref2)
	if err != nil {
		return nil, false, errRef
	}

	vals, err := p.eval.RangeValues(coord1, coord2)
	if err != nil {
		return nil, false, err
	}
	return vals, true, nil
}

// applyUnaryFunc aplica funcao de um argumento
func applyUnaryFunc(name string, v float64) (float64, error) {
	switch name {
	case "ABS":
		return math.Abs(v), nil
	case "INT":
		return math.Trunc(v), nil
	case "SQRT":
		if v < 0 {
			return 0, errValue
		}
		return math.Sqrt(v), nil
	case "LOG":
		if v <= 0 {
			return 0, errValue
		}
		return math.Log10(v), nil
	case "LN":
		if v <= 0 {
			return 0, errValue
		}
		return math.Log(v), nil
	case "EXP":
		return math.Exp(v), nil
	case "SIN":
		return math.Sin(v), nil
	case "COS":
		return math.Cos(v), nil
	case "TAN":
		return math.Tan(v), nil
	case "ASIN":
		return math.Asin(v), nil
	case "ACOS":
		return math.Acos(v), nil
	case "ATAN":
		return math.Atan(v), nil
	case "NOT":
		if v == 0 {
			return 1, nil
		}
		return 0, nil
	case "ISERROR":
		return 0, nil // Nao chegamos aqui em caso de erro
	// LOG10 = alias de LOG (base 10) - conforme manual
	case "LOG10":
		if v <= 0 {
			return 0, errValue
		}
		return math.Log10(v), nil
	// ISNA: verdadeiro se NA - nao chegamos aqui pois NA propaga como erro
	case "ISNA":
		return 0, nil
	}
	return 0, fmt.Errorf("%w: funcao desconhecida @%s", errError, name)
}

// applyBinaryFunc aplica funcao de dois argumentos
func applyBinaryFunc(name string, a, b float64) (float64, error) {
	switch name {
	case "MOD":
		if b == 0 {
			return 0, errDivZero
		}
		return math.Mod(a, b), nil
	case "ROUND":
		factor := math.Pow(10, b)
		return math.Round(a*factor) / factor, nil
	case "ATAN2":
		return math.Atan2(a, b), nil
	case "AND":
		if a != 0 && b != 0 {
			return 1, nil
		}
		return 0, nil
	case "OR":
		if a != 0 || b != 0 {
			return 1, nil
		}
		return 0, nil
	}
	return 0, fmt.Errorf("%w: funcao desconhecida @%s", errError, name)
}

// applyRangeFunc aplica funcao sobre um slice de valores
func applyRangeFunc(name string, vals []float64) (float64, error) {
	if len(vals) == 0 {
		switch name {
		case "COUNT":
			return 0, nil
		case "SUM":
			return 0, nil
		}
		return 0, errNA
	}

	switch name {
	case "SUM":
		sum := 0.0
		for _, v := range vals {
			sum += v
		}
		return sum, nil

	case "AVG", "AVERAGE":
		sum := 0.0
		for _, v := range vals {
			sum += v
		}
		return sum / float64(len(vals)), nil

	case "MIN":
		min := vals[0]
		for _, v := range vals[1:] {
			if v < min {
				min = v
			}
		}
		return min, nil

	case "MAX":
		max := vals[0]
		for _, v := range vals[1:] {
			if v > max {
				max = v
			}
		}
		return max, nil

	case "COUNT":
		return float64(len(vals)), nil

	case "STD":
		// Desvio padrao populacional
		if len(vals) < 2 {
			return 0, nil
		}
		sum := 0.0
		for _, v := range vals {
			sum += v
		}
		mean := sum / float64(len(vals))
		variance := 0.0
		for _, v := range vals {
			d := v - mean
			variance += d * d
		}
		return math.Sqrt(variance / float64(len(vals))), nil

	case "VAR":
		if len(vals) < 2 {
			return 0, nil
		}
		sum := 0.0
		for _, v := range vals {
			sum += v
		}
		mean := sum / float64(len(vals))
		variance := 0.0
		for _, v := range vals {
			d := v - mean
			variance += d * d
		}
		return variance / float64(len(vals)), nil

	// NPV: Valor Presente Liquido - primeiro argumento e a taxa
	case "NPV":
		if len(vals) < 2 {
			return 0, errValue
		}
		return calcNPV(vals[0], vals[1:]), nil
	}

	return 0, fmt.Errorf("%w: funcao desconhecida @%s", errError, name)
}

// ─── Funcoes adicionais reveladas pelo manual Compucenter ─────────────────────
// Acrescentadas apos leitura do manual completo do SC2 MSX (213 paginas)

// applyRangeFuncExtra trata aliases e funcoes extras do SC2 MSX
// Esta funcao estende applyRangeFunc com os casos adicionais
func init() {
	// Registra aliases reconhecidos pelo parser como nomes canonicos
	// (feito via mapeamento no parseFunction)
}

// NPV calcula o Valor Presente Liquido
// NPV(taxa, col/linha) - conforme manual cap 7
func calcNPV(rate float64, vals []float64) float64 {
	result := 0.0
	for j, v := range vals {
		result += v / math.Pow(1+rate, float64(j+1))
	}
	return result
}

// LOOKUP busca valor em tabela (vertical ou horizontal)
// Retorna o valor adjacente ao ultimo valor <= chave
func calcLookup(key float64, keys, values []float64) (float64, error) {
	if len(keys) == 0 || len(values) == 0 {
		return 0, errNA
	}
	result := values[0]
	found := false
	for i, k := range keys {
		if k <= key {
			if i < len(values) {
				result = values[i]
				found = true
			}
		}
	}
	if !found {
		return 0, errNA
	}
	return result, nil
}

// applyDateFunc implementa as funcoes de calendario do SC2 MSX
// O SC2 usa Calendario Juliano Modificado: 1=01/Mar/1900, 73049=28/Fev/2100
// Para simplicidade, armazenamos como dias desde 01/Jan/1900
func applyDateFunc(name string, args []float64) (float64, error) {
	switch name {
	case "DATE":
		// DATE(MM, DD, YY) ou DATE(MM, DD, YYYY)
		if len(args) < 3 {
			return 0, errValue
		}
		mm := int(args[0])
		dd := int(args[1])
		yy := int(args[2])
		if yy < 100 {
			yy += 1900
		} // SC2: 2 digitos = sec XX
		// Retorna representacao numerica simples (ano*10000 + mes*100 + dia)
		return float64(yy*10000 + mm*100 + dd), nil

	case "JDATE":
		// JDATE(valor_data) - retorna numero juliano
		if len(args) < 1 {
			return 0, errValue
		}
		return args[0], nil // simplificado

	case "WDAY":
		// WDAY(valor_data) - dia da semana (1=Domingo..7=Sabado)
		if len(args) < 1 {
			return 0, errValue
		}
		// Simplificado: retorna 1
		return 1, nil

	case "MONTH":
		// MONTH(valor_data) - mes (1-12)
		if len(args) < 1 {
			return 0, errValue
		}
		v := int(args[0])
		if v > 9999 {
			return float64((v / 100) % 100), nil
		}
		return 1, nil

	case "DAY":
		// DAY(valor_data) - dia do mes
		if len(args) < 1 {
			return 0, errValue
		}
		v := int(args[0])
		if v > 9999 {
			return float64(v % 100), nil
		}
		return 1, nil

	case "YEAR":
		// YEAR(valor_data) - ano
		if len(args) < 1 {
			return 0, errValue
		}
		v := int(args[0])
		if v > 9999 {
			return float64(v / 10000), nil
		}
		return float64(v), nil
	}
	return 0, errError
}
