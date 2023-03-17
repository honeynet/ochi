import { writable } from "svelte/store";

const storedIsAuthenticated = localStorage.getItem("isAuthenticated");
export const isAuthenticated = writable(storedIsAuthenticated === "true");
isAuthenticated.subscribe((value) => {
    localStorage.setItem("isAuthenticated", value === true ? "true" : "false");
});

export const user = writable({});

const storedToken = localStorage.getItem("token");
export const token = writable(storedToken);
token.subscribe((value) => {
    localStorage.setItem("token", value === null ? "" : value);
});
