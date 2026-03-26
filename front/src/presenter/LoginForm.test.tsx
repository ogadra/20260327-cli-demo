import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import LoginForm from "./LoginForm";

describe("LoginForm", () => {
  /** Mock onSuccess callback used across tests. */
  const onSuccess = vi.fn();

  beforeEach(() => {
    onSuccess.mockClear();
    vi.restoreAllMocks();
    sessionStorage.clear();
  });

  it("renders password input and submit button", () => {
    render(<LoginForm onSuccess={onSuccess} />);
    expect(screen.getByTestId("password-input")).toBeTruthy();
    expect(screen.getByTestId("login-button")).toBeTruthy();
    expect(screen.getByTestId("login-button").hasAttribute("disabled")).toBe(false);
  });

  it("calls onSuccess on successful login with status 200", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({ status: 200 });

    render(<LoginForm onSuccess={onSuccess} />);
    fireEvent.change(screen.getByTestId("password-input"), { target: { value: "secret" } });
    fireEvent.submit(screen.getByTestId("login-form"));

    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalledOnce();
    });

    expect(globalThis.fetch).toHaveBeenCalledWith("/login", {
      method: "POST",
      body: JSON.stringify({ password: "secret" }),
      credentials: "include",
      redirect: "manual",
    });
  });

  it("calls onSuccess on redirect status 302", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({ status: 302 });

    render(<LoginForm onSuccess={onSuccess} />);
    fireEvent.submit(screen.getByTestId("login-form"));

    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalledOnce();
    });
  });

  it("calls onSuccess on opaque redirect status 0", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({ status: 0 });

    render(<LoginForm onSuccess={onSuccess} />);
    fireEvent.submit(screen.getByTestId("login-form"));

    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalledOnce();
    });
  });

  it("shows Invalid password error on 401 response", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({ status: 401 });

    render(<LoginForm onSuccess={onSuccess} />);
    fireEvent.submit(screen.getByTestId("login-form"));

    await waitFor(() => {
      expect(screen.getByTestId("login-error").textContent).toBe("Invalid password");
    });

    expect(onSuccess).not.toHaveBeenCalled();
  });

  it("shows Login failed error on other status codes", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({ status: 500 });

    render(<LoginForm onSuccess={onSuccess} />);
    fireEvent.submit(screen.getByTestId("login-form"));

    await waitFor(() => {
      expect(screen.getByTestId("login-error").textContent).toBe("Login failed");
    });

    expect(onSuccess).not.toHaveBeenCalled();
  });

  it("shows Login failed error on fetch rejection", async () => {
    globalThis.fetch = vi.fn().mockRejectedValue(new Error("network error"));

    render(<LoginForm onSuccess={onSuccess} />);
    fireEvent.submit(screen.getByTestId("login-form"));

    await waitFor(() => {
      expect(screen.getByTestId("login-error").textContent).toBe("Login failed");
    });

    expect(onSuccess).not.toHaveBeenCalled();
  });

  it("disables button while loading", async () => {
    /** Promise that never resolves to keep the loading state active. */
    let resolvePromise: (value: { status: number }) => void;
    const pendingPromise = new Promise<{ status: number }>((resolve) => {
      resolvePromise = resolve;
    });
    globalThis.fetch = vi.fn().mockReturnValue(pendingPromise);

    render(<LoginForm onSuccess={onSuccess} />);
    fireEvent.submit(screen.getByTestId("login-form"));

    expect(screen.getByTestId("login-button").hasAttribute("disabled")).toBe(true);
    expect(screen.getByTestId("login-button").textContent).toBe("Logging in...");

    resolvePromise!({ status: 200 });

    await waitFor(() => {
      expect(screen.getByTestId("login-button").hasAttribute("disabled")).toBe(false);
    });
  });

  it("clears previous error on new submission", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({ status: 401 });

    render(<LoginForm onSuccess={onSuccess} />);
    fireEvent.submit(screen.getByTestId("login-form"));

    await waitFor(() => {
      expect(screen.getByTestId("login-error")).toBeTruthy();
    });

    globalThis.fetch = vi.fn().mockResolvedValue({ status: 200 });
    fireEvent.submit(screen.getByTestId("login-form"));

    await waitFor(() => {
      expect(screen.queryByTestId("login-error")).toBeNull();
    });
  });
});
