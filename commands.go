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
		Name:        "play",
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
		Name:        "listqueue",
		Description: "List the song queue, 0 is the current song",
	},
	{
		Name:        "skip",
		Description: "List the song queue, 0 is the current song",
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
		log.Println("Failed to deliver above error to discord")
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
	case "play":
		playCommand(session, event, optionMap, ytClient)
	case "listqueue":
		listQueueCommand(session, event)
	case "skip":
		skipCommand(session, event)
	}
}

func playCommand(session *discordgo.Session, event *discordgo.InteractionCreate, optionMap map[string]*discordgo.ApplicationCommandInteractionDataOption, client *youtube.Client) {
	guild := getGuildFromID(event.GuildID)

	realGuild, err := session.State.Guild(event.GuildID)
	if err != nil {
		handleError(err, "Internal Error: Could not retrieve guild", session, event)
		return
	}

	err, voiceChannelID := getUserVoiceChannel(realGuild, event)
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

	if guild.isPlaying {
		return
	}

	guild.isPlaying = true
	err, discordError = startPlaying(session, voiceChannelID, guild)
	guild.isPlaying = false
	if err != nil {
		handleError(err, discordError, session, event)
		return
	}
}

func listQueueCommand(session *discordgo.Session, event *discordgo.InteractionCreate) {
	guild := getGuildFromID(event.GuildID)

	response := "**Queue:**"
	for n, song := range guild.songQueue {
		seconds := int(math.Floor(song.Metadata.Duration.Minutes()))
		minutes := int(song.Metadata.Duration.Seconds() - 60*math.Floor(song.Metadata.Duration.Minutes()))
		response += fmt.Sprintf("\n%d: %s by %s (%d:%d)", n, song.Metadata.Title, song.Metadata.Author, seconds, minutes)
	}
	if len(guild.songQueue) == 0 {
		response += "\nno songs in queue!"
	}

	session.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: response,
		},
	})
}

func skipCommand(session *discordgo.Session, event *discordgo.InteractionCreate) {
	guild := getGuildFromID(event.GuildID)

	var response string
	if guild.isPlaying {
		guild.skipChannel <- true
		response = "Skipped song"
	} else {
		response = "No song is playing"
	}

	session.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: response,
		},
	})
}
