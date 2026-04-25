"use client";

import {
  ClerkProvider,
  SignInButton as ClerkSignInButton,
  SignUpButton as ClerkSignUpButton,
  useAuth as useClerkAuth,
  useClerk,
  useUser as useClerkUser,
} from "@clerk/nextjs";
import { cloneElement, createContext, isValidElement, type MouseEvent, type ReactNode, useContext, useEffect, useState } from "react";

import {
  isLocalTestAuthProfile,
  localTestAuthCookieName,
  localTestAuthProfileCookieName,
  localTestAuthProfileLabels,
  type LocalTestAuthProfile,
} from "../lib/local-test-auth";

type AuthState = {
  mode: "disabled" | "clerk" | "local_test";
  getToken: () => Promise<string | null>;
  isLoaded: boolean;
  isSignedIn: boolean;
  localTestProfile: LocalTestAuthProfile | null;
  signInLocalTest: (profile: LocalTestAuthProfile) => Promise<boolean>;
  signOutLocalTest: () => Promise<boolean>;
};

type AuthProviderProps = {
  children: ReactNode;
  localTestAuthEnabled?: boolean;
  publishableKey?: string;
};

type AuthButtonProps = {
  children: ReactNode;
  mode?: "modal" | "redirect";
};

const disabledAuthState: AuthState = {
  mode: "disabled",
  getToken: async () => null,
  isLoaded: true,
  isSignedIn: false,
  localTestProfile: null,
  signInLocalTest: async () => false,
  signOutLocalTest: async () => false,
};

const AuthContext = createContext<AuthState>(disabledAuthState);

export function AuthProvider({ children, localTestAuthEnabled = false, publishableKey }: AuthProviderProps) {
  if (localTestAuthEnabled) {
    return <LocalTestAuthProvider>{children}</LocalTestAuthProvider>;
  }

  if (publishableKey) {
    return (
      <ClerkProvider publishableKey={publishableKey}>
        <ClerkAuthBridge>{children}</ClerkAuthBridge>
      </ClerkProvider>
    );
  }

  return <AuthContext.Provider value={disabledAuthState}>{children}</AuthContext.Provider>;
}

export function SignInButton({ children, mode }: AuthButtonProps) {
  const auth = useAuth();
  if (auth.mode === "clerk") {
    return <ClerkSignInButton mode={mode}>{children}</ClerkSignInButton>;
  }

  if (auth.mode === "local_test" && isValidElement<{ onClick?: (event: MouseEvent<HTMLElement>) => void }>(children)) {
    return cloneElement(children, {
      onClick: (event: MouseEvent<HTMLElement>) => {
        children.props.onClick?.(event);
        if (!event.defaultPrevented) {
          void auth.signInLocalTest("author");
        }
      },
    });
  }

  return <>{children}</>;
}

export function SignUpButton({ children, mode }: AuthButtonProps) {
  const auth = useAuth();
  if (auth.mode === "clerk") {
    return <ClerkSignUpButton mode={mode}>{children}</ClerkSignUpButton>;
  }

  return <>{children}</>;
}

export function useAuth() {
  return useContext(AuthContext);
}

export function AppAuthControls() {
  const auth = useAuth();

  if (auth.mode === "local_test") {
    return <LocalTestAuthControls />;
  }

  if (auth.mode === "clerk") {
    return <ClerkAuthControls />;
  }

  return null;
}

export function LocalTestAuthControls() {
  const auth = useAuth();
  const [isPending, setIsPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (auth.mode !== "local_test") {
    return null;
  }

  async function handleSignIn(profile: LocalTestAuthProfile) {
    setError(null);
    setIsPending(true);
    const ok = await auth.signInLocalTest(profile);
    setIsPending(false);
    if (!ok) {
      setError("Local test auth sign-in failed.");
    }
  }

  async function handleSignOut() {
    setError(null);
    setIsPending(true);
    const ok = await auth.signOutLocalTest();
    setIsPending(false);
    if (!ok) {
      setError("Local test auth sign-out failed.");
    }
  }

  const nextProfile = auth.localTestProfile === "moderator" ? "author" : "moderator";

  return (
    <div className="app-auth-controls" aria-live="polite">
      <span className="app-auth-controls__avatar" aria-hidden="true">
        LC
      </span>
      <div className="app-auth-controls__status">
        <span className="app-auth-controls__label">Local test auth</span>
        <span className="app-auth-controls__value">
          {!auth.isLoaded ? "Loading" : auth.isSignedIn ? `${auth.localTestProfile ? localTestAuthProfileLabels[auth.localTestProfile] : "Signed in"} session` : "Signed out"}
        </span>
      </div>

      <div className="app-auth-controls__actions">
        {!auth.isSignedIn ? (
          <>
            <button className="app-auth-controls__button" disabled={isPending || !auth.isLoaded} onClick={() => void handleSignIn("author")} type="button">
              Use author
            </button>
            <button className="app-auth-controls__button" disabled={isPending || !auth.isLoaded} onClick={() => void handleSignIn("moderator")} type="button">
              Use moderator
            </button>
          </>
        ) : (
          <>
            <button className="app-auth-controls__button" disabled={isPending || !auth.isLoaded} onClick={() => void handleSignIn(nextProfile)} type="button">
              Switch to {localTestAuthProfileLabels[nextProfile].toLowerCase()}
            </button>
            <button className="app-auth-controls__button app-auth-controls__button--secondary" disabled={isPending || !auth.isLoaded} onClick={() => void handleSignOut()} type="button">
              Sign out
            </button>
          </>
        )}
      </div>

      {error ? <p className="app-auth-controls__error">{error}</p> : null}
    </div>
  );
}

function ClerkAuthControls() {
  const auth = useAuth();
  const clerk = useClerk();
  const { user } = useClerkUser();
  const [isPending, setIsPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const accountState = !auth.isLoaded ? "Loading" : auth.isSignedIn ? formatClerkIdentity(user) : "Signed out";

  async function handleSignOut() {
    setError(null);
    setIsPending(true);
    try {
      await clerk.signOut();
    } catch {
      setError("Clerk sign-out failed.");
      setIsPending(false);
      return;
    }

    setIsPending(false);
  }

  return (
    <div className="app-auth-controls" aria-live="polite">
      <span className="app-auth-controls__avatar" aria-hidden="true">
        {auth.isSignedIn ? initialsForIdentity(accountState) : "CB"}
      </span>
      <div className="app-auth-controls__status">
        <span className="app-auth-controls__label">Account</span>
        <span className="app-auth-controls__value">{accountState}</span>
        {auth.isSignedIn ? <span className="app-auth-controls__rating">Rating 1824</span> : null}
      </div>

      <div className="app-auth-controls__actions">
        {!auth.isSignedIn ? (
          <>
            <SignInButton mode="modal">
              <button className="app-auth-controls__button" type="button">
                Sign in
              </button>
            </SignInButton>
            <SignUpButton mode="modal">
              <button className="app-auth-controls__button app-auth-controls__button--secondary" type="button">
                Create account
              </button>
            </SignUpButton>
          </>
        ) : (
          <button className="app-auth-controls__button app-auth-controls__button--secondary" disabled={isPending || !auth.isLoaded} onClick={() => void handleSignOut()} type="button">
            {isPending ? "Signing out..." : "Sign out"}
          </button>
        )}
      </div>

      {error ? <p className="app-auth-controls__error">{error}</p> : null}
    </div>
  );
}

function ClerkAuthBridge({ children }: { children: ReactNode }) {
  const auth = useClerkAuth();

  return (
    <AuthContext.Provider
      value={{
        mode: "clerk",
        getToken: async () => auth.getToken(),
        isLoaded: auth.isLoaded,
        isSignedIn: auth.isSignedIn ?? false,
        localTestProfile: null,
        signInLocalTest: async () => false,
        signOutLocalTest: async () => false,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

function LocalTestAuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(null);
  const [profile, setProfile] = useState<LocalTestAuthProfile | null>(null);
  const [isLoaded, setIsLoaded] = useState(false);

  useEffect(() => {
    syncLocalTestState(setToken, setProfile);
    setIsLoaded(true);
  }, []);

  async function signInLocalTest(nextProfile: LocalTestAuthProfile) {
    try {
      const response = await fetch("/api/local-test-auth/session", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ profile: nextProfile }),
      });
      if (!response.ok) {
        return false;
      }

      syncLocalTestState(setToken, setProfile);
      return true;
    } catch {
      return false;
    }
  }

  async function signOutLocalTest() {
    try {
      const response = await fetch("/api/local-test-auth/session", {
        method: "DELETE",
      });
      if (!response.ok) {
        return false;
      }

      syncLocalTestState(setToken, setProfile);
      return true;
    } catch {
      return false;
    }
  }

  return (
    <AuthContext.Provider
      value={{
        mode: "local_test",
        getToken: async () => readLocalTestAuthToken(),
        isLoaded,
        isSignedIn: token !== null,
        localTestProfile: profile,
        signInLocalTest,
        signOutLocalTest,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

function syncLocalTestState(
  setToken: (token: string | null) => void,
  setProfile: (profile: LocalTestAuthProfile | null) => void,
) {
  setToken(readLocalTestAuthToken());
  setProfile(readLocalTestAuthProfile());
}

function formatClerkIdentity(user: ReturnType<typeof useClerkUser>["user"]) {
  if (!user) {
    return "Signed in";
  }

  const fullName = [user.firstName, user.lastName].filter(Boolean).join(" ").trim();
  return fullName || user.username || user.primaryEmailAddress?.emailAddress || "Signed in";
}

function initialsForIdentity(identity: string) {
  const initials = identity
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? "")
    .join("");

  return initials || "CB";
}

function readLocalTestAuthToken() {
  return readCookie(localTestAuthCookieName);
}

function readLocalTestAuthProfile(): LocalTestAuthProfile | null {
  const value = readCookie(localTestAuthProfileCookieName);
  return value && isLocalTestAuthProfile(value) ? value : null;
}

function readCookie(name: string) {
  if (typeof document === "undefined") {
    return null;
  }

  const prefix = `${name}=`;
  for (const entry of document.cookie.split(";")) {
    const trimmed = entry.trim();
    if (!trimmed.startsWith(prefix)) {
      continue;
    }

    const value = trimmed.slice(prefix.length);
    return value ? decodeURIComponent(value) : null;
  }

  return null;
}
