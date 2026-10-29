import type { CreatorSummary } from "../../lib/types";
import { handleInternalLinkClick } from "../navigation";
import { CreatorBilling, CreatorLoading } from "./CreatorBilling";
import { SectionHeader } from "./SectionHeader";

export function CreatorIndex({
  creators,
  loading,
}: {
  creators: CreatorSummary[];
  loading: boolean;
}) {
  return (
    <section id="creators" class="browse-section creator-index-section">
      <div class="container">
        <SectionHeader
          title="Creators"
          href="/creators"
          className="accent-creators"
        />

        {loading ? (
          <CreatorLoading />
        ) : creators.length === 0 ? (
          <p class="empty-state">No creators found.</p>
        ) : (
          <>
            <CreatorBilling creators={creators} />
            <a
              href="/creators"
              class="show-more"
              onClick={(event) => {
                handleInternalLinkClick(event, "/creators", {
                  scrollToTop: true,
                });
              }}
            >
              Show more
            </a>
          </>
        )}
      </div>
    </section>
  );
}
