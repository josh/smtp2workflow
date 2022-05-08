# smtp2workflow

SMTP to Webhook Relay.

```yml
version: "3"
services:
  smtp2workflow:
    restart: always
    image: ghcr.io/josh/smtp2workflow
    ports:
      - "25:25"
    environment:
      - SMTP2WORKFLOW_DOMAIN=example.com
      - SMTP2WORKFLOW_CODE=d039b5
      - SMTP2WORKFLOW_GITHUB_TOKEN=ghp_123abc
      - SMTP2WORKFLOW_REPOSITORY_TEST=owner/repo
      - SMTP2WORKFLOW_WORKFLOW_TEST=email.yml
```

Will forward mail to `d039b5+test@example.com` as a workflow dispatch event on the `owner/repo` repository.
