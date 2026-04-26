import type { Entry, EntryType } from "../../lib/types";
import { BrowseSection } from "../components/BrowseSection";
import { InlineError } from "../components/InlineError";
import { LoadingGrid } from "../components/LoadingGrid";
import { SectionHeader } from "../components/SectionHeader";
import { TYPE_META } from "../constants";
import { accentClassForEntryType } from "../entryUtils";
import { optionalActiveEntryId } from "../propUtils";

export function BrowsePage({
  entryType,
  entries,
  loading,
  error,
  activeEntryId,
  restoreFromHistory,
}: {
  entryType: EntryType;
  entries: Entry[];
  loading: boolean;
  error: string | null;
  activeEntryId?: string;
  restoreFromHistory: boolean;
}) {
  const meta = TYPE_META[entryType];

  return (
    <section class="browse-page">
      <div class="container">
        <p class="eyebrow">{meta.eyebrow}</p>
        <SectionHeader
          title={meta.title}
          className={accentClassForEntryType(entryType)}
        />
        {error ? <InlineError message={error} /> : null}
        {loading ? (
          <LoadingGrid />
        ) : (
          <BrowseSection
            key={entryType}
            title={meta.title}
            entries={entries}
            {...optionalActiveEntryId(activeEntryId)}
            hideHeader
            infiniteScroll
            showAllEntries={restoreFromHistory}
            animateCards={!restoreFromHistory}
            hideSeriesDurations={entryType === "series"}
          />
        )}
      </div>
    </section>
  );
}
