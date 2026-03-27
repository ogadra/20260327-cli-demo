/** Data definition for a title slide. */
interface TitleSlideData {
  type: "title";
  text: string;
}

/** Data definition for a text slide. */
interface TextSlideData {
  type: "text";
  lines: string[];
}

/** Data definition for a terminal slide with command input. */
interface TerminalSlideData {
  type: "terminal";
  instruction: string;
  commands: string[];
}

/** Data definition for a poll slide. */
interface PollSlideData {
  type: "poll";
  pollId: string;
  question: string;
  options: string[];
}

/** Union type representing all possible slide data variants. */
export type SlideData = TitleSlideData | TextSlideData | TerminalSlideData | PollSlideData;

/** Ordered array of all slide data for the presentation. */
export const slideData: ReadonlyArray<SlideData> = [
  { type: "title", text: "推しツールは強制布教しよう" },
  { type: "text", lines: ["みなさん"] },
  { type: "text", lines: ["もちろん、`Nix`って使ってますよね？"] },
  {
    type: "poll",
    pollId: "nix-experience-1",
    question: "あなたはNixを使ったことがありますか",
    options: ["はい", "いいえ"],
  },
  { type: "text", lines: ["そもそもNixとはなにか"] },
  { type: "text", lines: ["宣言的で再現可能な信頼性のあるパッケージマネージャ"] },
  { type: "text", lines: ["なんぞや"] },
  { type: "text", lines: ["難しそう"] },
  { type: "text", lines: ["（ここで長々とNixの説明をする）"] },
  { type: "text", lines: ["なるほど"] },
  { type: "text", lines: ["わかった"] },
  { type: "text", lines: ["カッコ良さそうだから触ってみたい"] },
  { type: "text", lines: ["メモを残す"] },
  { type: "text", lines: ["（飲酒）"] },
  { type: "text", lines: ["帰宅"] },
  { type: "text", lines: ["就寝"] },
  { type: "text", lines: ["普通の登壇ではここで終わりです。"] },
  { type: "text", lines: ["いやいや"] },
  { type: "text", lines: ["せっかく登壇したんだから触ってもらわないと"] },
  { type: "text", lines: ["そう、思うわけです"] },
  { type: "text", lines: ["では", "**今、ここでみなさんがNixを触ったら？**"] },
  { type: "text", lines: ["この会場のNix経験率は100%になるわけです。"] },
  { type: "text", lines: ["今回はそれを目指します。"] },
  { type: "text", lines: ["Nixの推しポイント3選"] },
  {
    type: "text",
    lines: [
      "1. 設定管理・ロールバックが簡単",
      "2. 環境を汚さずコマンドが実行できる",
      "3. nix develop",
    ],
  },
  {
    type: "text",
    lines: ["まず、1から。", "お手元の画面で、実行ボタンを押すと`date`コマンドが実行できます。"],
  },
  { type: "terminal", instruction: "", commands: ["date"] },
  { type: "text", lines: ["HST？ホノルルは午前1時らしいです。大変ですね。"] },
  { type: "text", lines: ["事前にずらしておきました。"] },
  { type: "text", lines: ["設定を戻してくれや"] },
  { type: "terminal", instruction: "", commands: ["home-manager switch --rollback", "date"] },
  { type: "text", lines: ["戻りました。良かったね。"] },
  { type: "text", lines: ["これが良さの1つ目、`設定管理・ロールバックの簡単さ`です。"] },
  { type: "terminal", instruction: "", commands: ["home-manager generations"] },
  {
    type: "text",
    lines: ["コマンドを実行すると、本体設定の履歴が管理されていることがわかります。"],
  },
  { type: "text", lines: ["2つ目", "環境を汚さずコマンドが実行させられます。"] },
  { type: "text", lines: ["ポケモンにNixって言わせたいこと、ありますよね"] },
  { type: "terminal", instruction: "", commands: ["nix run nixpkgs#pokemonsay 'Nix'"] },
  { type: "text", lines: ["ポケモンにものを言わせられます。"] },
  { type: "terminal", instruction: "", commands: ["which pokemonsay"] },
  { type: "text", lines: ["PATHが通ってないので、環境はクリーンなままです。"] },
  { type: "text", lines: ["複数パッケージを同時に使いたいときは？"] },
  { type: "text", lines: ["そこでflake.nixです。", "flake.nixのあるディレクトリで"] },
  {
    type: "terminal",
    instruction: "",
    commands: ["nix develop --command sh -c \"figlet 'Nix' | cowsay -n | lolcat -f\""],
  },
  { type: "text", lines: ["を使えます。"] },
  { type: "text", lines: ["というわけでもう一度アンケート"] },
  {
    type: "poll",
    pollId: "nix-experience-2",
    question: "あなたはNixを使ったことがありますか",
    options: ["はい", "いいえ"],
  },
  { type: "text", lines: ["強制布教を完了できました"] },
  { type: "text", lines: ["みんなも推しツールは強制布教、しよう！！！"] },
  { type: "text", lines: ["Nixはいいぞ！！！！"] },
  { type: "text", lines: ["ご清聴ありがとうございました"] },
];
