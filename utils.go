package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
)

//isGogMatch
func isGogMatch(key string) bool {
	return gog.MatchString(key)
}

//isSteamMatch
func isSteamMatch(key string) bool {
	if steamOne.MatchString(key) || steamTwo.MatchString(key) {
		return true
	}

	return false
}

//isPs3Match
func isPs3Match(key string) bool {
	return ps3.MatchString(key)
}

//isUplayMatch
func isUplayMatch(key string) bool {
	if uplayOne.MatchString(key) || uplayTwo.MatchString(key) {
		return true
	}

	return false
}

//isOriginMatch
func isOriginMatch(key string) bool {
	return origin.MatchString(key)
}

//isURLMatch
func isURLMatch(key string) bool {
	return url.MatchString(key)
}

//getGameServiceString
func getGameServiceString(key string) string {

	if isGogMatch(key) {
		return "GOG"
	} else if isSteamMatch(key) {
		return "Steam"
	} else if isPs3Match(key) {
		return "PS3"
	} else if isUplayMatch(key) {
		return "Uplay"
	} else if isOriginMatch(key) {
		return "Origin"
	} else if isURLMatch(key) {
		return "Gift Link"
	}

	return "Unknown"
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

// This function refreshes the roleID
func refreshRoles(s *discordgo.Session) {

	if config.KeyRole != "" {
		roles, err := s.GuildRoles(guildID)

		if err == nil {
			for roleids := range roles {
				if roles[roleids].Name == config.KeyRole {
					roleID = roles[roleids].ID
				}
			}
		}
	}
}

// This will determine if a user is a member of the
// proper keys role or not
func isUserRoleAllowed(s *discordgo.Session, m *discordgo.MessageCreate) bool {

	if config.KeyRole != "" {
		refreshRoles(s)
		member, _ := s.GuildMember(guildID, m.Author.ID)
		if len(member.Roles) > 0 {
			for memberrole := range member.Roles {
				if member.Roles[memberrole] == roleID {
					//We found our keys roleID in the members list
					return true
				}
			}
		}
	} else {
		// If no keyrole is set then just let everyone through
		return true
	}

	return false
}

// Check if a msg has a prefix we care about. This is for
// optimization so we can skip any messages we dont care about.
// If adding new message triggers they must be added here
func checkPrefix(msg string) bool {

	if (msg == "!listkeys") ||
		(strings.HasPrefix(msg, "!add ") == true) ||
		(strings.HasPrefix(msg, "!take ") == true) ||
		(strings.HasPrefix(msg, "!search ") == true) ||
		(strings.HasPrefix(msg, "!help") == true) ||
		(strings.HasPrefix(msg, "!speak") == true) ||
		(msg == "!totals") ||
		(msg == "!mygames") {
		return true
	}

	return false
}
