// cmd/sc2msx/main.go
// SC2MSX - SuperCalc 2 MSX compativel - Implementacao Go/TUI
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/rivo/tview"

	"github.com/sc2msx/internal/spreadsheet"
	"github.com/sc2msx/internal/ui"
)

func main() {
	// Força runewidth a tratar caracteres "East Asian Ambiguous" como largura 1.
	// Sem isso, letras latinas acentuadas (á é ã ç ó) aparecem com espaço extra
	// no Windows Terminal e outros terminais que usam fontes com ambiguous width.
	runewidth.DefaultCondition.EastAsianWidth = false

	app := tview.NewApplication()

	sheet := spreadsheet.NewSpreadsheet()
	populateExample(sheet)

	grid := ui.NewGridView(sheet)
	pages := tview.NewPages()

	showSpreadsheet := func() {
		pages.SwitchToPage("spreadsheet")
		app.SetFocus(grid)
	}

	splash := ui.NewSplashScreen(app, showSpreadsheet)

	// ── Handler de help contextual ──
	grid.SetHelpHandler(func(ctx ui.HelpContext) {
		helpView := ui.NewHelpView(ctx, func() {
			pages.RemovePage("help")
			pages.SwitchToPage("spreadsheet")
			app.SetFocus(grid)
		})
		pages.AddPage("help", helpView, true, true)
		app.SetFocus(helpView)
	})

	// ── Handler de comandos SC2 ──
	grid.SetCommandHandler(func(cmd string) {
		closeModal := func(name string) {
			pages.RemovePage(name)
			pages.SwitchToPage("spreadsheet")
			app.SetFocus(grid)
		}

		modal := func(name, text string, btns []string, fn func(string)) {
			app.QueueUpdateDraw(func() {
				m := tview.NewModal().
					SetText(text).
					AddButtons(btns).
					SetDoneFunc(func(_ int, label string) {
						closeModal(name)
						fn(label)
					})
				pages.AddPage(name, m, true, true)
				app.SetFocus(m)
			})
		}

		input := func(name, label string, fn func(string)) {
			app.QueueUpdateDraw(func() {
				// Declara form antes dos closures para evitar referencia indefinida
				var form *tview.Form
				form = tview.NewForm().
					AddInputField(label, "", 20, nil, nil).
					AddButton("OK", func() {
						field := form.GetFormItemByLabel(label).(*tview.InputField)
						val := field.GetText()
						closeModal(name)
						fn(val)
					}).
					AddButton("Cancelar", func() { closeModal(name) })
				form.SetBorder(true).SetTitle(" " + name + " ").SetTitleAlign(tview.AlignLeft)
				pages.AddPage(name, center(form, 40, 7), true, true)
				app.SetFocus(form)
			})
		}

		switch cmd {
		case "QUIT":
			modal("quit", "Deseja sair do SC2MSX?", []string{"Sim", "Nao"}, func(btn string) {
				if btn == "Sim" {
					app.Stop()
				}
			})

		case "ZAP":
			modal("zap", "Apagar TODA a planilha? (Zap)", []string{"Sim", "Nao"}, func(btn string) {
				if btn == "Sim" {
					*sheet = *spreadsheet.NewSpreadsheet()
					grid.SetStatus("Planilha zerada (Zap)")
				}
			})

		case "SAVE":
			// SC2: /S pede nome do arquivo, salva no formato SDI
			defName := sheet.Filename
			if defName == "" {
				defName = "PLANILHA.SDI"
			}
			input("Save /S", fmt.Sprintf("Nome do arquivo [%s]:", defName), func(val string) {
				if val == "" {
					val = defName
				}
				// Aceita .SDI ou .CAL (gravamos sempre em SDI texto)
				upper := strings.ToUpper(val)
				if !strings.HasSuffix(upper, ".SDI") && !strings.HasSuffix(upper, ".CAL") {
					val += ".SDI"
				}
				if err := sheet.SaveSDI(val); err != nil {
					grid.SetStatus(fmt.Sprintf("ERRO ao salvar: %v", err))
				} else {
					sheet.Filename = val
					sheet.Modified = false
					grid.SetStatus(fmt.Sprintf("Salvo: %s (%d celulas)", val, len(sheet.Cells)))
				}
			})

		case "LOAD":
			// SC2: /L pede nome do arquivo, carrega no formato SDI
			input("Load /L", "Nome do arquivo (.SDI ou .CAL):", func(val string) {
				if val == "" {
					return
				}
				upper := strings.ToUpper(val)
				if !strings.HasSuffix(upper, ".SDI") && !strings.HasSuffix(upper, ".CAL") {
					val += ".SDI"
				}
				loaded, err := spreadsheet.LoadSDI(val)
				if err != nil {
					grid.SetStatus(fmt.Sprintf("ERRO ao carregar: %v", err))
					return
				}
				*sheet = *loaded
				grid.SetStatus(fmt.Sprintf("Carregado: %s (%d celulas)", val, len(sheet.Cells)))
			})

		case "WIDTH":
			cur := sheet.Cursor
			curW := sheet.GetColWidth(cur.Col)
			input("Width", fmt.Sprintf("Largura da coluna %s (%d):", spreadsheet.ColName(cur.Col), curW),
				func(val string) {
					w, err := strconv.Atoi(strings.TrimSpace(val))
					if err != nil || w < 1 || w > 72 {
						grid.SetStatus("Largura invalida (1-72)")
						return
					}
					sheet.SetColWidth(cur.Col, w)
					grid.SetStatus(fmt.Sprintf("Coluna %s: largura = %d", spreadsheet.ColName(cur.Col), w))
				})

		case "FORMAT":
			cur := sheet.Cursor
			modal("format",
				fmt.Sprintf("Formato da celula %s:\n\nG=Geral  D=Default(2dec)  I=Inteiro\nF=Fixo  S=Cientifico  $=Moeda  %%=Percent",
					cur),
				[]string{"G", "D", "I", "F", "S", "$", "%", "Cancela"},
				func(btn string) {
					cell := sheet.GetCell(cur)
					if cell.Type == spreadsheet.CellEmpty || btn == "Cancela" {
						return
					}
					switch btn {
					case "G":
						cell.Format.Type = spreadsheet.FormatGeneral
					case "D":
						cell.Format.Type = spreadsheet.FormatDefault
					case "I":
						cell.Format.Type = spreadsheet.FormatInteger
					case "F":
						cell.Format.Type = spreadsheet.FormatFixed
					case "S":
						cell.Format.Type = spreadsheet.FormatScientific
					case "$":
						cell.Format.Type = spreadsheet.FormatDollar
					case "%":
						cell.Format.Type = spreadsheet.FormatPercent
					}
					sheet.SetCell(cur, cell)
					sheet.Recalc()
					grid.SetStatus(fmt.Sprintf("Formato '%s' aplicado em %s", btn, cur))
				})

		case "INSERT":
			modal("insert", fmt.Sprintf("Inserir em %s:", sheet.Cursor),
				[]string{"Linha", "Coluna", "Cancela"},
				func(btn string) {
					switch btn {
					case "Linha":
						sheet.InsertRow(sheet.Cursor.Row)
						sheet.Recalc()
						grid.SetStatus(fmt.Sprintf("Linha %d inserida", sheet.Cursor.Row))
					case "Coluna":
						sheet.InsertCol(sheet.Cursor.Col)
						sheet.Recalc()
						grid.SetStatus(fmt.Sprintf("Coluna %s inserida", spreadsheet.ColName(sheet.Cursor.Col)))
					}
				})

		case "DELETE":
			modal("delete", fmt.Sprintf("Apagar em %s:", sheet.Cursor),
				[]string{"Linha", "Coluna", "Cancela"},
				func(btn string) {
					switch btn {
					case "Linha":
						r := sheet.Cursor.Row
						sheet.DeleteRow(r)
						sheet.Recalc()
						grid.SetStatus(fmt.Sprintf("Linha %d removida", r))
					case "Coluna":
						c := sheet.Cursor.Col
						sheet.DeleteCol(c)
						sheet.Recalc()
						grid.SetStatus(fmt.Sprintf("Coluna %s removida", spreadsheet.ColName(c)))
					}
				})

		case "BLANK":
			// /B Blank: apaga celula atual ou intervalo
			// Enter sem digitar = apaga celula corrente
			// Range valido (ex: A1:G5) = apaga todas as celulas do range
			cur := sheet.Cursor
			input("Blank /B", fmt.Sprintf("Range a apagar [Enter=%s]:", cur), func(val string) {
				val = strings.TrimSpace(val)
				if val == "" {
					// Enter sem range: apaga somente a celula corrente
					sheet.SetCell(cur, nil)
					sheet.Recalc()
					grid.SetStatus(fmt.Sprintf("Celula %s apagada", cur))
					return
				}
				// Tenta interpretar como range (A1:G5) ou celula simples (B3)
				from, to, err := parseRangeStr(val)
				if err != nil {
					grid.SetStatus(fmt.Sprintf("Range invalido: %s", val))
					return
				}
				sheet.ClearRange(from, to)
				sheet.Recalc()
				if from == to {
					grid.SetStatus(fmt.Sprintf("Celula %s apagada", from))
				} else {
					grid.SetStatus(fmt.Sprintf("Range %s:%s apagado", from, to))
				}
			})

		case "ARRANGE":
			// /A Arrange: ordena linhas ou colunas por uma coluna/linha chave
			cur := sheet.Cursor
			modal("arrange",
				fmt.Sprintf("/A Arrange - Celula: %s\n\nOrdenar por linha ou coluna?", cur),
				[]string{"Linhas", "Colunas", "Cancela"},
				func(btn string) {
					switch btn {
					case "Linhas":
						// Ordena linhas usando a coluna atual como chave
						input("Arrange Linhas", fmt.Sprintf("Intervalo de linhas (ex: 2:20) [cursor=linha %d]:", cur.Row),
							func(rangeStr string) {
								fromRow, toRow := cur.Row, cur.Row
								if rangeStr != "" {
									var f, t int
									if _, err := fmt.Sscanf(rangeStr, "%d:%d", &f, &t); err == nil {
										fromRow, toRow = f, t
									}
								}
								modal("arrange-dir", "Direcao?", []string{"Ascendente", "Descendente"}, func(dir string) {
									d := spreadsheet.ArrangeAscending
									if dir == "Descendente" {
										d = spreadsheet.ArrangeDescending
									}
									sheet.ArrangeRows(fromRow, toRow, cur.Col, d, true)
									sheet.Recalc()
									grid.SetStatus(fmt.Sprintf("Linhas %d:%d ordenadas por col %s", fromRow, toRow, spreadsheet.ColName(cur.Col)))
								})
							})
					case "Colunas":
						input("Arrange Colunas", fmt.Sprintf("Intervalo de colunas (ex: B:G) [cursor=%s]:", spreadsheet.ColName(cur.Col)),
							func(rangeStr string) {
								fromCol, toCol := cur.Col, cur.Col
								if rangeStr != "" {
									parts := strings.SplitN(strings.ToUpper(rangeStr), ":", 2)
									if len(parts) == 2 {
										c1, e1 := spreadsheet.ParseCoord(parts[0] + "1")
										c2, e2 := spreadsheet.ParseCoord(parts[1] + "1")
										if e1 == nil && e2 == nil {
											fromCol, toCol = c1.Col, c2.Col
										}
									}
								}
								modal("arrange-dir2", "Direcao?", []string{"Ascendente", "Descendente"}, func(dir string) {
									d := spreadsheet.ArrangeAscending
									if dir == "Descendente" {
										d = spreadsheet.ArrangeDescending
									}
									sheet.ArrangeCols(fromCol, toCol, cur.Row, d, true)
									sheet.Recalc()
									grid.SetStatus(fmt.Sprintf("Colunas ordenadas por linha %d", cur.Row))
								})
							})
					}
				})

		case "EDIT":
			grid.EnterEditMode()

		case "GLOBAL":
			modal("global",
				fmt.Sprintf("/G Global\n\nCelula atual: %s", sheet.Cursor),
				[]string{"Recalc", "Decimais", "Cancela"},
				func(btn string) {
					switch btn {
					case "Recalc":
						sheet.Recalc()
						grid.SetStatus("Recalculo manual concluido")
					case "Decimais":
						input("Global Decimais", "Casas decimais padrao (0-9):", func(val string) {
							d, err := strconv.Atoi(strings.TrimSpace(val))
							if err != nil || d < 0 || d > 9 {
								grid.SetStatus("Valor invalido (0-9)")
								return
							}
							sheet.DefaultFormat.Decimals = d
							sheet.Recalc()
							grid.SetStatus(fmt.Sprintf("Decimais padrao: %d", d))
						})
					}
				})

		case "TITLE":
			cur := sheet.Cursor
			modal("title",
				fmt.Sprintf("/T Title - %s\n\nTrava linha/coluna de titulo (freeze)", cur),
				[]string{"Horizontal", "Vertical", "Ambos", "Limpar", "Cancela"},
				func(btn string) {
					switch btn {
					case "Horizontal":
						sheet.TitleRow = cur.Row
						grid.SetStatus(fmt.Sprintf("Titulo horizontal: linha %d", cur.Row))
					case "Vertical":
						sheet.TitleCol = cur.Col
						grid.SetStatus(fmt.Sprintf("Titulo vertical: coluna %s", spreadsheet.ColName(cur.Col)))
					case "Ambos":
						sheet.TitleRow = cur.Row
						sheet.TitleCol = cur.Col
						grid.SetStatus(fmt.Sprintf("Titulos: linha %d, col %s", cur.Row, spreadsheet.ColName(cur.Col)))
					case "Limpar":
						sheet.TitleRow = 0
						sheet.TitleCol = 0
						grid.SetStatus("Titulos removidos")
					}
				})

		case "PRINT":
			input("Print /O", "Arquivo de saida (ex: RELAT.TXT):", func(val string) {
				if val == "" {
					return
				}
				if err := exportText(sheet, val); err != nil {
					grid.SetStatus(fmt.Sprintf("Erro ao exportar: %v", err))
				} else {
					grid.SetStatus(fmt.Sprintf("Exportado: %s", val))
				}
			})

		case "COPY":
			cur := sheet.Cursor
			input("Copy /C - Origem", fmt.Sprintf("Intervalo origem [%s]:", cur), func(src string) {
				if src == "" {
					src = cur.String()
				}
				input("Copy /C - Destino", "Celula destino (canto sup esq):", func(dst string) {
					if dst == "" {
						return
					}
					if err := copyRange(sheet, src, dst); err != nil {
						grid.SetStatus(fmt.Sprintf("Copy: %v", err))
					} else {
						sheet.Recalc()
						grid.SetStatus(fmt.Sprintf("Copiado %s -> %s", src, dst))
					}
				})
			})

		case "MOVE":
			cur := sheet.Cursor
			input("Move /M - Origem", fmt.Sprintf("Intervalo origem [%s]:", cur), func(src string) {
				if src == "" {
					src = cur.String()
				}
				input("Move /M - Destino", "Celula destino:", func(dst string) {
					if dst == "" {
						return
					}
					if err := moveRange(sheet, src, dst); err != nil {
						grid.SetStatus(fmt.Sprintf("Move: %v", err))
					} else {
						sheet.Recalc()
						grid.SetStatus(fmt.Sprintf("Movido %s -> %s", src, dst))
					}
				})
			})

		case "REPLICATE":
			cur := sheet.Cursor
			input("Replicate /R - Origem", fmt.Sprintf("Origem [%s]:", cur), func(src string) {
				if src == "" {
					src = cur.String()
				}
				input("Replicate /R - Destino", "Destino (ex: B5:B20):", func(dst string) {
					if dst == "" {
						return
					}
					if err := replicateRange(sheet, src, dst); err != nil {
						grid.SetStatus(fmt.Sprintf("Replicate: %v", err))
					} else {
						sheet.Recalc()
						grid.SetStatus(fmt.Sprintf("Replicado %s -> %s", src, dst))
					}
				})
			})

		case "XFER":
			grid.SetStatus("/Xfer: use /S para salvar e /L para carregar")
		}
	})

	pages.AddPage("splash", splash, true, true)
	pages.AddPage("spreadsheet", grid, true, false)

	app.SetRoot(pages, true).SetFocus(splash).EnableMouse(false)
	if err := app.Run(); err != nil {
		panic(err)
	}
}

// center envolve um primitive em uma flex box centralizada
func center(p tview.Primitive, w, h int) tview.Primitive {
	return tview.NewFlex().
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(tview.NewBox(), 0, 1, false).
			AddItem(p, h, 1, true).
			AddItem(tview.NewBox(), 0, 1, false), w, 1, true).
		AddItem(tview.NewBox(), 0, 1, false)
}

// ── Dados de exemplo (planilha EXEMPLO.CAL do manual SC2 MSX) ────────────────

func setL(s *spreadsheet.Spreadsheet, col, row int, text string) {
	s.SetCell(spreadsheet.Coord{Row: row, Col: col},
		&spreadsheet.Cell{Type: spreadsheet.CellText, TextValue: text, RawInput: text})
}

func setV(s *spreadsheet.Spreadsheet, col, row int, val float64) {
	raw := fmt.Sprintf("%.2f", val)
	s.SetCell(spreadsheet.Coord{Row: row, Col: col},
		&spreadsheet.Cell{Type: spreadsheet.CellNumber, NumericValue: val, RawInput: raw})
}

func setF(s *spreadsheet.Spreadsheet, col, row int, formula string) {
	s.SetCell(spreadsheet.Coord{Row: row, Col: col},
		&spreadsheet.Cell{Type: spreadsheet.CellFormula, Formula: formula, RawInput: formula})
}

func populateExample(s *spreadsheet.Spreadsheet) {
	// Cabecalhos
	setL(s, 1, 1, "")
	setL(s, 2, 1, "JAN")
	setL(s, 3, 1, "FEV")
	setL(s, 4, 1, "MAR")
	setL(s, 5, 1, "ABR")
	setL(s, 6, 1, "MAI")
	setL(s, 7, 1, "JUN")
	setL(s, 8, 1, "TOTAL")

	// Linha 2: VENDA BRUTA
	setL(s, 1, 2, "VENDA BRUTA")
	setV(s, 2, 2, 1500)
	setV(s, 3, 2, 1300)
	setV(s, 4, 2, 1800)
	setV(s, 5, 2, 3500)
	setV(s, 6, 2, 3200)
	setV(s, 7, 2, 11200)
	setF(s, 8, 2, "@SUM(B2:G2)")

	// Linha 3: CUSTO1
	setL(s, 1, 3, "CUSTO1")
	setV(s, 2, 3, 756)
	setV(s, 3, 3, 650)
	setV(s, 4, 3, 900)
	setV(s, 5, 3, 1750)
	setV(s, 6, 3, 2600)
	setV(s, 7, 3, 3600)
	setF(s, 8, 3, "@SUM(B3:G3)")

	// Linha 4: CUSTO2
	setL(s, 1, 4, "CUSTO2")
	setV(s, 2, 4, 255)
	setV(s, 3, 4, 221)
	setV(s, 4, 4, 306)
	setV(s, 5, 4, 595)
	setV(s, 6, 4, 884)
	setV(s, 7, 4, 1704)
	setF(s, 8, 4, "@SUM(B4:G4)")

	// Linha 5: VENDA LIQUIDA = BRUTA - CUSTO1 - CUSTO2
	setL(s, 1, 5, "VENDA LIQ.")
	setF(s, 2, 5, "+B2-B3-B4")
	setF(s, 3, 5, "+C2-C3-C4")
	setF(s, 4, 5, "+D2-D3-D4")
	setF(s, 5, 5, "+E2-E3-E4")
	setF(s, 6, 5, "+F2-F3-F4")
	setF(s, 7, 5, "+G2-G3-G4")
	setF(s, 8, 5, "@SUM(B5:G5)")

	// Linha 6: separador visual
	setL(s, 1, 6, strings.Repeat("-", 11))
	for col := 2; col <= 8; col++ {
		setL(s, col, 6, strings.Repeat("-", s.GetColWidth(col)-1))
	}

	// Linha 7-9: estatisticas
	setL(s, 1, 7, "MEDIA/MES")
	setF(s, 2, 7, "@AVG(B2:G2)")
	setL(s, 1, 8, "MAX MES")
	setF(s, 2, 8, "@MAX(B2:G2)")
	setL(s, 1, 9, "MIN MES")
	setF(s, 2, 9, "@MIN(B2:G2)")
	setL(s, 1, 10, "DESVIO")
	setF(s, 2, 10, "@STD(B2:G2)")

	// Larguras de coluna (SC2 padrao: descricao=12, dados=9)
	s.ColWidths[1] = 12
	for col := 2; col <= 8; col++ {
		s.ColWidths[col] = 9
	}

	// Recalcula todas as formulas
	s.Recalc()
}

// ─── Operacoes de intervalo ───────────────────────────────────────────────────

// parseRangeStr interpreta string de intervalo "A1:G5" ou celula simples "A1"
func parseRangeStr(s string) (from, to spreadsheet.Coord, err error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if idx := strings.Index(s, ":"); idx >= 0 {
		from, err = spreadsheet.ParseCoord(s[:idx])
		if err != nil {
			return
		}
		to, err = spreadsheet.ParseCoord(s[idx+1:])
	} else {
		from, err = spreadsheet.ParseCoord(s)
		to = from
	}
	return
}

// copyRange copia celulas do intervalo src para destino a partir de dstCell
func copyRange(s *spreadsheet.Spreadsheet, src, dst string) error {
	from, to, err := parseRangeStr(src)
	if err != nil {
		return fmt.Errorf("origem invalida: %w", err)
	}
	dstCoord, err := spreadsheet.ParseCoord(strings.ToUpper(strings.TrimSpace(dst)))
	if err != nil {
		return fmt.Errorf("destino invalido: %w", err)
	}

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
			srcCoord := spreadsheet.Coord{Row: r, Col: c}
			cell := s.GetCell(srcCoord)
			if cell.Type == spreadsheet.CellEmpty {
				continue
			}

			dR := dstCoord.Row + (r - minR)
			dC := dstCoord.Col + (c - minC)
			if dR > 254 || dC > 63 {
				continue
			}

			clone := *cell
			s.SetCell(spreadsheet.Coord{Row: dR, Col: dC}, &clone)
		}
	}
	return nil
}

// moveRange move celulas do intervalo src para destino (apaga a origem)
func moveRange(s *spreadsheet.Spreadsheet, src, dst string) error {
	if err := copyRange(s, src, dst); err != nil {
		return err
	}
	from, to, err := parseRangeStr(src)
	if err != nil {
		return err
	}
	s.ClearRange(from, to)
	return nil
}

// replicateRange replica celulas ajustando referencias relativas de formulas
// Ex: +B2-B3 replicado de B5 para C5 vira +C2-C3
func replicateRange(s *spreadsheet.Spreadsheet, src, dst string) error {
	srcFrom, srcTo, err := parseRangeStr(src)
	if err != nil {
		return fmt.Errorf("origem invalida: %w", err)
	}

	dstFrom, dstTo, err := parseRangeStr(dst)
	if err != nil {
		return fmt.Errorf("destino invalido: %w", err)
	}

	// Calcula deslocamento
	dRow := dstFrom.Row - srcFrom.Row
	dCol := dstFrom.Col - srcFrom.Col

	// Se destino e um intervalo maior, replica em toda a area
	minDR, maxDR := dstFrom.Row, dstTo.Row
	minDC, maxDC := dstFrom.Col, dstTo.Col
	if minDR > maxDR {
		minDR, maxDR = maxDR, minDR
	}
	if minDC > maxDC {
		minDC, maxDC = maxDC, minDC
	}

	srcMinR, srcMaxR := srcFrom.Row, srcTo.Row
	srcMinC, srcMaxC := srcFrom.Col, srcTo.Col
	if srcMinR > srcMaxR {
		srcMinR, srcMaxR = srcMaxR, srcMinR
	}
	if srcMinC > srcMaxC {
		srcMinC, srcMaxC = srcMaxC, srcMinC
	}

	srcH := srcMaxR - srcMinR + 1
	srcW := srcMaxC - srcMinC + 1

	for r := minDR; r <= maxDR; r++ {
		for c := minDC; c <= maxDC; c++ {
			// Qual celula da origem corresponde?
			srcR := srcMinR + ((r - minDR) % srcH)
			srcC := srcMinC + ((c - minDC) % srcW)
			cell := s.GetCell(spreadsheet.Coord{Row: srcR, Col: srcC})
			if cell.Type == spreadsheet.CellEmpty {
				continue
			}

			clone := *cell
			// Ajusta formula se houver
			if clone.Type == spreadsheet.CellFormula {
				clone.Formula = adjustFormulaRefs(clone.Formula, dRow+(r-minDR), dCol+(c-minDC))
			}
			s.SetCell(spreadsheet.Coord{Row: r, Col: c}, &clone)
		}
	}
	return nil
}

// adjustFormulaRefs ajusta referencias de celulas em uma formula
// Ex: "+B2-B3" com dRow=1 dCol=1 vira "+C3-C4"
// Implementacao simples: busca padroes de coordenadas e ajusta
func adjustFormulaRefs(formula string, dRow, dCol int) string {
	if dRow == 0 && dCol == 0 {
		return formula
	}

	result := strings.Builder{}
	i := 0
	f := strings.ToUpper(formula)

	for i < len(f) {
		ch := f[i]

		// Detecta inicio de referencia de celula (letra seguida de digito)
		if ch >= 'A' && ch <= 'Z' {
			// Le letras
			j := i
			for j < len(f) && f[j] >= 'A' && f[j] <= 'Z' {
				j++
			}
			// Le digitos
			k := j
			for k < len(f) && f[k] >= '0' && f[k] <= '9' {
				k++
			}

			if k > j && j > i && j-i <= 2 {
				// E uma referencia de celula
				ref := f[i:k]
				coord, err := spreadsheet.ParseCoord(ref)
				if err == nil {
					newR := coord.Row + dRow
					newC := coord.Col + dCol
					if newR >= 1 && newR <= 254 && newC >= 1 && newC <= 63 {
						result.WriteString(spreadsheet.Coord{Row: newR, Col: newC}.String())
					} else {
						result.WriteString(ref) // fora dos limites: mantem original
					}
					i = k
					continue
				}
			}
			// Nao e referencia (ex: nome de funcao): copia literalmente
			result.WriteByte(formula[i])
			i++
			continue
		}

		result.WriteByte(formula[i])
		i++
	}
	return result.String()
}

// exportText exporta a planilha como texto simples (para /O Output)
func exportText(s *spreadsheet.Spreadsheet, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Descobre dimensoes usadas
	maxRow, maxCol := 0, 0
	for coord := range s.Cells {
		if coord.Row > maxRow {
			maxRow = coord.Row
		}
		if coord.Col > maxCol {
			maxCol = coord.Col
		}
	}

	for row := 1; row <= maxRow; row++ {
		for col := 1; col <= maxCol; col++ {
			cell := s.GetCell(spreadsheet.Coord{Row: row, Col: col})
			w := s.GetColWidth(col)
			val := s.FormatCellValue(cell, w)
			fmt.Fprint(f, val)
			if col < maxCol {
				fmt.Fprint(f, " ")
			}
		}
		fmt.Fprintln(f)
	}
	return nil
}
