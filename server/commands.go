package main

import (
	"fmt"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"google.golang.org/api/gmail/v1"
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
	if baseCommand != "/"+commandGmail {
		return &model.CommandResponse{}, nil
	}

	switch action {
	case "connect":
		return p.handleConnectCommand(c, args)
	case "disconnect":
		return p.handleDisconnectCommand(c, args)
	case "import":
		return p.handleImportCommand(c, args)
	case "subscribe":
		return p.handleSubscriptionCommands(c, args, action)
	case "unsubscribe":
		return p.handleSubscriptionCommands(c, args, action)
	case "subscriptions":
		return p.handleListSubscriptionsCommand(c, args)
	case "setoutputchannel":
		return p.handleSetOutputChannelCommand(c, args)
	case "addoutputchannel":
		return p.handleAddOutputChannelCommand(c, args)
	case "removeoutputchannel":
		return p.handleRemoveOutputChannelCommand(c, args)
	case "showoutputchannels":
		return p.handleShowOutputChannelCommand(c, args)
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
	if p.checkIfConnected(args.UserId) == true {
		p.sendMessageFromBot(args.ChannelId, args.UserId, true, "You are already connected to Gmail.")
		return &model.CommandResponse{}, nil
	}
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

func (p *Plugin) getChannelIdFromName(channelTarget string, teamId string) (string) {
	channels, _ := p.API.GetPublicChannelsForTeam(teamId, 0, 100)
	p.API.LogInfo(fmt.Sprintf("Found %d channels for team", len(channels)))

	p.API.LogInfo("Looking for channel named: '" + channelTarget + "'")
	var foundChannelId string
	for _, channel := range channels { 
		p.API.LogInfo("Candidate channel name: " + channel.Name + ", display name: " + channel.DisplayName + ", ID: " + channel.Id)
		if strings.EqualFold(channelTarget, channel.DisplayName) {
			foundChannelId = channel.Id
			p.API.LogInfo("Found channel name: " + channel.Name + ", display name: " + channel.DisplayName + ", ID: " + channel.Id)
			break
		}
	} 

	return foundChannelId
}

// handleSetOutputChannelCommand configures which channe bot will post notifications in
func (p *Plugin) handleSetOutputChannelCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {

	p.API.LogInfo("Handling configure channel command")

	arguments := strings.Fields(args.Command)

	if len(arguments) < 3 {
		p.sendMessageFromBot(args.ChannelId, args.UserId, true, "Too few arguments to configure channel command ")
		return &model.CommandResponse{}, nil
	}

	channelTarget := strings.Join(arguments[2:], " ") //combine all words to one channel display name
	channels, _ := p.API.GetPublicChannelsForTeam(args.TeamId, 0, 100)

	p.API.LogInfo(fmt.Sprintf("Found %d channels for team", len(channels)))
	p.API.LogInfo("Looking for channel named: '" + channelTarget + "'")
	p.sendMessageFromBot(args.ChannelId, args.UserId, true, "Looking for channel named: '" + channelTarget + "'")

	foundChannels := []string{}

	for _, channel := range channels { 

		p.sendMessageFromBot(args.ChannelId, args.UserId, true, "Candidate channel name: " + channel.Name + ", display name: " + channel.DisplayName + ", ID: " + channel.Id)

		if strings.EqualFold(channelTarget, channel.DisplayName) {
			foundChannels = append(foundChannels, channel.Id)
			p.API.LogInfo("Found channel name: " + channel.Name + ", display name: " + channel.DisplayName + ", ID: " + channel.Id)
			p.sendMessageFromBot(args.ChannelId, args.UserId, true, "Found channel name: " + channel.Name + ", display name: " + channel.DisplayName + ", ID: " + channel.Id)
			break
		}
	} 
	
	if len(foundChannels) < 1 {
		p.API.LogInfo("Found no such channel:" + channelTarget)
		p.sendMessageFromBot(args.ChannelId, args.UserId, true, "Found no such channel:" + channelTarget)
		return &model.CommandResponse{}, nil
	}

	p.updateOutputChannels(foundChannels)
	return &model.CommandResponse{}, nil
}

func (p *Plugin) handleAddOutputChannelCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	p.API.LogInfo("Handling add output channel command")
		
	channelIds, err := p.getOutputChannels()
	if (err != nil) {
		p.API.LogInfo("Error! Could not get OutputChannels: " + err.Message)
		p.sendMessageFromBot(args.ChannelId, args.UserId, true, "Error! Could not get OutputChannels: " + err.Message)
		return &model.CommandResponse{}, err
	}

	arguments := strings.Fields(args.Command)
	channelTarget := strings.Join(arguments[2:], " ") //combine all words to one channel display name
	newChannelId := p.getChannelIdFromName(channelTarget, args.TeamId)

	channelIds = append(channelIds, newChannelId)
	err = p.updateOutputChannels(channelIds)
	if (err != nil) {
		p.API.LogError("Error setting outputchannels: " + err.Message)
	}

	outputString := "Added channel " + channelTarget + " as output channel"
	p.sendMessageFromBot(args.ChannelId, args.UserId, true, outputString)
	p.API.LogInfo(outputString)

	return &model.CommandResponse{}, nil
}

func (p *Plugin) handleRemoveOutputChannelCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	p.API.LogInfo("Handling remove output channel command")

	//Get channelID of Channel to remove
	arguments := strings.Fields(args.Command)
	channelTarget := strings.Join(arguments[2:], " ") //combine all words to one channel display name
	channelIdToRemove := p.getChannelIdFromName(channelTarget, args.TeamId)

	//Get current outputchannels
	channelIds, err := p.getOutputChannels()
	if (err != nil) {
		p.API.LogInfo("Error! Could not get OutputChannels: " + err.Message)
		p.sendMessageFromBot(args.ChannelId, args.UserId, true, "Error! Could not get OutputChannels: " + err.Message)
		return &model.CommandResponse{}, err
	}

	//Create new array with channels, excepting the one to remove
	outputChannelNames  := ""
	newChannelIds := []string{}
	for _, channel := range channelIds {
		if channel != channelIdToRemove {
			newChannelIds = append(newChannelIds, channel)
			newChannel, _ := p.API.GetChannel(channel)
			outputChannelNames += newChannel.DisplayName + ", "
		}
	}

	err = p.updateOutputChannels(newChannelIds)
	if (err != nil) {
		p.API.LogError("Error setting outputchannels: " + err.Message)
	}
	outputString := "Removed channel " + channelTarget + ". Now outputting to channels: " + outputChannelNames
	p.sendMessageFromBot(args.ChannelId, args.UserId, true, outputString)
	p.API.LogInfo(outputString)

	return &model.CommandResponse{}, err
}

// handleConfigureChannelCommand configures which channe bot will post notifications in
func (p *Plugin) handleShowOutputChannelCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {

	p.API.LogInfo("Handling show output channel command")

	channelIds, err := p.getOutputChannels()

	if (err != nil) {
		p.API.LogError("Error! Could not get OutputChannels: " + err.Message)
		p.sendMessageFromBot(args.ChannelId, args.UserId, true, "Error! Could not get OutputChannels: " + err.Message)
		return &model.CommandResponse{}, err
	}

	if len(channelIds) < 1 {
		p.API.LogInfo("No channels saved for output")
		p.sendMessageFromBot(args.ChannelId, args.UserId, true, "No channels saved for output")
		return &model.CommandResponse{}, nil
	}

	//Concatenate output
	outputString := "Will show output in channels: "
	for _, channelId := range channelIds { 
		channel, _ := p.API.GetChannel(channelId)
		channelDisplayName := channel.DisplayName
		outputString = outputString + channelDisplayName + ", "
	}
	p.API.LogInfo(args.ChannelId, args.UserId, true, outputString)
	p.sendMessageFromBot(args.ChannelId, args.UserId, true, outputString)

	return &model.CommandResponse{}, nil
}

// handleDisconnectCommand disconnects the user with Gmail account
func (p *Plugin) handleDisconnectCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {

	if p.checkIfConnected(args.UserId) == false {
		p.sendMessageFromBot(args.ChannelId, args.UserId, true, "You are not currently connected with Gmail. Use `/gmail connect` to get connected.")
		return &model.CommandResponse{}, nil
	}

	// Check if SiteURL is defined in the app
	siteURL := p.API.GetConfig().ServiceSettings.SiteURL
	if siteURL == nil {
		p.sendMessageFromBot(args.ChannelId, args.UserId, true, "Error! Site URL is not defined in the App")
		return &model.CommandResponse{}, nil
	}

	actionSecret := p.getConfiguration().EncryptionKey

	deleteButton := &model.PostAction{
		Type: model.POST_ACTION_TYPE_BUTTON,
		Name: "Disconnect",
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("%s/plugins/%s/command/disconnect", *siteURL, manifest.Id),
			Context: map[string]interface{}{
				"action":       ActionDisconnectPlugin,
				"actionSecret": actionSecret,
			},
		},
	}

	cancelButton := &model.PostAction{
		Type: model.POST_ACTION_TYPE_BUTTON,
		Name: "Cancel",
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("%s/plugins/%s/command/disconnect", *siteURL, manifest.Id),
			Context: map[string]interface{}{
				"action":       ActionCancel,
				"actionSecret": actionSecret,
			},
		},
	}

	deleteMessageAttachment := &model.SlackAttachment{
		Title: "Disconnect Gmail plugin",
		Text: ":scissors: Are you sure you would like to disconnect Gmail from Mattermost?\n" +
			"If you have any question or concerns please [report](https://github.com/abdulsmapara/mattermost-plugin-gmail/issues/new)",
		Actions: []*model.PostAction{deleteButton, cancelButton},
	}

	deletePost := &model.Post{
		UserId:    p.gmailBotID,
		ChannelId: args.ChannelId,
		Props: map[string]interface{}{
			"attachments": []*model.SlackAttachment{deleteMessageAttachment},
		},
	}

	p.API.SendEphemeralPost(args.UserId, deletePost)

	return &model.CommandResponse{}, nil

}

// handleHelpCommand posts help about the plugin
func (p *Plugin) handleHelpCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	p.sendMessageFromBot(args.ChannelId, args.UserId, true, helpTextHeader+commonHelpText)
	return &model.CommandResponse{}, nil
}

// handleImportCommand handles the command `/gmail import thread [id]` and `/gmail import mail [id]`
func (p *Plugin) handleImportCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {

	if p.checkIfConnected(args.UserId) == false {
		p.sendMessageFromBot(args.ChannelId, args.UserId, true, "Please connect yourself to Gmail using `/gmail connect`.")
		return &model.CommandResponse{}, nil
	}

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
		thread, threadErr := gmailService.Users.Threads.Get(gmailID, threadID).Format("minimal").Do()
		if threadErr != nil {
			p.sendMessageFromBot(args.ChannelId, args.UserId, true, threadErr.Error())
			return &model.CommandResponse{}, nil
		}
		threadMessages := []*gmail.Message{}
		for _, messageInfo := range thread.Messages {
			message, mailErr := gmailService.Users.Messages.Get(gmailID, messageInfo.Id).Format("raw").Do()
			if mailErr != nil {
				p.sendMessageFromBot(args.ChannelId, args.UserId, true, "Unable to get the thread.")
				return &model.CommandResponse{}, nil
			}
			threadMessages = append(threadMessages, message)
		}
		p.handleMessages(threadMessages, args.ChannelId, args.UserId, false)

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

	// Message extracted successfully
	p.handleMessages([]*gmail.Message{message}, args.ChannelId, args.UserId, false)

	return &model.CommandResponse{}, nil
}

func (p *Plugin) handleSubscriptionCommands(c *plugin.Context, args *model.CommandArgs, action string) (*model.CommandResponse, *model.AppError) {

	if p.checkIfConnected(args.UserId) == false {
		p.sendMessageFromBot(args.ChannelId, args.UserId, true, "Please connect yourself to Gmail using `/gmail connect`.")
		return &model.CommandResponse{}, nil
	}

	if action == "subscribe" {
		return p.handleSubscribeCommand(c, args)
	}
	return p.handleUnsubscribeCommand(c, args)
}

// handleSubscribeCommand updates the subscriptions of user in the KV Store
func (p *Plugin) handleSubscribeCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	// `/gmail subscribe [LABELS for eg. INBOX, CATEGORY_PROMOTIONS]`
	// if no Label specified, assume all the supported labels

	allLabelIDs := strings.TrimSpace(strings.ToUpper(strings.TrimPrefix(args.Command, "/"+commandGmail+" subscribe")))
	labelIDs := p.getSupportedLabels()

	if allLabelIDs != "" {
		labelIDs = strings.Split(allLabelIDs, ",")
	}

	for labelIndex, labelID := range labelIDs {
		labelIDs[labelIndex] = strings.TrimSpace(labelID)
		if _, ok := supportedLabelIDs[labelIDs[labelIndex]]; !ok {
			p.sendMessageFromBot(args.ChannelId, args.UserId, true, "Label ID: "+labelID+" not supported")
			return &model.CommandResponse{}, nil
		}
	}

	p.updateSubscriptionsOfUser(args.UserId, labelIDs)

	p.sendMessageFromBot(args.ChannelId, args.UserId, true, "You have subscribed to the labels: "+strings.Join(labelIDs, ",")+" successfully. Any previous subscription is overwritten.")
	return &model.CommandResponse{}, nil
}

func (p *Plugin) handleUnsubscribeCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {

	allLabelIDs := strings.TrimSpace(strings.ToUpper(strings.TrimPrefix(args.Command, "/"+commandGmail+" unsubscribe")))

	labelIDs, _ := p.getSubscriptionsOfUser(args.UserId)
	subscribedIDs := labelIDs

	// if not subscribed to any of the labelID
	if len(labelIDs) == 0 {
		p.sendMessageFromBot(args.ChannelId, args.UserId, true, "You have not subscribed to any label ID. Use `/"+commandGmail+" subscribe <Label IDs>` to subscribe.")
		return &model.CommandResponse{}, nil
	}
	// get mentioned label IDs
	if allLabelIDs != "" {
		labelIDs = strings.Split(allLabelIDs, ",")
	}

	remainSubscribed := []string{}
	unsubscribedTo := ""

	for _, subscribedID := range subscribedIDs {
		// user is currently subscribed to subscribedID
		foundInGivenIDs := false
		for _, labelID := range labelIDs {
			if strings.TrimSpace(labelID) == subscribedID {
				unsubscribedTo += subscribedID + ", "
				foundInGivenIDs = true
				break
			}
		}
		if foundInGivenIDs == false {
			remainSubscribed = append(remainSubscribed, subscribedID)
		}
	}
	if unsubscribedTo == "" {
		p.sendMessageFromBot(args.ChannelId, args.UserId, true, "You have not been unsubscribed from any labels. Please check if you have specified correct label IDs.")
		return &model.CommandResponse{}, nil
	}

	p.updateSubscriptionsOfUser(args.UserId, remainSubscribed)
	remainSubscribedMessage := ""
	for labelIndex, labelID := range remainSubscribed {
		remainSubscribedMessage += labelID
		if labelIndex != len(remainSubscribed)-1 {
			remainSubscribedMessage += ", "
		}
	}
	if remainSubscribedMessage != "" {
		remainSubscribedMessage = "You are currently subscribed to the labels: " + remainSubscribedMessage
	} else {
		remainSubscribedMessage = "Currently, you have no active subscriptions"
	}

	p.sendMessageFromBot(args.ChannelId, args.UserId, true, "You have successfully unsubscribed from the labels: "+unsubscribedTo+" .\n"+remainSubscribedMessage)
	return &model.CommandResponse{}, nil
}

func (p *Plugin) handleListSubscriptionsCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	if p.checkIfConnected(args.UserId) == false {
		p.sendMessageFromBot(args.ChannelId, args.UserId, true, "You are not currently connected with Gmail. Use `/gmail connect` to get connected.")
		return &model.CommandResponse{}, nil
	}
	subscriptions, _ := p.getSubscriptionsOfUser(args.UserId)
	if len(subscriptions) == 0 {
		p.sendMessageFromBot(args.ChannelId, args.UserId, true, "You have not subscribed to any labels. Please use `/gmail subscribe <Label IDs>` to subscribe.")
		return &model.CommandResponse{}, nil
	}
	p.sendMessageFromBot(args.ChannelId, args.UserId, true, "Currently, you are subscribed to the label IDs: "+strings.Join(subscriptions, ", "))
	return &model.CommandResponse{}, nil
}

// handleInvalidCommand
func (p *Plugin) handleInvalidCommand(c *plugin.Context, args *model.CommandArgs, action string) (*model.CommandResponse, *model.AppError) {
	p.sendMessageFromBot(args.ChannelId, args.UserId, true, "##### Unknown Command: "+action+"\n"+helpTextHeader+commonHelpText)
	return &model.CommandResponse{}, nil
}
