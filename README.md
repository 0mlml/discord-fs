# Discord Arbitary File Storage
## Inspiration
I was inspired by [this](https://www.youtube.com/watch?v=c_arQ-6ElYI) video. It used node.js with a React frontend. I saw several improvements that could be made, so I decided to make my own version in Golang.

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

![Console](https://github.com/0mlml/discord-fs/blob/main/.github/console-ss.png)
### Test script
Here is a quick bash script to generate a large file to test with:
```bash
#!/bin/bash
size="80M"
dd if=/dev/urandom of="$size"file.bin bs="$size" count=1
``````

## Config
The config file is located at `config.json`. It contains the following fields:
- `discord_token` - The token of the bot, generated from the [Discord Developer Portal](https://discord.com/developers/applications)
- `server_id` - The ID of the server that the bot will be running on
- `your_key` - The key used to encrypt the file. This should be a long, random string. 
- `max_file_size` - The maximum file size in bytes. This should be less than 25MB, with a bit of wiggle room.
- `advanced_terminal` - Try to allow advanced features like moving the cursor, tab completion, history. This might not work on all terminals.