export const colRowGridStyle = (gridDim: number) => {
  return {
    gridTemplateColumns: `repeat(${gridDim}, 1fr)`,
    gridTemplateRows: `repeat(${gridDim}, 1fr)`,
  };
};
