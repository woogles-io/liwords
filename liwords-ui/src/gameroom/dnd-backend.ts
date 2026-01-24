import { TouchBackend } from "react-dnd-touch-backend";
import { MultiBackend, MouseTransition } from "react-dnd-multi-backend";

export const MultiBackendOptions = {
  backends: [
    {
      id: "touch",
      backend: TouchBackend,
      options: {
        enableMouseEvents: true,
        delayTouchStart: 0, // Remove delay for immediate response
        delayMouseStart: 0, // Remove delay for mouse events
        touchSlop: 8, // Increase tolerance for small movements during taps
        ignoreContextMenu: true, // Ignore context menu for faster response
        tapTolerance: 0, // Immediate tap recognition
        enableHover: false, // Disable hover to reduce event complexity
      },
      transition: MouseTransition, // Use mouse transition for all devices
      preview: true,
    },
  ],
};

export default MultiBackend;
