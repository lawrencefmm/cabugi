import { ModerationProblemDraftsPage } from "../../../src/components/moderation-problem-drafts-page";

export default function ModerationPage() {
  return <ModerationProblemDraftsPage authEnabled={Boolean(process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY)} />;
}
