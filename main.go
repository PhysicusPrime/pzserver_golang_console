package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/gorcon/rcon"
)

// RCON-Verbindung
var rconClient *rcon.Conn

// Admin-Befehle registrieren
var adminCommands = map[string]func(args string, dg *discordgo.Session){
	"say": func(args string, dg *discordgo.Session) {
		// Nachricht an festgelegten Discord-Channel senden
		channelID := os.Getenv("DISCORD_CHANNEL_ID")
		//channelID := "1398097436525723839" // <-- hier Channel-ID eintragen
		_, err := dg.ChannelMessageSend(channelID, args)
		if err != nil {
			fmt.Println("Fehler beim Senden:", err)
		} else {
			fmt.Println("Nachricht gesendet:", args)
		}
	},
	"kick": func(args string, dg *discordgo.Session) {
		// RCON-Kick-Befehl
		resp, err := rconClient.Execute("kick " + args)
		if err != nil {
			fmt.Println("RCON Fehler:", err)
		} else {
			fmt.Println("RCON Antwort:", resp)
		}
	},
	"listplayers": func(args string, dg *discordgo.Session) {
		resp, err := rconClient.Execute("getplayers")
		if err != nil {
			fmt.Println("RCON Fehler:", err)
		} else {
			fmt.Println("Spielerliste:\n", resp)
		}
	},
}

func main() {
	// Discord Token
	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		log.Fatal("Bitte setze DISCORD_BOT_TOKEN")
	}

	// RCON verbinden
	addr := "34.32.108.129:27015"
	password := "x3pc092201"
	var err error
	rconClient, err = rcon.Dial(addr, password)
	if err != nil {
		log.Fatalf("Fehler beim Verbinden mit RCON: %v", err)
	}
	defer rconClient.Close()
	fmt.Println("Verbunden mit RCON-Server.")

	// Discord Session
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("Fehler beim Erstellen des Bots: %v", err)
	}
	dg.AddHandler(messageCreate)
	err = dg.Open()
	if err != nil {
		log.Fatalf("Fehler beim Öffnen der Discord-Verbindung: %v", err)
	}
	defer dg.Close()

	fmt.Println("Bot läuft. Admin-Konsole aktiv. Drücke Strg+C zum Beenden.")

	// Goroutine: Admin-Konsole
	go adminConsole(dg)

	// Hauptthread: Signal für sauberes Beenden
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt, syscall.SIGTERM)
	<-sc
	fmt.Println("Bot beendet.")
}

func adminConsole(dg *discordgo.Session) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("[Admin-Konsole] > ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if input == "exit" {
			fmt.Println("Admin-Konsole wird beendet.")
			os.Exit(0)
		}

		// RCON-Befehle direkt ausführen
		if strings.HasPrefix(input, "rcon ") {
			command := strings.TrimPrefix(input, "rcon ")
			response, err := rconClient.Execute(command)
			if err != nil {
				fmt.Println("RCON-Fehler:", err)
			} else {
				fmt.Println("RCON-Antwort:", response)
			}
			continue
		}

		// Bot-Befehle ausführen
		parts := strings.SplitN(input, " ", 2)
		cmd := parts[0]
		args := ""
		if len(parts) > 1 {
			args = parts[1]
		}

		if handler, ok := adminCommands[cmd]; ok {
			handler(args, dg)
		} else {
			fmt.Println("Unbekannter Befehl:", cmd)
		}
	}
}

// Discord Event-Handler
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	content := strings.TrimSpace(m.Content)

	if strings.HasPrefix(content, "!rcon ") {
		command := strings.TrimPrefix(content, "!rcon ")
		response, err := rconClient.Execute(command)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("RCON Fehler: %v", err))
			return
		}
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("RCON Antwort: %s", response))
	}

	if strings.HasPrefix(content, "!bot ") {
		command := strings.TrimPrefix(content, "!bot ")
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Bot Konsole: Befehl '%s' erhalten", command))
	}
}
