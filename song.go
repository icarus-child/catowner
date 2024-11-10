package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/bogem/id3v2"
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
		return errors.New("fetchVideoMetadata failed: Process timed out"), discordError, nil
	}
}

func (song *Song) saveSong(client *youtube.Client) (err error, discordError string) {
	os.Mkdir("songs", 0755)
	if _, err := os.Stat("./songs/" + song.Metadata.ID + ".mp3"); err == nil {
		err := os.Chtimes("./songs/"+song.Metadata.ID+".mp3", time.Now(), time.Now())
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

	err, mp3_out := runFFMPEGFromStdin(populateStdinReader(ytStream), exec.Command("ffmpeg", "-i", "pipe:0", "-f", "mp3", "pipe:1"))
	if err != nil {
		discordError = "Internal Error: Process ffmpeg failed"
		return err, discordError
	}

	file, err := os.Create("./songs/" + song.Metadata.ID + ".mp3")
	if err != nil {
		discordError = "Internal Error: Could not create empty .mp3 file"
		return err, discordError
	}
	defer file.Close()

	_, err = io.Copy(file, mp3_out)
	if err != nil {
		discordError = "Internal Error: Could not save song to file system"
		return err, discordError
	}

	err = song.addMP3Metadata(file)
	if err != nil {
		log.Printf("Non-fatal error occured while writing mp3 metadata: %s", err)
	}

	err = checkSongCount(SONG_DOWNLOAD_MAX)
	if err != nil {
		log.Printf("Non-fatal error occured while removing old mp3s: %s", err)
	}

	return nil, ""
}

func populateStdinReader(input io.Reader) func(io.WriteCloser) {
	return func(stdin io.WriteCloser) {
		defer stdin.Close()
		io.Copy(stdin, input)
	}
}

func populateStdinFile(file []byte) func(io.WriteCloser) {
	return func(stdin io.WriteCloser) {
		defer stdin.Close()
		io.Copy(stdin, bytes.NewReader(file))
	}
}

func runFFMPEGFromStdin(populate_stdin_func func(io.WriteCloser), cmd *exec.Cmd) (error, *bytes.Buffer) {
	resultBuffer := new(bytes.Buffer)
	// cmd.Stderr = os.Stderr
	cmd.Stdout = resultBuffer
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err, nil
	}
	err = cmd.Start()
	if err != nil {
		return err, nil
	}
	populate_stdin_func(stdin)
	err = cmd.Wait()
	if err != nil {
		return err, nil
	}

	return nil, resultBuffer
}

func (song *Song) addMP3Metadata(file io.Reader) error {
	tag, err := id3v2.ParseReader(file, id3v2.Options{
		ParseFrames: []string{},
	})
	if err != nil {
		return err
	}
	tag.SetTitle(song.Metadata.Title)
	tag.SetArtist(song.Metadata.Author)

	if err = tag.Save(); err != nil {
		return err
	}

	return nil
}

func (song *Song) getDCA() (error, [][]byte) {
	songBuffer, err := os.Open("./songs/" + song.Metadata.ID + ".mp3")
	if err != nil {
		return err, nil
	}

	ffmpegBuff := new(bytes.Buffer)
	ffmpeg := exec.Command("ffmpeg", "-i", "pipe:0", "-f", "s16le", "-ar", "48000", "-ac", "2", "pipe:1")
	ffmpegStdin, _ := ffmpeg.StdinPipe()
	ffmpeg.Stdout = ffmpegBuff

	dcaBuffer := new(bytes.Buffer)
	dca := exec.Command("dca")
	dca.Stdout = dcaBuffer
	dcaStdin, _ := dca.StdinPipe()

	ffmpeg.Start()
	songBuffer.WriteTo(ffmpegStdin)
	ffmpegStdin.Close()
	ffmpeg.Wait()

	dca.Start()
	ffmpegBuff.WriteTo(dcaStdin)
	dcaStdin.Close()
	dca.Wait()

	err, ret := loadSound(dcaBuffer)
	if err != nil {
		return err, nil
	}

	return nil, ret
}

func loadSound(inBuffer *bytes.Buffer) (err error, outBuffer [][]byte) {
	outBuffer = make([][]byte, 0)

	var opuslen int16

	for {
		// Read opus frame length from dca file.
		err = binary.Read(inBuffer, binary.LittleEndian, &opuslen)

		// If this is the end of the file, just return.
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return nil, outBuffer
		} else if err != nil {
			return err, nil
		}

		// Read encoded pcm from dca file.
		InBuf := make([]byte, opuslen)
		err = binary.Read(inBuffer, binary.LittleEndian, &InBuf)
		// Should not be any end of file errors
		if err != nil {
			return err, nil
		}

		// Append encoded pcm data to the buffer.
		outBuffer = append(outBuffer, InBuf)
	}
}

func checkSongCount(maxCount int) (err error) {
	files, err := os.ReadDir("./songs")
	if err != nil {
		return err
	}

	filesTruncated := make([]fs.DirEntry, 0)
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".mp3") {
			filesTruncated = append(filesTruncated, file)
		}
	}
	files = filesTruncated

	if len(files) <= maxCount {
		return
	}
	slices.SortFunc(files, func(a, b fs.DirEntry) int {
		aInfo, _ := a.Info()
		bInfo, _ := b.Info()
		if aInfo.ModTime().Equal(bInfo.ModTime()) {
			return 0
		} else if aInfo.ModTime().Before(bInfo.ModTime()) {
			return 1
		} else {
			return -1
		}
	})
	for {
		file := files[len(files)-1]
		files = files[:len(files)-1]

		err = os.Remove("./songs/" + file.Name())
		if err != nil {
			return err
		}

		if len(files) <= maxCount {
			return nil
		}
	}
}
