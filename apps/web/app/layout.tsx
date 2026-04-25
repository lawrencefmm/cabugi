import "katex/dist/katex.min.css";

import "./globals.css";

import Link from "next/link";
import type { ReactNode } from "react";

import { AppProviders } from "../src/components/app-providers";
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
              <span className="app-brand__mark">&lt;/&gt;</span>
              <span className="app-brand__text">cabugi</span>
            </Link>

            <div className="app-header__actions">
              <nav className="app-nav" aria-label="Primary">
                <Link href="/">Problems</Link>
                <Link href="/drafts/new">Drafts</Link>
                <Link href="/moderation/problem-drafts">Moderation</Link>
                <Link href="/submissions">Submissions</Link>
              </nav>

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
