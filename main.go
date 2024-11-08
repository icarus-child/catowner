package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
	"github.com/jogramming/dca"
	"github.com/joho/godotenv"
	"github.com/kkdai/youtube/v2"
)

var (
	guildSongQueue map[string][]*Song
	dcaOptions     *dca.EncodeOptions
	ytClient       *youtube.Client
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}
	Token := os.Getenv("DISCORD_BOT_TOKEN")
	App := os.Getenv("DISCORD_APPLICATION_ID")

	dcaOptions = dca.StdEncodeOptions
	dcaOptions.Bitrate = 64
	dcaOptions.Application = "lowdelay"

	ytClient = &youtube.Client{}

	session, err := discordgo.New("Bot " + Token)
	if err != nil {
		panic(err)
	}

	_, err = session.ApplicationCommandBulkOverwrite(App, "394662250188636161", commands)
	if err != nil {
		log.Fatalf("Could not register commands: %s", err)
	}

	// coms, err := session.ApplicationCommands(App, "394662250188636161")
	// if err != nil {
	// 	log.Fatalf("Could not retrieve commands: %s", err)
	// }
	// for _, command := range coms {
	// 	log.Printf("Removing command %s", command.Name)
	// 	session.ApplicationCommandDelete(App, "394662250188636161", command.ID)
	// }

	session.AddHandler(ready)
	session.AddHandler(handleSlashCommands)
	handleLoop(session)
}

func ready(session *discordgo.Session, event *discordgo.Ready) {
	log.Printf("Logged in as %s", event.User.String())
	session.UpdateCustomStatus("/listento")
}

func handleLoop(session *discordgo.Session) {
	err := session.Open()
	if err != nil {
		log.Fatalf("Could not open session: %s", err)
	}
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt)
	out := <-signalChannel
	log.Printf("Receieved signal: " + out.String() + ", exiting")
	err = session.Close()
	if err != nil {
		log.Fatalf("Could not close session gracefully: %s", err)
	}
}
