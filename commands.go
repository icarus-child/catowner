package main

import (
	"fmt"
	"log"
	"math"

	"github.com/bwmarrin/discordgo"
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
		Description: "Add a song from YouTube to the queue, the bot will join the user's call and begin playing",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "url",
				Description: "YouTube URL to pull the song from",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			},
		},
	},
	{
		Name:        "join",
		Description: "Join or move to the sender's voice channel [TESTING COMMAND]",
	},
}

func handleError(err error, discordError string, session *discordgo.Session, event *discordgo.InteractionCreate) {
	log.Printf("Runtime Error: %s\nCustom Message: %s", err, discordError)
	err = session.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("%s\n*%s*", discordError, err.Error()),
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
		playCommand(session, event, optionMap, ytClient)
	case "join":
		joinCommand(session, event)
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

func playCommand(session *discordgo.Session, event *discordgo.InteractionCreate, optionMap map[string]*discordgo.ApplicationCommandInteractionDataOption, client *youtube.Client) {
	guild := getGuildFromID(event.GuildID)

	eventGuild, err := session.State.Guild(event.GuildID)
	if err != nil {
		handleError(err, "Internal Error: Could not retrieve guild", session, event)
		return
	}

	err, voiceChannelID := getUserVoiceChannel(eventGuild, event)
	if err != nil {
		handleError(err, "You aren't in a voice channel I can see!", session, event)
		return
	}

	err, discordError, song := newSong(client, optionMap["url"].StringValue())
	if err != nil {
		handleError(err, discordError, session, event)
		return
	}

	err, discordError = song.saveSong(client)
	if err != nil {
		handleError(err, discordError, session, event)
		return
	}

	seconds := int(math.Floor(song.Metadata.Duration.Minutes()))
	minutes := int(song.Metadata.Duration.Seconds() - 60*math.Floor(song.Metadata.Duration.Minutes()))
	session.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Added %s by %s (%d:%d) to the queue", song.Metadata.Title, song.Metadata.Author, seconds, minutes),
		},
	})

	guild.songQueue = append(guild.songQueue, song)
	err, discordError = startPlaying(session, voiceChannelID, guild, song)
	if err != nil {
		handleError(err, discordError, session, event)
		return
	}
}

func joinCommand(session *discordgo.Session, event *discordgo.InteractionCreate) {
	eventGuild, err := session.State.Guild(event.GuildID)
	if err != nil {
		handleError(err, "Internal Error: Could not retrieve guild", session, event)
		return
	}

	err, voiceChannelID := getUserVoiceChannel(eventGuild, event)
	if err != nil {
		handleError(err, "You aren't in a voice channel I can see!", session, event)
		return
	}

	_, err = session.ChannelVoiceJoin(event.GuildID, voiceChannelID, false, true)
	if err != nil {
		handleError(err, "Error while joining voice call", session, event)
		return
	}
}
