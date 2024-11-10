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
	mariahCarey bool
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

	vc.Speaking(true)
	for {
		if len(guild.songQueue) == 0 {
			break
		}

		type errorPackage struct {
			err          error
			discordError string
		}

		c1 := make(chan errorPackage, 1)
		song := guild.songQueue[0]
		go func() {
			for {
				select {
				case <-guild.skipChannel:
					c1 <- errorPackage{err: nil, discordError: ""}
					return
				default:
					err, discordError := playSong(vc, song)
					c1 <- errorPackage{
						err:          err,
						discordError: discordError,
					}
					return
				}
			}
		}()

		select {
		case errPackage := <-c1:
			err = errPackage.err
			discordError = errPackage.discordError
		}
		guild.songQueue = guild.songQueue[1:]

		if err != nil {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	vc.Speaking(false)
	vc.Disconnect()
	if err != nil {
		guild.songQueue = make([]*Song, 0)
		return err, discordError + ", clearing queue"
	}

	return nil, ""
}

func playSong(vc *discordgo.VoiceConnection, song *Song) (error, string) {
	err, dcaSong := song.getDCA()
	if err != nil {
		return err, "Internal Error: Failed to encode mp3 to dca"
	}

	for _, buff := range dcaSong {
		vc.OpusSend <- buff
	}

	return nil, ""
}
