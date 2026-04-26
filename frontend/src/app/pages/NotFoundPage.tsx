import { SectionHeader } from "../components/SectionHeader";

export function NotFoundPage() {
  return (
    <section class="browse-page">
      <div class="container">
        <SectionHeader title="Not found" />
        <p class="empty-state">
          This route does not exist in the current archive.
        </p>
      </div>
    </section>
  );
}
