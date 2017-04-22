package main

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/chzyer/readline"
	"github.com/fatih/color"
	"github.com/legolord208/stdutil"
)

func commandsSay(session *discordgo.Session, terminal bool, cmd string, args []string, nargs int, w io.Writer) (returnVal string) {
	switch cmd {
	case "tts":
		fallthrough
	case "say":
		if nargs < 1 {
			stdutil.PrintErr("say/tts <stuff>", nil)
			return
		}
		if loc.channel == nil && userType != typeWebhook {
			stdutil.PrintErr(tl("invalid.channel"), nil)
			return
		}
		toggle := false
		tts := cmd == "tts"
		parts := args

	outer:
		for {
			msgStr := strings.Join(parts, " ")
			if terminal && msgStr == "toggle" {
				toggle = !toggle
			} else {

				if len(msgStr) > msgLimit {
					stdutil.PrintErr(tl("invalid.limit.message"), nil)
					return
				}

				if userType == typeWebhook {
					err := session.WebhookExecute(userID, userToken, false, &discordgo.WebhookParams{
						Content: msgStr,
						TTS:     tts,
					})
					if err != nil {
						stdutil.PrintErr(tl("failed.msg.send"), err)
						return
					}
					return
				}

				msgObj := &discordgo.MessageSend{}
				msgObj.SetContent(msgStr)
				msgObj.Tts = tts
				msg, err := session.ChannelMessageSendComplex(loc.channel.ID, msgObj)
				if err != nil {
					stdutil.PrintErr(tl("failed.msg.send"), err)
					return
				}
				writeln(w, tl("status.msg.create")+" "+msg.ID)
				lastUsedMsg = msg.ID
				returnVal = msg.ID
			}

			if !toggle {
				break
			}

			for {
				color.Unset()
				colorChatMode.Set()

				text, err := rl.Readline()
				if err != nil {
					if err != readline.ErrInterrupt && err != io.EOF {
						stdutil.PrintErr(tl("failed.readline.read"), err)
					}
					break outer
				}

				color.Unset()

				parts = strings.Fields(text)
				if len(parts) >= 1 {
					break
				}
			}
		}
	case "embed":
		if nargs < 1 {
			stdutil.PrintErr("embed <embed json>", nil)
			return
		}
		if loc.channel == nil && userType != typeWebhook {
			stdutil.PrintErr(tl("invalid.channel"), nil)
			return
		}

		jsonstr := strings.Join(args, " ")
		var embed = &discordgo.MessageEmbed{}

		err := json.Unmarshal([]byte(jsonstr), embed)
		if err != nil {
			stdutil.PrintErr(tl("failed.json"), err)
			return
		}

		if userType == typeWebhook {
			err = session.WebhookExecute(userID, userToken, false, &discordgo.WebhookParams{
				Embeds: []*discordgo.MessageEmbed{embed},
			})
			if err != nil {
				stdutil.PrintErr(tl("failed.msg.send"), err)
				return
			}
		} else {
			msg, err := session.ChannelMessageSendEmbed(loc.channel.ID, embed)
			if err != nil {
				stdutil.PrintErr(tl("failed.msg.send"), err)
				return
			}
			writeln(w, tl("status.msg.create")+" "+msg.ID)
			lastUsedMsg = msg.ID
			returnVal = msg.ID
		}
	case "big":
		if nargs < 1 {
			stdutil.PrintErr("big <stuff>", nil)
			return
		}
		if loc.channel == nil && userType != typeWebhook {
			stdutil.PrintErr(tl("invalid.channel"), nil)
			return
		}

		buffer := ""
		for _, c := range strings.Join(args, " ") {
			str := toEmojiString(c)
			if len(buffer)+len(str) > msgLimit {
				_, ok := say(session, w, loc.channel.ID, buffer)
				if !ok {
					return
				}

				buffer = ""
			}
			buffer += str
		}
		msg, _ := say(session, w, loc.channel.ID, buffer)

		if msg != nil {
			lastUsedMsg = msg.ID
			returnVal = msg.ID
		}
	case "sayfile":
		if nargs < 1 {
			stdutil.PrintErr("sayfile <path>", nil)
			return
		}
		if loc.channel == nil && userType != typeWebhook {
			stdutil.PrintErr(tl("invalid.channel"), nil)
			return
		}

		path := args[0]
		err := fixPath(&path)
		if err != nil {
			stdutil.PrintErr(tl("failed.fixpath"), err)
			return
		}

		reader, err := os.Open(path)
		if err != nil {
			stdutil.PrintErr(tl("failed.file.open"), err)
			return
		}
		defer reader.Close()

		scanner := bufio.NewScanner(reader)
		buffer := ""

		for i := 1; scanner.Scan(); i++ {
			text := scanner.Text()
			if len(text) > msgLimit {
				stdutil.PrintErr("Line "+strconv.Itoa(i)+" exceeded "+strconv.Itoa(msgLimit)+" characters.", nil)
				return
			} else if len(buffer)+len(text) > msgLimit {
				_, ok := say(session, w, loc.channel.ID, buffer)
				if !ok {
					return
				}

				buffer = ""
			}
			buffer += text + "\n"
		}

		err = scanner.Err()
		if err != nil {
			stdutil.PrintErr(tl("failed.file.read"), err)
			return
		}
		msg, _ := say(session, w, loc.channel.ID, buffer)
		if msg != nil {
			returnVal = msg.ID
			lastUsedMsg = msg.ID
		}
	case "file":
		if nargs < 1 {
			stdutil.PrintErr("file <file>", nil)
			return
		}
		if loc.channel == nil {
			stdutil.PrintErr(tl("invalid.channel"), nil)
			return
		}
		name := strings.Join(args, " ")
		err := fixPath(&name)
		if err != nil {
			stdutil.PrintErr(tl("failed.fixpath"), err)
		}

		file, err := os.Open(name)
		if err != nil {
			stdutil.PrintErr(tl("failed.file.open"), nil)
			return
		}
		defer file.Close()

		msg, err := session.ChannelFileSend(loc.channel.ID, filepath.Base(name), file)
		if err != nil {
			stdutil.PrintErr(tl("failed.msg.send"), err)
			return
		}
		writeln(w, tl("status.msg.created")+" "+msg.ID)
		returnVal = msg.ID
	case "quote":
		if nargs < 1 {
			stdutil.PrintErr("quote <message id>", nil)
			return
		}
		if loc.channel == nil {
			stdutil.PrintErr(tl("invalid.channel"), nil)
			return
		}

		msg, err := getMessage(session, loc.channel.ID, args[0])
		if err != nil {
			stdutil.PrintErr(tl("failed.msg.query"), err)
			return
		}

		t, err := timestamp(msg)
		if err != nil {
			stdutil.PrintErr(tl("failed.timestamp"), err)
			return
		}

		msg, err = session.ChannelMessageSendEmbed(loc.channel.ID, &discordgo.MessageEmbed{
			Author: &discordgo.MessageEmbedAuthor{
				Name:    msg.Author.Username,
				IconURL: discordgo.EndpointUserAvatar(msg.Author.ID, msg.Author.Avatar),
			},
			Description: msg.Content,
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Sent " + t,
			},
		})
		if err != nil {
			stdutil.PrintErr(tl("failed.msg.send"), err)
			return
		}
		writeln(w, tl("status.msg.create")+" "+msg.ID)
		lastUsedMsg = msg.ID
		returnVal = msg.ID
	case "editembed":
		fallthrough
	case "edit":
		if nargs < 2 {
			stdutil.PrintErr("edit <message id> <stuff>", nil)
			return
		}
		if loc.channel == nil {
			stdutil.PrintErr(tl("invalid.channel"), nil)
			return
		}

		id := args[0]
		contents := strings.Join(args[1:], " ")

		var msg *discordgo.Message
		var err error
		if cmd == "editembed" {
			var embed = &discordgo.MessageEmbed{}
			err := json.Unmarshal([]byte(contents), embed)
			if err != nil {
				stdutil.PrintErr(tl("failed.json"), err)
				return
			}

			msg, err = session.ChannelMessageEditEmbed(loc.channel.ID, id, embed)
		} else {
			msg, err = session.ChannelMessageEdit(loc.channel.ID, id, contents)
		}
		if err != nil {
			stdutil.PrintErr(tl("failed.msg.edit"), err)
			return
		}
		lastUsedMsg = msg.ID
	case "del":
		if nargs < 1 {
			stdutil.PrintErr("del <message id>", nil)
			return
		}
		if loc.channel == nil {
			stdutil.PrintErr(tl("invalid.channel"), nil)
			return
		}

		err := session.ChannelMessageDelete(loc.channel.ID, args[0])
		if err != nil {
			stdutil.PrintErr(tl("failed.msg.delete"), err)
			return
		}
	case "delall":
		if loc.channel == nil {
			stdutil.PrintErr(tl("invalid.channel"), nil)
			return
		}
		since := ""
		if nargs >= 1 {
			since = args[0]
		}
		messages, err := session.ChannelMessages(loc.channel.ID, 100, "", since, "")
		if err != nil {
			stdutil.PrintErr(tl("failed.msg.query"), err)
			return
		}

		ids := make([]string, len(messages))
		for i, msg := range messages {
			ids[i] = msg.ID
		}

		err = session.ChannelMessagesBulkDelete(loc.channel.ID, ids)
		if err != nil {
			stdutil.PrintErr(tl("failed.msg.query"), err)
			return
		}
		returnVal := strconv.Itoa(len(ids))
		writeln(w, strings.Replace(tl("status.msg.delall"), "#", returnVal, -1))
	}
	return
}