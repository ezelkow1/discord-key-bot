package main

import (
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	//"github.com/rapidloop/skv"
	//"encoding/gob"
	"io/ioutil"
	"os"
	"os/signal"
	"regexp"
	//"runtime"
	"strconv"
	"strings"
	"syscall"
)

const file = "keys.db"

// Variables used for command line parameters
var (
	Token string
	re    = regexp.MustCompile("([a-z A-Z]* )")
	//store, err = skv.Open("keys.db")
	x      = make(map[string][]gamekey)
	myfile *os.File
)

func init() {

	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.Parse()
}

type gamekey struct {
	author   string
	gamename string
	serial   string
}

func main() {

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}
	myfile, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE, 0666)
	_ = err
	_ = myfile
	x, err := ioutil.ReadFile(file)
	_ = x
	_ = err
	/*
		fileInfo, err := os.Stat(file)
		_ = fileInfo
		if err != nil {
			if os.IsNotExist(err) {
				//Create file
				myfile, err := os.Create(file)
				_ = err
				_ = myfile
			} else if os.IsExist(err) {
				myfile := os.OpenFile(file, os.O_RDWR|os.O_CREATE, 0666)
				//Read file
				x, err := ioutil.ReadFile(file)
				_ = err
				fmt.Println("Current file", x)
			}
		} */
	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	// If the message is "ping" reply with "Pong!"
	if strings.HasPrefix(m.Content, "!add ") == true {
		m.Content = strings.TrimPrefix(m.Content, "!add ")
		regtest := re.Split(m.Content, -1)
		key := regtest[1]
		gamename := strings.TrimSuffix(m.Content, key)
		gamename = strings.TrimSpace(gamename)
		normalized := strings.ToLower(gamename)
		normalized = strings.Replace(normalized, " ", "", -1)
		stringout := []string{"Key: ", key, ", game: ", gamename, ", normalized: ", normalized}
		s.ChannelMessageSend(m.ChannelID, strings.Join(stringout, ""))

		var thiskey gamekey
		thiskey.author = m.Author.Username
		thiskey.gamename = gamename
		thiskey.serial = key

		fmt.Println("loaded file")
		for i := range x[normalized] {
			if thiskey.serial == x[normalized][i].serial {
				fmt.Println("Key already entered")
				return
			}
		}
		x[normalized] = append(x[normalized], thiskey)
		stringout = []string{"Thanks ", thiskey.author, " for adding a key for ", thiskey.gamename,
			". There are now ", strconv.Itoa(len(x[normalized])), " keys for ", thiskey.gamename}
		s.ChannelMessageSend(m.ChannelID, strings.Join(stringout, ""))
		fmt.Println("keys:", x)

	}

	// If the message is "pong" reply with "Ping!"
	if m.Content == "pong" {
		s.ChannelMessageSend(m.ChannelID, "Ping!")
	}
}
