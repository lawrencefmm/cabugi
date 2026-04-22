import { ProblemDetailPage } from "../../../src/components/problem-detail-page";

type ProblemPageProps = {
  params: Promise<{
    slug: string;
  }>;
};

export default async function ProblemPage({ params }: ProblemPageProps) {
  const { slug } = await params;
  return <ProblemDetailPage slug={slug} />;
}
