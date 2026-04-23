import { ClerkProvider } from "@clerk/nextjs";

import "katex/dist/katex.min.css";

import "./globals.css";

import type { ReactNode } from "react";

import { AppProviders } from "../src/components/app-providers";

type RootLayoutProps = {
  children: ReactNode;
};

export default function RootLayout({ children }: RootLayoutProps) {
  const content = (
    <AppProviders>
      <div className="app-shell">
        <header className="app-header">
          <div className="app-header__inner">
            <a className="app-brand" href="/">
              <span className="app-brand__mark">C</span>
              <span>Cabugi</span>
            </a>

            <nav className="app-nav" aria-label="Primary">
              <a href="/">Problems</a>
              <a href="/drafts/new">Drafts</a>
              <a href="/moderation/problem-drafts">Moderation</a>
              <a href="/submissions">Submissions</a>
            </nav>
          </div>
        </header>

        {children}
      </div>
    </AppProviders>
  );

  const publishableKey = process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY;

  return (
    <html lang="en">
      <body>
        {publishableKey ? <ClerkProvider publishableKey={publishableKey}>{content}</ClerkProvider> : content}
      </body>
    </html>
  );
}
