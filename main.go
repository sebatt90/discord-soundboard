package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	
	"github.com/sebatt90/discord-soundboard/db"
	"github.com/sebatt90/discord-soundboard/web"
	"github.com/sebatt90/discord-soundboard/bot"
	"github.com/bwmarrin/discordgo"	
)


func main() {
	// channel for os.Signal
	sc := make(chan os.Signal, 1)
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

	// start bot
	b, err := discordgo.New("Bot " + *token)
	if err != nil {
		fmt.Fprintf(os.Stderr,"[ERROR] Couldn't create Discord bot (%s)\n", err)
		os.Exit(1)
	}
	
	// open web interface
	go web.Start(sc)
	// start bot
	go bot.Start(b, sc)

	
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	// close everything
	db.Close()
	b.Close()
}
