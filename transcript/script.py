import whisper
from pathlib import Path

AUDIO_FILE = Path("1.mp3")

def main():
    if not AUDIO_FILE.exists():
        raise FileNotFoundError(f"Audio file not found: {AUDIO_FILE}")

    # Options: tiny, base, small, medium, large
    # For Hinglish/Hindi mixed audio, small/medium usually works better.
    model = whisper.load_model("small")

    result = model.transcribe(
        str(AUDIO_FILE),
        language=None,        # auto-detect language
        task="transcribe",
        fp16=False            # safer for CPU
    )

    transcript = result["text"].strip()

    output_file = AUDIO_FILE.with_suffix(".txt")
    output_file.write_text(transcript, encoding="utf-8")

    print("\n===== TRANSCRIPT =====\n")
    print(transcript)
    print(f"\nTranscript saved to: {output_file}")

if __name__ == "__main__":
    main()