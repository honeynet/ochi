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

1. `comment the dial() and uncomment the test() in src/App.svelte`
2. run `npm run dev`
3. Go to `http://localhost:8080` in your browser.

##### For Frontend and backend development

1. To build the project, run `make build`
2. To start a local server, run `make local`
3. Go to `localhost:3000` in your browser
4. To generate fake events, follow frontend development's step 1.
##### For using Ochi as a storage of Glutton events locally
1. Start Ochi server with `make build && make local`
2. Build Glutton server
3. Update the Glutton config to include:
   1. `producers.enabled` to `true` [here](https://github.com/mushorg/glutton/blob/305a9d23a58d065f49ac25edeaeb374f4fe9c59b/config/config.yaml#L9)
   2. `producers.http.enabled` to `true` [here](https://github.com/mushorg/glutton/blob/305a9d23a58d065f49ac25edeaeb374f4fe9c59b/config/config.yaml#L11)
   3. `producers.http.remote` to `http://localhost:3000/publish?token=token`
4. Start Glutton server.
5. Open http://localhost:3000 and you should see Glutton events if everything is working as expected.

### Notes
1. If you are uncommenting `test()` and commenting `dial()`, please revert it back to its original state before generating PRs.
2. In case you are still facing any issue while setup, feel free to ask in [discussion](https://github.com/honeynet/ochi/discussions).

