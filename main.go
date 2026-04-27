package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"strings"
	"io"
	"bytes"

	"github.com/sebatt90/discord-soundboard/db"
	"github.com/bwmarrin/discordgo"
	"github.com/matthew-balzan/dca"
)

var q Queue
var vc *discordgo.VoiceConnection

func startRoutine() {
	go func() {
		for true {
			for q.IsEmpty() { /* ackward wait*/ }
			track := q.Dequeue()
			encodeSession, err := dca.EncodeMem(bytes.NewReader(track.Data), dca.StdEncodeOptions)
			if err != nil {
				fmt.Fprintf(os.Stderr,"[ERROR] encode error (%s) \n", err)
				return
			}
			defer encodeSession.Cleanup()

			
			// Stream into the voice connection
			vc.Speaking(true)
			done := make(chan error)
			dca.NewStream(encodeSession, vc, done)
			if err := <-done; err != nil && err != io.EOF {
				fmt.Fprintf(os.Stderr,"[ERROR] stream error (%s) \n", err)
			}

			vc.Speaking(false)

			fmt.Printf("[INFO] ffmpeg messages: %s\n", encodeSession.FFMPEGMessages())
		}
	}()

}

func main() {
	token := flag.String("token", "", "Discord bot token")
	flag.Parse()

	if *token == "" {
		fmt.Fprintf(os.Stderr, "[ERROR] --token is required\n")
		os.Exit(1)
	}

	// init db
	if err := db.Init("data.db"); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] db init error (%s)\n", err)
		os.Exit(1)
	}
	defer db.Close()
	
	bot, err := discordgo.New("Bot " + *token)
	if err != nil {
		fmt.Fprintf(os.Stderr,"[ERROR] Couldn't create Discord session (%s)\n", err)
		os.Exit(1)
	}

	// Register event handlers here
	bot.AddHandler(onReady)
	bot.AddHandler(onMessage)

	// Declare which intents you need
	bot.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates

	if err = bot.Open(); err != nil {
		fmt.Fprintf(os.Stderr,"[ERROR] opening connection (%s)\n", err)
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	
	bot.Close()
}

func onReady(s *discordgo.Session, e *discordgo.Ready) {
	fmt.Printf("[INFO] Logged in as %s#%s\n", e.User.Username, e.User.Discriminator)
}

func onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, "!play") {
		play(s,m)
	}
}

func play(s *discordgo.Session, m *discordgo.MessageCreate){
	var author *discordgo.User = m.Author
	query := strings.TrimPrefix(m.Content, "!play ")

	guild, err := s.State.Guild(m.GuildID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Could not find guild.\n")
		return
	}


	var voiceChannelID string
	for _, vs := range guild.VoiceStates {
		if vs.UserID == author.ID {
			voiceChannelID = vs.ChannelID
			break
		}
	}

	if voiceChannelID == "" {
		s.ChannelMessageSend(m.ChannelID, "You need to be in a voice channel first!")
		return
	}

	// Join the voice channel and start routine
	if vc == nil {
		vc, err = s.ChannelVoiceJoin(m.GuildID, voiceChannelID, false, false)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Could not join voice channel.")
			fmt.Fprintf(os.Stderr,"[ERROR] joining voice channel (%s)\n", err)
			return
		}

		startRoutine()
	}
	s.ChannelMessageSend(m.ChannelID, author.Username + " is querying: "+query)

	// fetch record
	track, err := db.GetTrack(query)

	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "\"**"+query+"**\" yielded no results...")
		return		
	}

	
	s.ChannelMessageSend(m.ChannelID, "Enequeued closest match: **"+track.Name+"**")

	// enqueue
	q.Enqueue(track)
}
