import { handleInternalLinkClick } from "../navigation";

export function SectionHeader({
  title,
  href,
  className,
  compact = false,
}: {
  title: string;
  href?: string | undefined;
  className?: string | undefined;
  compact?: boolean | undefined;
}) {
  return (
    <div class={`section-header ${compact ? "is-compact" : ""}`}>
      <h2>
        {href ? (
          <a
            href={href}
            class={`section-title-link ${className ?? ""}`}
            onClick={(event) => {
              handleInternalLinkClick(event, href, { scrollToTop: true });
            }}
          >
            {title}
          </a>
        ) : (
          <span class={`section-title-text ${className ?? ""}`}>{title}</span>
        )}
      </h2>
      <span class="section-rule" />
    </div>
  );
}
