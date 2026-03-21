import type { ReactNode } from "react";
import { useCallback, useEffect, useRef, useState, type KeyboardEvent } from "react";

/** Props for the CommandInput component. */
interface Props {
  /** Callback invoked with the entered command string. */
  onSubmit: (command: string) => void;
  /** Whether the input is disabled. */
  disabled: boolean;
}

/** Text input with command history navigable via arrow keys. */
const CommandInput = ({ onSubmit, disabled }: Props): ReactNode => {
  const [value, setValue] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);
  const historyRef = useRef<string[]>([]);
  const historyIndexRef = useRef(-1);

  useEffect(() => {
    if (!disabled) {
      inputRef.current?.focus();
    }
  }, [disabled]);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent<HTMLInputElement>) => {
      const trimmed = value.trim();
      if (e.key === "Enter" && trimmed) {
        historyRef.current.push(trimmed);
        historyIndexRef.current = historyRef.current.length;
        onSubmit(trimmed);
        setValue("");
        requestAnimationFrame(() => inputRef.current?.focus());
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
    [value, onSubmit],
  );

  return (
    <input
      ref={inputRef}
      type="text"
      value={value}
      onChange={(e) => setValue(e.target.value)}
      onKeyDown={handleKeyDown}
      disabled={disabled}
      placeholder="Enter command..."
      style={{
        width: "100%",
        padding: "8px",
        fontFamily: "monospace",
        fontSize: "14px",
        boxSizing: "border-box",
      }}
    />
  );
};

export default CommandInput;
