package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"go-chat/internal/client"
	"log"
)

func main() {
	p := tea.NewProgram(client.NewModel())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
