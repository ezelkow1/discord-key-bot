package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	_ "github.com/mattn/go-sqlite3"
)

//Configuration for bot
type Configuration struct {
	Token            string
	BroadcastChannel string
	DbFile           string
	KeyRole          string
	UserFile         string
	ListPMOnly       bool
	OwnerID          string
}

// Variables used for command line parameters or global
var (
	config   = Configuration{}
	re       = regexp.MustCompile(`.*\s`)
	gog      = regexp.MustCompile(`^[a-z,A-Z,0-9]{5}-[a-z,A-Z,0-9]{5}-[a-z,A-Z,0-9]{5}-[a-z,A-Z,0-9]{5}$`)
	steamOne = regexp.MustCompile(`^[a-z,A-Z,0-9]{5}-[a-z,A-Z,0-9]{5}-[a-z,A-Z,0-9]{5}$`)
	steamTwo = regexp.MustCompile(`^[a-z,A-Z,0-9]{5}-[a-z,A-Z,0-9]{5}-[a-z,A-Z,0-9]{5}-[a-z,A-Z,0-9]{5}-[a-z,A-Z,0-9]{5}$`)
	ps3      = regexp.MustCompile(`^[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}$`)
	uplayOne = regexp.MustCompile(`^[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}$`)
	uplayTwo = regexp.MustCompile(`^[a-z,A-Z,0-9]{3}-[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}$`)
	origin   = regexp.MustCompile(`^[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}$`)
	url      = regexp.MustCompile(`^http`)

	configfile  string
	embedColor  = 0x00ff00
	initialized = false
	guildID     string
	roleID      string
	limitUsers  = false
	database    *sql.DB
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

	database, err = sql.Open("sqlite3", config.DbFile)
	if err != nil {
		log.Fatal(err)
		return
	}

	statement, err := database.Prepare("CREATE TABLE IF NOT EXISTS games (id INTEGER PRIMARY KEY, normalized TEXT NOT NULL, pretty TEXT NOT NULL, UNIQUE(normalized, pretty))")
	defer statement.Close()
	_, err = statement.Exec()
	if err != nil {
		log.Fatal(err)
		return
	}

	statement, _ = database.Prepare("CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, discordid INTEGER UNIQUE)")
	_, err = statement.Exec()
	if err != nil {
		log.Fatal(err)
		return
	}
	statement, err =
		database.Prepare("CREATE TABLE IF NOT EXISTS" +
			" keys (id INTEGER PRIMARY KEY, key TEXT NOT NULL, game_id INTEGER NOT NULL, userid_entered INTEGER NOT NULL, userid_taken INTEGER, service TEXT," +
			"UNIQUE(key) " +
			"FOREIGN KEY (game_id) REFERENCES games(id) " +
			"FOREIGN KEY (userid_entered) REFERENCES users(id) " +
			"FOREIGN KEY (userid_taken) REFERENCES users(id))")

	_, err = statement.Exec()
	if err != nil {
		log.Fatal(err)
		return
	}
}

func main() {

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + config.Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Close the DB on shutdown
	defer database.Close()

	// Register ready as a callback for the ready events.
	dg.AddHandler(ready)

	// Register messageCreate as a callback for message events
	dg.AddHandler(messageCreate)

	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening connection to discord servers, ", err)
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
	// Discord just loves to send ready events during server hiccups
	// This prevents spamming
	if initialized == false {
		// Set the playing status.
		s.UpdateStatus(0, "")
		//SendEmbed(s, config.BroadcastChannel, "", "I iz here", "Keybot has arrived. You may now use me like the dumpster I am")
		if config.KeyRole != "" {
			guildID = event.Guilds[0].ID
			refreshRoles(s)
		}

		initialized = true
	}
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Only allow messages in either DM or broadcast channel
	dmchan, err := s.UserChannelCreate(m.Author.ID)

	if err != nil {
		fmt.Println("error: ", err)
		fmt.Println("messageCreate err in creating dmchan")
		return
	}

	if (m.ChannelID != config.BroadcastChannel) && (m.ChannelID != dmchan.ID) {
		return
	}

	// Skip any messages we dont care about
	if checkPrefix(m.Content) == false {
		return
	}

	// Check if a user has the proper role, if a non-empty role is set
	if !isUserRoleAllowed(s, m) {
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

	if m.Content == "!mygames" {
		if limitUsers {
			PrintMyGames(s, m)
		}
	}

	if m.Content == "!totals" {
		PrintTotals(s, m)
	}
	if strings.HasPrefix(m.Content, "!speak") == true {
		if m.Author.ID == config.OwnerID {
			Speak(s, m)
		}
	}
}

// PrintTotals prints the total number of games and keys
func PrintTotals(s *discordgo.Session, m *discordgo.MessageCreate) {
	counts, err := database.Query("SELECT COUNT(*) FROM keys")
	defer counts.Close()
	if err != nil {
		SendEmbed(s, m.ChannelID, "", "Total Games", "Error getting keys totals")
	}
	var keys int
	var games int
	for counts.Next() {
		counts.Scan(&keys)
	}
	counts, err = database.Query("SELECT COUNT(*) FROM games")
	if err != nil {
		SendEmbed(s, m.ChannelID, "", "Total Games", "Error getting games totals")
	}
	for counts.Next() {
		counts.Scan(&games)
	}

	SendEmbed(s, m.ChannelID, "", "Total Games", "There are "+strconv.Itoa(games)+" games with "+strconv.Itoa(keys)+" keys")
	return
}

// Speak lets a specified owner ID speak from the bot
func Speak(s *discordgo.Session, m *discordgo.MessageCreate) {
	m.Content = strings.TrimPrefix(m.Content, "!speak ")
	s.ChannelMessageSend(config.BroadcastChannel, m.Content)
}

//PrintHelp will print out the help dialog
func PrintHelp(s *discordgo.Session, m *discordgo.MessageCreate) {
	var buffer bytes.Buffer
	buffer.WriteString("!add game name key - this will add a new key to the database. This should be done in a DM with the bot\n")
	buffer.WriteString("!listkeys - Lists current games and the number of available keys. This should be done in a DM with the bot\n")
	buffer.WriteString("!take game name - Will give you one of the keys for the game in a DM\n")
	if limitUsers {
		buffer.WriteString("!mygames - Will give a list of games you have taken\n")
	}
	buffer.WriteString("!search search-string - Will search the database for matching games\n")
	buffer.WriteString("!totals - Will give a total number of games and keys stored")
	SendEmbed(s, m.ChannelID, "", "Available Commands", buffer.String())
}

//SearchGame will scan the map keys for a match on the substring
func SearchGame(s *discordgo.Session, m *discordgo.MessageCreate) {

	m.Content = strings.TrimPrefix(m.Content, "!search ")

	search := NormalizeGame(m.Content)

	// First lets find how many matches we have, maybe its less efficient then doing it all at once?
	// Either way we dont want to flood a channel if its >20 results so grab the count
	counts, err := database.Query("SELECT COUNT(*) FROM games WHERE normalized like ?", "%"+search+"%")
	defer counts.Close()
	if err != nil {
		SendEmbed(s, m.ChannelID, "", "Search Results", "Error getting count of games")
		return
	}

	var gamecount int
	for counts.Next() {
		counts.Scan(&gamecount)
	}
	if gamecount <= 0 {
		SendEmbed(s, m.ChannelID, "", "Search results", "No Matches Found for: "+search)
		return
	}

	var buffer bytes.Buffer
	k := 0
	dmchan, err := s.UserChannelCreate(m.Author.ID)
	if err != nil {
		return
	}

	if gamecount > 20 && (m.ChannelID != dmchan.ID) {
		SendEmbed(s, m.ChannelID, "", "Too Many Results", "There are too many search results, please do this search in a DM")
		return
	}

	// Collect the matching games for the search and the count of keys based on the games ID value
	list, err := database.Query("SELECT DISTINCT pretty as name, (SELECT COUNT(*) FROM keys WHERE games.id = game_id) gamecount FROM games INNER JOIN keys on keys.game_id = games.id WHERE games.normalized like ? order by pretty", "%"+search+"%")
	defer list.Close()
	if err != nil {
		SendEmbed(s, m.ChannelID, "", "Search Results", "Error getting list of matching games")
		return
	}

	for list.Next() {
		var name string
		var count int
		list.Scan(&name, &count)
		buffer.WriteString(name)
		buffer.WriteString(": ")
		buffer.WriteString(strconv.Itoa(count))
		buffer.WriteString(" keys\n")
		k++
		if k == 20 {
			SendEmbed(s, m.ChannelID, "", "Search Results", buffer.String())
			buffer.Reset()
			k = 0
		}
	}
	if k != 0 {
		SendEmbed(s, m.ChannelID, "", "Search Results", buffer.String())
		buffer.Reset()
	}
}

//CheckUserLimitAllowed to see if a user has this normalized game name in their taken list
func CheckUserLimitAllowed(s *discordgo.Session, m *discordgo.MessageCreate, gamename string) bool {

	if limitUsers == false {
		// We aren't doing user limiting, always allow
		return true
	}

	//normalized := NormalizeGame(gamename)

	return true
}

//GrabKey will grab one of the keys for the current game, pop it, and save
func GrabKey(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Clean up content
	m.Content = strings.TrimPrefix(m.Content, "!take ")
	gamename := m.Content
	//normalized := NormalizeGame(gamename)

	allowed := CheckUserLimitAllowed(s, m, gamename)

	// User already took this game
	if !allowed {
		dmchan, _ := s.UserChannelCreate(m.Author.ID)
		SendEmbed(s, dmchan.ID, "", "Access Denied", "You have already received a copy of "+gamename)
		return
	} else if limitUsers {
		//User limiting enabled and user allowed, add game
		//userList[m.Author.Username] = append(userList[m.Author.Username], gamename)
	}

	//We need to pop off a game, if it exists
	if true {
		//Send the key to the user
		//Announce to channel

		//If no more keys, remove entry in map

	} else {
		SendEmbed(s, m.ChannelID, "", "WHY U DO DIS?", gamename+" doesn't exist you cheeky bastard!")
	}
}

//PrintMyGames will print out the list of games a user has taken
func PrintMyGames(s *discordgo.Session, m *discordgo.MessageCreate) {

	//SendEmbed(s, m.ChannelID, "", "Game List", buffer.String())
	return
}

//ListKeys lists what games and how many keys for each
func ListKeys(s *discordgo.Session, m *discordgo.MessageCreate) {

	if config.ListPMOnly {
		if m.ChannelID == config.BroadcastChannel {
			SendEmbed(s, m.ChannelID, "", "Not in here", "Don't do that in here")
			return
		}
	}

	k := 0
	var buffer bytes.Buffer
	list, err := database.Query("SELECT DISTINCT pretty as name, (SELECT COUNT(*) FROM keys WHERE games.id = game_id) gamecount FROM games INNER JOIN keys on keys.game_id = games.id order by pretty")
	defer list.Close()
	if err != nil {
		SendEmbed(s, m.ChannelID, "", "Search Results", "Error getting list of games")
		return
	}

	for list.Next() {
		var pretty string
		var count int
		err = list.Scan(&pretty, &count)
		if err == sql.ErrNoRows {
			SendEmbed(s, m.ChannelID, "", "Game List", "No Games in Database")
			return
		}

		buffer.WriteString(pretty)
		buffer.WriteString(": ")
		buffer.WriteString(strconv.Itoa(count))
		buffer.WriteString(" keys\n")
		k++

		if k == 20 {
			SendEmbed(s, m.ChannelID, "", "Game List", buffer.String())
			buffer.Reset()
			k = 0
		}
	}

	if k != 0 {
		SendEmbed(s, m.ChannelID, "", "Game List", buffer.String())
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

	userid := CheckAndUpdateUser(s, m)
	// Strip the cmd, split off key from regex and grab name
	//m.Content = strings.TrimPrefix(m.Content, "!add ")
	//regtest := re.Split(m.Content, -1)
	//key := regtest[1]
	//gamename := CleanKey(m.Content, key)
	//normalized := NormalizeGame(gamename)

	/*var thiskey GameKey
	thiskey.Author = m.Author.Username
	thiskey.GameName = gamename
	thiskey.Serial = key
	thiskey.ServiceType = getGameServiceString(thiskey.Serial)
	*/
	//Check if key already exists

	//Add new key and notify user and channel, then save to disk
	/*
		SendEmbed(s, config.BroadcastChannel, "", "All Praise "+thiskey.Author, "Thanks "+thiskey.Author+
			" for adding a key for "+thiskey.GameName+" ("+thiskey.ServiceType+"). There are now "+strconv.Itoa(len(x[normalized]))+
			" keys for "+thiskey.GameName)

		SendEmbed(s, m.ChannelID, "", "All Praise "+thiskey.Author, "Thanks "+thiskey.Author+
			" for adding a key for "+thiskey.GameName+" ("+thiskey.ServiceType+"). There are now "+strconv.Itoa(len(x[normalized]))+
			" keys for "+thiskey.GameName)
	*/
}

// CheckAndUpdateUser Check current user info against database, update if necessary, and return DB ID
func CheckAndUpdateUser(s *discordgo.Session, m *discordgo.MessageCreate) int {

	var noMatchName bool
	var noMatchDiscordid bool

	// Search for users discordid in the DB, this should be the fast case as people are added

	// Search for users name in the DB, here we will need to check/update their discordid
	// What to do if someone shadows a name?
	list, err := database.Query("SELECT name, id, discordid FROM users where name = ?", m.Author.Username)
	defer list.Close()
	if err != nil {
		SendEmbed(s, m.ChannelID, "", "Add Game", "Error looking up user "+m.Author.Username+" in database")
		return
	}

	for list.Next() {
		var name string
		var id int
		var discordid int
		err = list.Scan(&name, &id, &discordid)

		if err == sql.ErrNoRows {
			// User has no matching DB entry to name, jump to add new user
		}
	}

	// User has no entry in DB, add a new one
}
