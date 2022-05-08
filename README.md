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
      - SMTP2WORKFLOW_URL_TEST=https://d039b5.requestcatcher.com/test
```

Will forward mail to `d039b5+test@example.com` to `https://d039b5.requestcatcher.com/test`.
