import { describe, expect, it } from "vitest";
import { TILE_HEIGHT, TILE_WIDTH, gridToScreen, screenToGrid, tileCenter } from "./iso";

describe("gridToScreen", () => {
  it("places the origin tile at 0,0", () => {
    expect(gridToScreen({ col: 0, row: 0 })).toEqual({ x: 0, y: 0 });
  });

  it("moves right and down along a column", () => {
    expect(gridToScreen({ col: 1, row: 0 })).toEqual({ x: TILE_WIDTH / 2, y: TILE_HEIGHT / 2 });
  });

  it("moves left and down along a row", () => {
    expect(gridToScreen({ col: 0, row: 1 })).toEqual({ x: -TILE_WIDTH / 2, y: TILE_HEIGHT / 2 });
  });
});

describe("screenToGrid", () => {
  it("round-trips tile centers", () => {
    for (let col = 0; col < 8; col++) {
      for (let row = 0; row < 8; row++) {
        expect(screenToGrid(tileCenter({ col, row }))).toEqual({ col, row });
      }
    }
  });

  it("resolves points just inside the diamond corners", () => {
    const c = tileCenter({ col: 2, row: 3 });

    expect(screenToGrid({ x: c.x, y: c.y - TILE_HEIGHT / 2 + 2 })).toEqual({ col: 2, row: 3 });
    expect(screenToGrid({ x: c.x + TILE_WIDTH / 2 - 4, y: c.y })).toEqual({ col: 2, row: 3 });
    expect(screenToGrid({ x: c.x - TILE_WIDTH / 2 + 4, y: c.y })).toEqual({ col: 2, row: 3 });
  });

  it("returns negative coordinates above the grid", () => {
    expect(screenToGrid({ x: 0, y: -10 })).toEqual({ col: -1, row: -1 });
  });
});
