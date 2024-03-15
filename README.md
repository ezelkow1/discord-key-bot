![Go](https://github.com/ezelkow1/discord-key-bot/workflows/Go/badge.svg?branch=master)

# discord-key-bot
A bot for discord that accepts, announces, and gives out keys

This requires a conf.json file (or whatever you choose to name the config file). An editable example is provided. It requires your bot token, the guildID of your server, appID of your bot from discord, and the name of the json/db file for key storage

The commands are:

/add - This will open a popup dialog to enter a new game and it's key

/search - search the game database for anything that matches

/list - print a list of all games in the database

/take - take a key

The bot will take any added keys, normalize the name by stripping whitespaces and lowercasing, and that becomes the key to a map. Within each key in the map are individual gamekeys which record the original "pretty" version of the game name, the user who donated, the gamekey, and the service for redeeming the key.  Currently, the bot can recognize Steam, Uplay, Origin, PS3, GOG, and URLs. Any other key will be stored as an 'unknown' type.  If a key is Steam or GOG, it will also generate a redemption link on a key `take`.

Finally, the bot supports searching with /search string, comparing a search substring to any key names, essentially a \*(stripped tolower string)\*

If you wish to limit user access this can now be done via discord server controls in the integration section. From there you can limit to channels and user roles for any of the commands or the bot itself.

Note: no releases have been generated for this version that uses slash commands, only prebuilt releases exist of the deprecated version

The latest release can be found here: https://github.com/ezelkow1/discord-key-bot/releases/latest
