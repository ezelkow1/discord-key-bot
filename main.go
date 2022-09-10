package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

// GameKey struct
type GameKey struct {
	Author      string
	GameName    string
	Serial      string
	ServiceType string
}

// Configuration for bot
type Configuration struct {
	Token   string
	DbFile  string
	GuildID string
	AppID   string
}

// Variables used for command line parameters or global
var (
	config   = Configuration{}
	gog      = regexp.MustCompile(`^[a-z,A-Z,0-9]{5}-[a-z,A-Z,0-9]{5}-[a-z,A-Z,0-9]{5}-[a-z,A-Z,0-9]{5}$`)
	steamOne = regexp.MustCompile(`^[a-z,A-Z,0-9]{5}-[a-z,A-Z,0-9]{5}-[a-z,A-Z,0-9]{5}$`)
	steamTwo = regexp.MustCompile(`^[a-z,A-Z,0-9]{5}-[a-z,A-Z,0-9]{5}-[a-z,A-Z,0-9]{5}-[a-z,A-Z,0-9]{5}-[a-z,A-Z,0-9]{5}$`)
	ps3      = regexp.MustCompile(`^[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}$`)
	uplayOne = regexp.MustCompile(`^[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}$`)
	uplayTwo = regexp.MustCompile(`^[a-z,A-Z,0-9]{3}-[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}$`)
	origin   = regexp.MustCompile(`^[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}-[a-z,A-Z,0-9]{4}$`)
	url      = regexp.MustCompile(`^http`)
	// Game key Database
	x = make(map[string][]GameKey)

	configfile  string
	embedColor  = 0x00ff00
	initialized = false
	fileLock    sync.RWMutex
)

var (
	commands = []discordgo.ApplicationCommand{
		{
			Name:        "add",
			Description: "Add a game key",
		},
		{
			Name:        "search",
			Description: "Search game database",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "search",
					Description: "Thing you are searching for",
					Required:    true,
				},
			},
		},
		{
			Name:        "take",
			Description: "Take a game key",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "game",
					Description: "Name of game to take",
					Required:    true,
				},
			},
		},
		{
			Name:        "list",
			Description: "List all games in database",
		},
	}
	commandsHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"add": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			add(s, i)
		},
		"take": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			// Create a slash command here
			take(s, i)
		},
		"search": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			// Create a slash cmd for search
			search(s, i)
		},
		"list": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			list(s, i)
		},
	}
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

	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			if h, ok := commandsHandlers[i.ApplicationCommandData().Name]; ok {
				h(s, i)
			}
		case discordgo.InteractionModalSubmit:
			var embedslice []*discordgo.MessageEmbed
			data := i.ModalSubmitData()

			if !strings.HasPrefix(data.CustomID, "add") {
				return
			}

			userid := strings.Split(data.CustomID, "_")[1]
			member, err := s.GuildMember(config.GuildID, userid)

			if err != nil {
				panic(err)
			}

			numkey := AddGame(data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value,
				data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value,
				member.User.Username)
			if numkey > 0 {
				embedslice = append(embedslice, NewEmbed().SetTitle("All Praise "+member.User.Username).SetColor(embedColor).SetDescription("Thanks "+member.User.Username+" for adding a key for "+
					data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value+". There are now "+strconv.Itoa(numkey)+
					" keys for "+data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value).MessageEmbed)
			} else {
				embedslice = append(embedslice, NewEmbed().AddField("Already In Database", "Key already exists in the database").MessageEmbed)
			}
			err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: embedslice,
				},
			})
			if err != nil {
				panic(err)
			}
		}

	})

	cmdIDs := make(map[string]string, len(commands))

	for _, cmd := range commands {
		rcmd, err := dg.ApplicationCommandCreate(config.AppID, config.GuildID, &cmd)
		if err != nil {
			log.Fatalf("cannot create slash command %q: %v", cmd.Name, err)
		}

		cmdIDs[rcmd.ID] = rcmd.Name
	}

	if _, err := os.Stat(config.DbFile); os.IsNotExist(err) {
		fmt.Println("Db File does not exist, creating")
		newFile, _ := os.Create(config.DbFile)
		newFile.Close()

	}

	fileLock.Lock()
	Load(config.DbFile, &x)
	checkDB()
	fileLock.Unlock()
	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}
	defer dg.Close()

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	//dg.Close()

	for id, name := range cmdIDs {
		err := dg.ApplicationCommandDelete(config.AppID, config.GuildID, id)
		if err != nil {
			log.Fatalf("cant delete slash cmd %q: %v", name, err)
		}
	}
}

// checkDB
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

func list(s *discordgo.Session, i *discordgo.InteractionCreate) {
	list, num := ListKeys()
	var embedslice []*discordgo.MessageEmbed
	embedslice = append(embedslice, NewEmbed().SetTitle("Game List").AddField("Total Games", strconv.Itoa(num+1)).SetColor(embedColor).MessageEmbed)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: embedslice,
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})

	split_list := strings.Split(list, "\n")
	k := 0
	var buffer bytes.Buffer
	for _, str := range split_list {
		buffer.WriteString(str + "\n")
		k++

		if k == 20 {
			embedslice = append(embedslice, NewEmbed().AddField("Search Results", buffer.String()).SetColor(embedColor).MessageEmbed)
			s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Embeds: embedslice,
				Flags:  discordgo.MessageFlagsEphemeral,
			})
			buffer.Reset()
			k = 0
			embedslice = nil
		}
	}

	if k != 0 {
		embedslice = append(embedslice, NewEmbed().AddField("Search Results", buffer.String()).SetColor(embedColor).MessageEmbed)
		buffer.Reset()

		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: embedslice,
			Flags:  discordgo.MessageFlagsEphemeral,
		})
	}
}

func add(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "add_" + i.Interaction.Member.User.ID,
			Title:    "Add Game",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:  "gamename",
							Label:     "Name of Game",
							Style:     discordgo.TextInputShort,
							Required:  true,
							MaxLength: 1000,
							MinLength: 3,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:  "key",
							Label:     "Game Key",
							Style:     discordgo.TextInputShort,
							Required:  true,
							MaxLength: 2000,
						},
					},
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}
}

func search(s *discordgo.Session, i *discordgo.InteractionCreate) {

	foundgame := false
	var embedslice []*discordgo.MessageEmbed
	options := i.ApplicationCommandData().Options
	fileLock.RLock()
	defer fileLock.RUnlock()

	Load(config.DbFile, &x)
	if len(x) == 0 {
		embedslice = append(embedslice, NewEmbed().AddField("Empty Database", "No Games in Database").SetColor(embedColor).MessageEmbed)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: embedslice,
				Flags:  discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	search := NormalizeGame(options[0].StringValue())
	// Lets make this pretty, sort keys by name
	keys := make([]string, 0, len(x))
	for key := range x {
		if strings.Contains(key, search) {
			keys = append(keys, key)
			foundgame = true
		}
	}

	if !foundgame {
		embedslice = append(embedslice, NewEmbed().AddField("Search Results", "No Matches Found").SetColor(embedColor).MessageEmbed)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: embedslice,
				Flags:  discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	embedslice = append(embedslice, NewEmbed().AddField("Search Results", ".....").SetColor(embedColor).MessageEmbed)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: embedslice,
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})

	embedslice = nil

	sort.Strings(keys)
	var buffer bytes.Buffer
	k := 0
	for j := range keys {
		buffer.WriteString(x[keys[j]][0].GameName)
		buffer.WriteString(" (")
		buffer.WriteString(getGameServiceString(x[keys[j]][0].Serial))
		buffer.WriteString(")")
		buffer.WriteString(": ")
		buffer.WriteString(strconv.Itoa(len(x[keys[j]])))
		buffer.WriteString(" keys\n")
		k++

		if k == 20 {
			embedslice = append(embedslice, NewEmbed().AddField("Search Results", buffer.String()).SetColor(embedColor).MessageEmbed)
			s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Embeds: embedslice,
				Flags:  discordgo.MessageFlagsEphemeral,
			})
			buffer.Reset()
			k = 0
			embedslice = nil
		}
	}

	if k != 0 {
		embedslice = append(embedslice, NewEmbed().AddField("Search Results", buffer.String()).SetColor(embedColor).MessageEmbed)
		buffer.Reset()

		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: embedslice,
			Flags:  discordgo.MessageFlagsEphemeral,
		})
	}

}

func take(s *discordgo.Session, i *discordgo.InteractionCreate) {
	//var embedslice = make([]*discordgo.MessageEmbed, 2)
	var embedslice []*discordgo.MessageEmbed
	options := i.ApplicationCommandData().Options

	normalized := NormalizeGame(options[0].StringValue())
	fileLock.Lock()
	defer fileLock.Unlock()
	Load(config.DbFile, &x)

	//We need to pop off a game, if it exists
	if len(x[normalized]) > 0 {
		var userkey GameKey
		userkey, x[normalized] = x[normalized][0], x[normalized][1:]
		embedslice = append(embedslice, NewEmbed().AddField("Here Is your key", userkey.GameName+" ("+userkey.ServiceType+")"+": "+userkey.Serial+"\nThis game was brought to you by "+userkey.Author).SetColor(embedColor).MessageEmbed)
		if userkey.ServiceType == "Steam" {
			embedslice = append(embedslice, NewEmbed().AddField("Steam Redeem Link", "https://store.steampowered.com/account/registerkey?key="+userkey.Serial).SetColor(embedColor).MessageEmbed)
		}

		if userkey.ServiceType == "GOG" {
			embedslice = append(embedslice, NewEmbed().AddField("GOG Redeem Link", "https://www.gog.com/redeem/"+userkey.Serial).SetColor(embedColor).MessageEmbed)

		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: embedslice,
				Flags:  discordgo.MessageFlagsEphemeral,
			},
		})
		//Announce to channel
		var username string

		if i.User != nil {
			username = i.User.Username
		} else {
			username = i.Member.User.Username
		}
		embedslice = nil
		embedslice = append(embedslice, NewEmbed().AddField("Another Satisfied Customer", username+" has just taken a key for "+userkey.GameName+". There are "+strconv.Itoa(len(x[normalized]))+" keys remaining").SetColor(embedColor).MessageEmbed)
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: embedslice,
		})
		//If no more keys, remove entry in map
		if len(x[normalized]) == 0 {
			delete(x, normalized)
		}

		Save(config.DbFile, &x)

	} else {
		embedslice = append(embedslice, NewEmbed().AddField("WHY U DO DIS?", options[0].StringValue()+" doesn't exist you cheeky bastard!").SetColor(embedColor).MessageEmbed)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: embedslice,
			},
		})
	}
}

// This function will be called (due to AddHandler above) when the bot receives
// the "ready" event from Discord.
func ready(s *discordgo.Session, event *discordgo.Ready) {

	// Discord just loves to send ready events during server hiccups
	// This prevents spamming
	if !initialized {
		// Set the playing status.

		if len(event.Guilds) <= 0 {
			fmt.Println("Error: No servers returned from discord. Make sure to invite your bot to your server first")
			fmt.Println("Error: You can do that with https://discord.com/oauth2/authorize?client_id=123451234512345&scope=bot")
			fmt.Println("Error: Replace the number with your bot's clientid value from your developer portal")
			os.Exit(4)
		}

		initialized = true
	}
}

// ListKeys lists what games and how many keys for each
func ListKeys() (string, int) {

	fileLock.RLock()
	defer fileLock.RUnlock()
	Load(config.DbFile, &x)

	if len(x) == 0 {
		//	SendEmbed(s, m.ChannelID, "", "EMPTY DATABASE", "No Keys present in Database")
		return "", 0
	}

	// Lets make this pretty, sort keys by name
	keys := make([]string, 0, len(x))
	for key := range x {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Build the output message
	var buffer bytes.Buffer

	i := 0
	for i = range keys {
		buffer.WriteString(x[keys[i]][0].GameName)
		buffer.WriteString(" (")
		buffer.WriteString(getGameServiceString(x[keys[i]][0].Serial))
		buffer.WriteString(")")
		buffer.WriteString(" : ")
		buffer.WriteString(strconv.Itoa(len(x[keys[i]])))
		buffer.WriteString(" keys\n")
	}
	return buffer.String(), i
}

// AddGame will add a new key to the db
// It will also check to see if the key was put in the
// broadcast chan, remove if necessary
func AddGame(name string, inkey string, user string) int {

	// Strip the cmd, split off key from regex and grab name
	gamename := strings.TrimSpace(name)
	normalized := NormalizeGame(gamename)
	key := strings.TrimSpace(inkey)
	var thiskey GameKey
	thiskey.Author = user
	thiskey.GameName = gamename
	thiskey.Serial = key
	thiskey.ServiceType = getGameServiceString(thiskey.Serial)

	fileLock.Lock()
	defer fileLock.Unlock()

	Load(config.DbFile, &x)

	//Check if key already exists
	for i := range x[normalized] {
		if thiskey.Serial == x[normalized][i].Serial {
			return 0
		}
	}

	//Add new key and notify user and channel, then save to disk
	x[normalized] = append(x[normalized], thiskey)

	Save(config.DbFile, &x)
	return len(x[normalized])
}
