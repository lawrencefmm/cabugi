import { formatSubmissionStatus, type SubmissionStatus } from "../lib/api";

type VerdictBadgeProps = {
  verdict: SubmissionStatus;
};

export function VerdictBadge({ verdict }: VerdictBadgeProps) {
  return <span className={`verdict-badge verdict-badge--${verdictTone(verdict)}`}>{formatSubmissionStatus(verdict)}</span>;
}

function verdictTone(verdict: SubmissionStatus) {
  switch (verdict) {
    case "accepted":
      return "accepted";
    case "queued":
    case "running":
      return "info";
    case "time_limit_exceeded":
      return "warning";
    case "wrong_answer":
    case "compile_error":
    case "runtime_error":
    case "judge_failed":
      return "error";
  }
}
