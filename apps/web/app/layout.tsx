import "katex/dist/katex.min.css";

import "./globals.css";

import type { ReactNode } from "react";

type RootLayoutProps = {
  children: ReactNode;
};

export default function RootLayout({ children }: RootLayoutProps) {
  return (
    <html lang="en">
      <body>
        <div className="app-shell">
          <header className="app-header">
            <div className="app-header__inner">
              <a className="app-brand" href="/">
                <span className="app-brand__mark">C</span>
                <span>Cabugi</span>
              </a>

              <nav className="app-nav" aria-label="Primary">
                <span>Published problems</span>
                <span>Modern practice flow</span>
              </nav>
            </div>
          </header>

          {children}
        </div>
      </body>
    </html>
  );
}
