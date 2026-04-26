export function InlineError({ message }: { message: string }) {
  return (
    <div class="container">
      <div class="error-banner">{message}</div>
    </div>
  );
}
