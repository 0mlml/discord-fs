# Discord Arbitary File Storage
## Inspiration
I was inspired by [this](https://www.youtube.com/watch?v=c_arQ-6ElYI) video. It used node.js with a React frontend. I saw several improvements that could be made, so I decided to make my own version in Golang.
## How it works
### Important - The bot will create lots of channels. Do not use this in a server where this is a problem.

## Config
The config file is located at `config.json`. It contains the following fields:
- `discord_token` - The token of the bot, generated from the [Discord Developer Portal](https://discord.com/developers/applications)
- `server_id` - The ID of the server that the bot will be running on