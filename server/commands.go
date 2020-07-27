package main

import (
	"fmt"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"math"
	"strings"
)

// ExecuteCommand executes the commands registered on getCommand() via RegisterCommand hook
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	// Obtain base command and its associated action -
	// Split the entered command based on white space (" ")
	arguments := strings.Fields(args.Command)

	// Example "gmail" in command "/gmail"
	baseCommand := arguments[0]

	// Example "connect" in command "/gmail connect"
	action := ""
	if len(arguments) > 1 {
		action = arguments[1]
	}

	// if command not '/gmail', then return
	if baseCommand != "/gmail" {
		return &model.CommandResponse{}, nil
	}

	switch action {
	case "connect":
		return p.handleConnectCommand(c, args)
	case "disconnect":
		return p.handleDisconnectCommand(c, args)
	case "import":
		return p.handleImportCommand(c, args)
	case "":
		return p.handleHelpCommand(c, args)
	case "help":
		return p.handleHelpCommand(c, args)
	default:
		return p.handleInvalidCommand(c, args, action)
	}
}

// handleConnectCommand connects the user with Gmail account
func (p *Plugin) handleConnectCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	// Check if SiteURL is defined in the app
	siteURL := p.API.GetConfig().ServiceSettings.SiteURL
	if siteURL == nil {
		p.sendMessageFromBot(args.ChannelId, args.UserId, true, "Error! Site URL is not defined in the App")
		return &model.CommandResponse{}, nil
	}

	// Send an ephemeral post with the link to connect gmail
	p.sendMessageFromBot(args.ChannelId, args.UserId, true, fmt.Sprintf("[Click here to connect your Gmail account with Mattermost.](%s/plugins/%s/oauth/connect)", *siteURL, manifest.Id))

	return &model.CommandResponse{}, nil
}

// handleDisconnectCommand disconnects the user with Gmail account
func (p *Plugin) handleDisconnectCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	// Send an ephemeral post with the link to disconnect gmail
	p.sendMessageFromBot(args.ChannelId, args.UserId, true, fmt.Sprintf(""))
	return &model.CommandResponse{}, nil
}

// handleHelpCommand posts help about the plugin
func (p *Plugin) handleHelpCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	p.sendMessageFromBot(args.ChannelId, args.UserId, true, helpText)
	return &model.CommandResponse{}, nil
}

// handleImportCommand handles the command `/gmail import thread [id]` and `/gmail import mail [id]`
func (p *Plugin) handleImportCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {

	arguments := strings.Fields(args.Command)
	// validate arguments of the command
	if len(arguments) < 3 {
		p.sendMessageFromBot(args.ChannelId, args.UserId, true, "Please use `thread` or `mail` after `/gmail import`. Also provide the ID of thread/mail.")
		return &model.CommandResponse{}, nil
	}
	queryType := arguments[2]
	if queryType != "thread" && queryType != "mail" {
		p.sendMessageFromBot(args.ChannelId, args.UserId, true, "Only `thread` and `mail` are supported after `/gmail import`.")
		return &model.CommandResponse{}, nil
	}
	if len(arguments) < 4 {
		p.sendMessageFromBot(args.ChannelId, args.UserId, true, "Please provide the ID of "+arguments[2])
		return &model.CommandResponse{}, nil
	}
	rfcID := arguments[3]

	gmailID, err := p.getGmailID(args.UserId)
	if err != nil {
		p.sendMessageFromBot(args.ChannelId, args.UserId, true, err.Error())
		return &model.CommandResponse{}, nil
	}

	gmailService, err := p.getGmailService(args.UserId)
	if err != nil {
		p.sendMessageFromBot(args.ChannelId, args.UserId, true, err.Error())
		return &model.CommandResponse{}, nil
	}
	p.API.LogInfo("gmailService created successfully")

	if queryType == "thread" {
		threadID, threadIDErr := p.getThreadID(args.UserId, gmailID, rfcID)
		if threadIDErr != nil {
			p.sendMessageFromBot(args.ChannelId, args.UserId, true, threadIDErr.Error())
			return &model.CommandResponse{}, nil
		}
		thread, threadErr := gmailService.Users.Threads.Get(gmailID, threadID).Format("raw").Do()
		if threadErr != nil {
			p.sendMessageFromBot(args.ChannelId, args.UserId, true, threadErr.Error())
			return &model.CommandResponse{}, nil
		}
		p.sendMessageFromBot(args.ChannelId, "", false, thread.Messages[0].Raw)
		return &model.CommandResponse{}, nil
	}
	// if queryType == "mail" =>
	// Note that explicit condition check is not required

	messageID, err := p.getMessageID(args.UserId, gmailID, rfcID)
	if err != nil {
		p.sendMessageFromBot(args.ChannelId, args.UserId, true, err.Error())
		p.API.LogInfo(err.Error())
		return &model.CommandResponse{}, nil
	}
	p.API.LogInfo("Extracted Message ID from rfc ID successfully")

	message, err := gmailService.Users.Messages.Get(gmailID, messageID).Format("raw").Do()
	if err != nil {
		p.sendMessageFromBot(args.ChannelId, args.UserId, true, "Unable to get the mail.")
		return &model.CommandResponse{}, nil
	}
	p.API.LogInfo("Message extracted successfully")

	base64URLMessage := message.Raw
	plainTextMessage, err := p.decodeBase64URL(base64URLMessage)
	if err != nil {
		p.sendMessageFromBot(args.ChannelId, args.UserId, true, "Error: "+err.Error())
		return &model.CommandResponse{}, nil
	}
	// Extract Subject and Body (base64url) from the message. TODO: Add attachments.
	subject, body, attachments, err := p.parseMessage(plainTextMessage)
	if err != nil {
		p.sendMessageFromBot(args.ChannelId, args.UserId, true, "An error has occured while trying to parse the mail. Please try again later or report to the System Administrator.")
		return &model.CommandResponse{}, nil
	}
	fileIDArray := []string{}
	fileNameArray := []string{}
	for _, attachment := range attachments {
		fileName, fileData := p.getAttachmentDetails(attachment)
		fileInfo, fileErr := p.API.UploadFile(fileData, args.ChannelId, fileName)
		if fileErr != nil {
			p.sendMessageFromBot(args.ChannelId, args.UserId, true, "Attachment "+fileName+" was not uploaded. Please report this to the System Administrator")
		}
		fileNameArray = append(fileNameArray, fileName)
		fileIDArray = append(fileIDArray, fileInfo.Id)
	}

	postID, _ := p.sendMessageFromBot(args.ChannelId, "", false, "###### Date: \n"+"###### Subject: "+subject+"\n"+body)

	if len(fileIDArray) > 0 {
		countFiles := 0
		parentID := postID
		// One Post can contain atmost 5 attachments
		for countFiles = 0; countFiles <= len(fileIDArray); countFiles += 5 {
			post := &model.Post{
				UserId:    p.gmailBotID,
				ChannelId: args.ChannelId,
				RootId:    postID,
				ParentId:  parentID,
				FileIds:   fileIDArray[countFiles:int(math.Min(float64(countFiles+5), float64(len(fileIDArray))))],
			}
			postInfo, err := p.API.CreatePost(post)
			if err != nil {
				p.sendMessageFromBot(args.ChannelId, args.UserId, true, "An error has occured : "+err.Error())
			}
			parentID = postInfo.Id
		}
	}

	return &model.CommandResponse{}, nil
}

// handleInvalidCommand
func (p *Plugin) handleInvalidCommand(c *plugin.Context, args *model.CommandArgs, action string) (*model.CommandResponse, *model.AppError) {
	p.sendMessageFromBot(args.ChannelId, args.UserId, true, "##### Unknown Command "+action+"\n"+helpText)
	return &model.CommandResponse{}, nil
}
