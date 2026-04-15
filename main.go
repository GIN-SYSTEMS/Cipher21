package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	_ "modernc.org/sqlite"
)

// --- DESIGN SYSTEM ---
var (
	c_neon  = lipgloss.Color("#00FFD1")
	c_gold  = lipgloss.Color("#FFCC00")
	c_red   = lipgloss.Color("#FF3333")
	c_green = lipgloss.Color("#00FF41")
	c_gray  = lipgloss.Color("#555555")
	c_white = lipgloss.Color("#FFFFFF")

	cardStyle      = lipgloss.NewStyle().Width(10).Height(5).Border(lipgloss.RoundedBorder()).Align(lipgloss.Center, lipgloss.Center).Background(c_white)
	containerStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
	hudStyle       = lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).Padding(1, 2).Width(95).MarginBottom(1)

	// FIXED WIDTHS: 60 for game, 30 for feed = 90 total.
	gameAreaStyle  = lipgloss.NewStyle().Width(62)
	sidePanelStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(c_gray).PaddingLeft(2).MarginLeft(2).Width(28).Height(15)
)

type Room struct {
	Name, Emoji             string
	MinBet, XPMult, MinRank int
	Desc                    string
}

var Rooms = []Room{
	{"The Alley", "🪵 ", 10, 1, 0, "Low Stakes - Entry Level"},
	{"The Lounge", "💎 ", 500, 3, 500, "High Risk - Elite Members"},
	{"The Node", "🖥️ ", 2500, 8, 2000, "Cyber Node - Pro Hackers"},
	{"The Vault", "🏦 ", 10000, 20, 10000, "Deep Storage - High Rollers"},
	{"The Sanctum", "🕍 ", 50000, 50, 50000, "Inner Circle - Secret Stakes"},
	{"The Singularity", "🌀 ", 250000, 150, 250000, "The End - For the Legend"},
}

type Card struct{ Value, Suit string }

type model struct {
	db                                  *sql.DB
	balance, bet, wins, loses, xp, debt int
	playerCards, dealerCards            []Card
	deck                                []Card
	state, msg                          string
	cursor, roomIdx, shuffleFrame       int
	currentHash                         string
	width, height                       int
	lastBet, luckyCoin                  int
	history                             []string
}

// --- DB CORE ---
func initDB() *sql.DB {
	db, _ := sql.Open("sqlite", "./cipher_sovereign_final.db")
	db.Exec("CREATE TABLE IF NOT EXISTS stats (id INTEGER PRIMARY KEY, balance INTEGER, wins INTEGER, loses INTEGER, xp INTEGER, debt INTEGER, coins INTEGER)")
	var count int
	db.QueryRow("SELECT COUNT(*) FROM stats").Scan(&count)
	if count == 0 {
		db.Exec("INSERT INTO stats (balance, wins, loses, xp, debt, coins) VALUES (1600, 0, 0, 0, 0, 0)")
	}
	return db
}

func (m *model) sync(save bool) {
	if save {
		m.db.Exec("UPDATE stats SET balance = ?, wins = ?, loses = ?, xp = ?, debt = ?, coins = ?", m.balance, m.wins, m.loses, m.xp, m.debt, m.luckyCoin)
	} else {
		m.db.QueryRow("SELECT balance, wins, loses, xp, debt, coins FROM stats").Scan(&m.balance, &m.wins, &m.loses, &m.xp, &m.debt, &m.luckyCoin)
	}
}

func (m *model) addLog(text string, color lipgloss.Color) {
	styled := lipgloss.NewStyle().Foreground(color).Render(text)
	m.history = append(m.history, styled)
	if len(m.history) > 12 {
		m.history = m.history[1:]
	}
}

func score(cards []Card) int {
	t, a := 0, 0
	for _, c := range cards {
		if c.Value == "A" {
			a++
			t += 11
		} else if c.Value == "J" || c.Value == "Q" || c.Value == "K" {
			t += 10
		} else {
			var v int
			fmt.Sscanf(c.Value, "%d", &v)
			t += v
		}
	}
	for t > 21 && a > 0 {
		t -= 10
		a--
	}
	return t
}

func createShoe() []Card {
	v := []string{"2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K", "A"}
	s := []string{"♠", "♣", "♥", "♦"}
	var d []Card
	for i := 0; i < 6; i++ {
		for _, suit := range s {
			for _, val := range v {
				d = append(d, Card{val, suit})
			}
		}
	}
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(d), func(i, j int) { d[j], d[i] = d[i], d[j] })
	return d
}

// --- APP ---
func (m model) Init() tea.Cmd { return nil }

func initialModel() model {
	m := model{db: initDB(), state: "TIERS"}
	m.sync(false)
	return m
}

func tick(msg string, d int) tea.Cmd {
	return tea.Tick(time.Duration(d)*time.Millisecond, func(t time.Time) tea.Msg { return msg })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case string:
		if msg == "SHUFFLE_ANIM" {
			m.shuffleFrame++
			if m.shuffleFrame < 15 {
				return m, tick("SHUFFLE_ANIM", 80)
			}
			m.state = "DEAL"
			m.deck = createShoe()
			return m, tick("DEAL_T", 200)
		}
		if msg == "DEAL_T" {
			if len(m.playerCards) < 2 || len(m.dealerCards) < 2 {
				c := m.deck[0]
				m.deck = m.deck[1:]
				if len(m.playerCards) <= len(m.dealerCards) {
					m.playerCards = append(m.playerCards, c)
				} else {
					m.dealerCards = append(m.dealerCards, c)
				}
				return m, tick("DEAL_T", 250)
			}
			m.state = "PLAY"
			if score(m.playerCards) >= 20 {
				return m, m.toD()
			}
		}
		if msg == "DEALER_T" {
			if score(m.dealerCards) < 17 {
				m.dealerCards = append(m.dealerCards, m.deck[0])
				m.deck = m.deck[1:]
				return m, tick("DEALER_T", 600)
			}
			m.resolve()
		}
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			m.sync(true)
			return m, tea.Quit
		}
		switch m.state {
		case "TIERS":
			switch msg.String() {
			case "up":
				if m.roomIdx > 0 {
					m.roomIdx--
				}
			case "down":
				if m.roomIdx < len(Rooms)-1 {
					m.roomIdx++
				}
			case "enter":
				r := Rooms[m.roomIdx]
				if m.xp < r.MinRank {
					m.msg = "ERROR: INSUFFICIENT XP"
					return m, nil
				}
				m.state = "BET"
				m.msg = ""
			case "s":
				m.state = "SHOP"
			case "d":
				m.state = "DEBTS"
			case "h":
				m.state = "HISTORY"
			}
		case "SHOP", "DEBTS", "HISTORY":
			if msg.String() == "b" {
				m.state = "TIERS"
			}
			if m.state == "SHOP" && msg.String() == "1" && m.xp >= 1000 {
				m.xp -= 1000
				m.luckyCoin++
				m.sync(true)
			}
		case "BET":
			switch msg.String() {
			case "up":
				if m.bet+100 <= m.balance {
					m.bet += 100
				}
			case "down":
				if m.bet > Rooms[m.roomIdx].MinBet {
					m.bet -= 100
				}
			case "m":
				m.bet = m.balance
			case "enter":
				if m.balance < m.bet {
					m.state = "LOAN"
					return m, nil
				}
				m.lastBet = m.bet
				m.playerCards, m.dealerCards = nil, nil
				m.state = "SHUFFLE"
				m.shuffleFrame = 0
				m.generateHash()
				m.addLog(fmt.Sprintf("STAKE: %d$", m.bet), c_gold)
				return m, tick("SHUFFLE_ANIM", 80)
			case "b":
				m.state = "TIERS"
			}
		case "PLAY":
			switch msg.String() {
			case "up":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down":
				if m.cursor < 2 {
					m.cursor++
				}
			case "enter":
				if m.cursor == 0 {
					m.playerCards = append(m.playerCards, m.deck[0])
					m.deck = m.deck[1:]
					if score(m.playerCards) >= 20 {
						return m, m.toD()
					}
					if score(m.playerCards) > 21 {
						m.resolve()
					}
				} else if m.cursor == 1 {
					return m, m.toD()
				} else if m.cursor == 2 {
					if m.balance >= m.bet*2 {
						m.bet *= 2
						m.addLog(fmt.Sprintf("DOUBLE: %d$", m.bet), c_gold)
						m.playerCards = append(m.playerCards, m.deck[0])
						m.deck = m.deck[1:]
						return m, m.toD()
					}
				}
			}
		case "RESULT":
			switch msg.String() {
			case "enter":
				m.bet = m.lastBet
				if m.balance < m.bet {
					m.state = "BET"
				} else {
					m.playerCards, m.dealerCards, m.state, m.msg = nil, nil, "DEAL", ""
					m.addLog(fmt.Sprintf("STAKE: %d$", m.bet), c_gold)
					return m, tick("DEAL_T", 100)
				}
			case "t":
				m.state = "TIERS"
				m.msg = ""
			case "b":
				m.state = "BET"
				m.msg = ""
			}
		case "LOAN":
			if msg.String() == "enter" {
				m.balance += 1600
				m.debt += 1900
				m.state = "TIERS"
			}
		}
	}
	return m, nil
}

func (m *model) generateHash() {
	seed := fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Intn(100000))
	h := sha256.New()
	h.Write([]byte(seed))
	m.currentHash = hex.EncodeToString(h.Sum(nil))[:16]
}

func (m *model) toD() tea.Cmd { m.state = "DEALER"; return tick("DEALER_T", 400) }

func (m *model) resolve() {
	m.state = "RESULT"
	m.xp += 10 * Rooms[m.roomIdx].XPMult
	p, d := score(m.playerCards), score(m.dealerCards)
	if p > 21 {
		if m.luckyCoin > 0 {
			m.luckyCoin--
			m.msg = "🛡️ FAIL-SAFE: LUCKY COIN SAVED YOU"
			m.addLog("SAFE: 0$", c_neon)
		} else {
			m.msg = "BUSTED!"
			m.balance -= m.bet
			m.addLog(fmt.Sprintf("LOSS: -%d$", m.bet), c_red)
		}
	} else if d > 21 || p > d {
		win := m.bet
		if m.debt > 0 {
			repay := int(float64(win) * 0.2)
			if repay > m.debt {
				repay = m.debt
			}
			m.debt -= repay
			win -= repay
		}
		m.balance += win
		m.xp += 40 * Rooms[m.roomIdx].XPMult
		m.msg = "WINNER!"
		m.addLog(fmt.Sprintf("WIN: +%d$", win), c_green)
	} else if p == d {
		m.msg = "PUSH (DRAW)"
		m.addLog("PUSH: 0$", c_white)
	} else {
		m.msg = "DEALER WINS"
		m.balance -= m.bet
		m.addLog(fmt.Sprintf("LOSS: -%d$", m.bet), c_red)
	}
	m.bet = m.lastBet
	m.sync(true)
}

func (m model) View() string {
	var sb strings.Builder
	rank := "ROOKIE"
	if m.xp >= 500 {
		rank = "PRO"
	}
	if m.xp >= 5000 {
		rank = "EXECUTIVE"
	}
	if m.xp >= 50000 {
		rank = "SHADOW"
	}

	// --- HUD (ALWAYS AT TOP) ---
	hudContent := fmt.Sprintf("💰 %-7d$ | 🎲 %-7d$ | 🏆 %-10s (XP: %d) | 🪙 %d", m.balance, m.bet, rank, m.xp, m.luckyCoin)
	sb.WriteString(hudStyle.Copy().BorderForeground(c_neon).Render(
		lipgloss.NewStyle().Foreground(c_gold).Bold(true).SetString("🎰 CIPHER STAKES: SOVEREIGN ELITE 🎰").String()+"\n"+hudContent) + "\n\n")

	var body strings.Builder
	isGameplay := m.state == "DEAL" || m.state == "PLAY" || m.state == "DEALER" || m.state == "RESULT"

	switch m.state {
	case "TIERS":
		body.WriteString("🛋️  SELECT TABLE TIER\n\n")
		for i, r := range Rooms {
			cur := "   "
			if m.roomIdx == i {
				cur = " ▶ "
			}
			lock := ""
			if m.xp < r.MinRank {
				lock = fmt.Sprintf(" 🔒 (REQ: %d XP)", r.MinRank)
			}
			line := fmt.Sprintf("%s%-18s (%-7d$) - %-18s %s", cur+r.Emoji, r.Name, r.MinBet, r.Desc, lock)
			if m.roomIdx == i {
				body.WriteString(lipgloss.NewStyle().Foreground(c_neon).Render(line) + "\n")
			} else {
				body.WriteString(lipgloss.NewStyle().Foreground(c_gray).Render(line) + "\n")
			}
		}
		if m.msg != "" {
			body.WriteString("\n " + lipgloss.NewStyle().Foreground(c_red).Render(m.msg))
		}
		body.WriteString("\n\n [S] Shop  |  [D] Debts  |  [H] History  |  [Enter] Join")
	case "SHOP":
		body.WriteString("🛒 BLACK MARKET\n\n1. Lucky Coin (🪙) - 1000 XP\n   - Consumed to protect balance on bust.\n\n [B] Back")
	case "DEBTS":
		body.WriteString("🏦 FINANCIAL OVERVIEW\n\n")
		if m.debt > 0 {
			body.WriteString(fmt.Sprintf(" > OUTSTANDING DEBT: %d$\n > Tax: 20%% from winnings.", m.debt))
		} else {
			body.WriteString(" > DEBT STATUS: CLEAN.")
		}
		body.WriteString("\n\n [B] Back")
	case "HISTORY":
		body.WriteString("📜 SESSION LOGS\n\n")
		for _, h := range m.history {
			body.WriteString(" " + h + "\n")
		}
		body.WriteString("\n [B] Back")
	case "SHUFFLE":
		f := m.shuffleFrame
		for i := 0; i < 6; i++ {
			offset := (f - i) * 5
			if offset < 0 {
				offset = 0
			}
			if offset > 45 {
				offset = 45
			}
			body.WriteString(strings.Repeat(" ", offset) + " .-------. \n")
		}
		body.WriteString("\n" + lipgloss.NewStyle().Foreground(c_neon).Italic(true).Render("DECRYPTING DECK..."))
	case "BET":
		body.WriteString("\n LOCATION: " + Rooms[m.roomIdx].Name + "\n " + "STAKE: " + fmt.Sprintf("%d$\n\n", m.bet))
		body.WriteString(" [M] All-in  |  [B] Back  |  [Enter] Start")
	case "LOAN":
		body.WriteString("\n\n DEBT PROTOCOL ACTIVATED.\n Take 1600$ (Repay 1900$)\n\n [Enter] SIGN CONTRACT")
	default:
		dScore := "?"
		if m.state == "DEALER" || m.state == "RESULT" {
			dScore = fmt.Sprintf("%d", score(m.dealerCards))
		}
		body.WriteString(fmt.Sprintf("🤵  DEALER [%s]\n", dScore) + m.renderHand(m.dealerCards, m.state == "PLAY" || m.state == "DEAL") + "\n\n")
		body.WriteString(fmt.Sprintf("👤  PLAYER [%d]\n", score(m.playerCards)) + m.renderHand(m.playerCards, false) + "\n\n")
		if m.state == "PLAY" {
			opts := []string{"HIT", "STAND", "DOUBLE"}
			for i, o := range opts {
				cur := "   "
				if m.cursor == i {
					cur = " ▶ "
				}
				body.WriteString(cur + o + "\n")
			}
		} else if m.state == "RESULT" {
			body.WriteString(lipgloss.NewStyle().Bold(true).Foreground(c_neon).Render(" "+m.msg) + "\n\n")
			body.WriteString(" [ENTER] Re-bet  |  [B] Change Bet  |  [T] Switch Table")
		}
	}

	var finalView string
	if isGameplay {
		var side strings.Builder
		side.WriteString(lipgloss.NewStyle().Bold(true).Foreground(c_gold).Render(" FINANCIAL FEED\n") + strings.Repeat("─", 20) + "\n")
		start := 0
		if len(m.history) > 8 {
			start = len(m.history) - 8
		}
		for i := start; i < len(m.history); i++ {
			side.WriteString(" " + m.history[i] + "\n")
		}
		finalView = lipgloss.JoinHorizontal(lipgloss.Top, gameAreaStyle.Render(body.String()), sidePanelStyle.Render(side.String()))
	} else {
		finalView = body.String()
	}

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, containerStyle.Copy().BorderForeground(c_neon).Render(sb.String()+finalView))
}

func (m model) renderHand(cards []Card, hide bool) string {
	var rendered []string
	for i, c := range cards {
		val, suit := c.Value, c.Suit
		if hide && i == 1 {
			val, suit = "?", "?"
		}
		col := lipgloss.Color("#000000")
		if (suit == "♥" || suit == "♦") && val != "?" {
			col = c_red
		}
		row1 := fmt.Sprintf("%-2s     ", val)
		row2 := fmt.Sprintf("   %s   ", suit)
		row3 := fmt.Sprintf("     %2s", val)
		cardTxt := fmt.Sprintf("\n%s\n\n%s\n\n%s", row1, row2, row3)
		rendered = append(rendered, cardStyle.Copy().Foreground(col).Render(cardTxt))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	p.Run()
}
