import type { ComponentChildren } from "preact";
import { handleInternalLinkClick } from "../navigation";

export function NavLink({
  href,
  active,
  className,
  onNavigate,
  children,
}: {
  href: string;
  active: boolean;
  className?: string;
  onNavigate?: () => void;
  children: ComponentChildren;
}) {
  return (
    <a
      href={href}
      class={`nav-link ${className ?? ""} ${active ? "is-active" : ""}`}
      onClick={(event) => {
        onNavigate?.();
        handleInternalLinkClick(event, href);
      }}
    >
      {children}
    </a>
  );
}
