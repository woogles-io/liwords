import { HTML5Backend } from "react-dnd-html5-backend";
import { TouchBackend } from "react-dnd-touch-backend";
import {
  MultiBackend,
  TouchTransition,
  MouseTransition,
} from "react-dnd-multi-backend";

export const MultiBackendOptions = {
  backends: [
    {
      id: "touch",
      backend: TouchBackend,
      options: {
        enableMouseEvents: true,
        delayTouchStart: 0, // Remove delay for immediate response
      },
      transition: MouseTransition, // Use mouse transition for all devices
      preview: true,
    },
  ],
};

export default MultiBackend;
