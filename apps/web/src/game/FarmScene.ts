import { Application, Container, Graphics, type FederatedPointerEvent } from "pixi.js";
import { palette } from "./art/palette";
import { drawGrassTuft, drawSoilPlot, drawTileHighlight } from "./art/tiles";
import { TILE_HEIGHT, gridToScreen, screenToGrid, type GridPoint } from "./iso";

export type FarmSceneOptions = {
  cols: number;
  rows: number;
};

export class FarmScene {
  private readonly app = new Application();
  private readonly world = new Container();
  private readonly highlight = drawTileHighlight();
  private initialized = false;
  private destroyed = false;

  constructor(private readonly options: FarmSceneOptions) {}

  async mount(host: HTMLElement): Promise<void> {
    await this.app.init({
      resizeTo: host,
      background: palette.grass,
      antialias: true,
      autoDensity: true,
      resolution: window.devicePixelRatio,
    });
    this.initialized = true;

    // React may unmount us while init is still pending.
    if (this.destroyed) {
      this.app.destroy(true, { children: true });
      return;
    }

    host.appendChild(this.app.canvas);
    this.app.stage.addChild(this.world);

    this.drawGround();
    this.drawPlots();

    this.highlight.visible = false;
    this.world.addChild(this.highlight);

    this.app.stage.eventMode = "static";
    this.app.stage.hitArea = this.app.screen;
    this.app.stage.on("pointermove", this.onPointerMove);
    this.app.stage.on("pointerleave", this.hideHighlight);

    this.app.renderer.on("resize", this.centerWorld);
    this.centerWorld();
  }

  destroy(): void {
    if (this.destroyed) return;
    this.destroyed = true;

    if (this.initialized) {
      this.app.renderer.off("resize", this.centerWorld);
      this.app.destroy(true, { children: true });
    }
  }

  private drawGround() {
    const { cols, rows } = this.options;
    const margin = 4;
    const tufts = new Container();

    for (let i = 0; i < 40; i++) {
      const tuft = drawGrassTuft();
      const { x, y } = gridToScreen({
        col: -margin + Math.random() * (cols + margin * 2),
        row: -margin + Math.random() * (rows + margin * 2),
      });
      tuft.position.set(x, y);
      tufts.addChild(tuft);
    }

    this.world.addChild(tufts);
  }

  private drawPlots() {
    const { cols, rows } = this.options;
    const top = gridToScreen({ col: 0, row: 0 });
    const right = gridToScreen({ col: cols, row: 0 });
    const bottom = gridToScreen({ col: cols, row: rows });
    const left = gridToScreen({ col: 0, row: rows });

    const bed = new Graphics()
      .poly([top.x, top.y - 4, right.x + 8, right.y, bottom.x, bottom.y + 14, left.x - 8, left.y])
      .fill(palette.soilBorder);
    this.world.addChild(bed);

    for (let row = 0; row < rows; row++) {
      for (let col = 0; col < cols; col++) {
        const plot = drawSoilPlot();
        const { x, y } = gridToScreen({ col, row });
        plot.position.set(x, y);
        this.world.addChild(plot);
      }
    }
  }

  private readonly onPointerMove = (e: FederatedPointerEvent) => {
    const tile = screenToGrid(this.world.toLocal(e.global));

    if (!this.isInside(tile)) {
      this.hideHighlight();
      return;
    }

    const { x, y } = gridToScreen(tile);
    this.highlight.position.set(x, y);
    this.highlight.visible = true;
  };

  private readonly hideHighlight = () => {
    this.highlight.visible = false;
  };

  private isInside({ col, row }: GridPoint) {
    return col >= 0 && row >= 0 && col < this.options.cols && row < this.options.rows;
  }

  private readonly centerWorld = () => {
    const { cols, rows } = this.options;
    const farmHeight = (cols + rows) * (TILE_HEIGHT / 2);
    const offsetX = ((rows - cols) * TILE_HEIGHT) / 2;

    this.world.position.set(
      this.app.screen.width / 2 + offsetX,
      (this.app.screen.height - farmHeight) / 2,
    );
  };
}
