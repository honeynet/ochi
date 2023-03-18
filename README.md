# Ochi

UI for events from [Glutton](https://github.com/mushorg/glutton), streamed live from a development instance.

## Development Requirements -
1. [Golang version > 1.17](https://go.dev/doc/install)
2. [Node LTS version](https://nodejs.org/en/download/)

#### For windows system, some additional setup is needed (only in case of backend development)
> if you have `make` command working in your system, you can directly jump to point 4.
1. download and setup [mingw 64 bit version](https://sourceforge.net/projects/mingw-w64/files/). (These steps are required to make sure, your `make command` works.)
2. add path of `mingw64/bin` folder in your system environment variables. (In case if don't know, you can have search on internet).
3. then go to `mingw64/bin` folder and search for file named `mingw32-make` and rename it to `make`
4. go to your downloaded `ochi` folder, run command `go env` and check if your go environment have `CGO_ENABLED="1"` or not, if not then make appropriate changes in your system to have `CGO_ENABLED="1"`.

### Steps for development -
1. Clone the repo `git clone https://github.com/glaslos/ochi.git`
2. run `cd ochi`
2. run `npm install`

##### For Frontend development only
1. `comment the dial() and uncomment the test() in src/App.svelte`
2. run `npm run dev`
3. Go to  `http://localhost:8080` in your browser.

##### For Frontend and backend development
1. To build the project, run `make build`
2. To start a local server, run `make local`
3. Go to `localhost:3000` in your browser
4. To generate fake events, follow frontend development's step 1.

> If you are uncommenting `test()` and commenting `dial()`, please revert it back to its original state before generating PRs.

### Notes
1. The Development setup runs best on Linux based systems.
2. The backend development need to run `go-sqlite` which is difficult to setup on windows, so you may have to face some difficulties while setting it up.
4. In case you are still facing any issue while setup, feel free to ask.