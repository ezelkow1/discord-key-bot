package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

//GameKey struct
type GameKey struct {
	Author      string
	GameName    string
	Serial      string
	ServiceType string
}

//Configuration for bot
type Configuration struct {
	Token            string
	BroadcastChannel string
	DbFile           string
	KeyRole          string
}

// Variables used for command line parameters or global
var (
	config      = Configuration{}
	re          = regexp.MustCompile(`.*\s`)
	gog         = regexp.MustCompile(`^[a-z,A-Z,0-9]{5}-[a-z,A-Z,0-9]{5}-[a-z,A-Z,0-9]{5}-[a-z,A-Z,0-9]{5}$`)
	steamOne    = regexp.MustCompile(`^[a-z,A-Z,0-9]{5}-[a-z,A-Z,0-9]{5}-[a-z,A-Z,0-9]{5}$`)
	steamTwo    = regexp.MustCompile(`^[a-z,A-Z,0-9]{5}-[a-z,A-Z,0-9]{5}-[a-z,A-Z,0-9]{5}-[a-z,A-Z,0-9]{5}-[a-z,A-Z,0-9]{5}$`)
	ps3         = regexp.MustCompile(`^[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}$`)
	uplayOne    = regexp.MustCompile(`^[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}$`)
	uplayTwo    = regexp.MustCompile(`^[a-z,A-Z,0-9]{3}-[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}$`)
	origin      = regexp.MustCompile(`^[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}$`)
	url         = regexp.MustCompile(`^http`)
	x           = make(map[string][]GameKey)
	configfile  string
	embedColor  = 0x00ff00
	initialized = false
)

func init() {

	flag.StringVar(&configfile, "c", "", "Configuration file location")
	flag.Parse()

	if configfile == "" {
		fmt.Println("No config file entered")
		os.Exit(1)
	}

	if _, err := os.Stat(configfile); os.IsNotExist(err) {
		fmt.Println("Configfile does not exist, you should make one")
		os.Exit(2)
	}

	fileh, _ := os.Open(configfile)
	decoder := json.NewDecoder(fileh)
	err := decoder.Decode(&config)
	if err != nil {
		fmt.Println("error: ", err)
		os.Exit(3)
	}
}

func main() {

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + config.Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Register ready as a callback for the ready events.
	dg.AddHandler(ready)

	// Register messageCreate as a callback for message events
	dg.AddHandler(messageCreate)

	if _, err := os.Stat(config.DbFile); os.IsNotExist(err) {
		fmt.Println("Db File does not exist, creating")
		newFile, _ := os.Create(config.DbFile)
		newFile.Close()

	}

	Load(config.DbFile, &x)
	checkDB()
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

//checkDB
// This scans the map to make sure all fields exist properly and if not
// populate them
func checkDB() {
	//Check servicetype exists, can add future entries here
	for i := range x {
		for k := range x[i] {
			if x[i][k].ServiceType == "" {
				x[i][k].ServiceType = getGameServiceString(x[i][k].Serial)
			}
		}
	}

	Save(config.DbFile, &x)
}

// This function will be called (due to AddHandler above) when the bot receives
// the "ready" event from Discord.
func ready(s *discordgo.Session, event *discordgo.Ready) {

	// Discord just loves to send ready events during server hiccups
	// This prevents spamming
	if initialized == false {
		// Set the playing status.
		s.UpdateStatus(0, "")
		//SendEmbed(s, config.BroadcastChannel, "", "I iz here", "Keybot has arrived. You may now use me like the dumpster I am")
		initialized = true
	}
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Only allow messages in either DM or broadcast channel
	dmchan, _ := s.UserChannelCreate(m.Author.ID)
	if (m.ChannelID != config.BroadcastChannel) && (m.ChannelID != dmchan.ID) {
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

	// Search for substring of request
	if strings.HasPrefix(m.Content, "!search ") == true {
		SearchGame(s, m)
	}

	if strings.HasPrefix(m.Content, "!help") == true {
		PrintHelp(s, m)
	}
}

//PrintHelp will print out the help dialog
func PrintHelp(s *discordgo.Session, m *discordgo.MessageCreate) {
	var buffer bytes.Buffer
	buffer.WriteString("!add game name key - this will add a new key to the database. This should be done in a DM with the bot\n")
	buffer.WriteString("!listkeys - PLEASE USE THIS IN A PRIVATE MESSAGE WITH THE BOT. Lists current games and the number of available keys\n")
	buffer.WriteString("!take game name - Will give you one of the keys for the game in a DM\n")
	buffer.WriteString("!search search-string - Will search the database for matching games")
	SendEmbed(s, m.ChannelID, "", "Available Commands", buffer.String())
}

//SearchGame will scan the map keys for a match on the substring
func SearchGame(s *discordgo.Session, m *discordgo.MessageCreate) {

	foundgame := false
	m.Content = strings.TrimPrefix(m.Content, "!search ")

	Load(config.DbFile, &x)
	if len(x) == 0 {
		SendEmbed(s, config.BroadcastChannel, "", "Empty Database", "No Keys present in Database")
		return
	}

	search := NormalizeGame(m.Content)

	// Lets make this pretty, sort keys by name
	keys := make([]string, 0, len(x))
	for key := range x {
		keys = append(keys, key)
	}

	var buffer bytes.Buffer
	var output string
	for i := range keys {
		if strings.Contains(keys[i], search) {
			if !foundgame {
				foundgame = true
			}
			buffer.WriteString(x[keys[i]][0].GameName)
			buffer.WriteString(" (")
			buffer.WriteString(getGameServiceString(x[keys[i]][0].Serial))
			buffer.WriteString(")")
			buffer.WriteString(": ")
			buffer.WriteString(strconv.Itoa(len(x[keys[i]])))
			buffer.WriteString(" keys\n")
		}
	}

	if foundgame {
		output = buffer.String()
	} else {
		output = "No Matches Found"
	}

	SendEmbed(s, m.ChannelID, "", "Search Results", output)
}

//GrabKey will grab one of the keys for the current game, pop it, and save
func GrabKey(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Clean up content
	m.Content = strings.TrimPrefix(m.Content, "!take ")
	gamename := m.Content
	normalized := NormalizeGame(gamename)

	Load(config.DbFile, &x)
	//We need to pop off a game, if it exists
	if len(x[normalized]) > 0 {
		var userkey GameKey
		userkey, x[normalized] = x[normalized][0], x[normalized][1:]
		dmchan, _ := s.UserChannelCreate(m.Author.ID)

		//Send the key to the user
		SendEmbed(s, dmchan.ID, "", "Here is your key", userkey.GameName+" ("+userkey.ServiceType+")"+": "+userkey.Serial+"\nThis game was brought to you by "+userkey.Author)
		if userkey.ServiceType == "Steam" {
			SendEmbed(s, dmchan.ID, "", "Steam Redeem Link", "https://store.steampowered.com/account/registerkey?key="+userkey.Serial)
		}

		if userkey.ServiceType == "GOG" {
			SendEmbed(s, dmchan.ID, "", "GOG Redeem Link", "https://www.gog.com/redeem/"+userkey.Serial)
		}
		//Announce to channel
		SendEmbed(s, config.BroadcastChannel, "", "Another satisfied customer", m.Author.Username+" has just taken a key for "+userkey.GameName+". There are "+
			strconv.Itoa(len(x[normalized]))+" keys remaining")

		//If no more keys, remove entry in map
		if len(x[normalized]) == 0 {
			delete(x, normalized)
		}

		Save(config.DbFile, &x)

	} else {
		SendEmbed(s, m.ChannelID, "", "WHY U DO DIS?", gamename+" doesn't exist you cheeky bastard!")
	}
}

//ListKeys lists what games and how many keys for each
func ListKeys(s *discordgo.Session, m *discordgo.MessageCreate) {
	Load(config.DbFile, &x)
	if len(x) == 0 {
		SendEmbed(s, m.ChannelID, "", "EMPTY DATABASE", "No Keys present in Database")
		return
	}

	// Lets make this pretty, sort keys by name
	keys := make([]string, 0, len(x))
	for key := range x {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Build the output message
	var buffer bytes.Buffer
	k := 0
	i := 0
	for i = range keys {
		buffer.WriteString(x[keys[i]][0].GameName)
		buffer.WriteString(" (")
		buffer.WriteString(getGameServiceString(x[keys[i]][0].Serial))
		buffer.WriteString(")")
		buffer.WriteString(" : ")
		buffer.WriteString(strconv.Itoa(len(x[keys[i]])))
		buffer.WriteString(" keys\n")
		k++

		if k == 20 {
			SendEmbed(s, m.ChannelID, "Current Key List",
				"Games: "+strconv.Itoa((i+1)-(k-1))+" to "+strconv.Itoa((i+1)),
				buffer.String())
			buffer.Reset()
			k = 0
		}
	}

	if k != 0 {
		SendEmbed(s, m.ChannelID, "Current Key List",
			"Games: "+strconv.Itoa((i+1)-(k-1))+" to "+strconv.Itoa(i+1),
			buffer.String())
		buffer.Reset()
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
		SendEmbed(s, m.ChannelID, "", "WHY U NO READ INSTRUCTION", "Don't be silly "+m.Author.Username+". Send me your key in a private message")
		return
	}

	addcount := strings.Count(m.Content, "!add")

	if addcount > 1 {
		SendEmbed(s, m.ChannelID, "", "Too many keys!", "Only one !add at a time please")
		return
	}

	// Strip the cmd, split off key from regex and grab name
	m.Content = strings.TrimPrefix(m.Content, "!add ")
	regtest := re.Split(m.Content, -1)
	key := regtest[1]
	gamename := CleanKey(m.Content, key)
	normalized := NormalizeGame(gamename)

	var thiskey GameKey
	thiskey.Author = m.Author.Username
	thiskey.GameName = gamename
	thiskey.Serial = key
	thiskey.ServiceType = getGameServiceString(thiskey.Serial)

	Load(config.DbFile, &x)

	//Check if key already exists
	for i := range x[normalized] {
		if thiskey.Serial == x[normalized][i].Serial {
			SendEmbed(s, m.ChannelID, "", "Already Exists", "Key already entered in database")
			return
		}
	}

	//Add new key and notify user and channel, then save to disk
	x[normalized] = append(x[normalized], thiskey)

	SendEmbed(s, config.BroadcastChannel, "", "All Praise "+thiskey.Author, "Thanks "+thiskey.Author+
		" for adding a key for "+thiskey.GameName+" ("+thiskey.ServiceType+"). There are now "+strconv.Itoa(len(x[normalized]))+
		" keys for "+thiskey.GameName)

	SendEmbed(s, m.ChannelID, "", "All Praise "+thiskey.Author, "Thanks "+thiskey.Author+
		" for adding a key for "+thiskey.GameName+" ("+thiskey.ServiceType+"). There are now "+strconv.Itoa(len(x[normalized]))+
		" keys for "+thiskey.GameName)

	Save(config.DbFile, &x)
}
