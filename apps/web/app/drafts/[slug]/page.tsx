import { ProblemAuthoringPage } from "../../../src/components/problem-authoring-page";

type DraftPageProps = {
  params: Promise<{
    slug: string;
  }>;
};

export default async function DraftPage({ params }: DraftPageProps) {
  const { slug } = await params;
  return <ProblemAuthoringPage authEnabled={Boolean(process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY)} slug={slug} />;
}
