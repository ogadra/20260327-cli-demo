import type { ReactNode } from "react";

/** Props for the PollView component. */
interface PollViewProps {
  /** Available poll options. */
  options: string[];
  /** Maximum number of choices a voter can select. */
  maxChoices: number;
  /** Current vote counts per option. */
  votes: Record<string, number>;
  /** Options the current user has voted for. */
  myChoices: string[];
  /** Callback to cast a vote for an option. */
  onVote: (choice: string) => void;
  /** Callback to withdraw a vote for an option. */
  onUnvote: (choice: string) => void;
  /** Callback to switch a vote from one option to another. */
  onSwitch: (from: string, to: string) => void;
}

/** Renders a poll with vote bars and interactive voting buttons. */
const PollView = ({
  options,
  maxChoices,
  votes,
  myChoices,
  onVote,
  onUnvote,
  onSwitch,
}: PollViewProps): ReactNode => {
  const totalVotes = options.reduce((sum, option) => sum + (votes[option] ?? 0), 0);

  /** Handle a click on a poll option. */
  const handleClick = (option: string): void => {
    const isSelected = myChoices.includes(option);
    if (isSelected) {
      if (maxChoices === 1) return;
      onUnvote(option);
      return;
    }
    if (maxChoices === 1 && myChoices.length === 1) {
      onSwitch(myChoices[0], option);
      return;
    }
    if (myChoices.length < maxChoices) {
      onVote(option);
    }
  };

  return (
    <div role="group" style={{ padding: "16px", fontFamily: "sans-serif", color: "#fff" }}>
      {options.map((option) => {
        const count = votes[option] ?? 0;
        const pct = totalVotes > 0 ? (count / totalVotes) * 100 : 0;
        const selected = myChoices.includes(option);
        const disabled = !selected && myChoices.length >= maxChoices && maxChoices !== 1;

        return (
          <button
            key={option}
            type="button"
            onClick={() => handleClick(option)}
            disabled={disabled}
            style={{
              display: "flex",
              alignItems: "center",
              width: "100%",
              padding: "12px",
              marginBottom: "8px",
              background: selected ? "#1a3a5c" : "#222",
              border: selected ? "2px solid #4a9eff" : "2px solid #444",
              borderRadius: "8px",
              color: "#fff",
              fontSize: "16px",
              cursor: disabled ? "not-allowed" : "pointer",
              position: "relative",
              overflow: "hidden",
              opacity: disabled ? 0.5 : 1,
            }}
          >
            <div
              aria-hidden="true"
              style={{
                position: "absolute",
                left: 0,
                top: 0,
                bottom: 0,
                width: `${pct}%`,
                background: selected ? "rgba(74, 158, 255, 0.3)" : "rgba(255, 255, 255, 0.1)",
                transition: "width 0.3s ease",
              }}
            />
            <span style={{ position: "relative", zIndex: 1, flexGrow: 1, textAlign: "left" }}>
              {option}
            </span>
            <span style={{ position: "relative", zIndex: 1, marginLeft: "8px", color: "#aaa" }}>
              {count}
            </span>
          </button>
        );
      })}
    </div>
  );
};

export default PollView;
