# Slack Notify Action

This action sends action result notification to Slack.

## Example usage

```yaml
uses: onionyst/action-slack-notify@v1
env:
  SLACK_AUTHOR: ${{ github.event.head_commit.author.name }}
  SLACK_AVATAR_URL: ${{ github.event.repository.owner.avatar_url }}
  SLACK_COMMIT_ID: ${{ github.event.head_commit.id }}
  SLACK_COMMIT_MSG: ${{ github.event.head_commit.message }}
  SLACK_COMMIT_URL: ${{ github.event.head_commit.url }}
  SLACK_COMPARE_URL: ${{ github.event.compare }}
  SLACK_EMAIL: ${{ github.event.head_commit.author.email }}
  SLACK_STATUS: ${{ job.status }}
  SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK_URL }}
```
