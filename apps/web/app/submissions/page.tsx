import { SubmissionHistoryPage } from "../../src/components/submission-history-page";
import { frontendAuthEnabled } from "../../src/lib/runtime";

export default function SubmissionsPage() {
  return <SubmissionHistoryPage authEnabled={frontendAuthEnabled()} />;
}
