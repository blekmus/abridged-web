import { LOADING_CARD_KEYS } from "../constants";
import { SectionHeader } from "./SectionHeader";

export function LoadingGrid({
  title,
  titleClassName,
  compactHeader = false,
}: {
  title?: string;
  titleClassName?: string;
  compactHeader?: boolean;
}) {
  return (
    <div class="container">
      {title ? (
        <SectionHeader
          title={title}
          className={titleClassName}
          compact={compactHeader}
        />
      ) : null}
      <div class="card-grid">
        {LOADING_CARD_KEYS.map((key) => (
          <div key={key} class="browse-card skeleton-card">
            <div class="card-thumbnail-shell skeleton-block" />
            <div class="card-copy">
              <div class="skeleton-line short" />
              <div class="skeleton-line" />
              <div class="skeleton-line short" />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
