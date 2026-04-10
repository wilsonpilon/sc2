// cmd/sc2msx/main.go
// SC2MSX - SuperCalc 2 MSX compativel - Implementacao Go/TUI
package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rivo/tview"

	"github.com/sc2msx/internal/spreadsheet"
	"github.com/sc2msx/internal/ui"
)

func main() {
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

	// ── Handler de comandos SC2 ──
	grid.SetCommandHandler(func(cmd string) {
		closeModal := func(name string) {
			pages.RemovePage(name)
			pages.SwitchToPage("spreadsheet")
			app.SetFocus(grid)
		}

		modal := func(name, text string, btns []string, fn func(string)) {
			m := tview.NewModal().
				SetText(text).
				AddButtons(btns).
				SetDoneFunc(func(_ int, label string) {
					closeModal(name)
					fn(label)
				})
			pages.AddPage(name, m, true, true)
			app.SetFocus(m)
		}

		input := func(name, label string, fn func(string)) {
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

		case "GLOBAL":
			modal("global",
				"Configuracao Global:\n\nD=Decimais padrao  W=Largura padrao\nR=Recalc manual/auto  C=Virgulas",
				[]string{"Recalc", "Cancela"},
				func(btn string) {
					if btn == "Recalc" {
						sheet.Recalc()
						grid.SetStatus("Recalculo manual concluido")
					}
				})

		case "TITLE":
			grid.SetStatus("/Title: em desenvolvimento (fixa linha/coluna de titulo)")

		case "PRINT":
			grid.SetStatus("/Print: em desenvolvimento")

		case "COPY":
			grid.SetStatus("/Copy: em desenvolvimento")

		case "MOVE":
			grid.SetStatus("/Move: em desenvolvimento")

		case "REPLICATE":
			grid.SetStatus("/Replicate: em desenvolvimento")

		case "XFER":
			grid.SetStatus("/Xfer (SDI): em desenvolvimento")
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
