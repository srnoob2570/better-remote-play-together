import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LoadingButton } from "./loading-button";

describe("LoadingButton", () => {
  it("keeps the action label and shows a spinner while loading", () => {
    render(<LoadingButton loading>Guardar</LoadingButton>);

    const button = screen.getByRole("button", { name: "Guardar" });
    expect(button).toBeDisabled();
    expect(button).toHaveAttribute("aria-busy", "true");
    expect(button.querySelector('[data-slot="spinner"]')).toBeInTheDocument();
  });

  it("shows no spinner when idle", () => {
    render(<LoadingButton>Guardar</LoadingButton>);

    const button = screen.getByRole("button", { name: "Guardar" });
    expect(button).toBeEnabled();
    expect(button).not.toHaveAttribute("aria-busy");
    expect(button.querySelector('[data-slot="spinner"]')).not.toBeInTheDocument();
  });
});
