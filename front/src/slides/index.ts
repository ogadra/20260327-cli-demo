import type { ComponentType } from "react";
import type { SlideProps } from "../components/SlideView";
import Slide0 from "./Slide0";
import Slide1 from "./Slide1";
import Slide2 from "./Slide2";

/** Ordered array of slide components indexed by page number. */
const slides: ReadonlyArray<ComponentType<SlideProps>> = [Slide0, Slide1, Slide2];

export default slides;
