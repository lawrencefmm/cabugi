import "katex/dist/katex.min.css";

import "./globals.css";

import Link from "next/link";
import type { ReactNode } from "react";

import { AppProviders } from "../src/components/app-providers";
import { AppShellNav } from "../src/components/app-shell-nav";
import { AppAuthControls, AuthProvider } from "../src/components/auth";

type RootLayoutProps = {
  children: ReactNode;
};

export default function RootLayout({ children }: RootLayoutProps) {
  const content = (
    <AppProviders>
      <div className="app-shell">
        <header className="app-header">
          <div className="app-header__inner">
            <Link className="app-brand" href="/">
              <span className="app-brand__mark" aria-hidden="true">
                <svg viewBox="0 0 48 32" role="img">
                  <path d="M4 28 18 4l7 12-4 7 5-1 4 6H4Z" />
                  <path d="M27 13 39 28H28l-4-6 4-9Z" />
                </svg>
              </span>
              <span className="app-brand__text">Cabugi</span>
            </Link>

            <AppShellNav />

            <div className="app-header__actions">
              <div className="app-header__score" aria-label="Cabugi rating">
                <span aria-hidden="true">F</span>
                <strong>1824</strong>
              </div>
              <span className="app-header__bell" aria-hidden="true" />
              <AppAuthControls />
            </div>
          </div>
        </header>

        {children}
      </div>
    </AppProviders>
  );

  const localTestAuthEnabled = process.env.NEXT_PUBLIC_LOCAL_TEST_AUTH_ENABLED === "true";
  const publishableKey = process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY;

  return (
    <html lang="en">
      <body>
        <AuthProvider localTestAuthEnabled={localTestAuthEnabled} publishableKey={publishableKey}>
          {content}
        </AuthProvider>
      </body>
    </html>
  );
}
