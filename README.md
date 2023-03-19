# Ochi

UI for events from [Glutton](https://github.com/mushorg/glutton), streamed live from a development instance.

## Development Requirements

1. [Golang version > 1.17](https://go.dev/doc/install)
2. [Node LTS version](https://nodejs.org/en/download/)

#### For windows system, some additional setup is needed (only in case of backend development)

> If you have `make` command working in your system and got `gcc compiler` installed already, you can directly jump to point 2.

1. Follow the steps mentioned [here](https://github.com/mattn/go-sqlite3#windows)
2. Go to your downloaded `ochi` folder, run command `go env` and check if your go environment have `CGO_ENABLED="1"` or not, if not then make appropriate changes in your system to have `CGO_ENABLED="1"`.

### Steps for development

1. Clone the repo `git clone https://github.com/glaslos/ochi.git`
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

### Notes
> If you are uncommenting `test()` and commenting `dial()`, please revert it back to its original state before generating PRs.
In case you are still facing any issue while setup, feel free to ask in [discussion](https://github.com/glaslos/ochi/discussions).

