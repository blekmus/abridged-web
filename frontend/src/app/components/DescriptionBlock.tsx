import { useLayoutEffect, useRef, useState } from "preact/hooks";

export function DescriptionBlock({ text }: { text: string }) {
  const [expanded, setExpanded] = useState(false);
  const [canExpand, setCanExpand] = useState(false);
  const copyRef = useRef<HTMLDivElement | null>(null);
  const paragraphs = text
    .split(/\n{2,}/)
    .map((paragraph) => paragraph.trim())
    .filter(Boolean);
  const isCollapsed = canExpand && !expanded;

  useLayoutEffect(() => {
    setExpanded(false);
  }, []);

  useLayoutEffect(() => {
    const copy = copyRef.current;
    if (!copy) {
      return;
    }

    const updateCanExpand = () => {
      const paragraph = copy.querySelector("p");
      const lineHeight = Number.parseFloat(
        window.getComputedStyle(paragraph ?? copy).lineHeight,
      );
      const collapsedHeight = lineHeight * 6;

      setCanExpand(copy.scrollHeight > collapsedHeight + 1);
    };

    updateCanExpand();

    if (typeof ResizeObserver === "undefined") {
      window.addEventListener("resize", updateCanExpand);
      return () => {
        window.removeEventListener("resize", updateCanExpand);
      };
    }

    const observer = new ResizeObserver(updateCanExpand);
    observer.observe(copy);

    return () => {
      observer.disconnect();
    };
  }, []);

  return (
    <section class="description-section">
      <h2 class="watch-section-title">
        <span>Youtube Description</span>
      </h2>
      <div class="description-block">
        <div
          class={`description-copy ${isCollapsed ? "collapsed" : "expanded"}`}
        >
          <div ref={copyRef}>
            {paragraphs.map((paragraph) => (
              <p key={paragraph}>{paragraph}</p>
            ))}
          </div>
        </div>
        {canExpand ? (
          <button
            type="button"
            class="description-toggle"
            onClick={() => setExpanded((value) => !value)}
          >
            {expanded ? "Collapse" : "Read more"}
          </button>
        ) : null}
      </div>
    </section>
  );
}
