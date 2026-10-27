// import type { ComponentChildren } from "preact";
// import { useState } from "preact/hooks";
import { useEffect } from "preact/hooks";
import type { CatalogResponse } from "../../lib/types";
import { BrowseSection } from "../components/BrowseSection";
import { CreatorIndex } from "../components/CreatorIndex";
import { HeroSearch } from "../components/HeroSearch";
import { InlineError } from "../components/InlineError";
import { scheduleEntrySurprisePreparation } from "../effects/entrySurprisePreparation";
import { shuffledEntries } from "../entryUtils";
import { optionalActiveEntryId } from "../propUtils";

type RandomizedHomeEntries = Pick<
  CatalogResponse,
  "series" | "shorts" | "shots" | "songs" | "songAmvs" | "creators"
>;

let cachedRandomizedHomeEntries:
  | { source: CatalogResponse; entries: RandomizedHomeEntries }
  | undefined;

export function HomePage({
  catalog,
  loading,
  error,
  restoreFromHistory,
  activeEntryId,
}: {
  catalog: CatalogResponse;
  loading: boolean;
  error: string | null;
  restoreFromHistory: boolean;
  activeEntryId?: string;
}) {
  useEffect(() => scheduleEntrySurprisePreparation(), []);

  const randomizedHomeEntries = getRandomizedHomeEntries(catalog);

  return (
    <>
      <section class="hero">
        <div class="container hero-content">
          <div class="hero-heading">
            <span class="hero-the">The</span>
            <h1 class="hero-title">
              <span class="hero-upright">Abridged</span>{" "}
              <span class="hero-italic">Catalogue</span>
            </h1>
          </div>
          <p class="hero-copy">
            A collection of abridged anime,{" "}
            <i>including the ones the platforms tried to erase.</i>
          </p>
          <HeroSearch catalog={catalog} loading={loading} />
        </div>
      </section>

      {error ? <InlineError message={error} /> : null}
      <BrowseSection
        title="Shows"
        titleHref="/series"
        titleClassName="accent-shows"
        entries={randomizedHomeEntries.series}
        {...optionalActiveEntryId(activeEntryId)}
        limited
        animateCards={!restoreFromHistory}
        hideSeriesDurations
        loading={loading}
      />
      <BrowseSection
        title="Shorts"
        titleHref="/shorts"
        titleClassName="accent-shorts"
        entries={randomizedHomeEntries.shorts}
        {...optionalActiveEntryId(activeEntryId)}
        limited
        animateCards={!restoreFromHistory}
        loading={loading}
      />
      <BrowseSection
        title="One-Shots"
        titleHref="/shots"
        titleClassName="accent-one-shots"
        entries={randomizedHomeEntries.shots}
        {...optionalActiveEntryId(activeEntryId)}
        limited
        animateCards={!restoreFromHistory}
        loading={loading}
      />
      <BrowseSection
        title="Songs"
        titleHref="/songs"
        titleClassName="accent-songs"
        entries={randomizedHomeEntries.songs}
        {...optionalActiveEntryId(activeEntryId)}
        limited
        animateCards={!restoreFromHistory}
        loading={loading}
      />
      <CreatorIndex
        creators={randomizedHomeEntries.creators}
        loading={loading}
      />
      {/* <HomeQuestions /> */}
    </>
  );
}

function getRandomizedHomeEntries(
  catalog: CatalogResponse,
): RandomizedHomeEntries {
  if (cachedRandomizedHomeEntries?.source === catalog) {
    return cachedRandomizedHomeEntries.entries;
  }

  const entries = {
    series: shuffledEntries(catalog.series),
    shorts: shuffledEntries(catalog.shorts),
    shots: shuffledEntries(catalog.shots),
    songs: shuffledEntries(catalog.songs),
    songAmvs: shuffledEntries(catalog.songAmvs),
    creators: shuffledEntries(catalog.creators).slice(0, 10),
  };

  cachedRandomizedHomeEntries = { source: catalog, entries };

  return entries;
}

// function HomeQuestions() {
//   const [openQuestion, setOpenQuestion] = useState<string | null>(
//     "Why does this exist?",
//   );
//   const questions: Array<{ question: string; answer: ComponentChildren }> = [
//     {
//       question: "Why does this exist?",
//       answer: (
//         <p>
//           I love going through this collection about once per year and I've done
//           everything I can to make that as seamless as possible. Maybe I'm
//           overengineering this a whole lot more than it needs to be, but I enjoy
//           it. I've already built a{" "}
//           <a
//             href="https://github.com/blekmus/abridged-cli"
//             target="_blank"
//             rel="noreferrer"
//           >
//             CLI
//           </a>{" "}
//           and a{" "}
//           <a href="https://github.com/blekmus/raycast-abridged">
//             Raycast Plugin
//           </a>{" "}
//           so a website is sort of the natural evolution.
//         </p>
//       ),
//     },
//     {
//       question: "Where can I find more?",
//       answer: (
//         <p>
//           Once upon a time I made a{" "}
//           <a
//             href="https://nyaa.si/view/1979033"
//             target="_blank"
//             rel="noreferrer"
//           >
//             torrent
//           </a>{" "}
//           with a bunch of the entries here. Other than that, there's always the{" "}
//           <a
//             target="_blank"
//             href="https://abridgedseries.fandom.com/wiki/Abridged_Archive"
//             rel="noreferrer"
//           >
//             abridged archive.
//           </a>
//         </p>
//       ),
//     },
//     {
//       question: "Why are some classics missing?",
//       answer: (
//         <p>
//           As much as I want to include every abridgedment out there, I'm limited
//           by how much storage I have on my server. So I decided to not host the
//           big titles that are already easily accessible.
//         </p>
//       ),
//     },
//     {
//       question: "How are shows, shorts, and one-shots split?",
//       answer: (
//         <>
//           <p>
//             <strong>Shows: </strong> A set of entries that run episodically or
//             are part of a series of works by a creator or group of creators.
//           </p>
//           <p>
//             <strong>One-shots: </strong> Single videos that are not part of a
//             continuous series but share an overarching story. These are usually
//             longer than 5 minutes.
//           </p>
//           <p>
//             <strong>Shorts: </strong>Similar to Shots, but shorter in length and
//             may not have a well-defined plot.
//           </p>
//         </>
//       ),
//     },
//   ];

//   return (
//     <section class="qa-section">
//       <div class="container qa-container">
//         <div class="qa-intro">
//           <h2 class="qa-title">Q&A</h2>
//         </div>
//         <div class="qa-list">
//           {questions.map((item) => (
//             <details
//               class="qa-item"
//               key={item.question}
//               open={openQuestion === item.question}
//             >
//               <summary
//                 onClick={(event) => {
//                   event.preventDefault();
//                   setOpenQuestion((current) =>
//                     current === item.question ? null : item.question,
//                   );
//                 }}
//                 onKeyDown={(event) => {
//                   if (event.key !== "Enter" && event.key !== " ") {
//                     return;
//                   }

//                   event.preventDefault();
//                   setOpenQuestion((current) =>
//                     current === item.question ? null : item.question,
//                   );
//                 }}
//               >
//                 {item.question}
//               </summary>
//               <div class="qa-answer">{item.answer}</div>
//             </details>
//           ))}
//         </div>
//       </div>
//     </section>
//   );
// }
