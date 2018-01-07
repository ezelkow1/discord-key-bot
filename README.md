# discord-key-bot
A bot for discord that accepts, announces, and gives out keys

This requires a conf.json file (or whatever you choose to name the config file). An editable example is provided. It will need your bot token, a channel (using the channel ID number) to use for broadcast messages, and the name of the json/db file for key storage

In the channel itself the bot will need 'Manage Messages' permission, as the bot will erase any '!add' commands sent in in the channel so that keys do not appear publicly

As far as structure the bot will take any !add-ed keys, normalize the name by stripping whitespace and lowercasing, and that becomes the key to a map. Within each key in the map are individual gamekeys which record the original "pretty" version of the game name, the user who donated, the gamekey, and the service to redeem the key on.  Currently the bot can recognize steam, uplay, origin, ps3, gog, and urls. Any other key will be stored as an 'unknown' type.  If  a key is steam or gog it will also generate a redemption link on a key !take.

Finally the bot supports searching with !search, just comparing a search substring to any key names, so basically a \*(stripped tolower string)\*
