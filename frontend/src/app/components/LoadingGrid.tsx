import { LOADING_CARD_KEYS } from "../constants";

export function LoadingGrid() {
  return (
    <div class="container">
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
