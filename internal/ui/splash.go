// internal/ui/splash.go
// Tela de entrada do SuperCalc 2 MSX - versão Go/tview
package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// SplashScreen exibe a tela de apresentação do SC2
// Visual fiel ao original MSX: fundo escuro, texto simples, sem bordas fancy
func NewSplashScreen(app *tview.Application, onEnter func()) *tview.Box {
	box := tview.NewBox()

	box.SetDrawFunc(func(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
		// Fundo preto (simula tela MSX)
		bgStyle := tcell.StyleDefault.
			Background(tcell.ColorBlack).
			Foreground(tcell.ColorWhite)

		// Preenche o fundo
		for row := y; row < y+height; row++ {
			for col := x; col < x+width; col++ {
				screen.SetContent(col, row, ' ', nil, bgStyle)
			}
		}

		// Estilo de destaque (ciano brilhante - cores MSX típicas)
		titleStyle := tcell.StyleDefault.
			Background(tcell.ColorBlack).
			Foreground(tcell.ColorAqua).
			Bold(true)

		// Estilo normal branco
		normalStyle := tcell.StyleDefault.
			Background(tcell.ColorBlack).
			Foreground(tcell.ColorWhite)

		// Estilo amarelo para destaques
		hiliteStyle := tcell.StyleDefault.
			Background(tcell.ColorBlack).
			Foreground(tcell.ColorYellow)

		// Estilo verde para versão
		greenStyle := tcell.StyleDefault.
			Background(tcell.ColorBlack).
			Foreground(tcell.ColorLime)

		// Centro horizontal da tela de 80 colunas
		cx := x + width/2

		// Linha de duplo traço no topo (simula borda MSX)
		drawHLine(screen, x, y+1, width, '═', titleStyle)

		// ── Título principal ──
		// SC2 original mostra "SuperCalc 2" centralizado
		drawCentered(screen, cx, y+3, "╔══════════════════════════════╗", titleStyle)
		drawCentered(screen, cx, y+4, "║                              ║", titleStyle)
		drawCentered(screen, cx, y+5, "║       S U P E R C A L C      ║", titleStyle)
		drawCentered(screen, cx, y+6, "║                              ║", titleStyle)
		drawCentered(screen, cx, y+7, "║    Planilha de Cálculo  v2   ║", titleStyle)
		drawCentered(screen, cx, y+8, "║         para MSX             ║", titleStyle)
		drawCentered(screen, cx, y+9, "║                              ║", titleStyle)
		drawCentered(screen, cx, y+10, "╚══════════════════════════════╝", titleStyle)

		// Versão
		drawCentered(screen, cx, y+12, "Versão 2.0 - Compatível com SC2 MSX", greenStyle)

		// Separador
		drawHLine(screen, x+10, y+14, width-20, '─', normalStyle)

		// Informações
		drawCentered(screen, cx, y+16, "Copyright (C) 1989 PRACTICA Informática Ltda.", normalStyle)
		drawCentered(screen, cx, y+17, "Implementação Go/TUI - Compatibilidade total SC2 MSX", normalStyle)

		// Separador
		drawHLine(screen, x+10, y+19, width-20, '─', normalStyle)

		// Instrução para continuar
		drawCentered(screen, cx, y+21, "Pressione  ENTER  para continuar", hiliteStyle)
		drawCentered(screen, cx, y+22, "Pressione  ?     para ajuda", normalStyle)

		// Linha de rodapé
		drawHLine(screen, x, y+height-2, width, '═', titleStyle)
		drawCentered(screen, cx, y+height-1, " SC2MSX - Go Edition ", normalStyle)

		return x, y, width, height
	})

	// Captura teclas na splash screen
	box.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			onEnter()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case '?':
				onEnter() // Por hora, também entra (help será implementado depois)
			}
		case tcell.KeyEscape:
			onEnter()
		}
		return event
	})

	return box
}

// drawCentered desenha texto centralizado em x, linha y
func drawCentered(screen tcell.Screen, cx, y int, text string, style tcell.Style) {
	startX := cx - len([]rune(text))/2
	drawAt(screen, startX, y, text, style)
}

// drawAt desenha texto na posição x, y
func drawAt(screen tcell.Screen, x, y int, text string, style tcell.Style) {
	for i, ch := range text {
		screen.SetContent(x+i, y, ch, nil, style)
	}
}

// drawHLine desenha linha horizontal
func drawHLine(screen tcell.Screen, x, y, width int, ch rune, style tcell.Style) {
	for i := 0; i < width; i++ {
		screen.SetContent(x+i, y, ch, nil, style)
	}
}
