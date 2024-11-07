package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/switchupcb/dasgo/dasgo"
)

type DiscordUser dasgo.GetSticker

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "ping",
		Description: "Bot will respond with pong",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "message",
				Description: "Will be repeated back to you",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			},
			{
				Name:        "author",
				Description: "Whether to prepend message's author",
				Type:        discordgo.ApplicationCommandOptionBoolean,
			},
		},
	},
}

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}
	Token := os.Getenv("DISCORD_BOT_TOKEN")
	App := os.Getenv("DISCORD_APPLICATION_ID")

	session, err := discordgo.New("Bot " + Token)
	if err != nil {
		panic(err)
	}

	session.AddHandler(ready)
	err = session.Open()
	if err != nil {
		log.Fatalf("Could not open session: %s", err)
	}

	_, err := session.ApplicationCommandBulkOverwrite(App)

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt)
	out := <-signalChannel
	log.Printf("receieved signal: " + out.String() + ", exiting")
	err = session.Close()
	if err != nil {
		log.Fatalf("Could not close session gracefully: %s", err)
	}
}

func ready(session *discordgo.Session, event *discordgo.Ready) {
	log.Printf("Logged in as %s", event.User.String())
	session.UpdateCustomStatus("/listento")
}

func handleSlashCommands(session *discordgo.Session, event *discordgo.InteractionCreate) {
	if event.Type != discordgo.InteractionApplicationCommand {
		return
	}

	data := event.ApplicationCommandData()
	switch data.Name {
	case "ping":
		fmt.Print("pong")
	}
}
