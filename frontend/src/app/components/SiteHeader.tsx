import { handleInternalLinkClick } from "../navigation";
import { NavLink } from "./NavLink";

export function SiteHeader({ currentPathname }: { currentPathname: string }) {
  const links = [
    { href: "/series", label: "Shows", accentClass: "accent-shows" },
    { href: "/shorts", label: "Shorts", accentClass: "accent-shorts" },
    { href: "/shots", label: "One-Shots", accentClass: "accent-one-shots" },
    { href: "/songs", label: "Songs", accentClass: "accent-songs" },
  ];

  return (
    <header class="site-header">
      <div class="container site-header-row">
        <div class="site-header-rail">
          <a
            href="/"
            class="site-mark"
            onClick={(event) => {
              handleInternalLinkClick(event, "/");
            }}
          >
            The abridged catalogue
          </a>
        </div>
        <nav class="site-nav" aria-label="Primary">
          {links.map((link) => (
            <NavLink
              key={link.href}
              href={link.href}
              active={currentPathname === link.href}
              className={link.accentClass}
            >
              {link.label}
            </NavLink>
          ))}
        </nav>
      </div>
    </header>
  );
}
