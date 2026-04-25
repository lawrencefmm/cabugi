export const localTestAuthCookieName = "cabugi_local_test_auth_token";
export const localTestAuthProfileCookieName = "cabugi_local_test_auth_profile";

export type LocalTestAuthProfile = "author" | "moderator";

export const localTestAuthProfileLabels: Record<LocalTestAuthProfile, string> = {
  author: "Author",
  moderator: "Moderator",
};

export function isLocalTestAuthProfile(value: string): value is LocalTestAuthProfile {
  return value === "author" || value === "moderator";
}
