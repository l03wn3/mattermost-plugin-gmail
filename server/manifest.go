// This file is automatically generated. Do not modify it manually.

package main

import (
	"strings"

	"github.com/mattermost/mattermost-server/v5/model"
)

var manifest *model.Manifest

const manifestStr = `
{
  "id": "mattermost-plugin-gmail",
  "name": "Mattermost Gmail Bot",
  "description": "Gmail Integration for Mattermost",
  "homepage_url": "https://github.com/abdulsmapara/mattermost-plugin-gmail",
  "support_url": "https://github.com/abdulsmapara/mattermost-plugin-gmail/issues",
  "version": "0.1.0",
  "min_server_version": "5.19.0",
  "server": {
    "executables": {
      "linux-amd64": "server/dist/plugin-linux-amd64",
      "darwin-amd64": "server/dist/plugin-darwin-amd64",
      "windows-amd64": "server/dist/plugin-windows-amd64.exe"
    },
    "executable": ""
  },
  "settings_schema": {
    "header": "The Gmail plugin for Mattermost",
    "footer": "Made with Love and Support from Mattermost by Abdul Sattar Mapara",
    "settings": [
      {
        "key": "GmailOAuthClientID",
        "display_name": "Google (Gmail) Client ID",
        "type": "text",
        "help_text": "The client ID for the OAuth app registered with Google Cloud",
        "placeholder": "Please copy client ID over from Google API console for Gmail API",
        "default": null
      },
      {
        "key": "GmailOAuthSecret",
        "display_name": "Gmail Secret",
        "type": "text",
        "help_text": "The client secret for the OAuth app registered with Google Cloud.",
        "placeholder": "Please copy secret over from Google (gmail) OAuth application",
        "default": null
      },
      {
        "key": "TopicName",
        "display_name": "Topic Name",
        "type": "text",
        "help_text": "Topic Name is used to subscribe user for notifications from Gmail.",
        "placeholder": "Create a topic in Google Cloud pubsub",
        "default": null
      },
      {
        "key": "EncryptionKey",
        "display_name": "Plugin Encryption Key",
        "type": "generated",
        "help_text": "The AES encryption key internally used in plugin to encrypt stored access tokens.",
        "placeholder": "Generate the key and store before connecting the account",
        "default": null
      }
    ]
  }
}
`

func init() {
	manifest = model.ManifestFromJson(strings.NewReader(manifestStr))
}
