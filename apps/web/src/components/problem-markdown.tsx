import Markdown from "react-markdown";
import rehypeKatex from "rehype-katex";
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";

type ProblemMarkdownProps = {
  content: string;
};

export function ProblemMarkdown({ content }: ProblemMarkdownProps) {
  return (
    <div className="problem-markdown">
      <Markdown remarkPlugins={[remarkGfm, remarkMath]} rehypePlugins={[rehypeKatex]}>
        {content}
      </Markdown>
    </div>
  );
}
