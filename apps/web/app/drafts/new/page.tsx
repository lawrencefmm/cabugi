import { ProblemAuthoringPage } from "../../../src/components/problem-authoring-page";

export default function NewDraftPage() {
  return <ProblemAuthoringPage authEnabled={Boolean(process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY)} />;
}
