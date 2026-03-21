import { forwardRef, useEffect, useImperativeHandle, useRef } from "react";
import { Terminal as XTerm } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";

/** Imperative handle exposed by the Terminal component via ref. */
export interface TerminalHandle {
  /** Write raw data to the terminal. */
  write(data: string): void;
  /** Write data followed by a newline to the terminal. */
  writeln(data: string): void;
}

/** xterm.js terminal component with auto-fit on resize. */
const Terminal = forwardRef<TerminalHandle>((_, ref) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<XTerm | null>(null);

  useImperativeHandle(
    ref,
    () => ({
      write: (data: string) => {
        xtermRef.current?.write(data);
      },
      writeln: (data: string) => {
        xtermRef.current?.writeln(data);
      },
    }),
    [],
  );

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;

    const xterm = new XTerm({ convertEol: true });
    const fitAddon = new FitAddon();
    xterm.loadAddon(fitAddon);
    xterm.open(el);
    fitAddon.fit();
    xterm.write("$ ");
    xtermRef.current = xterm;

    const onResize = () => fitAddon.fit();
    window.addEventListener("resize", onResize);

    return () => {
      window.removeEventListener("resize", onResize);
      xterm.dispose();
      xtermRef.current = null;
    };
  }, []);

  return <div ref={containerRef} style={{ width: "100%", flexGrow: 1 }} />;
});

Terminal.displayName = "Terminal";

export default Terminal;
