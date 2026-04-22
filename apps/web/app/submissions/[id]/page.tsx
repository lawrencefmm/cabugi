import { SubmissionSummaryPage } from "../../../src/components/submission-summary-page";

type SubmissionPageProps = {
  params: Promise<{
    id: string;
  }>;
};

export default async function SubmissionPage({ params }: SubmissionPageProps) {
  const { id } = await params;
  return <SubmissionSummaryPage authEnabled={Boolean(process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY)} submissionId={id} />;
}
