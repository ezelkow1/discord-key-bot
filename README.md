![Go](https://github.com/ezelkow1/discord-key-bot/workflows/Go/badge.svg?branch=master)

# discord-key-bot
A bot for discord that accepts, announces, and gives out keys

This requires a conf.json file (or whatever you choose to name the config file). An editable example is provided. It requires your bot token, a channel (using the channel ID number) to use for broadcasting messages, the name of the json/db file for key storage, and a KeysRole name if you wish to limit bot communication to a specific role.

The bot will need 'Manage Messages' permission in the channel, as the bot will erase any `!add` commands sent in the channel so that keys do not appear publicly

The bot will take any `!add`-ed keys, normalize the name by stripping whitespaces and lowercasing, and that becomes the key to a map. Within each key in the map are individual gamekeys which record the original "pretty" version of the game name, the user who donated, the gamekey, and the service for redeeming the key.  Currently, the bot can recognize Steam, Uplay, Origin, PS3, GOG, and URLs. Any other key will be stored as an 'unknown' type.  If a key is Steam or GOG, it will also generate a redemption link on a key `!take`.

Finally, the bot supports searching with `!search`, comparing a search substring to any key names, essentially a \*(stripped tolower string)\*

With the addition of roles' security, multi-server usage is _unavailable_. If the bot is used across multiple servers/guilds, the role management will be unavailable, and should set the field to the default of "" to disable it.

The commands are (from the `!help` command):
```
!add game name key - this will add a new key to the database. This should be done in a DM with the bot
!listkeys - PLEASE USE THIS IN A PRIVATE MESSAGE WITH THE BOT. Lists current games and the number of available keys
!take game name - Will give you one of the keys for the game in a DM
!mygames - Will give a list of games you have taken (this is only active when you are using a user database explained below)
!search search-string - Will search the database for matching games
!totals - Will give a total count of games and the total number of keys stored
```

The latest release can be found here: https://github.com/ezelkow1/discord-key-bot/releases/latest




# Addendum
There are a couple of new options that dont really have default values but you can use them by setting the values in the config file:

UserFile - this is a string just like the database file. If this is set the bot will start keeping track of users and what games they take. It will prevent users from taking more than one key from any single game. This can be handy if you are using the bot for some sort of mass key distribution setup.

When enabled this also adds a new bot command, !mygames, which a user can do to receive a list of all the games they have taken from the bot

ListPMOnly - this is a boolean (set using true or false without quotations). The default is already false so you only need to use this if you want to set it to true. Setting true will prevent the bot from doing a full key list inside the broadcast channel. This can be really handy for when you start to get past ~100 keys or so, so people dont inadvertently (or on purpose) end up having the bot spam your channel with a key listing
