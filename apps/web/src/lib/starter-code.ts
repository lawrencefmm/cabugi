export type SubmissionLanguage = "cpp17" | "python";

export const starterCodeTemplates: Record<SubmissionLanguage, string> = {
  cpp17: `#include <iostream>

int main() {
  std::ios::sync_with_stdio(false);
  std::cin.tie(nullptr);

  return 0;
}
`,
  python: `def main() -> None:
    pass


if __name__ == "__main__":
    main()
`,
};
