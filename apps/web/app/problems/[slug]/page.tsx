import { ProblemDetailPage } from "../../../src/components/problem-detail-page";
import { frontendAuthEnabled } from "../../../src/lib/runtime";

type ProblemPageProps = {
  params: Promise<{
    slug: string;
  }>;
};

export default async function ProblemPage({ params }: ProblemPageProps) {
  const { slug } = await params;
  return <ProblemDetailPage authEnabled={frontendAuthEnabled()} slug={slug} />;
}
