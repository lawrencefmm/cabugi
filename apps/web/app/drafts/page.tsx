import { ProblemDraftsPage } from "../../src/components/problem-drafts-page";
import { frontendAuthEnabled } from "../../src/lib/runtime";

export default function DraftsPage() {
  return <ProblemDraftsPage authEnabled={frontendAuthEnabled()} />;
}
