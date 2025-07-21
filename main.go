package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// GitHub Actions environment variables
const (
	EnvGitHubEventName = "GITHUB_EVENT_NAME"
	EnvGitHubJob       = "GITHUB_JOB"
	EnvGitHubRef       = "GITHUB_REF"
	EnvGitHubRepo      = "GITHUB_REPOSITORY"
	EnvGitHubRepoOwner = "GITHUB_REPOSITORY_OWNER"
	EnvGitHubRunID     = "GITHUB_RUN_ID"
	EnvGitHubRunNumber = "GITHUB_RUN_NUMBER"
	EnvGitHubWorkflow  = "GITHUB_WORKFLOW"
)

// Slack environment variables
const (
	EnvSlackAuthor     = "SLACK_AUTHOR"
	EnvSlackAvatarURL  = "SLACK_AVATAR_URL"
	EnvSlackCommitID   = "SLACK_COMMIT_ID"
	EnvSlackCommitMsg  = "SLACK_COMMIT_MSG"
	EnvSlackCommitURL  = "SLACK_COMMIT_URL"
	EnvSlackCompareURL = "SLACK_COMPARE_URL"
	EnvSlackEmail      = "SLACK_EMAIL"
	EnvSlackStatus     = "SLACK_STATUS"
	EnvSlackWebhookURL = "SLACK_WEBHOOK_URL"
)

// Slack attachment color
const (
	ColorSuccess   = "#2eb886"
	ColorFailure   = "#951e13"
	ColorCancelled = "#dddddd"
)

// Message Slack incoming webhook message
type Message struct {
	Text        string       `json:"text,omitempty"` // fallback string
	Blocks      []any        `json:"blocks,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	ThreadTS    string       `json:"thread_ts,omitempty"`
	Markdown    bool         `json:"mrkdwn,omitempty"` // default: true
}

// Attachment Slack incoming webhook attachment
type Attachment struct {
	Blocks []any  `json:"blocks,omitempty"`
	Color  string `json:"color,omitempty"`
}

// Context Slack incoming webhook context block
type Context struct {
	Type     string `json:"type"`               // always `context`
	Elements []any  `json:"elements"`           // one of Image and Text, maximum size: 10
	BlockID  string `json:"block_id,omitempty"` // maximum length: 255
}

// Section Slack incoming webhook section block
type Section struct {
	Type      string  `json:"type"`                // always `section`
	Text      *Text   `json:"text,omitempty"`      // maximum length: 3000
	BlockID   string  `json:"block_id,omitempty"`  // maximum length: 255
	Fields    []*Text `json:"fields,omitempty"`    // maximum size: 10
	Accessory any     `json:"accessory,omitempty"` // one of block elements
}

// Image Slack incoming webhook image block element
type Image struct {
	Type     string `json:"type"` // always `image`
	ImageURL string `json:"image_url"`
	AltText  string `json:"alt_text"` // plain text
}

// Text Slack incoming webhook text composition object
type Text struct {
	Type     string `json:"type"` // `plain_text` or `mrkdwn`
	Text     string `json:"text"`
	Emoji    bool   `json:"emoji,omitempty"`    // only usable for `plain_text`
	Verbatim bool   `json:"verbatim,omitempty"` // only usable for `mrkdwn`
}

func main() {
	webhookURL := getEnv(EnvSlackWebhookURL)
	status := getEnv(EnvSlackStatus)
	author := getEnv(EnvSlackAuthor)
	email := getEnv(EnvSlackEmail)
	commitID := getEnv(EnvSlackCommitID)
	commitMsg := getEnv(EnvSlackCommitMsg)
	commitURL := getEnv(EnvSlackCommitURL)
	avatarURL := getEnv(EnvSlackAvatarURL)
	compareURL := getEnv(EnvSlackCompareURL)

	commitMsg = strings.Split(commitMsg, "\n")[0]

	statusColors := map[string]string{
		"success":   ColorSuccess,
		"failure":   ColorFailure,
		"cancelled": ColorCancelled,
	}
	color, ok := statusColors[status]
	if !ok {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Invalid %s", EnvSlackStatus))
		os.Exit(1)
	}

	event := os.Getenv(EnvGitHubEventName)
	ref := os.Getenv(EnvGitHubRef)
	repo := os.Getenv(EnvGitHubRepo)
	owner := os.Getenv(EnvGitHubRepoOwner)
	runID := os.Getenv(EnvGitHubRunID)
	runNumber := os.Getenv(EnvGitHubRunNumber)
	workflow := os.Getenv(EnvGitHubWorkflow)

	payload := Message{
		Text: fmt.Sprintf("GitHub Actions (%s): %s %s", repo, workflow, status),
		Blocks: []any{
			&Context{
				Type: "context",
				Elements: []any{
					&Image{
						Type:     "image",
						ImageURL: avatarURL,
						AltText:  owner,
					},
					&Text{
						Type: "mrkdwn",
						Text: fmt.Sprintf("*%s*", repo),
					},
				},
			},
		},
		Attachments: []Attachment{
			{
				Blocks: []any{
					&Section{
						Type: "section",
						Text: &Text{
							Type: "mrkdwn",
							Text: fmt.Sprintf("*<https://github.com/%s/actions/runs/%s|%s #%s>*", repo, runID, workflow, runNumber),
						},
					},
					&Section{
						Type: "section",
						Fields: []*Text{
							{
								Type: "mrkdwn",
								Text: fmt.Sprintf("*Ref:*\n<%s|%s>", compareURL, ref),
							},
							{
								Type: "mrkdwn",
								Text: fmt.Sprintf("*Author:*\n<mailto:%s|%s>", email, author),
							},
							{
								Type: "mrkdwn",
								Text: fmt.Sprintf("*Event:*\n%s", event),
							},
							{
								Type: "mrkdwn",
								Text: fmt.Sprintf("*Status:*\n%s", status),
							},
						},
					},
					&Section{
						Type: "section",
						Text: &Text{
							Type: "mrkdwn",
							Text: fmt.Sprintf("*Message:*\n<%s|%s (%s)>", commitURL, commitMsg, commitID[:8]),
						},
					},
				},
				Color: color,
			},
		},
	}

	if err := send(webhookURL, payload); err != nil {
		fmt.Fprintf(os.Stderr, "Payload send failed: %s\n", err)
		os.Exit(2)
	}
}

func getEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Need to provide %s", key))
		os.Exit(1)
	}

	return value
}

func jsonMarshal(t any) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(t)
	return buffer.Bytes(), err
}

func send(webhookURL string, payload Message) error {
	enc, err := jsonMarshal(payload)
	if err != nil {
		return err
	}

	b := bytes.NewBuffer(enc)

	res, err := http.Post(webhookURL, "application/json", b)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("Error on message: %s", res.Status)
	}

	fmt.Println(res.Status)
	return nil
}
