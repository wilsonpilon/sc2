#!/bin/bash
# setup.sh - Configura dependências e compila o SC2MSX
set -e

echo "=== SC2MSX Setup ==="
echo ""

# Verifica Go
if ! command -v go &> /dev/null; then
    echo "ERRO: Go não encontrado. Instale Go 1.21+ de https://go.dev/dl/"
    exit 1
fi

echo "Go: $(go version)"
echo ""

# Recria go.mod limpo
cat > go.mod << 'GOMOD'
module github.com/sc2msx

go 1.21
GOMOD

rm -f go.sum

echo "Buscando versão mais recente do tview..."
go get github.com/rivo/tview@latest
go get github.com/gdamore/tcell/v2@latest
go get github.com/mattn/go-sqlite3@latest

echo ""
echo "Ajustando dependências..."
go mod tidy

echo ""
echo "Compilando..."
go build -o sc2msx ./cmd/sc2msx/

echo ""
echo "=== Pronto! Execute: ./sc2msx ==="
