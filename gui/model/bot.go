package model

import (
	"fmt"
	"log"
	"strings"
)

// Bot represents a kelp bot instance
type Bot struct {
	Name     string `json:"name"`
	Strategy string `json:"strategy"`
	Running  bool   `json:"running"`
	Test     bool   `json:"test"`
	Warnings uint16 `json:"warnings"`
	Errors   uint16 `json:"errors"`
}

// MakeAutogeneratedBot factory method
func MakeAutogeneratedBot() *Bot {
	return &Bot{
		Name:     "George the Auspicious Octopus",
		Strategy: "buysell",
		Running:  false,
		Test:     true,
		Warnings: 0,
		Errors:   0,
	}
}

// FromFilenames creates a Bot representing the bot that uses the provided filenames
func FromFilenames(traderFilename string, strategyFilename string) *Bot {
	log.Printf("creating Bot struct from filenames: %s, %s\n", traderFilename, strategyFilename)

	botNameSnake := strings.TrimSuffix(traderFilename, "__trader.cfg")
	strategy := strings.TrimSuffix(strings.TrimPrefix(strategyFilename, botNameSnake+"__strategy_"), ".cfg")
	botName := strings.Replace(botNameSnake, "_", " ", -1)
	return &Bot{
		Name:     strings.Title(botName),
		Strategy: strategy,
		// Running:  false,
		// Test:     true,
		// Warnings: 0,
		// Errors:   0,
	}
}

// FilenamePair represents the two config filenames associated with a bot
type FilenamePair struct {
	Trader   string
	Strategy string
}

// Filenames where we should save bot config file
func (b *Bot) Filenames() *FilenamePair {
	return GetBotFilenames(b.Name, b.Strategy)
}

// GetBotFilenames from botName
func GetBotFilenames(botName string, strategy string) *FilenamePair {
	converted := strings.ToLower(strings.Replace(botName, " ", "_", -1))
	return &FilenamePair{
		Trader:   fmt.Sprintf("%s__trader.%s", converted, "cfg"),
		Strategy: fmt.Sprintf("%s__strategy_%s.%s", converted, strategy, "cfg"),
	}
}

// GetLogPrefix from botName
func GetLogPrefix(botName string, strategy string) string {
	converted := strings.ToLower(strings.Replace(botName, " ", "_", -1))
	return fmt.Sprintf("%s__%s_", converted, strategy)
}
