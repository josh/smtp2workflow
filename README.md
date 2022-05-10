# smtp2workflow

SMTP to GitHub Actions workflow Relay.

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

Will forward mail to `d039b5+test@example.com` as a workflow dispatch event on the `owner/repo` repository. The raw email is passed as a git blob SHA to work around dispatch event input size limits.

Here's an example workflow that receives the email payload.

```yml
name: Email

on:
  workflow_dispatch:
    inputs:
      email_sha:
        description: "Email Blob SHA"

jobs:
  process:
    runs-on: ubuntu-latest

    steps:
      - name: Fetch email blob
        run: |
          gh api --jq '.content | @base64d' "/repos/$REPOSITORY/git/blobs/$FILE_SHA" >input.eml
        env:
          GITHUB_TOKEN: ${{ github.token }}
          REPOSITORY: ${{ github.repository }}
          FILE_SHA: ${{ github.event.inputs.email_sha }}
```
