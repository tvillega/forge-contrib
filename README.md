# forge-contrib

A small GitHub App that turns issue comments into actions. Mention the bot in a
comment and it will add/remove labels, add/remove a milestone, or assign/unassign
the commenter. It reacts with 👍 when the command succeeds and 👎 when it fails.

The app receives webhooks through [smee.io](https://smee.io), so it does not need
a public IP address or an open inbound port.

```
GitHub  ──webhook──▶  smee.io channel  ──SSE──▶  smee client  ──▶  forge-contrib  ──▶  GitHub API
```

## Requirements

- Docker Engine with Compose v2 (`docker compose version`)
- A GitHub App with permission to manage issues
- A smee.io channel

## 1. Create a smee.io channel

1. Open <https://smee.io> and click **Start a new channel**.
2. Copy the channel URL, for example `https://smee.io/AbCdEfGhIjKlMnOp`.
   You will use it as the GitHub webhook URL and as `SMEE_URL`.

The channel is public to anyone who knows the URL, but every delivery is still
verified inside the app with `GITHUB_WEBHOOK_SECRET`.

## 2. Create the GitHub App

1. Go to **GitHub → Settings → Developer settings → GitHub Apps → New GitHub App**.
2. Set **Webhook URL** to the smee.io channel URL from step 1.
3. Set **Webhook secret** to a random string and keep it for `GITHUB_WEBHOOK_SECRET`
   (`openssl rand -hex 32` works well).
4. Under **Permissions → Repository permissions**, set **Issues** to
   **Read and write**. Leave the rest at their defaults.
5. Under **Subscribe to events**, enable **Issue comment**.
6. Create the app, note the **App ID**, then scroll down and click
   **Generate a private key**.
7. Save the downloaded `.pem` file as `private-key.pem` next to
   `docker-compose.yml`.

Finally, install the app on the repositories it should manage (**Install App** in
the app settings) and add those repositories to `REPO_WHITELIST` below.

## 3. Configure

Create a `.env` file next to `docker-compose.yml`:

```dotenv
PORT=5050
GITHUB_APP_ID=123456
GITHUB_WEBHOOK_SECRET=replace-with-your-webhook-secret
SMEE_URL=https://smee.io/AbCdEfGhIjKlMnOp

# Only these users may trigger commands (comma-separated).
USER_WHITELIST=alice,bob

# Only these repositories are handled (comma-separated, "owner/repo").
REPO_WHITELIST=alice/project,bob/other-project

# Command prefix. Set this: with an empty value every comment is treated as a command.
BOT_NAME=arietta
```

`docker-compose.yml` mounts `./private-key.pem` into the container and sets
`GITHUB_PRIVATE_KEY_PATH` for you.

The container runs as UID 10001, so make sure the key is readable:

```bash
chmod 644 private-key.pem
```

## 4. Run

```bash
docker compose pull
docker compose up -d
```

Check that both services are running and the app is healthy:

```bash
docker compose ps
docker compose logs -f smee
```

`smee` waits for the app healthcheck before it starts forwarding events.

## 5. Use it

Comment on an issue in a whitelisted repository:

```text
arietta add label bug
arietta rm label bug
arietta add milestone v1.0
arietta rm milestone
arietta self assign
arietta self unassign
```

The bot adds 👍 to a successful command or 👎 if it could not be applied. The
comment must start with `BOT_NAME` exactly as configured.

## Configuration reference

| Variable                   | Required | Description                                                        |
| -------------------------- | -------- | ------------------------------------------------------------------ |
| `GITHUB_APP_ID`            | yes      | Numeric ID of the GitHub App.                                      |
| `GITHUB_WEBHOOK_SECRET`    | yes      | Secret configured on the GitHub App; used to verify deliveries.    |
| `SMEE_URL`                 | yes      | smee.io channel URL that receives the GitHub webhooks.             |
| `GITHUB_PRIVATE_KEY_PATH`  | no       | Path to the private key inside the container. Set by Compose.      |
| `USER_WHITELIST`           | yes      | Comma-separated GitHub logins allowed to run commands.             |
| `REPO_WHITELIST`           | yes      | Comma-separated `owner/repo` names the bot is allowed to act on.   |
| `BOT_NAME`                 | yes      | Comment prefix that triggers the bot, e.g. `arietta`.              |
| `PORT`                     | no       | Port the app listens on inside the container. Defaults to `5050`.  |

With empty whitelists the bot ignores everything. With empty `BOT_NAME` it treats
every comment as a command.

## Updating

```bash
docker compose pull
docker compose up -d
```

## Troubleshooting

- **Nothing happens on a comment** — check `docker compose logs -f smee`, then the
  **Recent Deliveries** tab of the GitHub App. Confirm the event is
  `issue_comment`, the repo is in `REPO_WHITELIST`, and the author is in
  `USER_WHITELIST`.
- **`failed to get access token` in the logs** — the app is not installed on the
  repository, the App ID is wrong, or the private key cannot be read.
- **`permission denied` reading the key** — run `chmod 644 private-key.pem`.
- **SELinux blocks the bind mount** (Fedora/RHEL) — append `,z` to the key volume
  in `docker-compose.yml`: `./private-key.pem:/etc/arietta/private-key.pem:ro,z`.
- **`smee` restarts because the app is unhealthy** — run `docker compose logs forge-contrib`.
  The container listens on `PORT`; no host port is published by design.

## Publishing the image

For maintainers, after building and testing locally:

```bash
docker build -t tvillega/forge-contrib:latest .
docker login
docker push tvillega/forge-contrib:latest
```
