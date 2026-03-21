import { useCallback, useRef, useState, type KeyboardEvent } from "react";

interface Props {
  onSubmit: (command: string) => void;
  disabled: boolean;
}

const CommandInput = ({ onSubmit, disabled }: Props) => {
  const [value, setValue] = useState("");
  const historyRef = useRef<string[]>([]);
  const historyIndexRef = useRef(-1);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent<HTMLInputElement>) => {
      if (e.key === "Enter" && value.trim()) {
        historyRef.current.push(value);
        historyIndexRef.current = historyRef.current.length;
        onSubmit(value);
        setValue("");
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        if (historyIndexRef.current > 0) {
          historyIndexRef.current--;
          setValue(historyRef.current[historyIndexRef.current]);
        }
      } else if (e.key === "ArrowDown") {
        e.preventDefault();
        if (historyIndexRef.current < historyRef.current.length - 1) {
          historyIndexRef.current++;
          setValue(historyRef.current[historyIndexRef.current]);
        } else {
          historyIndexRef.current = historyRef.current.length;
          setValue("");
        }
      }
    },
    [value, onSubmit],
  );

  return (
    <input
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
