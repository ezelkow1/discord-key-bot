package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

//GameKey struct
type GameKey struct {
	Author   string
	GameName string
	Serial   string
}

//Configuration for bot
type Configuration struct {
	Token            string
	BroadcastChannel string
	DbFile           string
}

// Variables used for command line parameters
var (
	config = Configuration{}
	re     = regexp.MustCompile("([a-z A-Z]* )")
	x      = make(map[string][]GameKey)
)

func init() {
	fileh, _ := os.Open("conf.json")
	decoder := json.NewDecoder(fileh)
	err := decoder.Decode(&config)
	if err != nil {
		fmt.Println("error: ", err)
	}
}

func main() {

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + config.Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	dg.AddHandler(messageCreate)

	if _, err := os.Stat(config.DbFile); os.IsNotExist(err) {
		fmt.Println("Db File does not exist, creating")
		newFile, _ := os.Create(config.DbFile)
		newFile.Close()

	}
	Load(config.DbFile, x)

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

	// Add a new key to the db
	if strings.HasPrefix(m.Content, "!add ") == true {
		AddGame(s, m)
	}

	// List off current games and amount of keys for each
	if m.Content == "!listkeys" {
		ListKeys(s, m)
	}

	// Grab a key for a game
	if strings.HasPrefix(m.Content, "!take ") == true {
		GrabKey(s, m)
	}
}

//GrabKey will grab one of the keys for the current game, pop it, and save
func GrabKey(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Clean up content
	m.Content = strings.TrimPrefix(m.Content, "!take ")
	gamename := CleanGame(m.Content, " ")
	normalized := NormalizeGame(gamename)
	//We need to pop off a game, if it exists

	if len(x[normalized]) > 0 {
		var userkey GameKey
		userkey, x[normalized] = x[normalized][0], x[normalized][1:]
		dmchan, _ := s.UserChannelCreate(m.Author.ID)
		stringout := []string{userkey.GameName, ": ", userkey.Serial, " was brought to you by ", userkey.Author}
		s.ChannelMessageSend(dmchan.ID, strings.Join(stringout, ""))

		stringout = []string{m.Author.Username, " has just taken a key for ", userkey.GameName, ". There are ", strconv.Itoa(len(x[normalized])), " keys remaining"}
		s.ChannelMessageSend(config.BroadcastChannel, strings.Join(stringout, ""))

		if len(x[normalized]) == 0 {
			delete(x, normalized)
		}
		Save(config.DbFile, x)
	} else {
		stringout := []string{gamename, " doesn't exist you cheeky bastard!"}
		s.ChannelMessageSend(m.ChannelID, strings.Join(stringout, ""))
	}

	//Now we need to broadcast that key was taken, by who
}

//ListKeys lists what games and how many keys for each
func ListKeys(s *discordgo.Session, m *discordgo.MessageCreate) {
	Load(config.DbFile, &x)
	if len(x) == 0 {
		s.ChannelMessageSend(config.BroadcastChannel, "No Keys present in Database")
		return
	}
	for i := range x {
		stringout := []string{x[i][0].GameName, " : ", strconv.Itoa(len(x[i])), " keys"}
		s.ChannelMessageSend(m.ChannelID, strings.Join(stringout, ""))
	}
}

//AddGame will add a new key to the db
// It will also check to see if the key was put in the
// broadcast chan, remove if necessary
func AddGame(s *discordgo.Session, m *discordgo.MessageCreate) {

	//Check if the user added the key from the broadcast chan, if so
	//immediately delete the msg and warn
	if m.ChannelID == config.BroadcastChannel {
		s.ChannelMessageDelete(m.ChannelID, m.ID)
		stringout := []string{"Don't be silly ", m.Author.Username, " putting keys in here is for kids"}
		s.ChannelMessageSend(m.ChannelID, strings.Join(stringout, ""))
		return
	}

	m.Content = strings.TrimPrefix(m.Content, "!add ")
	regtest := re.Split(m.Content, -1)
	key := regtest[1]
	gamename := CleanGame(m.Content, key)
	normalized := NormalizeGame(gamename)

	var thiskey GameKey
	thiskey.Author = m.Author.Username
	thiskey.GameName = gamename
	thiskey.Serial = key
	Load(config.DbFile, &x)

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
	s.ChannelMessageSend(config.BroadcastChannel, strings.Join(stringout, ""))
	s.ChannelMessageSend(m.ChannelID, strings.Join(stringout, ""))
	Save(config.DbFile, x)
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
