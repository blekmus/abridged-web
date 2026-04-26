import { Fragment } from "preact";
import { useEffect, useState } from "preact/hooks";
import { api } from "../../lib/api";
import type { Entry } from "../../lib/types";
import { BrowseSection } from "../components/BrowseSection";
import { InlineError } from "../components/InlineError";
import { LoadingGrid } from "../components/LoadingGrid";
import { CREATOR_SECTIONS } from "../constants";
import { groupEntriesByType } from "../entryUtils";
import { optionalActiveEntryId } from "../propUtils";
import type { AsyncState } from "../types";

export function CreatorPage({
  slug,
  restoreFromHistory,
  activeEntryId,
}: {
  slug: string;
  restoreFromHistory: boolean;
  activeEntryId?: string;
}) {
  const [state, setState] = useState<AsyncState<Entry[]>>({
    data: null,
    loading: true,
    error: null,
  });

  useEffect(() => {
    let active = true;
    setState({ data: null, loading: true, error: null });
    api
      .fetchCreator(slug)
      .then((data) => {
        if (!active) return;
        setState({ data, loading: false, error: null });
      })
      .catch((error: Error) => {
        if (!active) return;
        setState({ data: [], loading: false, error: error.message });
      });

    return () => {
      active = false;
    };
  }, [slug]);

  const entries = state.data ?? [];
  const title = entries[0]?.creator ?? slug.replace(/-/g, " ");
  const groupedEntries = groupEntriesByType(entries);

  return (
    <section class="browse-page">
      <div class="container">
        <header class="creator-header">
          <h1 class="creator-title">{title}</h1>
          <p class="eyebrow">Creator catalog</p>
        </header>
        {state.error ? <InlineError message={state.error} /> : null}
        {state.loading ? (
          <LoadingGrid />
        ) : entries.length > 0 ? (
          <div class="creator-sections">
            {CREATOR_SECTIONS.map((section) => {
              const sectionEntries = groupedEntries[section.entryType];
              if (sectionEntries.length === 0) {
                return null;
              }

              if (section.entryType === "song") {
                const songEntries = sectionEntries.filter(
                  (entry) => entry.subtype !== "amv",
                );
                const songAmvEntries = sectionEntries.filter(
                  (entry) => entry.subtype === "amv",
                );

                return (
                  <Fragment key={section.entryType}>
                    {songEntries.length > 0 ? (
                      <BrowseSection
                        title={section.title}
                        titleClassName={section.titleClassName}
                        entries={songEntries}
                        {...optionalActiveEntryId(activeEntryId)}
                        showAllEntries
                        animateCards={!restoreFromHistory}
                        compactHeader
                        hideSeriesDurations
                      />
                    ) : null}
                    {songAmvEntries.length > 0 ? (
                      <BrowseSection
                        title="Songs / AMVs"
                        titleClassName={section.titleClassName}
                        entries={songAmvEntries}
                        {...optionalActiveEntryId(activeEntryId)}
                        showAllEntries
                        animateCards={!restoreFromHistory}
                        compactHeader
                        hideSeriesDurations
                      />
                    ) : null}
                  </Fragment>
                );
              }

              return (
                <BrowseSection
                  key={section.entryType}
                  title={section.title}
                  titleClassName={section.titleClassName}
                  entries={sectionEntries}
                  {...optionalActiveEntryId(activeEntryId)}
                  showAllEntries
                  animateCards={!restoreFromHistory}
                  compactHeader
                  hideSeriesDurations
                />
              );
            })}
          </div>
        ) : (
          <p class="empty-state">No items found for this creator.</p>
        )}
      </div>
    </section>
  );
}
