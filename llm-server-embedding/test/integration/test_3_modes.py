#!/usr/bin/env python3
"""
Test des 3 modes PDF de l'API MLX-OlmOCR
Démontre : mode 'ocr', 'text', et 'auto'
"""

import os
import sys
import base64
import json
import requests
from pathlib import Path


API_URL = "http://127.0.0.1:11999/v1/chat/completions"
API_KEY = os.getenv("API_KEY", "DmdPeuln8ox8k8CFfphuKscUW5xNvSPhtNSh-CT2ieI")


def read_pdf_base64(pdf_path: str) -> str:
    """Lit un PDF et le retourne en base64"""
    with open(pdf_path, 'rb') as f:
        return base64.b64encode(f.read()).decode('utf-8')


def test_pdf_mode(pdf_path: str, mode: str, page: int = 0):
    """
    Teste un PDF avec le mode spécifié.

    Args:
        pdf_path: Chemin du PDF
        mode: 'ocr', 'text', ou 'auto'
        page: Numéro de page (0-indexed)
    """
    print(f"\n{'='*80}")
    print(f"📄 Test: {Path(pdf_path).name} (page {page + 1})")
    print(f"🔧 Mode: {mode.upper()}")
    print(f"{'='*80}")

    # Lire le PDF
    pdf_base64 = read_pdf_base64(pdf_path)

    # Construire la requête
    payload = {
        "model": "olmocr",
        "messages": [{
            "role": "user",
            "content": [
                {"type": "text", "text": "Extrait le texte de ce document."},
                {
                    "type": "pdf",
                    "data": pdf_base64,
                    "page": page,
                    "mode": mode
                }
            ]
        }],
        "max_tokens": 4096,
        "temperature": 0.1
    }

    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "Content-Type": "application/json"
    }

    print(f"🚀 Envoi requête à l'API...")

    try:
        response = requests.post(API_URL, json=payload, headers=headers, timeout=120)

        if response.status_code == 200:
            result = response.json()
            text = result['choices'][0]['message']['content']
            usage = result['usage']

            print(f"✅ Succès!")
            print(f"📊 Tokens: {usage['total_tokens']} "
                  f"(prompt: {usage['prompt_tokens']}, "
                  f"completion: {usage['completion_tokens']})")
            print(f"\n{'─'*80}")
            print(f"📝 RÉSULTAT:")
            print(f"{'─'*80}")
            print(text[:500] + "..." if len(text) > 500 else text)
            print(f"{'─'*80}\n")
        else:
            print(f"❌ Erreur API: {response.status_code}")
            print(response.text)

    except Exception as e:
        print(f"❌ Erreur: {e}")


def main():
    """Démontre les 3 modes sur différents types de PDFs"""
    test_dir = Path(__file__).parent.parent / "pdf"

    if not test_dir.exists():
        print(f"❌ Dossier {test_dir} introuvable")
        return

    print("\n" + "="*80)
    print("🧪 TEST DES 3 MODES PDF")
    print("="*80)

    # Test 1: Mode OCR (force) sur PDF texte
    print("\n📌 TEST 1: Mode 'ocr' (force OCR même sur PDF texte)")
    pdf_texte = test_dir / "texte.pdf"
    if pdf_texte.exists():
        test_pdf_mode(str(pdf_texte), mode="ocr", page=0)
    else:
        print("❌ texte.pdf introuvable")

    # Test 2: Mode TEXT (extraction native) sur PDF texte
    print("\n📌 TEST 2: Mode 'text' (extraction native rapide)")
    if pdf_texte.exists():
        test_pdf_mode(str(pdf_texte), mode="text", page=0)
    else:
        print("❌ texte.pdf introuvable")

    # Test 3: Mode AUTO sur PDF texte (doit détecter et utiliser extraction native)
    print("\n📌 TEST 3: Mode 'auto' sur PDF texte (détecte → extraction native)")
    if pdf_texte.exists():
        test_pdf_mode(str(pdf_texte), mode="auto", page=0)
    else:
        print("❌ texte.pdf introuvable")

    # Test 4: Mode AUTO sur PDF scanné (doit détecter et utiliser OCR)
    print("\n📌 TEST 4: Mode 'auto' sur PDF scanné (détecte → OCR)")
    pdf_image = test_dir / "image.pdf"
    if pdf_image.exists():
        test_pdf_mode(str(pdf_image), mode="auto", page=0)
    else:
        print("❌ image.pdf introuvable")

    # Test 5: Mode TEXT sur PDF scanné (doit échouer avec message clair)
    print("\n📌 TEST 5: Mode 'text' sur PDF scanné (doit échouer)")
    if pdf_image.exists():
        test_pdf_mode(str(pdf_image), mode="text", page=0)
    else:
        print("❌ image.pdf introuvable")

    print("\n" + "="*80)
    print("✅ Tests terminés !")
    print("="*80)
    print("\n💡 RECOMMANDATIONS:")
    print("   - Pour n8n: utilisez mode='ocr' (toujours marche)")
    print("   - Pour optimisation: utilisez mode='auto' (détection intelligente)")
    print("   - Pour PDFs texte garantis: utilisez mode='text' (ultra rapide)")
    print()


if __name__ == "__main__":
    main()
