import { VerdictBadge } from "../src/components/verdict-badge";

export default function HomePage() {
  return (
    <main>
      <h1>Cabugi</h1>
      <p>Competitive programming practice platform.</p>
      <VerdictBadge verdict="accepted" />
    </main>
  );
}
