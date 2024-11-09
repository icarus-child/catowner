package main

import (
	"errors"
	"io"
	"sort"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/expiteRz/dca"
)

type Guild struct {
	id          string
	songQueue   []*Song
	mariahCarey bool
}

func getGuildFromID(guildID string) (guild Guild) {
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

func startPlaying(session *discordgo.Session, voiceChannelID string, guild Guild, song *Song) (err error, discordError string) {
	vc, err := session.ChannelVoiceJoin(guild.id, voiceChannelID, false, true)
	if err != nil {
		return err, "Error while joining voice call"
	}

	time.Sleep(250 * time.Millisecond)

	vc.Speaking(true)
	playSong(vc, song)
	vc.Speaking(false)

	return nil, ""
}

func playSong(vc *discordgo.VoiceConnection, song *Song) (error, string) {
	encodingSession, err := dca.EncodeFile("./song/"+song.Metadata.ID+".mp3", dcaOptions)
	if err != nil {
		return err, "Internal Error: Failed to encode mp3 to dca"
	}
	done := make(chan error)
	dca.NewStream(encodingSession, vc, done)
	err = <-done
	if err != nil && err != io.EOF {
		return err, "Internal Error: Playback failed for unknown reasons"
	}
	return nil, ""
}
