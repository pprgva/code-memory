#!/usr/bin/env python3
"""
Script de test pour l'OCR avec PDFs (scannés, texte, hybrides)
Teste l'API MLX-OlmOCR avec différents types de documents
"""

import os
import sys
import base64
import json
import requests
import fitz  # PyMuPDF
from pathlib import Path
from io import BytesIO
from PIL import Image


API_URL = "http://127.0.0.1:11999/v1/chat/completions"
API_KEY = os.getenv("API_KEY", "DmdPeuln8ox8k8CFfphuKscUW5xNvSPhtNSh-CT2ieI")


def pdf_to_image_base64(pdf_path: str, page_num: int = 0, dpi: int = 150) -> str:
    """
    Convertit une page PDF en image PNG encodée en base64.

    Args:
        pdf_path: Chemin du fichier PDF
        page_num: Numéro de page (0-indexed)
        dpi: Résolution de l'image (150 = qualité standard)

    Returns:
        str: Image encodée en base64 (format data:image/png;base64,...)
    """
    # Ouvrir le PDF
    pdf_doc = fitz.open(pdf_path)

    if page_num >= pdf_doc.page_count:
        page_num = 0

    # Récupérer la page
    page = pdf_doc[page_num]

    # Convertir en image (matrix pour la résolution)
    zoom = dpi / 72  # 72 DPI par défaut
    mat = fitz.Matrix(zoom, zoom)
    pix = page.get_pixmap(matrix=mat)

    # Convertir en PNG
    img_data = pix.tobytes("png")

    # Encoder en base64
    img_base64 = base64.b64encode(img_data).decode('utf-8')

    pdf_doc.close()

    return f"data:image/png;base64,{img_base64}"


def extract_text_from_pdf(pdf_path: str, page_num: int = 0) -> str:
    """
    Extrait le texte natif d'une page PDF (pour PDFs texte ou hybrides).

    Args:
        pdf_path: Chemin du fichier PDF
        page_num: Numéro de page

    Returns:
        str: Texte extrait (vide si PDF scanné)
    """
    pdf_doc = fitz.open(pdf_path)

    if page_num >= pdf_doc.page_count:
        page_num = 0

    page = pdf_doc[page_num]
    text = page.get_text()

    pdf_doc.close()

    return text.strip()


def detect_pdf_type(pdf_path: str, page_num: int = 0) -> str:
    """
    Détecte le type de PDF: scanné, texte, ou hybride.

    Returns:
        str: 'scanned', 'text', ou 'hybrid'
    """
    text = extract_text_from_pdf(pdf_path, page_num)

    if not text or len(text) < 50:
        return "scanned"  # Peu ou pas de texte = PDF scanné

    # Vérifier s'il y a des images dans la page
    pdf_doc = fitz.open(pdf_path)
    page = pdf_doc[page_num]
    images = page.get_images()
    pdf_doc.close()

    if images:
        return "hybrid"  # Texte + images
    else:
        return "text"  # Seulement du texte


def test_ocr(pdf_path: str, page_num: int = 0, prompt: str = None):
    """
    Teste l'OCR avec un PDF (adaptatif selon le type).

    Args:
        pdf_path: Chemin du PDF
        page_num: Numéro de page à tester
        prompt: Question/instruction personnalisée (optionnel)
    """
    print(f"\n{'='*80}")
    print(f"📄 Test OCR: {Path(pdf_path).name} (page {page_num + 1})")
    print(f"{'='*80}")

    # Détecter le type de PDF
    pdf_type = detect_pdf_type(pdf_path, page_num)
    print(f"🔍 Type détecté: {pdf_type.upper()}")

    # Préparer la requête selon le type
    if pdf_type == "text":
        # PDF texte: extraire le texte nativement (pas besoin d'OCR)
        text = extract_text_from_pdf(pdf_path, page_num)
        print(f"📝 Texte extrait ({len(text)} caractères)")
        print(f"\nPremiers 200 caractères:")
        print(text[:200] + "...\n")
        return  # Pas besoin d'appeler l'API pour du texte pur

    # Pour PDFs scannés ou hybrides: utiliser l'OCR
    print(f"🖼️  Conversion en image...")
    image_base64 = pdf_to_image_base64(pdf_path, page_num)

    # Construire le message
    if not prompt:
        if pdf_type == "hybrid":
            prompt = "Extrait tout le texte de ce document (texte natif + images)."
        else:
            prompt = "Extrait tout le texte de cette image scannée."

    content = [
        {"type": "text", "text": prompt},
        {"type": "image_url", "image_url": {"url": image_base64}}
    ]

    payload = {
        "model": "olmocr",
        "messages": [{"role": "user", "content": content}],
        "max_tokens": 4096,
        "temperature": 0.1  # Basse température pour OCR précis
    }

    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "Content-Type": "application/json"
    }

    print(f"🚀 Envoi requête OCR à l'API...")

    try:
        response = requests.post(API_URL, json=payload, headers=headers, timeout=120)

        if response.status_code == 200:
            result = response.json()
            text = result['choices'][0]['message']['content']
            usage = result['usage']

            print(f"✅ OCR réussi!")
            print(f"📊 Tokens: {usage['total_tokens']} (prompt: {usage['prompt_tokens']}, completion: {usage['completion_tokens']})")
            print(f"\n{'─'*80}")
            print(f"📝 RÉSULTAT OCR:")
            print(f"{'─'*80}")
            print(text)
            print(f"{'─'*80}\n")
        else:
            print(f"❌ Erreur API: {response.status_code}")
            print(response.text)

    except Exception as e:
        print(f"❌ Erreur: {e}")


def main():
    """Teste tous les PDFs du dossier test/pdf/"""
    test_dir = Path(__file__).parent.parent / "pdf"

    if not test_dir.exists():
        print(f"❌ Dossier {test_dir} introuvable")
        return

    # Lister les PDFs
    pdfs = sorted(test_dir.glob("*.pdf"))

    if not pdfs:
        print(f"❌ Aucun PDF dans {test_dir}")
        return

    print(f"\n🎯 {len(pdfs)} PDFs trouvés dans {test_dir}/")

    # Si argument, tester un seul fichier
    if len(sys.argv) > 1:
        pdf_name = sys.argv[1]
        pdf_path = test_dir / pdf_name

        if not pdf_path.exists():
            print(f"❌ PDF introuvable: {pdf_path}")
            return

        # Page optionnelle
        page_num = int(sys.argv[2]) - 1 if len(sys.argv) > 2 else 0

        test_ocr(str(pdf_path), page_num)
    else:
        # Tester le premier PDF de chaque type
        print("\n🧪 Test rapide avec texte.pdf (page 1)...")
        test_pdf = test_dir / "texte.pdf"
        if test_pdf.exists():
            test_ocr(str(test_pdf), page_num=0)


if __name__ == "__main__":
    main()
