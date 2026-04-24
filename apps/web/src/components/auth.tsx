"use client";

import { ClerkProvider, SignInButton as ClerkSignInButton, useAuth as useClerkAuth } from "@clerk/nextjs";
import { createContext, type ReactNode, useContext, useEffect, useState } from "react";

type AuthState = {
  mode: "disabled" | "clerk" | "local_test";
  getToken: () => Promise<string | null>;
  isLoaded: boolean;
  isSignedIn: boolean;
};

type AuthProviderProps = {
  children: ReactNode;
  localTestAuthEnabled?: boolean;
  publishableKey?: string;
};

type SignInButtonProps = {
  children: ReactNode;
  mode?: "modal" | "redirect";
};

const localTestAuthCookieName = "cabugi_local_test_auth_token";

const disabledAuthState: AuthState = {
  mode: "disabled",
  getToken: async () => null,
  isLoaded: true,
  isSignedIn: false,
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

export function SignInButton({ children, mode }: SignInButtonProps) {
  const auth = useAuth();
  if (auth.mode === "clerk") {
    return <ClerkSignInButton mode={mode}>{children}</ClerkSignInButton>;
  }

  return <>{children}</>;
}

export function useAuth() {
  return useContext(AuthContext);
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
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

function LocalTestAuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(null);
  const [isLoaded, setIsLoaded] = useState(false);

  useEffect(() => {
    setToken(readLocalTestAuthToken());
    setIsLoaded(true);
  }, []);

  return (
    <AuthContext.Provider
      value={{
        mode: "local_test",
        getToken: async () => readLocalTestAuthToken(),
        isLoaded,
        isSignedIn: token !== null,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

function readLocalTestAuthToken() {
  if (typeof document === "undefined") {
    return null;
  }

  const prefix = `${localTestAuthCookieName}=`;
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
