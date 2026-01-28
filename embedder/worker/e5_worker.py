"""E5 embedding STDIO worker. Reads JSON lines from stdin, writes JSON lines to stdout."""

import argparse
import json
import sys

import torch
from transformers import AutoModel, AutoTokenizer


def log(msg: str) -> None:
    """Log to stderr to keep stdout clean for protocol messages."""
    print(f"[e5_worker] {msg}", file=sys.stderr, flush=True)


def send(obj: dict) -> None:
    """Write a JSON line to stdout."""
    print(json.dumps(obj), file=sys.stdout, flush=True)


def load_model(model_path: str) -> tuple:
    """Load tokenizer and model, return (tokenizer, model, device)."""
    log(f"Loading model from {model_path}")
    tokenizer = AutoTokenizer.from_pretrained(model_path)
    device = "mps" if torch.backends.mps.is_available() else "cpu"
    model = AutoModel.from_pretrained(model_path).to(device)
    model.eval()
    log(f"Model loaded on {device}")
    return tokenizer, model, device


def embed(tokenizer, model, device: str, texts: list[str], input_type: str) -> list[list[float]]:
    """Generate L2-normalized E5 embeddings for the given texts."""
    prefix = "query: " if input_type == "query" else "passage: "
    prefixed = [prefix + t for t in texts]

    encoded = tokenizer(prefixed, padding=True, truncation=True, max_length=512, return_tensors="pt").to(device)

    with torch.no_grad():
        output = model(**encoded)

    token_emb = output[0]
    mask = encoded["attention_mask"].unsqueeze(-1).expand(token_emb.size()).float()
    pooled = torch.sum(token_emb * mask, 1) / torch.clamp(mask.sum(1), min=1e-9)
    embeddings = torch.nn.functional.normalize(pooled, p=2, dim=1)

    return embeddings.cpu().numpy().tolist()


def run_loop(tokenizer, model, device: str, dimensions: int) -> None:
    """Read JSON lines from stdin, process, write results to stdout."""
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        try:
            req = json.loads(line)
            req_id = req["id"]
            texts = req["texts"]
            input_type = req.get("input_type", "passage")
            result = embed(tokenizer, model, device, texts, input_type)
            send({"id": req_id, "embeddings": result, "dimensions": dimensions})
        except json.JSONDecodeError as e:
            log(f"Invalid JSON: {e}")
        except KeyError as e:
            send({"id": req.get("id", "unknown"), "error": f"Missing field: {e}"})
        except Exception as e:
            send({"id": req.get("id", "unknown"), "error": str(e)})


def main() -> None:
    parser = argparse.ArgumentParser(description="E5 embedding STDIO worker")
    parser.add_argument("--model-path", required=True, help="Path to the E5 model directory")
    args = parser.parse_args()

    tokenizer, model, device = load_model(args.model_path)

    dimensions = model.config.hidden_size
    send({"status": "ready", "dimensions": dimensions})

    run_loop(tokenizer, model, device, dimensions)


if __name__ == "__main__":
    try:
        main()
    except (KeyboardInterrupt, BrokenPipeError):
        pass
