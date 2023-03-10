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
    1. To build the project, run 
    `make build`
    2. To start a local server, run
    `make local`
    3. Go to `localhost:3000` in your browser


### Note - 
The Development setup runs best on Linux based systems.

The backend development need to run `go-sqlite` which is difficult to setup on windows, so you may have to face some difficulties while setting it up.

