# Ochi

UI for events from [Glutton](https://github.com/mushorg/glutton), streamed live from a development instance.

## Development Requirements
For backend development
1. [Golang version > 1.17](https://go.dev/doc/install)
2. [Node LTS version](https://nodejs.org/en/download/)

### Steps for developement

1. Clone the repo `git clone https://github.com/glaslos/ochi.git`
2. run `cd ochi`
2. run `npm install`

##### For Frontend development only
1. `comment the dial() and uncomment the test() in src/App.svelte`
2. run `npm run dev`
    
##### For Frontend and backend development
1. To build the project, run `make build`
2. To start a local server, run `make local`
3. Go to `localhost:3000` in your browser

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
The Development setup runs best on Linux based systems.

The backend development need to run `go-sqlite` which is difficult to setup on windows, so you may have to face some difficulties while setting it up.
