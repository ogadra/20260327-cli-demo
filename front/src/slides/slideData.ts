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
  { type: "title", text: "推しツールは\n強制布教しよう" },
  { type: "text", lines: ["みなさん"] },
  { type: "text", lines: ["もちろん、\n`Nix`\nって使ってますよね？"] },
  {
    type: "poll",
    pollId: "nix-experience-1",
    question: "あなたは`Nix`経験者？",
    options: ["はい", "いいえ"],
  },
  { type: "text", lines: ["そもそも\n`Nix`\nとはなにか"] },
  { type: "text", lines: ["宣言的で\n再現可能な\n信頼性のある\nパッケージマネージャ"] },
  { type: "text", lines: ["なんぞや"] },
  { type: "text", lines: ["難しそう"] },
  { type: "text", lines: ["（長々としたNixの説明）"] },
  { type: "text", lines: ["なるほど"] },
  { type: "text", lines: ["わかった"] },
  { type: "text", lines: ["カッコ良さそうだから\n触ってみたい"] },
  { type: "text", lines: ["メモを残す"] },
  { type: "text", lines: ["（飲酒）"] },
  { type: "text", lines: ["帰宅"] },
  { type: "text", lines: ["就寝"] },
  { type: "text", lines: ["普通の登壇では\nここで終わりです。"] },
  { type: "text", lines: ["いやいや"] },
  { type: "text", lines: ["せっかく\n登壇したんだから\n触ってもらわないと"] },
  { type: "text", lines: ["そう、思うわけです"] },
  { type: "text", lines: ["では", "**今、ここでみなさんが**\n`Nix`\n**を触ったら？**"] },
  { type: "text", lines: ["この会場の\nNix経験率は\n100%になるわけです。"] },
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
  { type: "text", lines: ["HST？\nホノルルは午前2時らしいです。\n大変ですね。"] },
  { type: "text", lines: ["事前に\nずらしておきました。"] },
  { type: "text", lines: ["設定を戻してくれや"] },
  { type: "terminal", instruction: "", commands: ["home-manager switch --rollback", "date"] },
  { type: "text", lines: ["戻りました。\n良かったね。"] },
  { type: "text", lines: ["これが良さの1つ目、\n`設定管理・ロールバックの簡単さ`\nです。"] },
  {
    type: "text",
    lines: ["コマンドを実行すると、\n本体設定の履歴が\n管理されていることが\nわかります。"],
  },
  { type: "terminal", instruction: "", commands: ["home-manager generations"] },
  { type: "text", lines: ["2つ目", "環境を汚さず\nコマンドが\n実行させられます。"] },
  { type: "text", lines: ["ポケモンに\n`Nix`\nって言わせたいこと、\nありますよね"] },
  { type: "terminal", instruction: "", commands: ["nix run nixpkgs#pokemonsay 'Nix'"] },
  { type: "text", lines: ["ポケモンに\nものを言わせられます。"] },
  { type: "terminal", instruction: "", commands: ["which pokemonsay"] },
  { type: "text", lines: ["PATHが通ってないので、\n環境はクリーンなままです。"] },
  { type: "text", lines: ["複数パッケージを\n同時に\n使いたいときは？"] },
  { type: "text", lines: ["そこで\n`flake.nix`\nです。", "`flake.nix`のある\nディレクトリで"] },
  {
    type: "terminal",
    instruction: "",
    commands: ["nix develop --command sh -c \"figlet 'Nix' | cowsay -n | lolcat -f\""],
  },
  { type: "text", lines: ["を使えます。"] },
  { type: "text", lines: ["というわけで\nもう一度アンケート"] },
  {
    type: "poll",
    pollId: "nix-experience-2",
    question: "あなたは`Nix`経験者？",
    options: ["はい", "いいえ"],
  },
  { type: "text", lines: ["強制布教を\n完了できました"] },
  { type: "text", lines: ["みんなも\n推しツールは強制布教、\nしよう！！！"] },
  { type: "text", lines: ["Nixはいいぞ！！！！"] },
  { type: "text", lines: ["ご清聴ありがとうございました"] },
];
