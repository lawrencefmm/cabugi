import { ModerationProblemDraftsPage } from "../../../src/components/moderation-problem-drafts-page";
import { frontendAuthEnabled } from "../../../src/lib/runtime";

export default function ModerationPage() {
  return <ModerationProblemDraftsPage authEnabled={frontendAuthEnabled()} />;
}
