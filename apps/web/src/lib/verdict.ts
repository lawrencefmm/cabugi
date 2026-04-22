export type Verdict =
  | "accepted"
  | "wrong_answer"
  | "compile_error"
  | "runtime_error"
  | "time_limit_exceeded";

export function formatVerdict(verdict: Verdict): string {
  switch (verdict) {
    case "accepted":
      return "Accepted";
    case "wrong_answer":
      return "Wrong Answer";
    case "compile_error":
      return "Compile Error";
    case "runtime_error":
      return "Runtime Error";
    case "time_limit_exceeded":
      return "Time Limit Exceeded";
  }
}
