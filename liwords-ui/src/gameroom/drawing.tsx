import React from 'react';

export const useDrawing = (isEnabled: boolean) => {
  // Drawing functionalities.
  // Right-drag = draw.
  // RightClick several times = clear drawing.
  // Shift+RightClick = clear drawing.
  // Shift+RightClick (when no drawing) = context menu.

  const boardEltRef = React.useRef<HTMLElement>();
  const [boardSize, setBoardSize] = React.useState({
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
  const [picture, setPicture] = React.useState<{
    drawing: boolean;
    strokes: Array<{
      points: Array<{ x: number; y: number }>; // scaled to [0,1)
      path: string; // Mx,yLx,yLx,y... based on current boardSize
    }>;
  }>({ drawing: false, strokes: [] });
  const hasPicture = picture.strokes.length > 0;
  const handleContextMenu = React.useCallback(
    (evt: React.MouseEvent) => {
      if (!evt.shiftKey) {
        // Draw when not holding shift.
        evt.preventDefault();
      } else if (hasPicture) {
        // Shift+RightClick clears drawing.
        setPicture((pic) => ({ ...pic, drawing: false, strokes: [] }));
        evt.preventDefault();
      } else {
        // Shift+RightClick accesses context menu if no drawing.
      }
    },
    [hasPicture]
  );
  const scaledXYStr = React.useCallback(
    ({ x, y }: { x: number; y: number }) =>
      `${x * boardSize.width},${y * boardSize.height}`,
    [boardSize.width, boardSize.height]
  );
  const handleMouseDown = React.useCallback(
    (evt: React.MouseEvent) => {
      if (evt.button === 2 && !evt.shiftKey) {
        const newXY = getXY(evt);
        setPicture((pic) => {
          pic.strokes.push({ points: [newXY], path: `M${scaledXYStr(newXY)}` }); // mutate
          return { ...pic, drawing: true }; // shallow clone for performance
        });
      }
    },
    [getXY, scaledXYStr]
  );
  const handleMouseUp = React.useCallback((evt: React.MouseEvent) => {
    setPicture((pic) => {
      if (!pic.drawing) return pic;
      // Right-click this many times to clear drawing.
      const howMany = 3;
      if (pic.strokes.length >= howMany) {
        const lastPoint = pic.strokes[pic.strokes.length - 1].points[0];
        let i = 0;
        for (; i < howMany; ++i) {
          const ithLastPoints =
            pic.strokes[pic.strokes.length - (i + 1)].points;
          if (
            !(
              ithLastPoints.length < 2 &&
              ithLastPoints[0].x === lastPoint.x &&
              ithLastPoints[0].y === lastPoint.y
            )
          )
            break;
        }
        if (i === howMany) {
          return { ...pic, drawing: false, strokes: [] };
        }
      }
      return { ...pic, drawing: false };
    });
  }, []);
  const handleMouseMove = React.useCallback(
    (evt: React.MouseEvent) => {
      const newXY = getXY(evt);
      setPicture((pic) => {
        if (!pic.drawing) return pic;
        const lastStroke = pic.strokes[pic.strokes.length - 1];
        const lastPoints = lastStroke.points;
        const lastPoint = lastPoints[lastPoints.length - 1];
        if (lastPoint.x === newXY.x && lastPoint.y === newXY.y) return pic;
        lastPoints.push(newXY); // mutate
        lastStroke.path += `L${scaledXYStr(newXY)}`;
        return { ...pic }; // shallow clone for performance
      });
    },
    [getXY, scaledXYStr]
  );
  const handlePointerDown = React.useCallback((evt: React.PointerEvent) => {
    (evt.target as Element).setPointerCapture(evt.pointerId);
  }, []);
  const handlePointerUp = React.useCallback((evt: React.PointerEvent) => {
    (evt.target as Element).releasePointerCapture(evt.pointerId);
  }, []);
  React.useEffect(() => {
    setPicture((pic) => {
      // Board size changed, recompute path.
      for (const stroke of pic.strokes) {
        let path = `M${scaledXYStr(stroke.points[0])}`;
        for (let i = 1; i < stroke.points.length; ++i) {
          path += `L${scaledXYStr(stroke.points[i])}`;
        }
        stroke.path = path; // mutate
      }
      return { ...pic }; // shallow clone for performance
    });
  }, [scaledXYStr]);
  const currentDrawing = React.useMemo(() => {
    let path = '';
    for (const { points, path: thisPath } of picture.strokes) {
      path += thisPath;
      if (points.length === 1) {
        // Draw a diamond to represent a single point.
        path += 'm-1,0l1,1l1,-1l-1,-1l-1,1l1,1';
      }
    }
    return <path d={path} fill="none" strokeWidth={5} stroke="red" />;
  }, [picture]);

  return {
    outerDivProps: isEnabled
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
    svgDrawing: isEnabled ? (
      <svg
        viewBox={`0 0 ${boardSize.width} ${boardSize.height}`}
        style={{
          position: 'absolute',
          left: 0,
          top: 0,
          width: boardSize.width,
          height: boardSize.height,
          pointerEvents: 'none',
        }}
      >
        {currentDrawing}
      </svg>
    ) : null,
  };
};
