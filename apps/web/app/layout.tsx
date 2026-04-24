import { ClerkProvider } from "@clerk/nextjs";

import "katex/dist/katex.min.css";

import "./globals.css";

import Link from "next/link";
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
            <Link className="app-brand" href="/">
              <span className="app-brand__mark">&lt;/&gt;</span>
              <span className="app-brand__text">cabugi</span>
            </Link>

            <nav className="app-nav" aria-label="Primary">
              <Link href="/">Problems</Link>
              <Link href="/drafts/new">Drafts</Link>
              <Link href="/moderation/problem-drafts">Moderation</Link>
              <Link href="/submissions">Submissions</Link>
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
