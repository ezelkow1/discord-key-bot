package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// isGogMatch
func isGogMatch(key string) bool {
	return gog.MatchString(key)
}

// isSteamMatch
func isSteamMatch(key string) bool {
	if steamOne.MatchString(key) || steamTwo.MatchString(key) {
		return true
	}

	return false
}

// isPs3Match
func isPs3Match(key string) bool {
	return ps3.MatchString(key)
}

// isUplayMatch
func isUplayMatch(key string) bool {
	if uplayOne.MatchString(key) || uplayTwo.MatchString(key) {
		return true
	}

	return false
}

// isOriginMatch
func isOriginMatch(key string) bool {
	return origin.MatchString(key)
}

// isURLMatch
func isURLMatch(key string) bool {
	return url.MatchString(key)
}

// getGameServiceString
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

// CleanKey cleans up the input name. Strips trailing key from input
func CleanKey(name string, key string) string {
	tmp := strings.TrimSuffix(name, key)
	tmp = strings.TrimSpace(tmp)
	return tmp
}

// NormalizeGame the name of the game, removes spaces, lowercases
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
	fileh, _ := os.Create(path)
	n, _ := fileh.Write(b)
	b = b[:n]
	fileh.Close()
}

// Load json file
func Load(path string, object interface{}) {
	fileh, _ := os.Open(path)
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
}
