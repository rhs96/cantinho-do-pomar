import { Graphics } from "pixi.js";
import { TILE_HEIGHT, TILE_WIDTH } from "../iso";
import { palette } from "./palette";

const SOIL_DEPTH = 10;
const SOIL_INSET = 6;

export function drawSoilPlot(): Graphics {
  const g = new Graphics();
  const hw = TILE_WIDTH / 2 - SOIL_INSET * 2;
  const hh = TILE_HEIGHT / 2 - SOIL_INSET;
  const cy = TILE_HEIGHT / 2;

  g.poly([-hw, cy, 0, cy + hh, 0, cy + hh + SOIL_DEPTH, -hw, cy + SOIL_DEPTH]).fill(palette.soilSideLeft);
  g.poly([hw, cy, 0, cy + hh, 0, cy + hh + SOIL_DEPTH, hw, cy + SOIL_DEPTH]).fill(palette.soilSideRight);
  g.poly([0, cy - hh, hw, cy, 0, cy + hh, -hw, cy]).fill(palette.soilTop);

  // Furrows run parallel to the top-left edge.
  for (let i = 1; i <= 3; i++) {
    const t = i / 4;
    g.moveTo(-hw + hw * t, cy - hh * t)
      .lineTo(hw * t, cy + hh - hh * t)
      .stroke({ width: 3, color: palette.soilFurrow, cap: "round" });
  }

  return g;
}

export function drawTileHighlight(): Graphics {
  const hw = TILE_WIDTH / 2 - 4;
  const hh = TILE_HEIGHT / 2 - 2;
  const cy = TILE_HEIGHT / 2;

  return new Graphics()
    .poly([0, cy - hh, hw, cy, 0, cy + hh, -hw, cy])
    .stroke({ width: 3, color: palette.highlight, alpha: 0.9 });
}

export function drawGrassTuft(): Graphics {
  const g = new Graphics();
  g.moveTo(-4, 0).quadraticCurveTo(-6, -8, -9, -12);
  g.moveTo(0, 0).quadraticCurveTo(0, -10, 1, -15);
  g.moveTo(4, 0).quadraticCurveTo(6, -7, 9, -11);
  return g.stroke({ width: 2, color: palette.grassDark, cap: "round" });
}
