import App from './App.svelte';
import {hexy} from "hexy";

const app = new App({
	target: document.body,
	props: {
		name: 'world'
	}
});

const messageLog = document.getElementById("message-log")
const content = document.getElementById("content")

var hexFormat = {}

function appendLog(event, error) {
	const p = document.createElement("p")
	if (error) {
		p.style.color = "red"
		p.style.fontStyle = "bold"
		p.innerText = `${event.srcHost}`
	} else {
		p.innerText = `${event.srcHost}:${event.dstPort} ${event.rule} `
		if (event.scanner) {
			p.innerText += event.scanner + ` `
		}
		if (event.payload) {
			const a = document.createElement("a")
			a.addEventListener('click', function () {
				content.innerHTML = `<pre>${hexy(atob(event.payload), hexFormat)}</pre>`;
			});
			a.innerText = "Payload"
			p.append(a)
		}
	}
	messageLog.append(p)
	return p
}

export function dial() {
	//const conn = new WebSocket(`ws://${location.host}/subscribe`)
	const conn = new WebSocket(`wss://${location.host}/subscribe`)

	conn.addEventListener("close", ev => {
		appendLog(`WebSocket Disconnected code: ${ev.code}, reason: ${ev.reason}`, true)
		if (ev.code !== 1001) {
			appendLog("Reconnecting in 1s", true)
			setTimeout(dial, 1000)
		}
	})
	conn.addEventListener("open", ev => {
		console.info("websocket connected")
	})
	conn.addEventListener("message", ev => {
		const obj = JSON.parse(ev.data);
		console.log(obj)
		const p = appendLog(obj,false)
		p.scrollIntoView()
	})
}

export default app;
