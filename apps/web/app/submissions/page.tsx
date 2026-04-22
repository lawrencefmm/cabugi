import { SubmissionHistoryPage } from "../../src/components/submission-history-page";

export default function SubmissionsPage() {
  return <SubmissionHistoryPage authEnabled={Boolean(process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY)} />;
}
