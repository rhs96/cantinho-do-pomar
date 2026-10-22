"use client";

import { useEffect, useRef } from "react";
import styles from "./FarmCanvas.module.css";

export function FarmCanvas() {
  const hostRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const host = hostRef.current;
    if (!host) return;

    let destroy: (() => void) | undefined;
    let cancelled = false;

    import("@/game/FarmScene").then(({ FarmScene }) => {
      if (cancelled) return;
      const scene = new FarmScene({ cols: 6, rows: 6 });
      destroy = () => scene.destroy();
      void scene.mount(host);
    });

    return () => {
      cancelled = true;
      destroy?.();
    };
  }, []);

  return <div ref={hostRef} className={styles.host} />;
}
