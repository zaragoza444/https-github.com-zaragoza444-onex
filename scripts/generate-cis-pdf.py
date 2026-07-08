#!/usr/bin/env python3
"""Convert CIS markdown files to PDF via wkhtmltopdf."""

import html
import os
import subprocess
import sys

try:
    import markdown
except ImportError:
    subprocess.check_call([sys.executable, "-m", "pip", "install", "markdown", "-q"])
    import markdown

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
CIS_DIR = os.path.join(ROOT, "docs", "cis")
CSS_PATH = os.path.join(CIS_DIR, "pdf-style.css")

FILES = [
    "CIS-Nova-Bank-Online-v1.md",
    "CIS-Nova-1-Chain-22016-v1.md",
    "CIS-Nova-Integration-Matrix-v1.md",
    "README.md",
]


def load_css():
    if os.path.isfile(CSS_PATH):
        with open(CSS_PATH, encoding="utf-8") as f:
            return f.read()
    return ""


def md_to_html(md_path, title):
    with open(md_path, encoding="utf-8") as f:
        text = f.read()
    # Mermaid blocks are not renderable in static PDF; show as preformatted note.
    import re
    text = re.sub(
        r"```mermaid\n(.*?)```",
        r"*[Architecture diagram — see markdown source]*\n\n```\n\1```",
        text,
        flags=re.DOTALL,
    )
    body = markdown.markdown(
        text,
        extensions=["tables", "fenced_code", "toc"],
    )
    css = load_css()
    return f"""<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>{html.escape(title)}</title>
<style>{css}</style>
</head>
<body>{body}</body>
</html>"""


def html_to_pdf(html_path, pdf_path):
    cmd = [
        "wkhtmltopdf",
        "--quiet",
        "--enable-local-file-access",
        "--page-size", "A4",
        "-T", "20mm", "-B", "20mm", "-L", "18mm", "-R", "18mm",
        html_path,
        pdf_path,
    ]
    subprocess.run(cmd, check=True, capture_output=True, timeout=30)


def convert(md_name):
    md_path = os.path.join(CIS_DIR, md_name)
    base = os.path.splitext(md_name)[0]
    html_path = os.path.join(CIS_DIR, f"{base}.html")
    pdf_path = os.path.join(CIS_DIR, f"{base}.pdf")

    title = base.replace("-", " ")
    doc = md_to_html(md_path, title)
    with open(html_path, "w", encoding="utf-8") as f:
        f.write(doc)

    html_to_pdf(html_path, pdf_path)
    os.remove(html_path)
    size = os.path.getsize(pdf_path)
    print(f"  {pdf_path} ({size:,} bytes)")


def main():
    print("Generating CIS PDFs...")
    for name in FILES:
        print(f"Converting {name}...")
        convert(name)
    print("Done.")


if __name__ == "__main__":
    main()
