package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
)

const (
	apiBase = "https://discord.com/api/v10"
)

var (
	token     string
	idHistory = make([]string, 0) // autocomplete
)

func requestHeaders() *http.Header {
	return &http.Header{
		"Authorization": []string{fmt.Sprintf("Bot %s", token)},
		"User-Agent":    []string{"DiscordBot (0mlml/discord-fs)"},
		"Content-Type":  []string{"application/json; charset=utf-8"},
	}
}

func setToken(t string) bool {
	if t == "" {
		logger.Printf("Token is empty\n")
		return false
	}

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/users/@me", apiBase),
		nil,
	)

	if err != nil {
		logger.Printf("Error setting token: %v\n", err)
		return false
	}

	req.Header = *requestHeaders()
	req.Header.Set("Authorization", fmt.Sprintf("Bot %s", t))

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		logger.Printf("Error setting token: %v\n", err)
		return false
	}

	if resp.StatusCode >= 300 {
		logger.Printf("Error setting token: %v\n", resp.Status)
		return false
	}

	token = t

	return true
}

var (
	manfiestChannelID string
	dataChannels      []string
)

func getChannels() error {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/guilds/%s/channels", apiBase, config.String("server_id")),
		nil,
	)

	req.Header = *requestHeaders()

	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return err
	}

	if resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status getting channels: %v", resp.Status)
	}

	type discordChannel struct {
		ID    string `json:"id"`
		Topic string `json:"topic"`
		Type  int    `json:"type"`
	}

	var channels []discordChannel

	if err := json.NewDecoder(resp.Body).Decode(&channels); err != nil {
		return err
	}

	for _, channel := range channels {
		if channel.Type != 0 {
			continue
		}
		if channel.Topic == "discord-fs-manifest" {
			manfiestChannelID = channel.ID
		} else if channel.Topic == "discord-fs-data" {
			dataChannels = append(dataChannels, channel.ID)
		}
	}

	return nil
}

type discordAttachment struct {
	Filename string `json:"filename"`
	Size     int    `json:"size"`
	URL      string `json:"url"`
}

type discordMessageReference struct {
	MessageID string `json:"message_id"`
}

type discordMessage struct {
	ID          string                  `json:"id"`
	Content     string                  `json:"content"`
	Attachments []discordAttachment     `json:"attachments"`
	Reference   discordMessageReference `json:"message_reference"`
}

func getMessage(channelID string, messageID string) (message *discordMessage, err error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/channels/%s/messages/%s", apiBase, channelID, messageID),
		nil,
	)

	req.Header = *requestHeaders()

	if err != nil {
		return message, err
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return message, err
	}

	if resp.StatusCode >= 300 {
		return message, fmt.Errorf("unexpected status getting message: %v", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(&message); err != nil {
		return message, err
	}

	return message, nil
}

func createChannel(name string, topic string, t int) (ch *map[string]interface{}, err error) {
	payload := make(map[string]interface{})
	payload["name"] = name
	payload["topic"] = topic
	payload["type"] = t

	payloadJSON, err := json.Marshal(payload)

	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/guilds/%s/channels", apiBase, config.String("server_id")),
		bytes.NewBuffer(payloadJSON),
	)

	req.Header = *requestHeaders()

	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode > 300 {
		return nil, fmt.Errorf("undexpected response creating channel: %v", resp.Status)
	}

	return nil, nil
}

func intialize() {
	if err := getChannels(); err != nil {
		logger.Printf("Error getting channels: %v", err)
		return
	}

	if manfiestChannelID == "" {
		logger.Printf("Manifest channel not found, creating...\n")
		if _, err := createChannel("discord-fs-manifest", "discord-fs-manifest", 0); err != nil {
			panic(fmt.Sprintf("Error creating manifest channel: %v", err))
		}

		if err := getChannels(); err != nil {
			logger.Printf("Error getting channels: %v", err)
			return
		}
	}

	if len(dataChannels) == 0 {
		logger.Printf("Data channel not found, creating...\n")
		if _, err := createChannel("discord-fs-data", "discord-fs-data", 0); err != nil {
			panic(fmt.Sprintf("Error creating data channel: %v", err))
		}

		if err := getChannels(); err != nil {
			logger.Printf("Error getting channels: %v", err)
			return
		}
	}

	logger.Printf("Found manifest channel: %s\nFound %d data channels\n", manfiestChannelID, len(dataChannels))
}

type messageCreate struct {
	ChannelID   string
	ReferenceID string
	Content     string
	Data        []byte
	FileName    string
}

func sendDiscordAttachment(message messageCreate) (string, error) {
	var requestBody bytes.Buffer
	multipartWriter := multipart.NewWriter(&requestBody)

	fileWriter, err := multipartWriter.CreateFormFile("file", message.FileName)
	if err != nil {
		return "", err
	}
	fileWriter.Write(message.Data)

	payload := make(map[string]interface{})
	if message.Content != "" {
		payload["content"] = message.Content
	}
	if message.ReferenceID != "" {
		payload["message_reference"] = map[string]string{"message_id": message.ReferenceID}
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	_ = multipartWriter.WriteField("payload_json", string(payloadJSON))

	err = multipartWriter.Close()
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/channels/%s/messages", apiBase, message.ChannelID),
		&requestBody,
	)

	if err != nil {
		return "", err
	}

	req.Header = *requestHeaders()
	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("discord API returned error status code: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	messageID, ok := result["id"].(string)
	if !ok {
		return "", fmt.Errorf("invalid response format")
	}

	return messageID, nil
}

func sendChunkedFile(f *chunkedFile) (err error) {
	metaString := generateMeta(f.name, f.salt)

	dataSize := 0
	for _, chunk := range f.data {
		dataSize += len(chunk)
	}

	lastMessageID := ""
	attempt := 1
	for n := 0; n < len(f.data); n++ {
		message := messageCreate{
			ChannelID:   dataChannels[n%len(dataChannels)],
			ReferenceID: lastMessageID,
			Data:        f.data[n],
			FileName:    fmt.Sprintf("%d.enc", n),
		}

		if n == 0 {
			message.Content = metaString
		}

		logger.AddLine(
			fmt.Sprintf("send_%s", f.name),
			fmt.Sprintf("%s: sending attachment %d (size %d); %s", f.name, n+1, dataSize, ProgressBarUtil(n+1, len(f.data))),
		)

		lastMessageID, err = sendDiscordAttachment(message)

		if err != nil {
			logger.Printf("Error sending chunk %d: %v. Attempt %d/%d\n", n, err, attempt, config.Int("max_retry"))

			if attempt >= config.Int("max_retry") {
				logger.Printf("Aborting send of %s\n", f.name)
				return err
			}

			attempt++
			n--
		}
	}

	logger.RemoveLine(fmt.Sprintf("send_%s", f.name))
	logger.Printf("%s: sending attachment %d (size %d); %s\n", f.name, len(f.data), dataSize, ProgressBarUtil(1, 1))

	var payload = map[string]interface{}{
		"content": fmt.Sprintf("%s\n%s", metaString, lastMessageID),
	}

	payloadJSON, err := json.Marshal(payload)

	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/channels/%s/messages", apiBase, manfiestChannelID),
		bytes.NewBuffer(payloadJSON),
	)

	if err != nil {
		return err
	}

	req.Header = *requestHeaders()

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return err
	}

	if resp.StatusCode >= 300 {
		return fmt.Errorf("Error sending manifest: %v", resp.Status)
	}

	logger.Printf("Sent file %s, reference %s\n", f.name, lastMessageID)

	idHistory = append(idHistory, lastMessageID)

	return nil
}

func downloadChunk(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download chunk: status code %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func fetchChunkedFile(chainEndId string) (cf *chunkedFile, err error) {
	cf = &chunkedFile{}

	lastMessage, err := getMessage(dataChannels[0], chainEndId)
	if err != nil {
		return nil, err
	}

	chunkNumberStr := strings.Split(lastMessage.Attachments[0].Filename, ".")[0]
	chunkNumber, err := strconv.Atoi(chunkNumberStr)

	if err != nil {
		logger.Printf("Error parsing chunk number: %v\n", err)
		chunkNumber = -1
	}

	chunkNumber++

	chunkCount := chunkNumber

	logger.Printf("Starting download for reference %s, inferring %d chunks\n", chainEndId, chunkNumber)
	logger.AddLine(
		fmt.Sprintf("download_%s", chainEndId),
		fmt.Sprintf("%s: downloading attachment %d (size %d); %s", chainEndId, chunkNumber, lastMessage.Attachments[0].Size, ProgressBarUtil(chunkCount-chunkNumber, chunkCount)),
	)

	logger.Flush()

	data, err := downloadChunk(lastMessage.Attachments[0].URL)

	if err != nil {
		return nil, err
	}

	cf.data = append(cf.data, data)

	metaString := ""
	for {
		metaString = lastMessage.Content

		if lastMessage.Reference.MessageID == "" {
			break
		}

		lastMessage, err = getMessage(dataChannels[0], lastMessage.Reference.MessageID)
		if err != nil {
			return nil, err
		}

		chunkNumberStr = strings.Split(lastMessage.Attachments[0].Filename, ".")[0]

		chunkNumber, err := strconv.Atoi(chunkNumberStr)

		if err != nil {
			logger.Printf("Error parsing chunk number: %v\n", err)
			chunkNumber = -1
		}

		chunkNumber++

		logger.AddLine(
			fmt.Sprintf("download_%s", chainEndId),
			fmt.Sprintf("%s: downloading attachment %d (size %d); %s", chainEndId, chunkNumber, lastMessage.Attachments[0].Size, ProgressBarUtil(chunkCount-chunkNumber, chunkCount)),
		)

		logger.Flush()

		data, err := downloadChunk(lastMessage.Attachments[0].URL)

		if err != nil {
			return nil, err
		}

		cf.data = append(cf.data, data)
	}

	for i := len(cf.data)/2 - 1; i >= 0; i-- {
		opp := len(cf.data) - 1 - i
		cf.data[i], cf.data[opp] = cf.data[opp], cf.data[i]
	}

	cf.name, cf.salt = parseMeta(metaString)

	logger.Printf("%s: downloading attachment %d (size %d); %s", chainEndId, chunkNumber, lastMessage.Attachments[0].Size, ProgressBarUtil(1, 1))
	logger.RemoveLine(fmt.Sprintf("download_%s", chainEndId))
	logger.Printf("Fetched file %s out of %d chunks\n", cf.name, len(cf.data))

	return cf, nil
}
