import React from 'react';
import { useMountedState } from '../utils/mounted';

// Feature flag.
const drawingCanBeEnabled =
  localStorage?.getItem('enableScreenDrawing') === 'true';

export const makeDrawingHandlersSetterContext = () => {
  const keyDownHandlers = new Set<(evt: React.KeyboardEvent) => void>();

  return {
    setHandleKeyDown: (handler: (evt: React.KeyboardEvent) => void) => {
      keyDownHandlers.add(handler);
    },
    unsetHandleKeyDown: (handler: (evt: React.KeyboardEvent) => void) => {
      keyDownHandlers.delete(handler);
    },
    drawingCanBeEnabled, // Just a constant.
    handleKeyDown: (evt: React.KeyboardEvent) => {
      keyDownHandlers.forEach((handler) => handler(evt));
    },
  };
};

// Just abusing this as a global variable.
export const DrawingHandlersSetterContext = React.createContext(
  makeDrawingHandlersSetterContext()
);

export const useDrawing = () => {
  const { useState } = useMountedState();

  // Drawing functionalities.
  // Right-drag = draw.
  // RightClick several times = clear drawing.
  // Shift+RightClick = context menu.

  const canBeEnabled = drawingCanBeEnabled;

  const [isEnabledState, setIsEnabled] = useState(false);
  const isEnabled = canBeEnabled && isEnabledState;

  const boardEltRef = React.useRef<HTMLElement>();
  const [boardSize, setBoardSize] = useState({
    left: 0,
    top: 0,
    width: 1,
    height: 1,
  });
  const resizeFunc = React.useCallback(() => {
    const boardElt = boardEltRef.current;
    if (boardElt) {
      const { left, top, width, height } = boardElt.getBoundingClientRect();
      setBoardSize({
        left,
        top,
        width: Math.max(1, width),
        height: Math.max(1, height),
      });
    }
  }, []);
  const boardRef = React.useCallback(
    (elt) => {
      boardEltRef.current = elt;
      resizeFunc();
    },
    [resizeFunc]
  );
  React.useEffect(() => {
    if (!isEnabled) return;
    window.addEventListener('resize', resizeFunc);
    return () => window.removeEventListener('resize', resizeFunc);
  }, [isEnabled, resizeFunc]);
  const getXY = React.useCallback(
    (evt: React.MouseEvent): { x: number; y: number } => {
      const x = Math.max(
        0,
        Math.min(1, (evt.clientX - boardSize.left) / boardSize.width)
      );
      const y = Math.max(
        0,
        Math.min(1, (evt.clientY - boardSize.top) / boardSize.height)
      );
      return { x, y };
    },
    [boardSize]
  );
  const scaledXYStr = React.useCallback(
    ({ x, y }: { x: number; y: number }) =>
      `${x * boardSize.width},${y * boardSize.height}`,
    [boardSize.width, boardSize.height]
  );

  const [penColor, setPenColor] = useState('red');
  const [drawMode, setDrawMode] = useState('freehand');
  const boardResizedSinceLastPaintRef = React.useRef(true);
  const penRef = React.useRef<{ pen: string; mode: string }>();
  const strokesRef = React.useRef<
    Array<{
      points: Array<{ x: number; y: number }>; // scaled to [0,1)
      path: string; // Mx,yLx,yLx,y... based on current boardSize
      pen: string; // "red"
      mode: string; // "freehand"
      elt: React.ReactElement | undefined;
    }>
  >([]);
  const [currentDrawing, setCurrentDrawing] = useState<JSX.Element | undefined>(
    undefined
  );
  const plannedRepaintRef = React.useRef<number>();

  // For hopefully-unique id generation.
  const imagePrefixRef = React.useRef(`img${Date.now() + Math.random()}`);

  const repaintNow = React.useCallback(() => {
    if (plannedRepaintRef.current != null) {
      cancelAnimationFrame(plannedRepaintRef.current);
      plannedRepaintRef.current = undefined;
    }

    if (boardResizedSinceLastPaintRef.current) {
      // Rescale everything.
      for (const stroke of strokesRef.current) {
        let path = `M${scaledXYStr(stroke.points[0])}`;
        for (let i = 1; i < stroke.points.length; ++i) {
          path += `L${scaledXYStr(stroke.points[i])}`;
        }
        stroke.path = path;
        stroke.elt = undefined;
      }
      boardResizedSinceLastPaintRef.current = false;
    }

    for (let i = 0; i < strokesRef.current.length; ++i) {
      const stroke = strokesRef.current[i];
      if (!stroke.elt) {
        let path = stroke.path;
        const { x: x1, y: y1 } = stroke.points[0];
        const { x: x2, y: y2 } = stroke.points[stroke.points.length - 1];
        if (
          stroke.points.length === 1 ||
          (stroke.mode !== 'freehand' && x1 === x2 && y1 === y2)
        ) {
          // Draw a diamond to represent a single point.
          path += 'm-1,0l1,1l1,-1l-1,-1l-1,1l1,1';
        } else if (stroke.mode === 'line') {
          path = `M${scaledXYStr(stroke.points[0])}L${scaledXYStr(
            stroke.points[stroke.points.length - 1]
          )}`;
        } else if (stroke.mode === 'arrow') {
          // h12 = line length
          const h12 = Math.hypot(x2 - x1, y2 - y1);
          // h23 = arrow length before rotation
          const h23 = Math.min(0.5 * h12, 1 / 30);
          const h23oh12 = h23 / h12;
          // (x23,y23) = arrow endpoint relative to (x2,y2) before rotation
          const x23 = h23oh12 * (x2 - x1);
          const y23 = h23oh12 * (y2 - y1);
          // using rotation matrix
          //   [ cos(theta) -sin(theta) ][x] = [ x*cos(theta)-y*sin(theta) ]
          //   [ sin(theta) cos(theta)  ][y] = [ x*sin(theta)+y*cos(theta) ]
          const radPerDeg = Math.PI / 180;
          const angle1 = 135 * radPerDeg;
          const cos1 = Math.cos(angle1);
          const sin1 = Math.sin(angle1);
          const x3 = x2 + x23 * cos1 - y23 * sin1;
          const y3 = y2 + x23 * sin1 + y23 * cos1;
          const angle2 = -angle1;
          const cos2 = Math.cos(angle2);
          const sin2 = Math.sin(angle2);
          const x4 = x2 + x23 * cos2 - y23 * sin2;
          const y4 = y2 + x23 * sin2 + y23 * cos2;
          if (isFinite(x3) && isFinite(y3) && isFinite(x4) && isFinite(y4)) {
            path = `M${scaledXYStr(stroke.points[0])}L${scaledXYStr(
              stroke.points[stroke.points.length - 1]
            )}M${scaledXYStr({ x: x3, y: y3 })}L${scaledXYStr(
              stroke.points[stroke.points.length - 1]
            )}L${scaledXYStr({ x: x4, y: y4 })}`;
          } else {
            // sometimes there's NaN somewhere
            path = `M${scaledXYStr(stroke.points[0])}L${scaledXYStr(
              stroke.points[stroke.points.length - 1]
            )}`;
          }
        } else if (stroke.mode === 'quadrilateral') {
          path = `M${scaledXYStr(stroke.points[0])}L${scaledXYStr({
            x: x1,
            y: y2,
          })}L${scaledXYStr(
            stroke.points[stroke.points.length - 1]
          )}L${scaledXYStr({ x: x2, y: y1 })}Z`;
        } else if (stroke.mode === 'circle') {
          const cx = (x1 + x2) / 2;
          const cy = (y1 + y2) / 2;
          const rx = Math.abs(x2 - x1) / 2;
          const ry = Math.abs(y2 - y1) / 2;
          path = `M${scaledXYStr({ x: cx - rx, y: cy })}a${scaledXYStr({
            x: rx,
            y: ry,
          })},0,1,0,${scaledXYStr({ x: 2 * rx, y: 0 })}a${scaledXYStr({
            x: rx,
            y: ry,
          })},0,1,0,${scaledXYStr({ x: -2 * rx, y: 0 })}`;
        }
        stroke.elt =
          stroke.pen === 'erase' ? (
            <path key={i} d={path} fill="none" strokeWidth={5} stroke="black" />
          ) : (
            <path
              key={i}
              d={path}
              fill="none"
              strokeWidth={5}
              stroke={stroke.pen}
            />
          );
      }
    }

    let toDraw: Array<React.ReactElement> = [];
    let toErase: Array<React.ReactElement> = [];
    const eraseMasks: Array<React.ReactElement> = [];
    let numMasks = 0;
    for (let i = 0; i < strokesRef.current.length; ) {
      for (
        ;
        i < strokesRef.current.length && strokesRef.current[i].pen !== 'erase';
        ++i
      ) {
        toDraw.push(strokesRef.current[i].elt!);
      }
      for (
        ;
        i < strokesRef.current.length && strokesRef.current[i].pen === 'erase';
        ++i
      ) {
        toErase.push(strokesRef.current[i].elt!);
      }
      if (toErase.length > 0) {
        if (toDraw.length > 0) {
          // Otherwise nothing worth erasing.
          const maskId = `${imagePrefixRef.current}.${++numMasks}`;
          eraseMasks.push(
            <mask key={i - 1} id={maskId}>
              <rect
                width={boardSize.width}
                height={boardSize.height}
                fill="white"
              />
              {toErase}
            </mask>
          );
          toDraw = [
            <g key={i - 1} mask={`url(#${maskId})`}>
              {toDraw}
            </g>,
          ];
        }
        toErase = [];
      } else if (toDraw.length > 0) {
        toDraw = [<g key={i - 1}>{toDraw}</g>];
      }
    }

    const ret = (
      <React.Fragment>
        {eraseMasks}
        {toDraw}
      </React.Fragment>
    );

    setCurrentDrawing(ret);

    setPenColor((x) => {
      if (x === 'erase' && strokesRef.current.length === 0) {
        return 'red'; // Deactivate eraser when no drawing.
      }
      return x;
    });
  }, [scaledXYStr, boardSize.width, boardSize.height]);

  const scheduleRepaint = React.useCallback(() => {
    if (plannedRepaintRef.current != null) {
      cancelAnimationFrame(plannedRepaintRef.current);
    }
    plannedRepaintRef.current = requestAnimationFrame(repaintNow);
  }, [repaintNow]);

  const handleContextMenu = React.useCallback((evt: React.MouseEvent) => {
    if (!evt.shiftKey) {
      // Draw when not holding shift.
      evt.preventDefault();
    } else {
      // Shift+RightClick accesses context menu.
    }
  }, []);

  const handleMouseDown = React.useCallback(
    (evt: React.MouseEvent) => {
      if (evt.button === 2 && !evt.shiftKey) {
        const newXY = getXY(evt);
        penRef.current = { pen: penColor, mode: drawMode };
        strokesRef.current.push({
          points: [newXY],
          path: `M${scaledXYStr(newXY)}`,
          pen: penRef.current.pen,
          mode: penRef.current.mode,
          elt: undefined,
        });
        scheduleRepaint();
      }
    },
    [penColor, drawMode, getXY, scaledXYStr, scheduleRepaint]
  );

  const handleMouseUp = React.useCallback(
    (evt: React.MouseEvent) => {
      if (!penRef.current) return;
      // Right-click this many times to clear drawing.
      const howMany = 3;
      if (strokesRef.current.length >= howMany) {
        const lastPoint =
          strokesRef.current[strokesRef.current.length - 1].points[0];
        let i = 0;
        for (; i < howMany; ++i) {
          const ithLastPoints =
            strokesRef.current[strokesRef.current.length - (i + 1)].points;
          if (
            !(
              ithLastPoints.length < 2 &&
              ithLastPoints[0].x === lastPoint.x &&
              ithLastPoints[0].y === lastPoint.y
            )
          )
            break;
        }
        if (i === howMany) strokesRef.current = [];
      }
      penRef.current = undefined;
      scheduleRepaint();
    },
    [scheduleRepaint]
  );

  const handleMouseMove = React.useCallback(
    (evt: React.MouseEvent) => {
      if (!penRef.current) return;
      const newXY = getXY(evt);
      const lastStroke = strokesRef.current[strokesRef.current.length - 1];
      const lastPoints = lastStroke.points;
      const lastPoint = lastPoints[lastPoints.length - 1];
      if (lastPoint.x === newXY.x && lastPoint.y === newXY.y) return;
      lastPoints.push(newXY);
      lastStroke.path += `L${scaledXYStr(newXY)}`;
      lastStroke.elt = undefined; // will be recomputed later
      scheduleRepaint();
    },
    [getXY, scaledXYStr, scheduleRepaint]
  );

  const handlePointerDown = React.useCallback((evt: React.PointerEvent) => {
    (evt.target as Element).setPointerCapture(evt.pointerId);
  }, []);
  const handlePointerUp = React.useCallback(
    (evt: React.PointerEvent) => {
      (evt.target as Element).releasePointerCapture(evt.pointerId);
      handleMouseUp(evt);
    },
    [handleMouseUp]
  );

  React.useEffect(() => {
    // Board size changed, invalidate path.
    if (strokesRef.current.length > 0) {
      boardResizedSinceLastPaintRef.current = true;
      scheduleRepaint();
    }
  }, [scaledXYStr, scheduleRepaint]);

  React.useEffect(() => {
    // Auto pen-up if disabling.
    if (!isEnabled && penRef.current) {
      penRef.current = undefined;
      scheduleRepaint();
    }
  }, [isEnabled, scheduleRepaint]);

  const handleKeyDown = React.useCallback(
    (evt: React.KeyboardEvent) => {
      if (evt.ctrlKey || evt.altKey || evt.metaKey) {
        return;
      }
      const key = evt.key.toUpperCase();
      if (key === '0') {
        // Toggle drawing.
        setIsEnabled((x) => !x);
      } else if (isEnabled) {
        if (key === 'F') {
          setDrawMode('freehand');
        }
        if (key === 'L') {
          setDrawMode('line');
        }
        if (key === 'A') {
          setDrawMode('arrow');
        }
        if (key === 'Q') {
          setDrawMode('quadrilateral');
        }
        if (key === 'C') {
          setDrawMode('circle');
        }
        if (key === 'R') {
          setPenColor('red');
        }
        if (key === 'G') {
          setPenColor('green');
        }
        if (key === 'B') {
          setPenColor('blue');
        }
        if (key === 'Y') {
          setPenColor('yellow');
        }
        if (key === 'E') {
          setPenColor('erase');
        }
        if (key === 'U') {
          // Undo.
          strokesRef.current.pop();
          penRef.current = undefined;
          scheduleRepaint();
        }
        if (key === 'W') {
          // Wipe.
          strokesRef.current = [];
          penRef.current = undefined;
          scheduleRepaint();
        }
      }
    },
    [isEnabled, scheduleRepaint]
  );

  const { setHandleKeyDown, unsetHandleKeyDown } = React.useContext(
    DrawingHandlersSetterContext
  );

  // Register handlers for board_panel to call.
  React.useEffect(() => {
    if (canBeEnabled) {
      setHandleKeyDown(handleKeyDown);
      return () => unsetHandleKeyDown(handleKeyDown);
    }
  }, [canBeEnabled, handleKeyDown, setHandleKeyDown, unsetHandleKeyDown]);

  // Instructions text for now, until there's a better UI.
  React.useEffect(() => {
    if (canBeEnabled) {
      if (isEnabled) {
        console.log('Drawing enabled.');
      } else {
        console.log('Drawing disabled. To enable, type 00.');
      }
    }
  }, [canBeEnabled, isEnabled]);
  React.useEffect(() => {
    if (canBeEnabled) {
      if (isEnabled) {
        console.log(
          `Pen color: ${penColor}. Mode: ${drawMode}. To draw on the board, use the right mouse button. For menu, press 0.`
        );
      }
    }
  }, [canBeEnabled, isEnabled, penColor, drawMode]);

  const outerDivProps = React.useMemo(
    () =>
      isEnabled
        ? {
            ref: boardRef,
            onContextMenu: handleContextMenu,
            onMouseDown: handleMouseDown,
            onMouseUp: handleMouseUp,
            onMouseMove: handleMouseMove,
            onPointerDown: handlePointerDown,
            onPointerUp: handlePointerUp,
          }
        : {},
    [
      isEnabled,
      boardRef,
      handleContextMenu,
      handleMouseDown,
      handleMouseUp,
      handleMouseMove,
      handlePointerDown,
      handlePointerUp,
    ]
  );
  const svgProps: React.SVGProps<SVGSVGElement> = React.useMemo(
    () =>
      isEnabled
        ? {
            viewBox: `0 0 ${boardSize.width} ${boardSize.height}`,
            style: {
              position: 'absolute',
              left: 0,
              top: 0,
              width: boardSize.width,
              height: boardSize.height,
              pointerEvents: 'none',
            },
          }
        : {},
    [isEnabled, boardSize.width, boardSize.height]
  );
  const svgDrawing = React.useMemo(
    () =>
      isEnabled && currentDrawing ? (
        <svg {...svgProps}>{currentDrawing}</svg>
      ) : null,
    [isEnabled, svgProps, currentDrawing]
  );
  const ret = React.useMemo(() => ({ outerDivProps, svgDrawing }), [
    outerDivProps,
    svgDrawing,
  ]);
  return ret;
};
