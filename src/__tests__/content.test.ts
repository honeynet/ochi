/**
 * @jest-environment jsdom
 */

import { render, screen } from "@testing-library/svelte";
import Content from "../Content.svelte";

describe("Content", () => {
  test("render download button", () => {
    const { getByText } = render(Content, { props: { content: { payload: "test" } } });
    const node = getByText("Download")
    expect(node).not.toBeNull();
  });
});