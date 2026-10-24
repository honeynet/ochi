# Ochi

UI for events from [Glutton](https://github.com/mushorg/glutton), events streamed [live](https://ochi.mushmush.org/) from a development instance.

## Motivation

Any publicly available IP address is under a constant barrage of attacks. We want to identify trends and attacks which are interesting for honeypot development. The majority of events are internet background noise, so we need to be able to identify truly new events worth investigating.
Threats are happening in real time and we don't scale to preserve history. We work on a live stream with the requirement to be able to easily filter and classify events which are interesting. 
Eventually we want to be able to enabled to quickly react to new trends, improve our sensors to collect valuable information.

## Development Requirements

1. [Golang version 1.26 or later](https://go.dev/doc/install)
2. [Node.js 26 or later](https://nodejs.org/en/download/)

### Steps for development

1. Clone the repo `git clone https://github.com/honeynet/ochi.git`
2. run `cd ochi`
3. run `npm install`

##### For Frontend development only

1. Run `npm run dev`
2. Open `http://localhost:5000` (or `http://localhost:8080`) in your browser.
3. To generate simulated test events without a live feed, click the **Config** button in the top navigation bar and select **Development**. No code changes or manual commenting is required. (Optionally, changing the default `env` store value in `frontend/store.ts` to `ENV_DEV` makes development mode the default on load).

##### For Frontend and backend development

1. To build the project, run `make build` (or `npm run build && go build -o server .`)
2. To start a local server, run `make local` (or `./server` / `./server.exe`)
3. Open `http://localhost:3000` in your browser.
4. To generate simulated events, switch to **Development** mode via the **Config** button in the UI.

Set `require_registered_sensors: true` to accept published events only from sensors whose UUID is in the `sensors` table. It is off by default, because a sensor that was never registered through `/sensors` is rejected once it is on.

##### For using Ochi as a storage of Glutton events locally
1. Start Ochi server with `make build && make local`
2. Build Glutton server
3. Update the Glutton config to include:
   1. `producers.enabled` to `true` [here](https://github.com/mushorg/glutton/blob/305a9d23a58d065f49ac25edeaeb374f4fe9c59b/config/config.yaml#L9)
   2. `producers.http.enabled` to `true` [here](https://github.com/mushorg/glutton/blob/305a9d23a58d065f49ac25edeaeb374f4fe9c59b/config/config.yaml#L11)
   3. `producers.http.remote` to `http://sensor:token@localhost:3000/api/publish` (use the `publish_token` from your `config.yaml` as the password). Glutton sends the userinfo as an HTTP Basic auth header, keeping the token out of the URL and so out of proxy access logs. The older `?token=token` query form still works but is logged, so prefer the userinfo form.
4. Start Glutton server.
5. Open `http://localhost:3000` and you should see Glutton events if everything is working as expected.

## Configuration & Security

On first start, the server creates `config.yaml` with local defaults (`localhost:3000`, publish token `token`, JWT secret `secret`). Pass `-config path/to/config.yaml` to use a different file.

### Roles & Permissions

Users can hold roles, and each role grants a fixed set of permissions (defined in `backend/entities/role.go`). The only role so far is `admin`, which grants:

- `users:list`: list every registered user (`GET /api/users`, and the **Users** page).
- `sensors:add_by_id`: register a sensor under a chosen UUID (`POST /api/sensors` with an `id`). Without it, the server generates the sensor ID.

To make someone an admin, add their Google account email to `config.yaml`; the role is granted the next time they log in or reload the page:

```yaml
admin_emails:
  - you@example.com
```

Removing an address from `admin_emails` does not revoke the role. Delete the row from the `user_roles` table to do that.

### Security Best Practices

> [!WARNING]
> **Do not use default credentials in production!** The auto-generated `config.yaml` uses placeholder credentials intended solely for offline/local development.

- **Secrets & Authentication**: Replace default values (`publish_token: "token"`, `jwt_secret: "secret"`) with strong, randomly generated secrets before deploying or exposing the server.
- **Sensitive Files**: Keep `config.yaml`, the SQLite database (`data.db`), and SQLite WAL files (`data.db-wal`, `data.db-shm`) private. These files contain operational secrets and data, and are excluded by `.gitignore`.
- **TLS / HTTPS & WSS**: When deploying to production, run Ochi behind a reverse proxy (e.g., Caddy, NGINX) with TLS enabled to protect publisher tokens and WebSocket traffic in transit.
- **Rate Limiting**: Adjust `publish_rate_per_sec` and `publish_burst` in `config.yaml` according to your sensor traffic to prevent denial-of-service and ingestion flooding.

### Notes
1. In case you are still facing any issue while setup, feel free to ask in [discussion](https://github.com/honeynet/ochi/discussions).

