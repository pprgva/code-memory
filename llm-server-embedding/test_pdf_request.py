#!/usr/bin/env python3
"""
Script de test pour envoyer un PDF réel au serveur OCR.
"""

import base64
import json
import os
import sys
from pathlib import Path

import requests

# Configuration
API_URL = "http://localhost:11999/v1/chat/completions"
API_KEY = os.getenv("API_KEY", "your-secret-key-here")


def create_test_pdf() -> bytes:
    """Crée un mini PDF de test."""
    # Mini PDF valide avec du texte
    pdf_content = b"""%PDF-1.4
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj
2 0 obj
<< /Type /Pages /Kids [3 0 R] /Count 1 >>
endobj
3 0 obj
<< /Type /Page /Parent 2 0 R /Resources 4 0 R /MediaBox [0 0 612 792] /Contents 5 0 R >>
endobj
4 0 obj
<< /Font << /F1 << /Type /Font /Subtype /Type1 /BaseFont /Helvetica >> >> >>
endobj
5 0 obj
<< /Length 44 >>
stream
BT
/F1 12 Tf
100 700 Td
(Hello World!) Tj
ET
endstream
endobj
xref
0 6
0000000000 65535 f
0000000009 00000 n
0000000058 00000 n
0000000115 00000 n
0000000214 00000 n
0000000304 00000 n
trailer
<< /Size 6 /Root 1 0 R >>
startxref
398
%%EOF
"""
    return pdf_content


def test_pdf_request(mode: str = "auto"):
    """
    Teste l'envoi d'un PDF au serveur.

    Args:
        mode: 'text', 'ocr', ou 'auto'
    """
    print(f"\n{'='*60}")
    print(f"Test PDF Request - Mode: {mode}")
    print(f"{'='*60}\n")

    # Créer ou lire un PDF de test
    pdf_data = create_test_pdf()
    print(f"✅ PDF créé: {len(pdf_data)} bytes")

    # Encoder en base64
    pdf_base64 = base64.b64encode(pdf_data).decode('utf-8')
    print(f"✅ PDF encodé: {len(pdf_base64)} caractères base64")

    # Préparer la requête selon le format attendu par le serveur
    request_body = {
        "model": "olmocr",
        "messages": [
            {
                "role": "user",
                "content": [
                    {
                        "type": "text",
                        "text": "Extrait le texte de ce document"
                    },
                    {
                        "type": "pdf",
                        "data": pdf_base64,
                        "page": 0,
                        "mode": mode
                    }
                ]
            }
        ],
        "max_tokens": 2048,
        "temperature": 1.0
    }

    print(f"\n📤 Envoi de la requête...")
    print(f"   URL: {API_URL}")
    print(f"   Mode PDF: {mode}")

    # Envoyer la requête
    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "Content-Type": "application/json"
    }

    try:
        response = requests.post(
            API_URL,
            headers=headers,
            json=request_body,
            timeout=30
        )

        print(f"\n📥 Réponse reçue:")
        print(f"   Status: {response.status_code}")
        print(f"   Headers: {dict(response.headers)}")

        # Afficher le corps de la réponse
        try:
            response_json = response.json()
            print(f"\n   Body:")
            print(json.dumps(response_json, indent=2, ensure_ascii=False))
        except:
            print(f"\n   Body (raw):")
            print(response.text)

        if response.status_code == 200:
            print(f"\n✅ Succès!")
            return True
        else:
            print(f"\n❌ Erreur {response.status_code}")
            return False

    except requests.exceptions.RequestException as e:
        print(f"\n❌ Erreur de connexion: {e}")
        return False


if __name__ == "__main__":
    # Tester les 3 modes
    modes = ["text", "auto", "ocr"]

    if len(sys.argv) > 1:
        # Mode spécifique fourni
        mode = sys.argv[1]
        if mode not in modes:
            print(f"Mode invalide: {mode}")
            print(f"Modes disponibles: {', '.join(modes)}")
            sys.exit(1)
        test_pdf_request(mode)
    else:
        # Tester tous les modes
        for mode in modes:
            success = test_pdf_request(mode)
            if not success:
                print(f"\n⚠️  Mode {mode} a échoué, arrêt des tests.")
                break
