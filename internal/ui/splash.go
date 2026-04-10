// internal/ui/splash.go
// Tela de entrada do SuperCalc 2 MSX - versao Go/tview
package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/rivo/tview"
)

func init() {
	// Garante que caracteres latinos acentuados sejam tratados como largura 1
	// em qualquer terminal, incluindo Windows Terminal com fontes ambiguous-width.
	runewidth.DefaultCondition.EastAsianWidth = false
}

// NewSplashScreen exibe a tela de apresentacao do SC2
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

		titleStyle := tcell.StyleDefault.
			Background(tcell.ColorBlack).
			Foreground(tcell.ColorAqua).
			Bold(true)

		normalStyle := tcell.StyleDefault.
			Background(tcell.ColorBlack).
			Foreground(tcell.ColorWhite)

		hiliteStyle := tcell.StyleDefault.
			Background(tcell.ColorBlack).
			Foreground(tcell.ColorYellow)

		greenStyle := tcell.StyleDefault.
			Background(tcell.ColorBlack).
			Foreground(tcell.ColorLime)

		cx := x + width/2

		// Linha horizontal no topo - ASCII puro, sem ambiguidade de largura
		splashDrawHLine(screen, x, y+1, width, '-', titleStyle)

		// Caixa do titulo usando apenas ASCII: +--+  |  |  +--+
		// Largura interna da caixa: 32 caracteres
		const boxW = 34 // largura total incluindo as duas bordas '|'
		const boxInner = 32

		splashDrawBoxLine(screen, cx, y+3, boxW, true, titleStyle) // topo
		splashDrawBoxMid(screen, cx, y+4, boxInner, "", titleStyle)
		splashDrawBoxMid(screen, cx, y+5, boxInner, "  S U P E R C A L C  2  M S X  ", titleStyle)
		splashDrawBoxMid(screen, cx, y+6, boxInner, "", titleStyle)
		splashDrawBoxMid(screen, cx, y+7, boxInner, "   Planilha de Calculo Eletron.  ", titleStyle)
		splashDrawBoxMid(screen, cx, y+8, boxInner, "         para  M S X             ", titleStyle)
		splashDrawBoxMid(screen, cx, y+9, boxInner, "", titleStyle)
		splashDrawBoxLine(screen, cx, y+10, boxW, false, titleStyle) // base

		// Versao - sem acentos para evitar problema de largura
		splashDrawCentered(screen, cx, y+12, "Versao 2.0 - Compativel com SC2 MSX", greenStyle)

		splashDrawHLine(screen, x+10, y+14, width-20, '-', normalStyle)

		splashDrawCentered(screen, cx, y+16, "Copyright (C) 1989 PRACTICA Informatica Ltda.", normalStyle)
		splashDrawCentered(screen, cx, y+17, "Implementacao Go/TUI - Compatibilidade SC2 MSX", normalStyle)

		splashDrawHLine(screen, x+10, y+19, width-20, '-', normalStyle)

		splashDrawCentered(screen, cx, y+21, "Pressione  ENTER  para continuar", hiliteStyle)
		splashDrawCentered(screen, cx, y+22, "Pressione  ?     para ajuda", normalStyle)

		splashDrawHLine(screen, x, y+height-2, width, '-', titleStyle)
		splashDrawCentered(screen, cx, y+height-1, " SC2MSX - Go Edition ", normalStyle)

		return x, y, width, height
	})

	box.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			onEnter()
			return nil
		case tcell.KeyRune:
			if event.Rune() == '?' {
				onEnter()
			}
		case tcell.KeyEscape:
			onEnter()
		}
		return event
	})

	return box
}

// splashDrawCentered centraliza texto usando runewidth para calcular largura real
func splashDrawCentered(screen tcell.Screen, cx, y int, text string, style tcell.Style) {
	w := runewidth.StringWidth(text)
	startX := cx - w/2
	splashDrawAt(screen, startX, y, text, style)
}

// splashDrawAt desenha texto caracter a caracter usando runewidth
// Avanca o X pela largura visual de cada rune (1 ou 2 celulas)
func splashDrawAt(screen tcell.Screen, x, y int, text string, style tcell.Style) {
	col := x
	for _, ch := range text {
		screen.SetContent(col, y, ch, nil, style)
		col += runewidth.RuneWidth(ch)
	}
}

// splashDrawHLine desenha linha horizontal com caractere ASCII simples
func splashDrawHLine(screen tcell.Screen, x, y, w int, ch rune, style tcell.Style) {
	for i := 0; i < w; i++ {
		screen.SetContent(x+i, y, ch, nil, style)
	}
}

// splashDrawBoxLine desenha linha de topo ou base da caixa ASCII
// +--------------------------------+
func splashDrawBoxLine(screen tcell.Screen, cx, y, boxW int, isTop bool, style tcell.Style) {
	startX := cx - boxW/2
	corner := '+'
	screen.SetContent(startX, y, corner, nil, style)
	for i := 1; i < boxW-1; i++ {
		screen.SetContent(startX+i, y, '-', nil, style)
	}
	screen.SetContent(startX+boxW-1, y, corner, nil, style)
}

// splashDrawBoxMid desenha linha do meio da caixa com conteudo centralizado
// |         texto aqui             |
func splashDrawBoxMid(screen tcell.Screen, cx, y, inner int, content string, style tcell.Style) {
	startX := cx - (inner+2)/2

	// Borda esquerda
	screen.SetContent(startX, y, '|', nil, style)

	// Conteudo centralizado dentro do inner
	contentW := runewidth.StringWidth(content)
	pad := inner - contentW
	padLeft := pad / 2
	padRight := pad - padLeft
	if padLeft < 0 {
		padLeft = 0
	}
	if padRight < 0 {
		padRight = 0
	}

	col := startX + 1
	for i := 0; i < padLeft; i++ {
		screen.SetContent(col, y, ' ', nil, style)
		col++
	}
	for _, ch := range content {
		screen.SetContent(col, y, ch, nil, style)
		col += runewidth.RuneWidth(ch)
	}
	for i := 0; i < padRight; i++ {
		screen.SetContent(col, y, ' ', nil, style)
		col++
	}

	// Borda direita
	screen.SetContent(startX+inner+1, y, '|', nil, style)
}
