import type { CreatorSummary } from "../../lib/types";
import { CreatorBilling, CreatorLoading } from "../components/CreatorBilling";
import { InlineError } from "../components/InlineError";
import { SectionHeader } from "../components/SectionHeader";

export function CreatorsPage({
  creators,
  loading,
  error,
}: {
  creators: CreatorSummary[];
  loading: boolean;
  error: string | null;
}) {
  const countLabel = loading
    ? "Loading creators"
    : `${creators.length} ${creators.length === 1 ? "creator" : "creators"}`;

  return (
    <section class="browse-page">
      <div class="container">
        <div class="browse-type-header">
          <h1 class="browse-type-title">Creators</h1>
          <p class="eyebrow">The people behind the catalogue</p>
        </div>
        <SectionHeader title={countLabel} className="accent-creators" />
        {error ? <InlineError message={error} /> : null}
        {loading ? (
          <CreatorLoading compact />
        ) : creators.length === 0 ? (
          <p class="empty-state">No creators found.</p>
        ) : (
          <CreatorBilling creators={creators} compact />
        )}
      </div>
    </section>
  );
}
