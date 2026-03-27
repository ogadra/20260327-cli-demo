import type { ReactNode } from "react";
import type { SlideProps } from "../components/SlideView";
import { TerminalSlide } from "./TerminalSlide";

/** Hands-on slide with embedded terminal and command input. */
export const Slide0 = (_props: SlideProps): ReactNode => {
  return <TerminalSlide instruction="Try running a command" commands={["echo hello"]} />;
};
