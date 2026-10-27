const clamp = (value: number, min: number, max: number) =>
  Math.min(max, Math.max(min, value));

/** The search bar holds the welcome panel while scrolling pulls its tether. */
export function attachSearchElastic({
  root,
  content,
  art,
}: {
  root: HTMLDivElement;
  content: HTMLDivElement;
  art: SVGSVGElement;
}) {
  const startScroll = window.scrollY;
  const motion = window.matchMedia("(prefers-reduced-motion: reduce)");
  const surface = art.querySelector<SVGPathElement>("[data-surface]");
  const fibres = Array.from(
    art.querySelectorAll<SVGPathElement>("[data-fibre]"),
  );
  const material = art.querySelector("linearGradient");
  let frame = 0;
  let releaseTimer = 0;
  let retained = 0;
  let pull = 0;
  let target = startScroll;
  let position = startScroll;
  let commandedScroll = startScroll;
  let dragging = false;
  let touchY: number | null = null;
  let previousTime = 0;

  const draw = () => {
    const travel = Math.max(0, position - startScroll);
    // Flex follows real scrolling; continue the same movement at the page edge.
    const offset = Math.max(0, position - window.scrollY) * 0.5;
    content.style.transform = `translateY(${offset}px)`;
    const bounds = root.getBoundingClientRect();
    const contentBounds = content.getBoundingClientRect();
    const width = bounds.width;
    const end = Math.max(0, contentBounds.top - bounds.top);
    const center = width / 2;
    const tension = travel / 200;
    const breadth = 30 / Math.sqrt(1 + tension);
    art.setAttribute(
      "viewBox",
      `0 0 ${Math.max(1, width)} ${Math.max(1, bounds.height)}`,
    );
    art.style.opacity = String(clamp(travel / 50, 0, 1));
    material?.setAttribute("x1", String(center - breadth * 2));
    material?.setAttribute("x2", String(center + breadth * 2));
    const points = Array.from({ length: 49 }, (_, index) => {
      const t = index / 48;
      const flare = Math.exp(-t * 12) + Math.exp(-(1 - t) * 12);
      return { y: t * end, width: breadth * (0.35 + flare) };
    });
    const left = points.map((p) => `${center - p.width},${p.y}`);
    const right = points.map((p) => `${center + p.width},${p.y}`).reverse();
    surface?.setAttribute("d", `M${left.join(" L")} L${right.join(" L")} Z`);
    for (const fibre of fibres) {
      const offset = Number(fibre.dataset.fibre);
      fibre.setAttribute(
        "d",
        points
          .map(
            (p, i) =>
              `${i === 0 ? "M" : "L"}${center + p.width * offset},${p.y}`,
          )
          .join(" "),
      );
    }
  };

  const animate = (now: number) => {
    frame = 0;
    const elapsed = previousTime ? Math.min(64, now - previousTime) : 16;
    previousTime = now;
    const difference = target - position;
    position += difference * (motion.matches ? 1 : 1 - Math.exp(-elapsed / 75));
    if (Math.abs(difference) <= 0.75) position = target;
    commandedScroll = Math.min(position, maxScroll());
    window.scrollTo({
      top: commandedScroll,
      behavior: "instant",
    });
    if (Math.abs(difference) > 0.75) {
      frame = requestAnimationFrame(animate);
    } else {
      previousTime = 0;
    }
    draw();
  };
  const schedule = () => {
    if (!frame) frame = requestAnimationFrame(animate);
  };
  const maxScroll = () =>
    Math.max(0, document.documentElement.scrollHeight - window.innerHeight);
  const release = () => {
    dragging = false;
    // Some deformation survives each pull, until this welcome panel unmounts.
    const limit = Math.min(maxScroll(), startScroll + 200 + retained);
    const extension = Math.max(0, target - limit);
    retained = Math.min(180, retained + extension * 0.25);
    pull = 0;
    target = Math.min(target, limit + extension * 0.25);
    schedule();
  };
  const move = (delta: number) => {
    dragging = true;
    window.clearTimeout(releaseTimer);
    const limit = Math.min(maxScroll(), startScroll + 200 + retained);
    if (delta > 0 && target + delta > limit) {
      const excess = Math.max(0, target + delta - limit);
      pull = Math.min(5000, pull + excess);
      target = limit + 200 * (1 - Math.exp(-pull / 450));
    } else {
      target = Math.max(0, target + delta);
      pull = Math.max(0, pull + delta);
    }
    schedule();
    releaseTimer = window.setTimeout(release, 130);
  };
  const wheel = (event: WheelEvent) => {
    if (
      event.ctrlKey ||
      !event.deltaY ||
      Math.abs(event.deltaX) > Math.abs(event.deltaY)
    )
      return;
    event.preventDefault();
    const unit =
      event.deltaMode === 1
        ? 16
        : event.deltaMode === 2
          ? window.innerHeight
          : 1;
    move(event.deltaY * unit);
  };
  const touchStart = (event: TouchEvent) => {
    touchY =
      event.touches.length === 1 ? (event.touches[0]?.clientY ?? null) : null;
  };
  const touchMove = (event: TouchEvent) => {
    const next = event.touches[0]?.clientY;
    if (touchY === null || next === undefined || event.touches.length !== 1)
      return;
    event.preventDefault();
    move(touchY - next);
    touchY = next;
  };
  const scroll = () => {
    // Scroll events arrive after scrollTo, including after the last spring frame.
    // Do not mistake our own final scroll for a new gesture and recoil again.
    if (Math.abs(window.scrollY - commandedScroll) <= 1) {
      draw();
      return;
    }
    // Also constrain scrollbar dragging and keyboard scrolling.
    if (!frame && !dragging) {
      position = window.scrollY;
      target = Math.min(window.scrollY, startScroll + 200 + retained);
      schedule();
    }
    draw();
  };
  const observer = new ResizeObserver(draw);
  observer.observe(root);
  observer.observe(content);
  window.addEventListener("wheel", wheel, { passive: false });
  window.addEventListener("touchstart", touchStart, { passive: true });
  window.addEventListener("touchmove", touchMove, { passive: false });
  window.addEventListener("scroll", scroll, { passive: true });
  window.addEventListener("resize", draw);
  draw();

  return () => {
    cancelAnimationFrame(frame);
    window.clearTimeout(releaseTimer);
    observer.disconnect();
    window.removeEventListener("wheel", wheel);
    window.removeEventListener("touchstart", touchStart);
    window.removeEventListener("touchmove", touchMove);
    window.removeEventListener("scroll", scroll);
    window.removeEventListener("resize", draw);
    content.style.removeProperty("transform");
  };
}
