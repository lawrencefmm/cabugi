import { ProblemDetailPage } from "../../../src/components/problem-detail-page";

type ProblemPageProps = {
  params: Promise<{
    slug: string;
  }>;
};

export default async function ProblemPage({ params }: ProblemPageProps) {
  const { slug } = await params;
  return <ProblemDetailPage authEnabled={Boolean(process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY)} slug={slug} />;
}
