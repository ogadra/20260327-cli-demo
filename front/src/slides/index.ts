import type { ComponentType } from "react";
import type { SlideProps } from "../components/SlideView";
import Slide0 from "./Slide0";

/** Ordered array of slide components indexed by page number. */
const slides: ReadonlyArray<ComponentType<SlideProps>> = [Slide0];

export default slides;
