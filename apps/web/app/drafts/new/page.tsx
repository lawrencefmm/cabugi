import { ProblemAuthoringPage } from "../../../src/components/problem-authoring-page";
import { frontendAuthEnabled } from "../../../src/lib/runtime";

export default function NewDraftPage() {
  return <ProblemAuthoringPage authEnabled={frontendAuthEnabled()} />;
}
