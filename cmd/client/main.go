package main

import (
	"go-chat/internal/client"
	tea "github.com/charmbracelet/bubbletea"
	"log"
)

func main() {
	p := tea.NewProgram(client.NewModel())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
