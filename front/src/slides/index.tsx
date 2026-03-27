import type { ComponentType } from "react";
import type { SlideProps } from "../components/SlideView";
import { slideData } from "./slideData";
import { TextSlide } from "./TextSlide";
import { TitleSlide } from "./TitleSlide";
import { TerminalSlide } from "./TerminalSlide";
import { PollSlide } from "./PollSlide";

/** Build a slide component from a single SlideData entry. */
const buildSlide = (data: (typeof slideData)[number]): ComponentType<SlideProps> => {
  switch (data.type) {
    case "title":
      return () => <TitleSlide text={data.text} />;
    case "text":
      return () => <TextSlide lines={data.lines} />;
    case "terminal":
      return () => <TerminalSlide instruction={data.instruction} commands={data.commands} />;
    case "poll":
      return (props: SlideProps) => (
        <PollSlide
          pollId={data.pollId}
          question={data.question}
          options={data.options}
          {...props}
        />
      );
  }
};

/** Ordered array of slide components indexed by page number. */
export const slides: ReadonlyArray<ComponentType<SlideProps>> = slideData.map(buildSlide);
