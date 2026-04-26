import type { ComponentChildren } from "preact";
import { handleInternalLinkClick } from "../navigation";

export function NavLink({
  href,
  active,
  className,
  children,
}: {
  href: string;
  active: boolean;
  className?: string;
  children: ComponentChildren;
}) {
  return (
    <a
      href={href}
      class={`nav-link ${className ?? ""} ${active ? "is-active" : ""}`}
      onClick={(event) => {
        handleInternalLinkClick(event, href);
      }}
    >
      {children}
    </a>
  );
}
