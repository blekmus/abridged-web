import { useId, useLayoutEffect, useRef } from "preact/hooks";
import "../../styles/search-elastic.css";
import { attachSearchElastic } from "../effects/searchElastic";
import { SearchEntrySurprise } from "./SearchEntrySurprise";

const FIBRES = [-0.84, -0.56, -0.28, 0, 0.28, 0.56, 0.84];

export function SearchWelcome({
  onBegin,
  onCheerStart,
}: {
  onBegin: () => void;
  onCheerStart: () => void;
}) {
  const rootRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const artRef = useRef<SVGSVGElement>(null);
  const materialId = `search-elastic-${useId()}`;

  useLayoutEffect(() => {
    const root = rootRef.current;
    const content = contentRef.current;
    const art = artRef.current;
    if (!root || !content || !art) return;
    return attachSearchElastic({ root, content, art });
  }, []);

  return (
    <div
      ref={rootRef}
      class="search-welcome"
      data-elastic="idle"
      data-tendon="graphite"
    >
      <svg ref={artRef} class="search-elastic" fill="none" aria-hidden="true">
        <defs>
          <linearGradient
            id={materialId}
            gradientUnits="userSpaceOnUse"
            x1="0"
            y1="0"
            x2="1"
            y2="0"
          >
            <stop offset="0" class="tendon-edge" stop-opacity="0.12" />
            <stop offset="0.25" class="tendon-mid" stop-opacity="0.22" />
            <stop offset="0.48" class="tendon-highlight" stop-opacity="0.32" />
            <stop offset="0.65" class="tendon-mid" stop-opacity="0.2" />
            <stop offset="1" class="tendon-edge" stop-opacity="0.1" />
          </linearGradient>
        </defs>
        <g>
          <path data-surface fill={`url(#${materialId})`} />
          {FIBRES.map((fibre) => (
            <path
              key={fibre}
              data-fibre={fibre}
              class="search-elastic-fibre"
              stroke-opacity={0.06 + (1 - Math.abs(fibre)) * 0.08}
            />
          ))}
        </g>
      </svg>
      <div ref={contentRef} class="search-welcome-content">
        <div class="search-welcome-copy">
          <h2>
            Welcome to the <em>catalogue</em>
          </h2>
          <p id="search-welcome-hint">
            Type a title or creator to start searching
          </p>
        </div>
        <SearchEntrySurprise onBegin={onBegin} onCheerStart={onCheerStart} />
      </div>
    </div>
  );
}
