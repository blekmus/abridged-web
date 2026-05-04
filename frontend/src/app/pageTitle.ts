export const SITE_TITLE = "Abridged Catelogue";

export function pageTitle(title?: string): string {
  const trimmedTitle = title?.trim();

  return trimmedTitle ? `${trimmedTitle} | ${SITE_TITLE}` : SITE_TITLE;
}

export function setPageTitle(title?: string): void {
  document.title = pageTitle(title);
}
