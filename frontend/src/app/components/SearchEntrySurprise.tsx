import { useLayoutEffect, useRef, useState } from "preact/hooks";
import { prepareEntrySurprise } from "../effects/entrySurprisePreparation";
import type { SurpriseStage } from "../effects/entrySurpriseScene";
import { startSurpriseCelebration } from "../effects/surpriseCelebration";
import "../../styles/search-surprise.css";

type Phase = "pressed" | "flattening" | "bridging" | SurpriseStage | "fallback";

export function SearchEntrySurprise({
  onBegin,
  onCheerStart,
}: {
  onBegin?: () => void;
  onCheerStart?: () => void;
} = {}) {
  const [phase, setPhase] = useState<Phase>("pressed");
  const sourceRef = useRef<HTMLButtonElement>(null);
  const hostRef = useRef<HTMLSpanElement>(null);
  const generationRef = useRef(0);
  const runningRef = useRef(false);
  const cleanupRef = useRef<(() => void) | undefined>();

  useLayoutEffect(
    () => () => {
      generationRef.current += 1;
      cleanupRef.current?.();
    },
    [],
  );

  const reveal = () => {
    if (runningRef.current || phase !== "pressed") return;
    const source = sourceRef.current;
    const host = hostRef.current;
    if (!source || !host) return;
    onBegin?.();
    runningRef.current = true;
    cleanupRef.current?.();
    const generation = ++generationRef.current;
    const reducedMotion = window.matchMedia(
      "(prefers-reduced-motion: reduce)",
    ).matches;
    let disposed = false;
    const current = () => !disposed && generationRef.current === generation;
    // A new user gesture replaces the previous celebration; panel cleanup lets it finish.
    const celebration = startSurpriseCelebration(() => {
      if (current()) onCheerStart?.();
    }, !reducedMotion);
    let scene:
      | { dispose: () => void; present: () => void; start: () => void }
      | undefined;
    let observer: ResizeObserver | undefined;
    let flattenFrame = 0;
    let bridgeFrame = 0;
    let handoffTimer: number | undefined;
    let finishFlattening = () => {};
    const release = () => {
      if (!current()) return;
      celebration.release();
    };
    const fallback = () => {
      if (!current()) return;
      scene?.dispose();
      scene = undefined;
      observer?.disconnect();
      setPhase("fallback");
      runningRef.current = false;
    };
    cleanupRef.current = () => {
      disposed = true;
      cancelAnimationFrame(flattenFrame);
      cancelAnimationFrame(bridgeFrame);
      if (handoffTimer !== undefined) window.clearTimeout(handoffTimer);
      finishFlattening();
      observer?.disconnect();
      scene?.dispose();
      celebration.finish();
    };
    if (reducedMotion) {
      release();
      fallback();
      return;
    }
    setPhase("flattening");
    const flattened = new Promise<void>((resolve) => {
      finishFlattening = resolve;
      flattenFrame = requestAnimationFrame(() => {
        if (!current()) {
          resolve();
          return;
        }
        // Blend into 3D before flattening ends, so the two surfaces never pause apart.
        source.getBoundingClientRect();
        const transitions = source.getAnimations({ subtree: true });
        handoffTimer = window.setTimeout(resolve, 160);
        void Promise.all(
          transitions.map((transition) => transition.finished.catch(() => {})),
        ).then(() => {
          if (handoffTimer !== undefined) window.clearTimeout(handoffTimer);
          resolve();
        });
      });
    });
    void Promise.all([prepareEntrySurprise(), flattened])
      .then(([{ createEntrySurprise }]) => {
        if (!current()) return;
        scene = createEntrySurprise({
          host,
          source,
          deferStart: true,
          startFlush: true,
          onStart: () => {
            if (current()) celebration.tension();
          },
          onStage: (stage) => {
            if (!current()) return;
            setPhase(stage);
            if (stage === "settled") {
              runningRef.current = false;
            }
          },
          onRelease: release,
          onUnavailable: fallback,
        });
        // Render a flush pose while hidden; CSS blends it in as the lift begins.
        scene.present();
        setPhase("bridging");
        bridgeFrame = requestAnimationFrame(() => {
          if (current()) scene?.start();
        });
        const initialWidth = source.offsetWidth;
        observer = new ResizeObserver(() => {
          if (Math.abs(source.offsetWidth - initialWidth) > 1) fallback();
        });
        observer.observe(source);
      })
      .catch(() => {
        if (!current()) return;
        release();
        fallback();
      });
  };

  const rendered =
    phase !== "pressed" && phase !== "flattening" && phase !== "fallback";
  const revealed =
    phase === "released" || phase === "settled" || phase === "fallback";
  return (
    <div
      class={`search-entry-surprise ${rendered ? "is-rendered" : ""} ${revealed ? "is-revealed" : ""}`}
      data-stage={phase}
    >
      <button
        ref={sourceRef}
        type="button"
        class="search-option search-entry-template"
        aria-label="Your next favourite abridged series"
        aria-busy={runningRef.current}
        onClick={reveal}
      >
        <span class="search-thumbnail">
          <svg
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.3"
            aria-hidden="true"
          >
            <rect x="2" y="5" width="20" height="14" rx="2" />
            <path d="m10 9 5 3-5 3Z" />
          </svg>
        </span>
        <span class="search-entry-copy">
          <span class="search-creator">[Creator]</span>
          <span class="search-title">Your next favourite abridged series</span>
        </span>
        <span class="search-type">series</span>
      </button>
      <span ref={hostRef} class="surprise-scene" aria-hidden="true" />
      <output class="surprise-announcement" aria-live="polite">
        {revealed ? "You found the hidden entry." : ""}
      </output>
    </div>
  );
}
