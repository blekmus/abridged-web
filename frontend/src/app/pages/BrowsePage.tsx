import type { Entry, EntryType } from "../../lib/types";
import { BrowseSection } from "../components/BrowseSection";
import { InlineError } from "../components/InlineError";
import { LoadingGrid } from "../components/LoadingGrid";
import { SectionHeader } from "../components/SectionHeader";
import { TYPE_META } from "../constants";
import { accentClassForEntryType, sortedEntriesByTitle } from "../entryUtils";
import { optionalActiveEntryId } from "../propUtils";

export function BrowsePage({
  entryType,
  entries,
  songAmvs,
  loading,
  error,
  activeEntryId,
  restoreFromHistory,
}: {
  entryType: EntryType;
  entries: Entry[];
  songAmvs: Entry[];
  loading: boolean;
  error: string | null;
  activeEntryId?: string;
  restoreFromHistory: boolean;
}) {
  const meta = TYPE_META[entryType];
  const entryCountLabel = loading
    ? "Loading entries"
    : `${entries.length} ${entries.length === 1 ? "title" : "titles"}`;

  return (
    <section class="browse-page">
      <div class="container">
        <div class="browse-type-header">
          <h1 class="browse-type-title">{meta.title}</h1>
          <p class="eyebrow">{meta.eyebrow}</p>
        </div>
        <SectionHeader
          title={entryCountLabel}
          className={accentClassForEntryType(entryType)}
        />
        {error ? <InlineError message={error} /> : null}
        {loading ? (
          <LoadingGrid />
        ) : (
          <div class="browse-type-sections">
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
            {!error && entryType === "song" ? (
              <BrowseSection
                sectionId="amvs"
                title="AMVs"
                titleClassName={accentClassForEntryType(entryType)}
                entries={sortedEntriesByTitle(songAmvs)}
                {...optionalActiveEntryId(activeEntryId)}
                infiniteScroll
                showAllEntries={restoreFromHistory}
                animateCards={!restoreFromHistory}
                compactHeader
              />
            ) : null}
          </div>
        )}
      </div>
    </section>
  );
}
