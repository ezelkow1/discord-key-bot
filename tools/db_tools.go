package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
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
	x               = make(map[string][]GameKey)
	author          string
	oldprintout     bool
	olddeleteauth   bool
	printout        bool
	deleteauth      bool
	convert         bool
	dbFile          string
	converteddbFile string
	//database        *sql.DB
)

func init() {
	flag.BoolVar(&oldprintout, "op", false, "print keys for author (Old style DB)")
	flag.BoolVar(&olddeleteauth, "od", false, "delete keys from author (Old style DB)")
	flag.StringVar(&author, "author", "", "-author=authorname")
	flag.BoolVar(&convert, "c", false, "Convert an old style database to the newer sql format")
	flag.StringVar(&dbFile, "olddb", "keys.db", "Optional name of old db (need to set if not default). -olddb=originalDBfile.db")
	flag.StringVar(&converteddbFile, "newdb", "keys-sqlite.db", "Optional name of the new sql formatted database output. -newdb=keys-sqlite.db")
	flag.BoolVar(&printout, "p", false, "print keys for author (New style DB)")
	flag.BoolVar(&deleteauth, "d", false, "delete keys from author (New style DB)")
	flag.Parse()

}

func main() {

	if author != "" {
		if oldprintout {
			// print out authors keys
			oldprintKeys()
		}

		if olddeleteauth {
			// delete authors keys
			olddeleteAuth()
		}

		if printout {
			printKeys()
		}

		if deleteauth {
			deleteAuth()
		}
	}

	if convert {
		convertDB()
	}
}

func printKeys() {
	database, err := sql.Open("sqlite3", converteddbFile)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer database.Close()

	statement, err := database.Query("SELECT key, users.name as user_name, games.pretty as game FROM keys INNER JOIN users on users.id = keys.userid_entered INNER JOIN games on games.id = keys.game_id WHERE user_name = (?)", author)

	if err != nil {
		log.Fatal(err)
		return
	}
	for statement.Next() {
		var key string
		var author string
		var game string
		statement.Scan(&key, &author, &game)
		println("\nGame: " + game + ", key: " + key + ", Author: " + author)
	}
	statement.Close()
}

func deleteAuth() {
	database, err := sql.Open("sqlite3", converteddbFile)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer database.Close()
	var userid int
	deleterow, err := database.Prepare("DELETE from keys where userid_entered = (?)")
	defer deleterow.Close()
	statement, err := database.Query("SELECT id from users where name = (?)", author)
	if err != nil {
		log.Fatal(err)
		return
	}
	for statement.Next() {
		statement.Scan(&userid)
	}
	println("Deleting all keys from " + author + ":" + strconv.Itoa(userid))
	deleterow.Exec(userid)
}

func insertGames() {
	Load(dbFile, &x)
	if len(x) <= 0 {
		println("db: " + dbFile + " is empty")
		return
	}
	database, err :=
		sql.Open("sqlite3", converteddbFile)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer database.Close()
	statement, err := database.Prepare("INSERT INTO games (normalized, pretty) VALUES (?,?)")
	if err != nil {
		panic(err)
	}

	var count int
	for key := range x {
		for game := range x[key] {
			_, err = statement.Exec(key, x[key][game].GameName)
			if err == nil {
				count++
			}
			continue
		}
	}

	println("Inserted " + strconv.Itoa(count) + " distinct games in to the database")
	statement.Close()
}

func insertUsers() {
	Load(dbFile, &x)
	if len(x) <= 0 {
		println("db: " + dbFile + " is empty")
		return
	}
	database, err :=
		sql.Open("sqlite3", converteddbFile)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer database.Close()
	statement, err := database.Prepare("INSERT INTO users (name) VALUES (?)")
	if err != nil {
		panic(err)
	}

	var count int
	for key := range x {
		for game := range x[key] {
			_, err = statement.Exec(x[key][game].Author)

			if err == nil {
				count++
			}
		}
	}
	statement.Close()
	println("Inserted " + strconv.Itoa(count) + " distinct users in to the database (this will include nickname changes users may have made while entering keys)")
}

func insertKeys() {
	Load(dbFile, &x)
	if len(x) <= 0 {
		println("db: " + dbFile + " is empty")
		return
	}
	database, err :=
		sql.Open("sqlite3", converteddbFile)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer database.Close()
	insert, err := database.Prepare("INSERT INTO keys (key, game_id, userid_entered, service) VALUES (?,?,?,?)")
	defer insert.Close()
	if err != nil {
		panic(err)
	}

	var count int
	var foundErr bool
	for key := range x {
		for game := range x[key] {
			findgame, err := database.Query("SELECT id from games where normalized=(?)", NormalizeGame(x[key][game].GameName))
			var gameid int
			var userid int
			for findgame.Next() {
				err = findgame.Scan(&gameid)
				if err != nil {
					panic(err)
				}
			}
			findgame.Close()
			finduser, err := database.Query("SELECT id from users where name=(?)", x[key][game].Author)
			for finduser.Next() {
				err = finduser.Scan(&userid)
				if err != nil {
					panic(err)
				}
			}
			finduser.Close()
			_, err = insert.Exec(x[key][game].Serial, gameid, userid, x[key][game].ServiceType)
			if err != nil {
				log.Print(err)
				println("   For Game: " + x[key][game].GameName + " , Serial: " + x[key][game].Serial + ", Author: " + x[key][game].Author)
				foundErr = true
				dupe, err := database.Query("SELECT key, users.name as user_name, games.pretty as game FROM keys INNER JOIN users on users.id = keys.userid_entered INNER JOIN games on games.id = keys.game_id WHERE key = (?)", x[key][game].Serial)
				if err != nil {
					log.Print(err)
				}
				for dupe.Next() {
					var key string
					var gameid string
					var userid string
					dupe.Scan(&key, &userid, &gameid)
					println("   Duplicate Game in DB: " + gameid + ", key: " + key + ", Author: " + userid)
				}
				dupe.Close()
			} else {
				count++
			}
		}
	}

	if foundErr {
		println("\n\nIf this is a unique failure this game key already matches an existing one so you have a bad entry somewhere.\nThis is only a notification, the duplicate game is already in the database and this duplicate will not be entered\n\n")
	}
	println("Inserted " + strconv.Itoa(count) + " game keys in to the database")
}

func convertDB() {

	Load(dbFile, &x)
	if len(x) <= 0 {
		println("db: " + dbFile + " is empty")
		return
	}

	database, err :=
		sql.Open("sqlite3", converteddbFile)
	if err != nil {
		log.Fatal(err)
		return
	}

	defer database.Close()
	statement, err := database.Prepare("CREATE TABLE IF NOT EXISTS games (id INTEGER PRIMARY KEY, normalized TEXT NOT NULL, pretty TEXT NOT NULL, UNIQUE(normalized, pretty))")
	_, err = statement.Exec()
	if err != nil {
		log.Fatal(err)
		return
	}

	statement, _ = database.Prepare("CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT NOT NULL UNIQUE, discordid INTEGER UNIQUE)")
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
	statement.Close()

	println("New database " + converteddbFile + " initialized, converting " + dbFile)

	insertGames()
	insertUsers()
	insertKeys()

}

func oldprintKeys() {

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

func olddeleteAuth() {
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

//NormalizeGame the name of the game, removes spaces, lowercases
func NormalizeGame(name string) string {
	tmp := strings.ToLower(name)
	tmp = strings.Replace(tmp, " ", "", -1)
	return tmp
}
