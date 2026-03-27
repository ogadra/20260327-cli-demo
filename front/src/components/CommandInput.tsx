import type { ReactNode } from "react";
import { useCallback, useEffect, useRef, useState, type KeyboardEvent } from "react";

/** Props for the CommandInput component. */
interface Props {
  /** Callback invoked with the entered command string. */
  onSubmit: (command: string) => void;
  /** Whether the input is disabled. */
  disabled: boolean;
  /** Placeholder text shown in the input field. */
  placeholder?: string;
}

/** Text input with run button and command history navigable via arrow keys. */
const CommandInput = ({ onSubmit, disabled, placeholder }: Props): ReactNode => {
  const [value, setValue] = useState(placeholder ?? "");
  const inputRef = useRef<HTMLInputElement>(null);
  const historyRef = useRef<string[]>([]);
  const historyIndexRef = useRef(-1);

  useEffect(() => {
    if (!disabled) {
      inputRef.current?.focus();
    }
  }, [disabled]);

  useEffect(() => {
    setValue(placeholder ?? "");
  }, [placeholder]);

  const submitValue = useCallback(() => {
    const trimmed = value.trim();
    if (!trimmed) return;
    historyRef.current.push(trimmed);
    historyIndexRef.current = historyRef.current.length;
    onSubmit(trimmed);
    setValue("");
    requestAnimationFrame(() => inputRef.current?.focus());
  }, [value, onSubmit]);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent<HTMLInputElement>) => {
      if (e.key === "Enter") {
        submitValue();
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        if (historyIndexRef.current > 0) {
          historyIndexRef.current--;
          setValue(historyRef.current[historyIndexRef.current]);
        }
      } else if (e.key === "ArrowDown") {
        e.preventDefault();
        if (historyRef.current.length === 0) return;
        if (historyIndexRef.current < historyRef.current.length - 1) {
          historyIndexRef.current++;
          setValue(historyRef.current[historyIndexRef.current]);
        } else if (historyIndexRef.current < historyRef.current.length) {
          historyIndexRef.current = historyRef.current.length;
          setValue("");
        }
      }
    },
    [submitValue],
  );

  return (
    <div style={{ display: "flex", width: "100%", boxSizing: "border-box" }}>
      <input
        ref={inputRef}
        type="text"
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onKeyDown={handleKeyDown}
        disabled={disabled}
        placeholder={placeholder ?? "Enter command..."}
        style={{
          flex: 1,
          padding: "8px",
          fontFamily: "monospace",
          fontSize: "18px",
          boxSizing: "border-box",
          minWidth: 0,
        }}
      />
      <button
        type="button"
        onClick={submitValue}
        disabled={disabled || !value.trim()}
        aria-label="Run"
        style={{
          padding: "8px 16px",
          fontFamily: "monospace",
          fontSize: "18px",
          cursor: disabled || !value.trim() ? "default" : "pointer",
          whiteSpace: "nowrap",
        }}
      >
        Run
      </button>
    </div>
  );
};

export default CommandInput;
