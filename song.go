package main

import (
	"io"
	"log"
	"os"
	"time"

	"github.com/jogramming/dca"
	"github.com/kkdai/youtube/v2"
)

type Song struct {
	Metadata *youtube.Video
}

func newSong(client *youtube.Client, link string) (err error, discordError string, song *Song) {
	c1 := make(chan *youtube.Video, 1)
	c2 := make(chan error, 1)
	go func() {
		video, err := client.GetVideo(link)
		if err != nil {
			c2 <- err
		}
		c1 <- video
	}()

	select {
	case video := <-c1:
		return nil, "", &Song{
			Metadata: video,
		}
	case err := <-c2:
		discordError = "I couldn't find a video at that link"
		return err, discordError, nil
	case <-time.After(2 * time.Second):
		discordError = "I couldn't find a video at that link"
		return err, discordError, nil
	}
}

func (song *Song) saveSong(client *youtube.Client, options *dca.EncodeOptions) (err error, discordError string) {
	os.Mkdir("songs", 0755)
	if _, err := os.Stat("./songs/" + song.Metadata.ID + ".dca"); err == nil {
		err := os.Chtimes("./songs/"+song.Metadata.ID+".dca", time.Now(), time.Now())
		if err != nil {
			log.Printf("Non-fatal error occured while updating existing song file: %s", err)
		}
		return nil, ""
	}

	formats := song.Metadata.Formats.WithAudioChannels().Itag(160)
	if formats == nil {
		formats = song.Metadata.Formats.WithAudioChannels()
	}
	ytStream, _, err := client.GetStream(song.Metadata, &formats[0])
	if err != nil {
		discordError = "Internal Error: Could not attach video download"
		return err, discordError
	}
	defer ytStream.Close()

	encodingSession, err := dca.EncodeMem(ytStream, options)
	if err != nil {
		discordError = "Internal Error: Could not encode video to dca"
		return err, discordError
	}
	defer encodingSession.Cleanup()

	file, err := os.Create("./songs/" + song.Metadata.ID + ".dca")
	if err != nil {
		discordError = "Internal Error: Could not create empty .dca file"
		return err, discordError
	}
	defer file.Close()

	_, err = io.Copy(file, encodingSession)
	if err != nil {
		discordError = "Internal Error: Could not save song to file system"
		return err, discordError
	}

	return nil, ""
}
