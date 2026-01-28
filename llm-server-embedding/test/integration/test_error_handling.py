#!/usr/bin/env python3
"""Test de la gestion d'erreur mode 'text' sur PDF scanné"""

import os
import base64
import json
import requests
from pathlib import Path

API_URL = "http://127.0.0.1:11999/v1/chat/completions"
API_KEY = "DmdPeuln8ox8k8CFfphuKscUW5xNvSPhtNSh-CT2ieI"

# Lire image.pdf (PDF scanné)
pdf_path = Path(__file__).parent.parent / "pdf" / "image.pdf"
with open(pdf_path, 'rb') as f:
    pdf_base64 = base64.b64encode(f.read()).decode('utf-8')

# Test: mode 'text' sur PDF scanné (doit retourner 400)
payload = {
    "messages": [{
        "role": "user",
        "content": [
            {"type": "text", "text": "Extraire le texte"},
            {"type": "pdf", "data": pdf_base64, "mode": "text"}
        ]
    }]
}

print("🧪 Test: Mode 'text' sur PDF scanné")
print("=" * 60)

response = requests.post(
    API_URL,
    json=payload,
    headers={"Authorization": f"Bearer {API_KEY}"},
    timeout=30
)

print(f"Status: {response.status_code}")
print(f"\nRéponse:\n{json.dumps(response.json(), indent=2, ensure_ascii=False)}")

if response.status_code == 400:
    print("\n✅ Erreur 400 retournée correctement!")
else:
    print(f"\n❌ Status incorrect: {response.status_code} (attendu: 400)")
