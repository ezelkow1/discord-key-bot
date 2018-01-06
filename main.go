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

// Variables used for command line parameters or global
var (
	config     = Configuration{}
	re         = regexp.MustCompile(`.*\s`)
	x          = make(map[string][]GameKey)
	configfile string
	embedColor = 0x00ff00
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

// This function will be called (due to AddHandler above) when the bot receives
// the "ready" event from Discord.
func ready(s *discordgo.Session, event *discordgo.Ready) {

	// Set the playing status.
	s.UpdateStatus(0, "keys go in my piehole")
	SendEmbed(s, config.BroadcastChannel, "", "I iz here", "Keybot has arrived. You may now use me like the dumpster I am")
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
	buffer.WriteString("!listkeys - Lists current games and the number of available keys\n")
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

	//We need to pop off a game, if it exists
	if len(x[normalized]) > 0 {
		var userkey GameKey
		userkey, x[normalized] = x[normalized][0], x[normalized][1:]
		dmchan, _ := s.UserChannelCreate(m.Author.ID)

		//Send the key to the user
		SendEmbed(s, dmchan.ID, "", "Here is your key", userkey.GameName+": "+userkey.Serial+"\nThis game was brought to you by "+userkey.Author)

		//Announce to channel
		SendEmbed(s, config.BroadcastChannel, "", "Another satisfied customer", m.Author.Username+" has just taken a key for "+userkey.GameName+". There are "+
			strconv.Itoa(len(x[normalized]))+" keys remaining")

		//If no more keys, remove entry in map
		if len(x[normalized]) == 0 {
			delete(x, normalized)
		}

		Save(config.DbFile, x)

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
		" for adding a key for "+thiskey.GameName+". There are now "+strconv.Itoa(len(x[normalized]))+
		" keys for "+thiskey.GameName)

	SendEmbed(s, m.ChannelID, "", "All Praise "+thiskey.Author, "Thanks "+thiskey.Author+
		" for adding a key for "+thiskey.GameName+". There are now "+strconv.Itoa(len(x[normalized]))+
		" keys for "+thiskey.GameName)

	Save(config.DbFile, x)
}

//CleanKey cleans up the input name. Strips trailing key from input
func CleanKey(name string, key string) string {
	tmp := strings.TrimSuffix(name, key)
	tmp = strings.TrimSpace(tmp)
	return tmp
}

//NormalizeGame the name of the game, removes spaces, lowercases
func NormalizeGame(name string) string {
	tmp := strings.ToLower(name)
	tmp = strings.Replace(tmp, " ", "", -1)
	return tmp
}

// Save via json to file
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

// Load json file
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

//SendEmbed will send an embed msg
func SendEmbed(s *discordgo.Session, channelID string, title string, fieldTitle string, field string) {
	if title != "" {
		embed := NewEmbed().
			SetTitle(title).
			AddField(fieldTitle, field).
			SetColor(embedColor).MessageEmbed
		s.ChannelMessageSendEmbed(channelID, embed)
	} else {
		embed := NewEmbed().
			AddField(fieldTitle, field).
			SetColor(embedColor).MessageEmbed
		s.ChannelMessageSendEmbed(channelID, embed)
	}

}

//Embed ...
type Embed struct {
	*discordgo.MessageEmbed
}

// Constants for message embed character limits
const (
	EmbedLimitTitle       = 256
	EmbedLimitDescription = 2048
	EmbedLimitFieldValue  = 1024
	EmbedLimitFieldName   = 256
	EmbedLimitField       = 25
	EmbedLimitFooter      = 2048
	EmbedLimit            = 4000
)

//NewEmbed returns a new embed object
func NewEmbed() *Embed {
	return &Embed{&discordgo.MessageEmbed{}}
}

//SetTitle ...
func (e *Embed) SetTitle(name string) *Embed {
	e.Title = name
	return e
}

//SetDescription [desc]
func (e *Embed) SetDescription(description string) *Embed {
	if len(description) > 2048 {
		description = description[:2048]
	}
	e.Description = description
	return e
}

//AddField [name] [value]
func (e *Embed) AddField(name, value string) *Embed {
	if len(value) > 1024 {
		value = value[:1024]
	}

	if len(name) > 1024 {
		name = name[:1024]
	}

	e.Fields = append(e.Fields, &discordgo.MessageEmbedField{
		Name:  name,
		Value: value,
	})

	return e

}

//SetFooter [Text] [iconURL]
func (e *Embed) SetFooter(args ...string) *Embed {
	iconURL := ""
	text := ""
	proxyURL := ""

	switch {
	case len(args) > 2:
		proxyURL = args[2]
		fallthrough
	case len(args) > 1:
		iconURL = args[1]
		fallthrough
	case len(args) > 0:
		text = args[0]
	case len(args) == 0:
		return e
	}

	e.Footer = &discordgo.MessageEmbedFooter{
		IconURL:      iconURL,
		Text:         text,
		ProxyIconURL: proxyURL,
	}

	return e
}

//SetImage ...
func (e *Embed) SetImage(args ...string) *Embed {
	var URL string
	var proxyURL string

	if len(args) == 0 {
		return e
	}
	if len(args) > 0 {
		URL = args[0]
	}
	if len(args) > 1 {
		proxyURL = args[1]
	}
	e.Image = &discordgo.MessageEmbedImage{
		URL:      URL,
		ProxyURL: proxyURL,
	}
	return e
}

//SetThumbnail ...
func (e *Embed) SetThumbnail(args ...string) *Embed {
	var URL string
	var proxyURL string

	if len(args) == 0 {
		return e
	}
	if len(args) > 0 {
		URL = args[0]
	}
	if len(args) > 1 {
		proxyURL = args[1]
	}
	e.Thumbnail = &discordgo.MessageEmbedThumbnail{
		URL:      URL,
		ProxyURL: proxyURL,
	}
	return e
}

//SetAuthor ...
func (e *Embed) SetAuthor(args ...string) *Embed {
	var (
		name     string
		iconURL  string
		URL      string
		proxyURL string
	)

	if len(args) == 0 {
		return e
	}
	if len(args) > 0 {
		name = args[0]
	}
	if len(args) > 1 {
		iconURL = args[1]
	}
	if len(args) > 2 {
		URL = args[2]
	}
	if len(args) > 3 {
		proxyURL = args[3]
	}

	e.Author = &discordgo.MessageEmbedAuthor{
		Name:         name,
		IconURL:      iconURL,
		URL:          URL,
		ProxyIconURL: proxyURL,
	}

	return e
}

//SetURL ...
func (e *Embed) SetURL(URL string) *Embed {
	e.URL = URL
	return e
}

//SetColor ...
func (e *Embed) SetColor(clr int) *Embed {
	e.Color = clr
	return e
}

// InlineAllFields sets all fields in the embed to be inline
func (e *Embed) InlineAllFields() *Embed {
	for _, v := range e.Fields {
		v.Inline = true
	}
	return e
}

// Truncate truncates any embed value over the character limit.
func (e *Embed) Truncate() *Embed {
	e.TruncateDescription()
	e.TruncateFields()
	e.TruncateFooter()
	e.TruncateTitle()
	return e
}

// TruncateFields truncates fields that are too long
func (e *Embed) TruncateFields() *Embed {
	if len(e.Fields) > 25 {
		e.Fields = e.Fields[:EmbedLimitField]
	}

	for _, v := range e.Fields {

		if len(v.Name) > EmbedLimitFieldName {
			v.Name = v.Name[:EmbedLimitFieldName]
		}

		if len(v.Value) > EmbedLimitFieldValue {
			v.Value = v.Value[:EmbedLimitFieldValue]
		}

	}
	return e
}

// TruncateDescription ...
func (e *Embed) TruncateDescription() *Embed {
	if len(e.Description) > EmbedLimitDescription {
		e.Description = e.Description[:EmbedLimitDescription]
	}
	return e
}

// TruncateTitle ...
func (e *Embed) TruncateTitle() *Embed {
	if len(e.Title) > EmbedLimitTitle {
		e.Title = e.Title[:EmbedLimitTitle]
	}
	return e
}

// TruncateFooter ...
func (e *Embed) TruncateFooter() *Embed {
	if e.Footer != nil && len(e.Footer.Text) > EmbedLimitFooter {
		e.Footer.Text = e.Footer.Text[:EmbedLimitFooter]
	}
	return e
}
