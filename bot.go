package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"

	tl "cloud.google.com/go/translate"
	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
	"golang.org/x/text/language"
	"google.golang.org/api/option"
)

func main() {
	error := godotenv.Load()
	if error != nil {
		log.Panic("Error loading .env file")
	}

	telegramToken := os.Getenv("TELEGRAM_TOKEN")
	translateAPIKey := os.Getenv("TRANSLATE_API_KEY")
	authedChat, error := strconv.ParseInt(os.Getenv("AUTHED_CHAT"), 10, 64)
	if error != nil {
		log.Panic("AUTHED_CHAT must be a 64-bit integer")
	}

	// Setup Telegram Bot
	bot, error := tg.NewBotAPI(telegramToken)
	if error != nil {
		log.Panic(error)
	}

	botUsername := bot.Self.UserName
	log.Printf("Using Account: %s", botUsername)

	updateConfig := tg.NewUpdate(0)
	updateConfig.Timeout = 60
	updates, error := bot.GetUpdatesChan(updateConfig)
	if error != nil {
		log.Panic(error)
	}

	// Setup translate
	ctx := context.Background()
	client, error := tl.NewClient(ctx, option.WithAPIKey(translateAPIKey))
	if error != nil {
		log.Panic(error)
	}

	defer client.Close()

	for update := range updates {
		message := update.Message

		// Skip if message is not a command
		if message == nil || !message.IsCommand() {
			continue
		}

		// Gatekeeping
		if message.Chat.ID != authedChat {
			log.Printf(
				"Skipping message from user: %s [%d] in chat: %s [%d]\n",
				message.From.UserName,
				message.From.ID,
				message.Chat.Title,
				message.Chat.ID,
			)

			continue
		}

		command, taggedUsername := commandAndUsername(message)

		switch {
		case command == "translate" || command == "tl":
			if quotedMessage := message.ReplyToMessage; quotedMessage != nil {
				// Translate the quoted message
				targetLang := message.CommandArguments()
				response, error := translate(client, ctx, targetLang, quotedMessage.Text)
				if error != nil {
					log.Printf("Error Translating: %s\n", error.Error())
				} else {
					reply := newReply(message.Chat.ID, quotedMessage.MessageID, response)
					bot.Send(reply)
				}
			} else {
				// Translate the command argument
				code, text := parseLangCode(message.CommandArguments())
				if len(text) > 2 {
					response, error := translate(client, ctx, code, text)
					if error != nil {
						log.Printf("Error Translating: %s\n", error.Error())
					} else {
						reply := newReply(message.Chat.ID, message.MessageID, response)
						bot.Send(reply)
					}
				}
			}
		case command == "start" && taggedUsername == botUsername:
			reply := tg.NewMessage(message.Chat.ID, "¡Soy la tranductora! 👩‍🏫")
			bot.Send(reply)
		case command == "ping" && taggedUsername == botUsername:
			reply := tg.NewMessage(message.Chat.ID, "Estoy corriendo. 🏃‍♀")
			bot.Send(reply)
		default:
			// Ignore unknown commands
		}
	}
}

func translate(
	client *tl.Client,
	ctx context.Context,
	targetLang string,
	message string,
) (string, error) {
	if targetLang == "" {
		targetLang = "en"
	}

	lang, error := language.Parse(targetLang)
	if error != nil {
		return "", error
	}

	response, error := client.Translate(ctx, []string{message}, lang, &tl.Options{Format: tl.Text})
	if error != nil {
		return "", error
	}

	return response[0].Text, nil
}

func newReply(chatID int64, messageID int, text string) tg.MessageConfig {
	return tg.MessageConfig{
		BaseChat: tg.BaseChat{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
		},
		Text:                  text,
		DisableWebPagePreview: true,
	}
}

func parseLangCode(rawText string) (string, string) {
	prefix := "-"
	if len(rawText) <= len(prefix)+2 || !strings.HasPrefix(rawText, prefix) {
		return "", rawText
	}

	firstSpaceIndex := strings.IndexRune(rawText, ' ')
	if firstSpaceIndex == -1 {
		return "", rawText
	}

	return rawText[len(prefix):firstSpaceIndex], rawText[firstSpaceIndex:]
}

func commandAndUsername(message *tg.Message) (string, string) {
	command := message.CommandWithAt()
	splitCommand := strings.Split(command, "@")

	if len(splitCommand) > 1 {
		return splitCommand[0], splitCommand[1]
	} else {
		return command, ""
	}
}
