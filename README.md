# Discord Arbitary File Storage
## Inspiration
I was inspired by [this](https://www.youtube.com/watch?v=c_arQ-6ElYI) video. It used node.js with a React frontend. I saw several improvements that could be made, so I decided to make my own version in Golang.

## Features
- `send <fname>` - Send a file to Discord by filename
- `fetch <reference>` - Get a file from Discord using the message ID printed in console and the manifest channel
- `init` - Refresh channel ids. Done automatically on startup.
- Tab completion and left+right arrow key movement - From scratch.
- Minimal dependencies - Only requires my [cfg package](https://github.com/0mlml/cfgparser) (which has zero dependencies), the [golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto) package, and the standard library.
## How it works
#### Chunking
The file is split into chunks of roughly 25MB. The actual size is a bit lower to account for encryption overhead. 
#### Encryption
A salt is generated for each file based on the key provided in the config. The salt is used to generate a key and IV for AES-256-GCM. The IV is then prepended to the encrypted data.

Why not encrypt and then chunk? - Yeah that's probably easier
#### Uploading
Each encrypted chunk is uploaded to a Discord channel. The first message is sent with the filename and salt. The rest of the messages are sent as replies to the first message. This allows you to easily walk backwards through the messages to get the file.

![Data](https://github.com/0mlml/discord-fs/blob/main/.github/fs-data-ss.png)

A copy of the filename-salt pair is sent to a manifest channel alongside the last message ID for convenience.

![Manifest](https://github.com/0mlml/discord-fs/blob/main/.github/fs-manifest-ss.png)

Why not a database? - I wanted to keep this as stateless as possible. For now you just need to copy over the config file with the key (password) to be able to retrieve the file.
#### Assembly 
The chunks are downloaded, reversed, and decrypted. The decrypted chunks are then written to a file.

![Demo](https://github.com/0mlml/discord-fs/blob/main/.github/demo.gif)

## Limitations
- Each chunk is limited to 25MB. This is a limitation of Discord's API.
- Uploads can only be performed one at a time. This is because we need the message ID of the previous message to chain the chunks.
- Downloads are only performed one at a time. This could hypothetically be fixed, where the chain is walked first and then the chunks are downloaded in parallel.
- The filename is not concealed in any way. I didn't think this was necessary, but it could be added in the future.
- There's no way to catalogue your files. The manifest channel is the best way to keep track of message IDs. 
## Useful scripts
Bash script to generate a large file to test with:
```bash
#!/bin/bash
size="80M"
dd if=/dev/urandom of="$size"file.bin bs="$size" count=1
```
Bash script to decode a meta into the filename:
```bash
#!/bin/bash
meta="Li9oZWxsb3dvcmxkLnR4dAAzSEcxSDJQUmhvcz0="
echo "$meta" | base64 -d | awk 'BEGIN{RS="\0"} NR==1'
```
## Config
The config file is located at `config.json`. It contains the following fields:
- `discord_token` - The token of the bot, generated from the [Discord Developer Portal](https://discord.com/developers/applications)
- `server_id` - The ID of the server that the bot will be running on
- `your_key` - The key used to encrypt the file. This should be a long, random string. 
- `max_file_size` - The maximum file size in bytes. This should be less than 25MB, with a bit of wiggle room.
- `advanced_terminal` - Try to allow advanced features like moving the cursor and tab completion. This might not work on all terminals.