export function frontendAuthEnabled() {
  return Boolean(process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY) || process.env.NEXT_PUBLIC_LOCAL_TEST_AUTH_ENABLED === "true";
}
