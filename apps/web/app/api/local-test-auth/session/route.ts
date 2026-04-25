import { NextResponse } from "next/server";

import { isLocalTestAuthProfile, localTestAuthCookieName, localTestAuthProfileCookieName, type LocalTestAuthProfile } from "../../../../src/lib/local-test-auth";
import { localTestAuthTokenForProfile } from "../../../../src/lib/local-test-auth-server";

export async function POST(request: Request) {
  if (process.env.NEXT_PUBLIC_LOCAL_TEST_AUTH_ENABLED !== "true") {
    return NextResponse.json({ error: "not_found" }, { status: 404 });
  }

  const requestedProfile = await parseRequestedProfile(request);
  if (requestedProfile === null) {
    return NextResponse.json({ error: "invalid_profile" }, { status: 400 });
  }

  const response = NextResponse.json({ profile: requestedProfile });
  response.cookies.set(localTestAuthCookieName, localTestAuthTokenForProfile(requestedProfile), {
    httpOnly: false,
    path: "/",
    sameSite: "lax",
  });
  response.cookies.set(localTestAuthProfileCookieName, requestedProfile, {
    httpOnly: false,
    path: "/",
    sameSite: "lax",
  });
  return response;
}

export async function DELETE() {
  if (process.env.NEXT_PUBLIC_LOCAL_TEST_AUTH_ENABLED !== "true") {
    return NextResponse.json({ error: "not_found" }, { status: 404 });
  }

  const response = NextResponse.json({ ok: true });
  response.cookies.set(localTestAuthCookieName, "", { expires: new Date(0), path: "/" });
  response.cookies.set(localTestAuthProfileCookieName, "", { expires: new Date(0), path: "/" });
  return response;
}

async function parseRequestedProfile(request: Request): Promise<LocalTestAuthProfile | null> {
  try {
    const payload = (await request.json()) as { profile?: string };
    if (typeof payload.profile === "string" && isLocalTestAuthProfile(payload.profile)) {
      return payload.profile;
    }
  } catch {
    return null;
  }

  return null;
}
