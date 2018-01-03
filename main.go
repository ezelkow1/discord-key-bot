package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	//"github.com/rapidloop/skv"
)

const file = "keys.db"

// Variables used for command line parameters
var (
	Token            string
	re               = regexp.MustCompile("([a-z A-Z]* )")
	broadcastChannel = "397967839572787202"
	//store, err = skv.Open("keys.db")
	x = make(map[string][]GameKey)
)

func init() {

	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.Parse()
}

//GameKey struct
type GameKey struct {
	Author   string
	GameName string
	Serial   string
}

func main() {

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	dg.AddHandler(messageCreate)

	Load(file, x)
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
		gamename := CleanGame(m.Content, key)
		normalized := NormalizeGame(gamename)

		var thiskey GameKey
		thiskey.Author = m.Author.Username
		thiskey.GameName = gamename
		thiskey.Serial = key
		Load(file, &x)

		//Check if key already exists
		for i := range x[normalized] {
			if thiskey.Serial == x[normalized][i].Serial {
				s.ChannelMessageSend(m.ChannelID, "Key already entered")
				return
			}
		}

		//Add new key and notify user and channel, then save to disk
		x[normalized] = append(x[normalized], thiskey)
		stringout := []string{"Thanks ", thiskey.Author, " for adding a key for ", thiskey.GameName,
			". There are now ", strconv.Itoa(len(x[normalized])), " keys for ", thiskey.GameName}
		s.ChannelMessageSend(broadcastChannel, strings.Join(stringout, ""))
		s.ChannelMessageSend(m.ChannelID, strings.Join(stringout, ""))
		Save(file, x)
	}

	// List off current games and amount of keys for each
	if m.Content == "!listkeys" {
		Load(file, &x)
		for i := range x {
			stringout := []string{x[i][0].GameName, " : ", strconv.Itoa(len(x[i])), " keys"}
			s.ChannelMessageSend(m.ChannelID, strings.Join(stringout, ""))
		}
	}

	// Grab a key for a game
	if strings.HasPrefix(m.Content, "!take ") == true {
		// Clean up content
		m.Content = strings.TrimPrefix(m.Content, "!take ")
		gamename := CleanGame(m.Content, " ")
		normalized := NormalizeGame(gamename)
		//We need to pop off a game, if it exists
		mykeys, ok := x[normalized]
		if ok {
			stringout := []string{gamename, " has ", strconv.Itoa(len(mykeys)), " keys"}
			s.ChannelMessageSend(broadcastChannel, strings.Join(stringout, ""))
		} else {
			stringout := []string{gamename, " doesn't exist you cheeky bastard!"}
			s.ChannelMessageSend(m.ChannelID, strings.Join(stringout, ""))
		}

		//Now we need to broadcast that key was taken, by who
	}
}

//CleanGame cleans up the input name
func CleanGame(name string, key string) string {
	tmp := strings.TrimSuffix(name, key)
	tmp = strings.TrimSpace(tmp)
	return tmp
}

//NormalizeGame the name of the game
func NormalizeGame(name string) string {
	tmp := strings.ToLower(name)
	tmp = strings.Replace(tmp, " ", "", -1)
	return tmp
}

// Save via Gob to file
func Save(path string, object interface{}) {
	b, err := json.Marshal(object)
	if err != nil {
		fmt.Println("error on marshall")
	}
	fileh, err := os.Create(path)
	n, err := fileh.Write(b)
	b = b[:n]
	fileh.Close()
	return
}

// Load Gob file
func Load(path string, object interface{}) {
	fileh, err := os.Open(path)
	fileinfo, err := fileh.Stat()
	_ = err
	b := make([]byte, fileinfo.Size())
	n, err := fileh.Read(b)
	if err != nil {
		fmt.Println(err)
		return
	}
	b = b[:n]
	json.Unmarshal(b, &object)
	fileh.Close()
	return
}
