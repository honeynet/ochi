
import { isAuthenticated, user, token } from "./store";

let jwt: string
token.subscribe(value => {
    jwt = value
});

export async function validate() {
    if (jwt !== "") {
        const res = await fetch("/session", {
            method: "POST",
            body: jwt,
        });

        if (res.ok) {
            const json = await res.json();
			console.log(json);
			login(json);
        } else {
            console.log("failed to validate");
            logout()
        }
    }
}

export function logout() {
    isAuthenticated.set(false);
    user.set({});
    token.set("");
}

export function login(data:any) {
    user.set(data["user"]);
    token.set(data["token"]);
	isAuthenticated.set(true);
}