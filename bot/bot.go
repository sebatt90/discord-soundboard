package bot

import (
	"fmt"
	"os"
	"strings"
	"io"
	"bytes"

	"github.com/sebatt90/discord-soundboard/queue"
	"github.com/sebatt90/discord-soundboard/db"
	"github.com/bwmarrin/discordgo"
	"github.com/matthew-balzan/dca"
)

type GuildPlayer struct {
    q queue.Queue
    vc    *discordgo.VoiceConnection
}

var players = make(map[string]*GuildPlayer)

func Start(bot *discordgo.Session, sc chan os.Signal) {
	// Register event handlers here
	bot.AddHandler(onReady)
	bot.AddHandler(onMessage)
	bot.AddHandler(func(s *discordgo.Session, e *discordgo.GuildCreate) {
		err := db.InsertGuild(e.ID, e.Name, e.Icon)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] insert guild error (%s)\n", err)
		}
	})

	// Declare which intents you need
	bot.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates

	if err := bot.Open(); err != nil {
		fmt.Fprintf(os.Stderr,"[ERROR] opening connection (%s)\n", err)
		sc <- os.Interrupt
	}

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
		playSoundboard(s,m)
	}

	if strings.HasPrefix(m.Content, "!disconnect") {
		p, ok := players[m.GuildID]
		if p.vc != nil && ok {
			p.vc.Disconnect()
			p.q.Clear()
			p.vc = nil
		}
	}
}

func playSoundboard(s *discordgo.Session, m *discordgo.MessageCreate){
	var author *discordgo.User = m.Author
	query := strings.TrimPrefix(m.Content, "!play ")

	guild, err := s.State.Guild(m.GuildID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Could not find guild.\n")
		return
	}

	// Insert guild to DB (or replace its data)
	err = db.InsertGuild(guild.ID, guild.Name, guild.Icon)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Cannot add guild to DB (%s)\n",err)
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

	// Create guild entry in hash map if not exists
	if _, ok := players[guild.ID]; !ok {
		players[guild.ID] = &GuildPlayer{}
	}

	var p *GuildPlayer
	p, ok := players[guild.ID]
	if !ok {
		fmt.Fprintf(os.Stderr, "[ERROR] guild not found in map\n")
		return
	}
	
	// Join the voice channel and start routine
	if p.vc == nil {
		p.vc, err = s.ChannelVoiceJoin(m.GuildID, voiceChannelID, false, false)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Could not join voice channel.")
			fmt.Fprintf(os.Stderr,"[ERROR] joining voice channel (%s)\n", err)
			return
		}

		go soundboardPlayer(p, guild.ID)
	}
	s.ChannelMessageSend(m.ChannelID, author.Username + " is querying: "+query)

	// fetch record
	track, err := db.GetTrack(query, guild.ID)

	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "\"**"+query+"**\" yielded no results...")
		return		
	}

	
	s.ChannelMessageSend(m.ChannelID, "Enqueued closest match: **"+track.Name+"**")

	// enqueue
	p.q.Enqueue(track)
}


func soundboardPlayer(p *GuildPlayer, guildid string) {
	for true {
		for p.q.IsEmpty() { /* ackward wait*/ }
		if p.vc == nil {
			break
		}
		track := p.q.Dequeue()
		if track == nil { continue }
		encodeSession, err := dca.EncodeMem(bytes.NewReader(track.Data), dca.StdEncodeOptions)
		if err != nil {
			fmt.Fprintf(os.Stderr,"[ERROR] encode error (%s) \n", err)
			return
		}
		defer encodeSession.Cleanup()

		
		// Stream into the voice connection
		p.vc.Speaking(true)
		done := make(chan error)
		dca.NewStream(encodeSession, p.vc, done)
		if err := <-done; err != nil && err != io.EOF {
			fmt.Fprintf(os.Stderr,"[ERROR] stream error (%s)\n[INFO] ffmpeg messages: %s\n", err, encodeSession.FFMPEGMessages())
			return
		}

		p.vc.Speaking(false)
	}

}
