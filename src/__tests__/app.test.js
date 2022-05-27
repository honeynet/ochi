/**
 * @jest-environment jsdom
 */

 import { render, screen } from "@testing-library/svelte";
 import App from "../App.svelte";
 
 describe("App", () => {
   test("app has 'Ochi' node", () => {
     render(App);
     const node = screen.queryByText("Ochi");
     expect(node).not.toBeNull();
   });
});
