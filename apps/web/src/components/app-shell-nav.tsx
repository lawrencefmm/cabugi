"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const navItems = [
  { href: "/", label: "Problems", icon: "</>", active: (pathname: string) => pathname === "/" || pathname.startsWith("/problems") },
  { href: "/drafts", label: "Drafts", icon: "E2", active: (pathname: string) => pathname.startsWith("/drafts") },
  { href: "/submissions", label: "Submissions", icon: "S", active: (pathname: string) => pathname.startsWith("/submissions") },
  { href: "/moderation/problem-drafts", label: "Moderation", icon: "M", active: (pathname: string) => pathname.startsWith("/moderation") },
  { href: "/api/openapi/v1.yaml", label: "Docs", icon: "D", active: () => false },
];

export function AppShellNav() {
  const pathname = usePathname();

  return (
    <nav className="app-nav" aria-label="Primary">
      {navItems.map((item) => {
        const isActive = item.active(pathname);

        return (
          <Link className={`app-nav__link${isActive ? " app-nav__link--active" : ""}`} href={item.href} key={item.href} aria-current={isActive ? "page" : undefined}>
            <span className="app-nav__icon" aria-hidden="true">
              {item.icon}
            </span>
            <span>{item.label}</span>
          </Link>
        );
      })}
    </nav>
  );
}
