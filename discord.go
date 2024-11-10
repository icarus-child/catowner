package main

import (
	"errors"
	"sort"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Guild struct {
	id          string
	songQueue   []*Song
	isPlaying   bool
	skipChannel chan bool
}

func getGuildFromID(guildID string) (guild *Guild) {
	guildIndex := sort.Search(len(guilds), func(i int) bool {
		return guilds[i].id == guildID
	})
	if guildIndex == len(guilds) {
		panic(errors.New("processing bot is being run in an unregistered guild"))
	}
	guild = guilds[guildIndex]
	return guild
}

func getUserVoiceChannel(guild *discordgo.Guild, event *discordgo.InteractionCreate) (error, string) {
	for _, vs := range guild.VoiceStates {
		if vs.UserID == event.Member.User.ID {
			return nil, vs.ChannelID
		}
	}
	return errors.New("could not find User.ID in VoiceStates slice"), ""
}

func startPlaying(session *discordgo.Session, voiceChannelID string, guild *Guild) (err error, discordError string) {
	vc, err := session.ChannelVoiceJoin(guild.id, voiceChannelID, false, true)
	if err != nil {
		guild.songQueue = make([]*Song, 0)
		return err, "Error while joining voice call, clearing queue"
	}

	time.Sleep(250 * time.Millisecond)

	for {
		if len(guild.songQueue) == 0 {
			break
		}

		song := guild.songQueue[0]
		err, discordError = playSong(vc, song, guild.skipChannel)
		guild.songQueue = guild.songQueue[1:]
		if err != nil {
			break
		}

		time.Sleep(250 * time.Millisecond)
	}
	vc.Disconnect()
	if err != nil {
		guild.songQueue = make([]*Song, 0)
		return err, discordError + ", clearing queue"
	}

	return nil, ""
}

func playSong(vc *discordgo.VoiceConnection, song *Song, skip chan bool) (error, string) {
	err, dcaSong := song.getDCA()
	if err != nil {
		return err, "Internal Error: Failed to encode mp3 to dca"
	}

	vc.Speaking(true)
	for _, buff := range dcaSong {
		select {
		case <-skip:
			return nil, ""
		default:
			vc.OpusSend <- buff
		}
	}
	vc.Speaking(false)

	return nil, ""
}
