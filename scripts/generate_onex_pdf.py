"""Generate OneX Token Lab deployment guide PDF."""

from pathlib import Path

from fpdf import FPDF

OUT = Path(__file__).resolve().parents[1] / "OneX-Token-Lab-Guide.pdf"


class GuidePDF(FPDF):
    def header(self):
        if self.page_no() > 1:
            self.set_font("Helvetica", "I", 8)
            self.set_text_color(100, 100, 100)
            self.cell(0, 8, "OneX Token Lab - Production Guide", align="R", new_x="LMARGIN", new_y="NEXT")
            self.ln(2)

    def footer(self):
        self.set_y(-15)
        self.set_font("Helvetica", "I", 8)
        self.set_text_color(120, 120, 120)
        self.cell(0, 10, f"Page {self.page_no()}", align="C")

    def section(self, title: str):
        self.ln(4)
        self.set_font("Helvetica", "B", 14)
        self.set_text_color(30, 30, 30)
        self.cell(0, 10, title, new_x="LMARGIN", new_y="NEXT")
        self.set_draw_color(99, 102, 241)
        self.set_line_width(0.6)
        self.line(self.l_margin, self.get_y(), self.w - self.r_margin, self.get_y())
        self.ln(6)

    def body(self, text: str):
        self.set_font("Helvetica", "", 10)
        self.set_text_color(40, 40, 40)
        self.multi_cell(0, 5.5, text)
        self.ln(2)

    def bullet(self, text: str):
        self.set_font("Helvetica", "", 10)
        self.set_text_color(40, 40, 40)
        x = self.l_margin + 4
        self.set_x(x)
        self.multi_cell(self.w - self.r_margin - x, 5.5, f"- {text}")

    def code(self, text: str):
        self.set_font("Courier", "", 9)
        self.set_fill_color(245, 247, 250)
        self.set_text_color(20, 20, 20)
        self.multi_cell(0, 5, text, fill=True)
        self.ln(3)


def build() -> Path:
    pdf = GuidePDF()
    pdf.set_auto_page_break(auto=True, margin=18)
    pdf.add_page()

    # Cover
    pdf.ln(35)
    pdf.set_font("Helvetica", "B", 28)
    pdf.set_text_color(30, 30, 30)
    pdf.cell(0, 14, "OneX Token Lab", align="C", new_x="LMARGIN", new_y="NEXT")
    pdf.set_font("Helvetica", "", 14)
    pdf.set_text_color(80, 80, 80)
    pdf.cell(0, 10, "Production & Deployment Guide", align="C", new_x="LMARGIN", new_y="NEXT")
    pdf.ln(8)
    pdf.set_font("Helvetica", "", 11)
    pdf.cell(0, 8, "Multi-chain ERC-20 token generator", align="C", new_x="LMARGIN", new_y="NEXT")
    pdf.cell(0, 8, "June 2026", align="C", new_x="LMARGIN", new_y="NEXT")

    pdf.add_page()

    pdf.section("Live Production")
    pdf.body(
        "OneX Token Lab is deployed and running in production mode on your VPS."
    )
    pdf.bullet("App URL: http://zblockchainsystem.com:9340")
    pdf.bullet("Health: GET /health  |  Ready: GET /ready")
    pdf.bullet("Mode: production (API key required for deploy/register)")
    pdf.bullet("Chains: 12 supported (BSC, Ethereum, Base, Polygon, Arbitrum, Optimism, Avalanche, Linea, Blast, Scroll, Solana track, Sui track)")
    pdf.ln(2)

    pdf.section("GitHub Repository")
    pdf.body("Source code is published on GitHub:")
    pdf.code("https://github.com/zaragoza444/onex")
    pdf.body("Main branch contains OneX Token Lab, OneXToken contract, multi-chain server, and production UI.")

    pdf.section("First-Time Setup (Browser)")
    pdf.bullet("Open http://zblockchainsystem.com:9340")
    pdf.bullet("Click Settings in the top navigation bar")
    pdf.bullet("Paste your BSC_LAUNCHER_API_KEY from the server .env file")
    pdf.bullet("Click Save API key")
    pdf.bullet("Connect MetaMask and select your target chain in the wizard")
    pdf.bullet("Complete the 4-step wizard and Deploy")
    pdf.ln(2)
    pdf.body("Liquidity tools (PancakeSwap V2, BSCScan $1 listing) are available on BSC only.")

    pdf.section("Server Environment (.env)")
    pdf.body("Production variables on the VPS at /home/ubuntu/onex-token-lab/bsc-launcher/.env:")
    pdf.code(
        "BSC_LAUNCHER_ENV=production\n"
        "BSC_LAUNCHER_LISTEN=:9340\n"
        "BSC_LAUNCHER_API_KEY=<your-secret-key>\n"
        "BSC_LAUNCHER_CORS_ORIGINS=http://zblockchainsystem.com:9340\n"
        "BSC_RPC_URL=https://bsc-dataseed.binance.org\n"
        "BSCSCAN_API_KEY=<etherscan-v2-key>"
    )
    pdf.body("Never commit .env or private keys to Git. The API key is stored in the browser via Settings (localStorage).")

    pdf.section("VPS Details")
    pdf.bullet("Host: ubuntu@zblockchainsystem.com")
    pdf.bullet("Install path: /home/ubuntu/onex-token-lab")
    pdf.bullet("Binary: /home/ubuntu/onex-token-lab/bin/bsc-launcher")
    pdf.bullet("Systemd service: onex-token-lab")
    pdf.bullet("Go version on server: 1.22+")
    pdf.ln(2)
    pdf.body("Useful commands on the VPS:")
    pdf.code(
        "sudo systemctl status onex-token-lab\n"
        "sudo systemctl restart onex-token-lab\n"
        "curl -s http://127.0.0.1:9340/health"
    )

    pdf.section("Update VPS from GitHub")
    pdf.body("After pushing changes to GitHub main, redeploy on the VPS from your Windows machine:")
    pdf.code(
        "cd /home/ubuntu/onex\n"
        "$env:SSH_PASS='your-vps-password'\n"
        "python scripts/vps_pull_github.py"
    )
    pdf.body("This script pulls main, rebuilds the Go binary, and restarts the systemd service.")

    pdf.section("Local Development")
    pdf.code("bsc-launcher\\run-onex-token-lab.bat")
    pdf.body("Opens http://127.0.0.1:9340 in development mode (API key optional).")

    pdf.section("Production Features")
    pdf.bullet("OneXToken.sol - 17 on-chain wizard features (taxes, limits, mintable, pausable, blacklist, permit, enableTrading, etc.)")
    pdf.bullet("MetaMask deploy (user pays gas) or backend deploy (platform wallet)")
    pdf.bullet("Multi-chain MetaMask network switching")
    pdf.bullet("Dashboard with DexScreener prices and explorer links")
    pdf.bullet("PancakeSwap V2 liquidity wizard for BSCScan USD price")
    pdf.bullet("API key auth, CORS, rate limits, health/ready endpoints")

    pdf.section("API Endpoints")
    pdf.code(
        "GET  /health, /ready\n"
        "GET  /api/config, /api/tokens\n"
        "GET  /api/bscscan/:addr?chain=bsc\n"
        "GET  /api/price/:addr?chain=bsc\n"
        "POST /api/tokens/register   (X-API-Key)\n"
        "POST /api/deploy            (X-API-Key)\n"
        "POST /api/liquidity/register (X-API-Key)"
    )

    pdf.section("Security Checklist")
    pdf.bullet("Change VPS password and use SSH keys")
    pdf.bullet("Rotate BSC_LAUNCHER_API_KEY periodically")
    pdf.bullet("Set BSC_LAUNCHER_CORS_ORIGINS to your real domain when using HTTPS")
    pdf.bullet("Add BSCSCAN_API_KEY for holder stats on dashboard")
    pdf.bullet("Fund platform deployer wallet with minimal BNB only if using backend deploy")

    pdf.section("Support Links")
    pdf.bullet("BSCScan: https://bscscan.com")
    pdf.bullet("DexScreener: https://dexscreener.com")
    pdf.bullet("PancakeSwap: https://pancakeswap.finance")
    pdf.bullet("Repo: https://github.com/zaragoza444/onex")

    pdf.output(str(OUT))
    return OUT


if __name__ == "__main__":
    path = build()
    print(path)
