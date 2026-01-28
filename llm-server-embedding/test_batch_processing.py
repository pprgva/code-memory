#!/usr/bin/env python3
"""
Test batch embedding processing with multilingual texts
"""
import requests
import json
import time

API_KEY = "DmdPeuln8ox8k8CFfphuKscUW5xNvSPhtNSh-CT2ieI"
BASE_URL = "http://127.0.0.1:12000"

def test_batch_embeddings():
    """Test batch processing with multiple texts"""
    print("🧪 Test: Batch Embeddings (3 texts)")
    print("=" * 80)

    start = time.time()

    response = requests.post(
        f"{BASE_URL}/v1/embeddings",
        headers={
            "Authorization": f"Bearer {API_KEY}",
            "Content-Type": "application/json"
        },
        json={
            "input": [
                "Hello world",
                "Bonjour le monde",
                "Hola mundo"
            ],
            "model": "multilingual-e5-large"
        }
    )

    elapsed = time.time() - start

    print(f"Status: {response.status_code}")
    print(f"Duration: {elapsed:.2f}s")

    if response.status_code == 200:
        data = response.json()
        print(f"✅ Success!")
        print(f"Model: {data['model']}")
        print(f"Embeddings count: {len(data['data'])}")
        print(f"Usage: {data['usage']}")

        for i, item in enumerate(data['data']):
            print(f"\nEmbedding {i}:")
            print(f"  Index: {item['index']}")
            print(f"  Dimension: {len(item['embedding'])}")
            print(f"  First 5 values: {item['embedding'][:5]}")
    else:
        print(f"❌ Error: {response.text}")

    print("\n" + "=" * 80)


def test_multilingual():
    """Test multilingual capabilities"""
    print("🌍 Test: Multilingual Capabilities")
    print("=" * 80)

    texts = [
        "The quick brown fox jumps over the lazy dog",  # English
        "Le renard brun rapide saute par-dessus le chien paresseux",  # French
        "El rápido zorro marrón salta sobre el perro perezoso",  # Spanish
        "Der schnelle braune Fuchs springt über den faulen Hund",  # German
        "速い茶色のキツネが怠け者の犬を飛び越える"  # Japanese
    ]

    start = time.time()

    response = requests.post(
        f"{BASE_URL}/v1/embeddings",
        headers={
            "Authorization": f"Bearer {API_KEY}",
            "Content-Type": "application/json"
        },
        json={
            "input": texts,
            "model": "multilingual-e5-large"
        }
    )

    elapsed = time.time() - start

    print(f"Status: {response.status_code}")
    print(f"Duration: {elapsed:.2f}s ({elapsed/len(texts):.2f}s per text)")

    if response.status_code == 200:
        data = response.json()
        print(f"✅ Success!")
        print(f"Processed {len(data['data'])} multilingual texts")
        print(f"Total tokens: {data['usage']['total_tokens']}")
    else:
        print(f"❌ Error: {response.text}")

    print("\n" + "=" * 80)


if __name__ == "__main__":
    # Test batch processing
    test_batch_embeddings()

    print("\n")

    # Test multilingual
    test_multilingual()
