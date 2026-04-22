import { formatVerdict, type Verdict } from "../lib/verdict";

type VerdictBadgeProps = {
  verdict: Verdict;
};

export function VerdictBadge({ verdict }: VerdictBadgeProps) {
  return <span>{formatVerdict(verdict)}</span>;
}
