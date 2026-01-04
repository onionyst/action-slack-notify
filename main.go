package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
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
	// Required
	webhookURL := mustEnv(EnvSlackWebhookURL)
	status := strings.ToLower(mustEnv(EnvSlackStatus))

	// Optional Slack bits
	author := envOr(EnvSlackAuthor, "unknown")
	email := envOr(EnvSlackEmail, "")
	commitID := envOr(EnvSlackCommitID, "")
	commitMsg := envOr(EnvSlackCommitMsg, "")
	commitURL := envOr(EnvSlackCommitURL, "")
	avatarURL := envOr(EnvSlackAvatarURL, "")
	compareURL := envOr(EnvSlackCompareURL, "")

	// GitHub bits
	event := envOr(EnvGitHubEventName, "")
	ref := envOr(EnvGitHubRef, "")
	repo := envOr(EnvGitHubRepo, "")
	owner := envOr(EnvGitHubRepoOwner, "")
	runID := envOr(EnvGitHubRunID, "")
	runNumber := envOr(EnvGitHubRunNumber, "")
	workflow := envOr(EnvGitHubWorkflow, "")

	// Normalize commit message to first line
	if i := strings.IndexByte(commitMsg, '\n'); i >= 0 {
		commitMsg = commitMsg[:i]
	}

	// Fallbacks for compareURL
	if compareURL == "" {
		compareURL = commitURL
	}
	if compareURL == "" && repo != "" {
		compareURL = fmt.Sprintf("https://github.com/%s", repo)
	}

	statusColors := map[string]string{
		"success":   ColorSuccess,
		"failure":   ColorFailure,
		"cancelled": ColorCancelled,
	}
	color, ok := statusColors[status]
	if !ok {
		fmt.Fprintf(os.Stderr, "Invalid %s\n", EnvSlackStatus)
		os.Exit(1)
	}

	shortCommit := commitID
	if len(shortCommit) > 8 {
		shortCommit = shortCommit[:8]
	}

	// Author field formatting: mailto only if email exists
	authorField := fmt.Sprintf("*Author:*\n%s", author)
	if email != "" {
		authorField = fmt.Sprintf("*Author:*\n<mailto:%s|%s>", email, author)
	}

	// Ref field formatting: link only if compareURL exists
	refField := fmt.Sprintf("*Ref:*\n%s", ref)
	if compareURL != "" && ref != "" {
		refField = fmt.Sprintf("*Ref:*\n<%s|%s>", compareURL, ref)
	} else if compareURL != "" {
		refField = fmt.Sprintf("*Ref:*\n<%s|%s>", compareURL, ref)
	}

	// Commit line: link only if commitURL exists
	commitLine := fmt.Sprintf("*Message:*\n%s", commitMsg)
	if commitURL != "" && shortCommit != "" {
		commitLine = fmt.Sprintf("*Message:*\n<%s|%s (%s)>", commitURL, commitMsg, shortCommit)
	} else if shortCommit != "" {
		commitLine = fmt.Sprintf("*Message:*\n%s (%s)", commitMsg, shortCommit)
	}

	runLink := fmt.Sprintf("https://github.com/%s/actions/runs/%s", repo, runID)

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
							Text: fmt.Sprintf("*<%s|%s #%s>*", runLink, workflow, runNumber),
						},
					},
					&Section{
						Type: "section",
						Fields: []*Text{
							{Type: "mrkdwn", Text: refField},
							{Type: "mrkdwn", Text: authorField},
							{Type: "mrkdwn", Text: fmt.Sprintf("*Event:*\n%s", event)},
							{Type: "mrkdwn", Text: fmt.Sprintf("*Status:*\n%s", status)},
						},
					},
					&Section{
						Type: "section",
						Text: &Text{
							Type: "mrkdwn",
							Text: commitLine,
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

func mustEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		fmt.Fprintf(os.Stderr, "Need to provide %s\n", key)
		os.Exit(1)
	}
	return value
}

func envOr(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
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

	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader(enc))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("error on message: %s: %s", res.Status, strings.TrimSpace(string(body)))
	}

	fmt.Println(strings.TrimSpace(string(body)))
	return nil
}
