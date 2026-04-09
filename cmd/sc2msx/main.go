// cmd/sc2msx/main.go
// SC2MSX - SuperCalc 2 MSX compativel - Implementacao Go/TUI
package main

import (
	"fmt"

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

	grid.SetCommandHandler(func(cmd string) {
		switch cmd {
		case "QUIT":
			modal := tview.NewModal().
				SetText("Deseja sair do SC2MSX?").
				AddButtons([]string{"Sim", "Nao"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					if buttonLabel == "Sim" {
						app.Stop()
					} else {
						pages.RemovePage("quit")
						pages.SwitchToPage("spreadsheet")
						app.SetFocus(grid)
					}
				})
			pages.AddPage("quit", modal, true, true)
			app.SetFocus(modal)

		case "ZAP":
			modal := tview.NewModal().
				SetText("Apagar TODA a planilha?").
				AddButtons([]string{"Sim", "Nao"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					pages.RemovePage("zap")
					if buttonLabel == "Sim" {
						newSheet := spreadsheet.NewSpreadsheet()
						*sheet = *newSheet
						grid.SetStatus("Planilha zerada")
					}
					pages.SwitchToPage("spreadsheet")
					app.SetFocus(grid)
				})
			pages.AddPage("zap", modal, true, true)
			app.SetFocus(modal)

		case "SAVE":
			grid.SetStatus("SAVE: ainda nao implementado (proximo passo)")

		case "LOAD":
			grid.SetStatus("LOAD: ainda nao implementado (proximo passo)")
		}
	})

	pages.AddPage("splash", splash, true, true)
	pages.AddPage("spreadsheet", grid, true, false)

	app.SetRoot(pages, true).
		SetFocus(splash).
		EnableMouse(false)

	if err := app.Run(); err != nil {
		panic(err)
	}
}

// setLabel insere celula de texto
func setLabel(sheet *spreadsheet.Spreadsheet, col, row int, text string) {
	sheet.SetCell(spreadsheet.Coord{Row: row, Col: col}, &spreadsheet.Cell{
		Type:      spreadsheet.CellText,
		TextValue: text,
		RawInput:  text,
	})
}

// setNum insere celula numerica
func setNum(sheet *spreadsheet.Spreadsheet, col, row int, val float64) {
	sheet.SetCell(spreadsheet.Coord{Row: row, Col: col}, &spreadsheet.Cell{
		Type:         spreadsheet.CellNumber,
		NumericValue: val,
		RawInput:     fmt.Sprintf("%.2f", val),
	})
}

// setFormula insere celula de formula
func setFormula(sheet *spreadsheet.Spreadsheet, col, row int, formula string) {
	sheet.SetCell(spreadsheet.Coord{Row: row, Col: col}, &spreadsheet.Cell{
		Type:    spreadsheet.CellFormula,
		Formula: formula,
		RawInput: formula,
	})
}

// populateExample popula com a planilha EXEMPLO.CAL do manual do SC2 MSX
// Inclui formulas @SUM para demonstrar o avaliador
func populateExample(sheet *spreadsheet.Spreadsheet) {
	// ── Cabecalhos de mes (linha 1) ──
	setLabel(sheet, 1, 1, "")
	setLabel(sheet, 2, 1, "JAN")
	setLabel(sheet, 3, 1, "FEV")
	setLabel(sheet, 4, 1, "MAR")
	setLabel(sheet, 5, 1, "ABR")
	setLabel(sheet, 6, 1, "MAI")
	setLabel(sheet, 7, 1, "JUN")
	setLabel(sheet, 8, 1, "TOTAL")

	// ── Linha 2: VENDA BRUTA ──
	setLabel(sheet, 1, 2, "VENDA BRUTA")
	setNum(sheet, 2, 2, 1500.00)
	setNum(sheet, 3, 2, 1300.00)
	setNum(sheet, 4, 2, 1800.00)
	setNum(sheet, 5, 2, 3500.00)
	setNum(sheet, 6, 2, 3200.00)
	setNum(sheet, 7, 2, 11200.00)
	setFormula(sheet, 8, 2, "@SUM(B2:G2)") // Total = soma dos meses

	// ── Linha 3: CUSTO1 ──
	setLabel(sheet, 1, 3, "CUSTO1")
	setNum(sheet, 2, 3, 756.00)
	setNum(sheet, 3, 3, 650.00)
	setNum(sheet, 4, 3, 900.00)
	setNum(sheet, 5, 3, 1750.00)
	setNum(sheet, 6, 3, 2600.00)
	setNum(sheet, 7, 3, 3600.00)
	setFormula(sheet, 8, 3, "@SUM(B3:G3)")

	// ── Linha 4: CUSTO2 ──
	setLabel(sheet, 1, 4, "CUSTO2")
	setNum(sheet, 2, 4, 255.00)
	setNum(sheet, 3, 4, 221.00)
	setNum(sheet, 4, 4, 306.00)
	setNum(sheet, 5, 4, 595.00)
	setNum(sheet, 6, 4, 884.00)
	setNum(sheet, 7, 4, 1704.00)
	setFormula(sheet, 8, 4, "@SUM(B4:G4)")

	// ── Linha 5: VENDA LIQUIDA = BRUTA - CUSTO1 - CUSTO2 ──
	setLabel(sheet, 1, 5, "VENDA LIQ.")
	setFormula(sheet, 2, 5, "+B2-B3-B4")
	setFormula(sheet, 3, 5, "+C2-C3-C4")
	setFormula(sheet, 4, 5, "+D2-D3-D4")
	setFormula(sheet, 5, 5, "+E2-E3-E4")
	setFormula(sheet, 6, 5, "+F2-F3-F4")
	setFormula(sheet, 7, 5, "+G2-G3-G4")
	setFormula(sheet, 8, 5, "@SUM(B5:G5)")

	// ── Linha 7: Estatisticas ──
	setLabel(sheet, 1, 7, "MEDIA/MES")
	setFormula(sheet, 2, 7, "@AVG(B2:G2)") // Media mensal da venda bruta
	setLabel(sheet, 1, 8, "MAX MES")
	setFormula(sheet, 2, 8, "@MAX(B2:G2)")
	setLabel(sheet, 1, 9, "MIN MES")
	setFormula(sheet, 2, 9, "@MIN(B2:G2)")

	// ── Largura das colunas ──
	sheet.ColWidths[1] = 12 // Descricao
	for col := 2; col <= 8; col++ {
		sheet.ColWidths[col] = 9
	}

	// ── Recalcula todas as formulas ──
	sheet.Recalc()
}
