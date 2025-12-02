package main

import (
	"fmt"
	"io"
	"os"
	"os/user"
	"os/exec"
	"regexp"
	"time"
	"path/filepath"
	
	"github.com/kkdai/youtube/v2"
)

const usage = "Usage: DownloadYTAudio <YouTubeLink>\n" +
	"Example: DownloadYTAudio https://www.youtube.com/watch?v=INbQpAoaWSw\n\n"

func main() {
	if len(os.Args) < 2 || len(os.Args) > 2 {
		fmt.Fprintf(os.Stderr, "Error: Missing required arguments.\n%s", usage)
		os.Exit(1)
	}

	videoLink := os.Args[1]
	regex := regexp.MustCompile(`v=([a-zA-Z0-9_-]*)`)
	matches := regex.FindStringSubmatch(videoLink)

	if len(matches) < 2 {
		fmt.Fprintf(os.Stderr, "Error: No video ID in YouTube link.\n%s", usage)
		os.Exit(1)
	}

	videoId := matches[1]

	client := youtube.Client{}

	video, err := client.GetVideo(videoId)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Unable to get YouTube link.\n%s", usage)
		os.Exit(1)
	}


	durationSeconds := fmt.Sprintf("%v", video.Duration.Seconds())
	formats := video.Formats.WithAudioChannels()

	var targetFormat int = -1
	for i, format := range formats {
		if format.MimeType == "audio/mp4; codecs=\"mp4a.40.2\"" {
			targetFormat = i
			break
		}
	}

	if targetFormat == -1 {
		fmt.Fprintf(os.Stderr, "Error: No audio channel for YouTube link.\n%s", usage)
		os.Exit(1)
	}

	stream, _, err := client.GetStream(video, &formats[targetFormat])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Unable to get stream for YouTube link.\n%s", usage)
		os.Exit(1)
	}
	defer stream.Close()

	tempFilename := fmt.Sprintf("%s-temp.mp4", videoId)
	filename := fmt.Sprintf("%s.mp3", videoId)
	today := time.Now().Format("2006-01-02")

	currentUser, err := user.Current()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Unable to get current user.\n%s", usage)
		os.Exit(1)
	}

	// The HomeDir field contains the path to the home directory
	homeDir := currentUser.HomeDir

	dirPath := filepath.Join(homeDir, "Desktop/YTVideos", today)
	const dirPerm os.FileMode = 0755

	err = os.MkdirAll(dirPath, dirPerm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Unable to create dir on Desktop.\n%s", usage)
		os.Exit(1)
	}

	tempFilePath := filepath.Join(dirPath, tempFilename)
	filePath := filepath.Join(dirPath, filename)

	file, err := os.Create(tempFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Unable to create audio file.\n%s", usage)
		os.Exit(1)
	}
	defer file.Close()

	fmt.Printf("Downloading audio... ")
	_, err = io.Copy(file, stream)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Error writing audio file.\n%s", usage)
		os.Exit(1)
	}

	cmd := exec.Command("ffmpeg",
		"-i", tempFilePath,    // Input file
		"-t", durationSeconds, // Duration to process (trim)
		"-c", "copy",       // Optional: copies streams for speed, but may require re-encoding if output format changes
		"-vn", "-acodec", "libmp3lame",
		filePath,
	)

	//fmt.Printf("Executing command: %v\n", cmd.Args)
	fmt.Println("Trimming & converting audio...")
	
	_, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Error trimming audio file.\n%s", usage)
		os.Exit(1)
	}

	exec.Command("rm", tempFilePath).CombinedOutput()

	fmt.Fprintf(os.Stdout, "Successfully wrote %s\n", filePath)
}

