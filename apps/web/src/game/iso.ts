export const TILE_WIDTH = 128;
export const TILE_HEIGHT = 64;

export type GridPoint = { col: number; row: number };
export type ScreenPoint = { x: number; y: number };

// Returns the top vertex of the tile's diamond.
export function gridToScreen({ col, row }: GridPoint): ScreenPoint {
  return {
    x: (col - row) * (TILE_WIDTH / 2),
    y: (col + row) * (TILE_HEIGHT / 2),
  };
}

export function screenToGrid({ x, y }: ScreenPoint): GridPoint {
  return {
    col: Math.floor(y / TILE_HEIGHT + x / TILE_WIDTH),
    row: Math.floor(y / TILE_HEIGHT - x / TILE_WIDTH),
  };
}

export function tileCenter(p: GridPoint): ScreenPoint {
  const top = gridToScreen(p);
  return { x: top.x, y: top.y + TILE_HEIGHT / 2 };
}
