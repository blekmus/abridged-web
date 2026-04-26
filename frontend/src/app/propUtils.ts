export function optionalActiveEntryId(activeEntryId: string | undefined): {
  activeEntryId?: string;
} {
  return activeEntryId ? { activeEntryId } : {};
}
