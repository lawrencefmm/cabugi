import { SubmissionSummaryPage } from "../../../src/components/submission-summary-page";
import { frontendAuthEnabled } from "../../../src/lib/runtime";

type SubmissionPageProps = {
  params: Promise<{
    id: string;
  }>;
};

export default async function SubmissionPage({ params }: SubmissionPageProps) {
  const { id } = await params;
  return <SubmissionSummaryPage authEnabled={frontendAuthEnabled()} submissionId={id} />;
}
