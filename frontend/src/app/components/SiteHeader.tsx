import { useState } from "preact/hooks";
import { handleInternalLinkClick } from "../navigation";
import { NavLink } from "./NavLink";

export function SiteHeader({ currentPathname }: { currentPathname: string }) {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
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
              setMobileMenuOpen(false);
              handleInternalLinkClick(event, "/");
            }}
          >
            The abridged catalogue
          </a>
          <button
            type="button"
            class="mobile-menu-toggle"
            aria-controls="primary-navigation"
            aria-expanded={mobileMenuOpen}
            aria-label={mobileMenuOpen ? "Close menu" : "Open menu"}
            onClick={() => {
              setMobileMenuOpen((open) => !open);
            }}
          >
            <span aria-hidden="true" />
            <span aria-hidden="true" />
            <span aria-hidden="true" />
          </button>
        </div>
        <nav
          id="primary-navigation"
          class={`site-nav ${mobileMenuOpen ? "is-open" : ""}`}
          aria-label="Primary"
        >
          {links.map((link) => (
            <NavLink
              key={link.href}
              href={link.href}
              active={currentPathname === link.href}
              className={link.accentClass}
              onNavigate={() => {
                setMobileMenuOpen(false);
              }}
            >
              {link.label}
            </NavLink>
          ))}
        </nav>
      </div>
    </header>
  );
}
