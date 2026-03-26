import { type FormEvent, type ReactNode, useCallback, useState } from "react";

/** Props for the LoginForm component. */
export interface LoginFormProps {
  /** Callback invoked when login succeeds. */
  onSuccess: () => void;
}

/**
 * Login form component for the presenter page.
 * Submits a password to the /login endpoint and calls onSuccess on success.
 */
const LoginForm = ({ onSuccess }: LoginFormProps): ReactNode => {
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  /** Handles form submission by posting credentials to the login endpoint. */
  const handleSubmit = useCallback(
    (e: FormEvent): void => {
      e.preventDefault();
      setError("");
      setLoading(true);

      fetch("/login", {
        method: "POST",
        body: JSON.stringify({ password }),
        credentials: "include",
        redirect: "manual",
      })
        .then((res): void => {
          if (res.status === 0 || res.status === 302 || res.status === 200) {
            onSuccess();
          } else if (res.status === 401) {
            setError("Invalid password");
          } else {
            setError("Login failed");
          }
        })
        .catch((): void => {
          setError("Login failed");
        })
        .finally((): void => {
          setLoading(false);
        });
    },
    [password, onSuccess],
  );

  return (
    <div
      data-testid="login-form-container"
      style={{
        background: "#1a1a1a",
        color: "#fff",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        minHeight: "100vh",
        fontFamily: "sans-serif",
      }}
    >
      <form onSubmit={handleSubmit} data-testid="login-form" style={{ width: "300px" }}>
        <h1 style={{ fontSize: "24px", marginBottom: "24px", textAlign: "center" }}>
          Presenter Login
        </h1>
        <input
          data-testid="password-input"
          type="password"
          value={password}
          onChange={(e): void => setPassword(e.target.value)}
          placeholder="Password"
          style={{
            width: "100%",
            padding: "12px",
            marginBottom: "12px",
            background: "#333",
            color: "#fff",
            border: "1px solid #555",
            borderRadius: "4px",
            fontSize: "16px",
            boxSizing: "border-box",
          }}
        />
        {error && (
          <div
            data-testid="login-error"
            style={{ color: "#f66", marginBottom: "12px", fontSize: "14px" }}
          >
            {error}
          </div>
        )}
        <button
          data-testid="login-button"
          type="submit"
          disabled={loading}
          style={{
            width: "100%",
            padding: "12px",
            background: loading ? "#333" : "#555",
            color: loading ? "#666" : "#fff",
            border: "none",
            borderRadius: "4px",
            cursor: loading ? "not-allowed" : "pointer",
            fontSize: "16px",
          }}
        >
          {loading ? "Logging in..." : "Login"}
        </button>
      </form>
    </div>
  );
};

export default LoginForm;
