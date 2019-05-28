package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
)

//GameKey struct
type GameKey struct {
	Author      string
	GameName    string
	Serial      string
	ServiceType string
}

var (
	// Game key Database
	x          = make(map[string][]GameKey)
	author     string
	printout   bool
	deleteauth bool
	dbFile     = "keys.db"
)

func init() {

	flag.BoolVar(&printout, "p", false, "print keys for author")
	flag.BoolVar(&deleteauth, "d", false, "delete keys from author")
	flag.StringVar(&author, "author", "", "-author=authorname")
	flag.Parse()

}

func main() {

	if author != "" {
		if printout {
			// print out authors keys
			printKeys()
		}

		if deleteauth {
			// delete authors keys
			deleteAuth()
		}
	}
}

func printKeys() {

	Load(dbFile, &x)
	if len(x) <= 0 {
		return
	}

	for key := range x {
		//println(key + " r: " + strconv.Itoa(len(x[key])))
		for game := range x[key] {
			if x[key][game].Author == author {
				println("Game: " + x[key][game].GameName)
				println("Key: " + x[key][game].Serial)
				println("\n")
			}
		}
	}
}

func deleteAuth() {
	println("In delete auth")
	Load(dbFile, &x)
	if len(x) <= 0 {
		println("len x: " + strconv.Itoa(len(x)))
		return
	}

	println("Len x: " + strconv.Itoa(len(x)))
	for key := range x {
		//println(key + " r: " + strconv.Itoa(len(x[key])))
		var curGame []GameKey
		//curGame := make([]GameKey, len(x[key]))
		println("# Game: " + strconv.Itoa(len(x[key])))
		for game := range x[key] {
			if x[key][game].Author == author {
				println("Saw game from " + author + " :" + x[key][game].GameName)
			} else {
				curGame = append(curGame, x[key][game])
				fmt.Printf("Appending: %v\n", x[key][game])
			}
		}
		if len(curGame) != 0 {
			fmt.Printf("Old Slice: %v\n", x[key])
			x[key] = nil
			x[key] = append(x[key], curGame...)
			fmt.Printf("New Slice: %v\n", x[key])
			if len(x[key]) == 0 {
				delete(x, key)
			}
			curGame = nil
		} else {
			delete(x, key)
		}
	}
	Save(dbFile, &x)
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
