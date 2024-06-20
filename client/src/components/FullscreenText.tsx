interface FullscreenTextProps {
  text: string;
}

function FullscreenText({ text }: FullscreenTextProps) {
  return (
    <div className="flex h-screen justify-center items-center text-xl">
      {text}
    </div>
  );
}

export default FullscreenText;
