import assert from "node:assert/strict";

import { chromium } from "playwright";

const authCookieName = "cabugi_local_test_auth_token";

const apiBaseUrl = requiredEnv("E2E_API_BASE_URL");
const webBaseUrl = requiredEnv("E2E_WEB_BASE_URL");
const authorToken = requiredEnv("E2E_AUTHOR_TOKEN");
const moderatorToken = requiredEnv("E2E_MODERATOR_TOKEN");

const runID = Date.now().toString();
const draftSlug = `e2e-mirror-number-${runID}`;
const draftTitle = `E2E Mirror Number ${runID}`;
const acceptedPythonSolution = `def main() -> None:\n    import sys\n\n    value = sys.stdin.read().strip()\n    print(value)\n\n\nif __name__ == "__main__":\n    main()\n`;
const draftBundle = JSON.stringify({
  cases: [
    { input: "7\n", expectedOutput: "7\n" },
    { input: "42\n", expectedOutput: "42\n" },
  ],
});

const browser = await chromium.launch({ headless: true });

try {
  const authorContext = await newSignedInContext(browser, authorToken);
  const authorPage = await authorContext.newPage();

  await verifyPublishedProblemBrowsing(authorPage);
  await verifyDraftAuthoring(authorPage);

  const moderatorContext = await newSignedInContext(browser, moderatorToken);
  const moderatorPage = await moderatorContext.newPage();
  await verifyModerationApproval(moderatorPage);

  const publicContext = await browser.newContext();
  const publicPage = await publicContext.newPage();
  await verifyProblemPublished(publicPage);
  const submissionID = await verifyPublishedDraftSubmission(authorPage);

  await Promise.all([authorContext.close(), moderatorContext.close(), publicContext.close()]);

  console.log(`browser smoke passed for submission ${submissionID} and draft ${draftSlug}`);
} finally {
  await browser.close();
}

async function verifyPublishedProblemBrowsing(page) {
  await goto(page, "/");
  await page.getByRole("link", { name: "A + B" }).click();
  await page.getByRole("heading", { name: "A + B" }).waitFor();
}

async function verifyPublishedDraftSubmission(page) {
  await goto(page, `/problems/${draftSlug}`);
  await page.getByRole("heading", { name: draftTitle }).waitFor();

  await page.getByLabel("Language").selectOption("python");
  await setSolveWorkspaceSource(page, acceptedPythonSolution);

  const createSubmissionResponse = page.waitForResponse((response) => {
    return response.url().endsWith("/v1/submissions") && response.request().method() === "POST" && response.status() === 201;
  });

  await page.getByRole("button", { name: "Submit solution" }).click();

  const createdSubmissionResponse = await createSubmissionResponse;
  const requestPayload = createdSubmissionResponse.request().postDataJSON();
  assert.equal(requestPayload.problemSlug, draftSlug, "browser submission must target the published draft problem");
  assert.match(requestPayload.sourceCode, /print\(value\)/, "browser submission must send the accepted Python source");

  const submissionPayload = await createdSubmissionResponse.json();
  assert.equal(typeof submissionPayload.id, "string", "submission creation must return an id");

  const finalSubmission = await waitForTerminalSubmission(authorToken, submissionPayload.id);
  assert.equal(
    finalSubmission.status,
    "accepted",
    `expected accepted verdict, got ${finalSubmission.status} with results ${JSON.stringify(finalSubmission.results ?? [])}`,
  );

  await goto(page, `/submissions/${submissionPayload.id}`);
  await page.getByText("Final verdict").waitFor();
  await page.locator(".submission-detail .verdict-badge--accepted").first().waitFor();

  return submissionPayload.id;
}

async function verifyDraftAuthoring(page) {
  await goto(page, "/drafts/new");
  await page.getByText("Problem Authoring").waitFor();
  await page.getByLabel("Slug").fill(draftSlug);
  await page.getByLabel("Title").fill(draftTitle);
  await page.getByLabel("Time limit (ms)").fill("5000");
  await page.getByLabel("Statement Markdown").fill("Read one integer and print the same integer.");
  await page.getByLabel("Input Markdown").fill("One integer n.");
  await page.getByLabel("Output Markdown").fill("Print n.");
  await page.getByLabel("Constraints Markdown").fill("|n| <= 10^9");
  await page.getByLabel("Notes Markdown").fill("Smoke-test draft for browser verification.");
  await page.getByLabel("Hidden test bundle file").setInputFiles({
    buffer: Buffer.from(draftBundle),
    mimeType: "application/json",
    name: "mirror-number.json",
  });

  await page.locator("button:has-text('Upload bundle')").click();
  await page.getByText("Hidden test bundle uploaded.").waitFor();

  const createDraftResponse = page.waitForResponse((response) => {
    return response.url().endsWith("/v1/problem-drafts") && response.request().method() === "POST";
  });

  await page.getByRole("button", { name: "Create draft" }).click();
  const createdDraftResponse = await createDraftResponse;
  assert.equal(createdDraftResponse.status(), 201, `draft creation failed with ${createdDraftResponse.status()} and body ${await createdDraftResponse.text()}`);
  await page.waitForURL(urlFor(`/drafts/${draftSlug}`));
  await page.getByRole("button", { name: "Submit for review" }).waitFor();

  await page.getByRole("button", { name: "Submit for review" }).click();
  await page.getByText("Draft submitted for review.").waitFor();
  await page.getByText("In Review").waitFor();
}

async function verifyModerationApproval(page) {
  await goto(page, "/moderation/problem-drafts");
  await page.getByRole("button", { name: new RegExp(draftTitle) }).click();
  await page.getByLabel("Moderation notes").fill("Browser smoke approval.");

  const moderationResponse = page.waitForResponse((response) => {
    return response.url().endsWith(`/v1/moderation/problem-drafts/${draftSlug}/decision`) && response.request().method() === "POST";
  });

  await page.getByRole("button", { name: "Approve" }).click();
  const appliedDecisionResponse = await moderationResponse;
  assert.equal(appliedDecisionResponse.status(), 200, `moderation approval failed with ${appliedDecisionResponse.status()} and body ${await appliedDecisionResponse.text()}`);
}

async function verifyProblemPublished(page) {
  await goto(page, "/");
  await page.getByRole("link", { name: draftTitle }).waitFor();
  await page.getByRole("link", { name: draftTitle }).click();
  await page.getByRole("heading", { name: draftTitle }).waitFor();
}

async function newSignedInContext(browserInstance, token) {
  const context = await browserInstance.newContext();
  await context.addCookies([
    {
      name: authCookieName,
      sameSite: "Lax",
      url: webBaseUrl,
      value: token,
    },
  ]);
  return context;
}

async function goto(page, path) {
  const response = await page.goto(urlFor(path), { waitUntil: "domcontentloaded" });
  assert(response?.ok(), `navigation to ${path} must succeed`);
}

async function setSolveWorkspaceSource(page, sourceCode) {
  await page.waitForFunction(() => Boolean(window.__CABUGI_E2E__?.solveWorkspace));
  await page.evaluate((value) => {
    window.__CABUGI_E2E__?.solveWorkspace?.setSourceCode(value);
  }, sourceCode);
}

function requiredEnv(name) {
  const value = process.env[name];
  assert(value, `missing required environment variable ${name}`);
  return value;
}

function urlFor(path) {
  return new URL(path, webBaseUrl).toString();
}

async function waitForTerminalSubmission(token, submissionID) {
  const startedAt = Date.now();
  while (Date.now() - startedAt < 30000) {
    const response = await fetch(`${apiBaseUrl}/v1/submissions/${submissionID}`, {
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${token}`,
      },
    });

    assert(response.ok, `submission polling failed with status ${response.status}`);
    const submission = await response.json();
    if (isTerminalStatus(submission.status)) {
      return submission;
    }

    await new Promise((resolve) => setTimeout(resolve, 500));
  }

  throw new Error(`submission ${submissionID} did not reach a terminal verdict in time`);
}

function isTerminalStatus(status) {
  return ["accepted", "wrong_answer", "compile_error", "runtime_error", "time_limit_exceeded", "judge_failed"].includes(status);
}
