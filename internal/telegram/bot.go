package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/Zoxine/v2/internal/config"
	"github.com/Zoxine/v2/internal/pipeline"
)

type Bot struct {
	api      *tgbotapi.BotAPI
	cfg      config.Config
	allowed  map[int64]struct{}
	sessions *SessionManager
	pipeline *pipeline.Runner
}

func NewBot(cfg config.Config) (*Bot, error) {
	if cfg.Telegram.Token == "" {
		return nil, fmt.Errorf("telegram token is required")
	}
	if len(cfg.Telegram.AllowedUserIDs) == 0 {
		return nil, fmt.Errorf("at least one allowed telegram user id is required")
	}

	api, err := tgbotapi.NewBotAPI(cfg.Telegram.Token)
	if err != nil {
		return nil, fmt.Errorf("create telegram bot: %w", err)
	}

	allowed := make(map[int64]struct{}, len(cfg.Telegram.AllowedUserIDs))
	for _, id := range cfg.Telegram.AllowedUserIDs {
		allowed[id] = struct{}{}
	}

	p := pipeline.New(cfg)
	b := &Bot{
		api:     api,
		cfg:     cfg,
		allowed: allowed,
		pipeline: p,
	}
	b.sessions = NewSessionManager(cfg.Telegram.FlushDebounceSeconds, b.handleFlush)
	return b, nil
}

func (b *Bot) Run(ctx context.Context) error {
	slog.Info("telegram bot started", "username", b.api.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.api.GetUpdatesChan(u)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-sigCh:
			slog.Info("shutdown signal received")
			return nil
		case update := <-updates:
			if update.Message == nil {
				continue
			}
			b.handleMessage(update.Message)
		}
	}
}

func (b *Bot) handleMessage(msg *tgbotapi.Message) {
	userID := msg.From.ID
	if !b.isAllowed(userID) {
		b.reply(msg.Chat.ID, "You are not authorized to use this bot.")
		return
	}

	text := strings.TrimSpace(msg.Text)
	if text == "/cancel" {
		cleared := b.sessions.Cancel(userID)
		b.reply(msg.Chat.ID, fmt.Sprintf("Cleared %d buffered config(s).", cleared))
		return
	}
	if text == "/submit" || text == "/check" {
		lines := b.sessions.Flush(userID)
		if len(lines) == 0 {
			b.reply(msg.Chat.ID, "Nothing to submit. Forward or send vless:// / vmess:// URIs first.")
			return
		}
		go b.runPipeline(msg.Chat.ID, lines)
		return
	}
	if text == "/status" {
		count := b.sessions.Count(userID)
		b.reply(msg.Chat.ID, fmt.Sprintf("Buffered configs: %d", count))
		return
	}

	body := text
	if msg.Caption != "" {
		body = msg.Caption
	}
	uris := ExtractURIs(body)
	if len(uris) == 0 {
		b.reply(msg.Chat.ID, "No vless:// or vmess:// URIs found in this message.")
		return
	}

	added := b.sessions.Add(userID, uris)
	total := b.sessions.Count(userID)
	b.reply(msg.Chat.ID, fmt.Sprintf("Added %d URI(s). Buffer now has %d config(s). Auto-submit in %ds, or send /submit.",
		added, total, b.cfg.Telegram.FlushDebounceSeconds))
}

func (b *Bot) handleFlush(userID int64, lines []string) {
	chatID := userID
	go b.runPipeline(chatID, lines)
}

func (b *Bot) runPipeline(chatID int64, lines []string) {
	b.reply(chatID, fmt.Sprintf("Checking %d config(s)...", len(lines)))

	result, err := b.pipeline.Run(context.Background(), lines)
	if err != nil {
		slog.Error("pipeline failed", "error", err)
		b.reply(chatID, fmt.Sprintf("Failed: %v", err))
		return
	}
	b.reply(chatID, result.Message)
}

func (b *Bot) isAllowed(userID int64) bool {
	_, ok := b.allowed[userID]
	return ok
}

func (b *Bot) reply(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.DisableWebPagePreview = true
	if _, err := b.api.Send(msg); err != nil {
		slog.Error("telegram reply failed", "error", err)
	}
}
