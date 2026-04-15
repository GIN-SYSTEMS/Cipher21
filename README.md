# Cipher21: Sovereign Elite

A high-stakes, terminal-native Blackjack simulation written in **Go**. Built for users who value cryptographic transparency, efficient TUI (Terminal User Interface) design, and progressive gameplay mechanics.

---

## ⚡ Core Engine

### Cryptographic Transparency (Provably Fair)
Cipher21 utilizes a **SHA-256 hashing protocol** to ensure total game integrity.
* **The Seal:** Before the first card is dealt, a unique hash of the shuffled deck is generated and displayed.
* **The Verification:** This hash acts as a digital fingerprint. At the end of the round, you can verify that the deck was not manipulated mid-game. 

### Live Financial Monitoring
The interface features a **Real-Time Financial Feed** anchored to the right panel. It tracks every transaction, doubling down, and payout with sub-second latency.

### Progression & Tiers
Access is restricted by XP. Move through 6 distinct tiers to reach the end-game:
* **The Alley:** Entry-level stakes.
* **The Vault:** High-roller environment.
* **The Singularity:** Max-multiplier protocol (250,000+ XP required).

---

## 🎮 Navigation

| Key | Action |
| :--- | :--- |
| **Up / Down** | Adjust Stakes / Menu Selection |
| **Enter** | Confirm / Hit / Start |
| **[M]** | **Max Bet:** Full-balance commitment |
| **[H]** | **History:** Access session logs |
| **[D]** | **Debt:** Check loan status & repayment |
| **[S]** | **Shop:** Purchase Lucky Coins |
| **[Q]** | **Quit:** Secure database sync and exit |

---

## 🚀 Getting Started

### Windows (Compiled Binary)
1. Download the latest `Cipher21.exe` from the [Releases](https://github.com/GIN-SYSTEMS/Cipher21/releases) page.
2. Run the executable directly via terminal.
3. Your progress is saved automatically to a local `.db` file.

### Build from Source
Requires **Go 1.20+**.
```bash
git clone [https://github.com/GIN-SYSTEMS/Cipher21.git](https://github.com/GIN-SYSTEMS/Cipher21.git)
go mod tidy
go run main.go