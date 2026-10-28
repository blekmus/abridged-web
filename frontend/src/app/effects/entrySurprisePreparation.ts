import { preloadSurpriseAudio } from "./surpriseCelebration";

type SceneModule = typeof import("./entrySurpriseScene");
let prepared: Promise<SceneModule> | undefined;
let owners = 0;

export function prepareEntrySurprise(): Promise<SceneModule> {
  prepared ??= import("./entrySurpriseScene")
    .then(async (module) => {
      await module.prewarmEntrySurprise();
      return module;
    })
    .catch((error: unknown) => {
      prepared = undefined;
      throw error;
    });
  return prepared;
}

/** Prepare after search opens, retaining warmed GPU resources for this page. */
export function scheduleEntrySurprisePreparation(): () => void {
  owners += 1;
  let cancelled = false;
  const start = () => {
    if (cancelled) return;
    void preloadSurpriseAudio();
    if (!window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
      void prepareEntrySurprise().catch(() => {});
    }
  };
  let cancelScheduled: () => void;
  if (typeof window.requestIdleCallback === "function") {
    const idle = window.requestIdleCallback(start, { timeout: 1500 });
    cancelScheduled = () => window.cancelIdleCallback(idle);
  } else {
    const timer = window.setTimeout(start, 300);
    cancelScheduled = () => window.clearTimeout(timer);
  }
  return () => {
    if (cancelled) return;
    cancelled = true;
    cancelScheduled();
    owners -= 1;
    // Let child effects dispose their active scenes before releasing the shared context.
    queueMicrotask(() => {
      if (owners !== 0) return;
      const pending = prepared;
      if (!pending) return;
      void pending
        .then((module) => {
          if (owners !== 0 || pending !== prepared) return;
          module.disposeEntrySurprisePreparation();
          prepared = undefined;
        })
        .catch(() => {});
    });
  };
}
