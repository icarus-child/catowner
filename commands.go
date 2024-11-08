package main

import (
	"github.com/bwmarrin/discordgo"
	"github.com/jogramming/dca"
	"github.com/kkdai/youtube/v2"
)

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "ping",
		Description: "Bot will respond with pong",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "message",
				Description: "Will be repeated back to you",
				Type:        discordgo.ApplicationCommandOptionString,
			},
			{
				Name:        "author",
				Description: "Whether to prepend message's author",
				Type:        discordgo.ApplicationCommandOptionBoolean,
			},
		},
	},
	{
		Name:        "playsong",
		Description: "Add song from YouTube to the queue, bot will join the user's call and begin playing",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "url",
				Description: "YouTube URL to pull the song from",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			},
		},
	},
}

func handleError(err error, discordError string, session *discordgo.Session, event *discordgo.InteractionCreate) {
	err = session.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: discordError + "\n" + err.Error(),
		},
	})
	if err != nil {
		panic(err)
	}
}

func handleSlashCommands(session *discordgo.Session, event *discordgo.InteractionCreate) {
	if event.Type != discordgo.InteractionApplicationCommand {
		return
	}

	options := event.ApplicationCommandData().Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}
	switch event.ApplicationCommandData().Name {
	case "ping":
		pingCommand(session, event, optionMap)
	case "playsong":
		playCommand(session, event, optionMap, ytClient, dcaOptions)
	}
}

func pingCommand(session *discordgo.Session, event *discordgo.InteractionCreate, optionMap map[string]*discordgo.ApplicationCommandInteractionDataOption) {
	response := "pong"
	if optionMap["message"] != nil {
		response += " " + optionMap["message"].StringValue()
	}
	if optionMap["author"] != nil && optionMap["author"].BoolValue() {
		response = event.Member.Mention() + " " + response
	}
	session.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: response,
		},
	})
}

func playCommand(session *discordgo.Session, event *discordgo.InteractionCreate, optionMap map[string]*discordgo.ApplicationCommandInteractionDataOption, client *youtube.Client, options *dca.EncodeOptions) {
	err, discordError, song := newSong(client, optionMap["url"].StringValue())
	if err != nil {
		handleError(err, discordError, session, event)
	}
	err, discordError = song.saveSong(client, options)
	if err != nil {
		handleError(err, discordError, session, event)
	}
}
