import { ProblemAuthoringPage } from "../../../src/components/problem-authoring-page";
import { frontendAuthEnabled } from "../../../src/lib/runtime";

type DraftPageProps = {
  params: Promise<{
    slug: string;
  }>;
};

export default async function DraftPage({ params }: DraftPageProps) {
  const { slug } = await params;
  return <ProblemAuthoringPage authEnabled={frontendAuthEnabled()} slug={slug} />;
}
