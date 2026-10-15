package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"lexbot/internal/service"
)

type commandDefinition struct {
	names       []string
	description string
	examples    []string
	details     string
	handle      func(context.Context, string, *service.User, string)
}

// commands is the single catalog used to dispatch commands and render help.
func (h *Handler) commands() []commandDefinition {
	return []commandDefinition{
		{
			names:       []string{"ajuda"},
			description: "Mostrar os comandos disponíveis e exemplos de uso.",
			examples:    []string{"/ajuda", "/ajuda lista"},
			handle:      h.handleHelp,
		},
		{
			names:       []string{"lista"},
			description: "Listar suas palavras salvas, com filtros opcionais.",
			examples:    []string{"/lista", "/lista novas", "/lista dificeis"},
			handle: func(ctx context.Context, chat string, user *service.User, args string) {
				h.list.Handle(ctx, chat, user.ID, args)
			},
		},
		{
			names:       []string{"quiz"},
			description: "Iniciar ou retomar uma revisão das suas palavras.",
			examples:    []string{"/quiz"},
			handle: func(ctx context.Context, chat string, user *service.User, _ string) {
				h.quiz.Start(ctx, chat, user)
			},
		},
		{
			names:       []string{"status"},
			description: "Consultar seu progresso, níveis e taxa de acerto.",
			examples:    []string{"/status"},
			handle: func(ctx context.Context, chat string, user *service.User, _ string) {
				h.status.Handle(ctx, chat, user.ID)
			},
		},
		{
			names:       []string{"dashboard"},
			description: "Abrir o gerenciamento das suas palavras no navegador.",
			examples:    []string{"/dashboard"},
			handle: func(ctx context.Context, chat string, user *service.User, _ string) {
				h.dashboard.Handle(ctx, chat, user.ID)
			},
		},
	}
}

func (h *Handler) findCommand(name string) (commandDefinition, bool) {
	name = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(name), "/"))
	for _, command := range h.commands() {
		for _, alias := range command.names {
			if name == alias {
				return command, true
			}
		}
	}
	return commandDefinition{}, false
}

func commandNames(command commandDefinition) string {
	names := make([]string, len(command.names))
	for i, name := range command.names {
		names[i] = "/" + name
	}
	return strings.Join(names, " ou ")
}

func (h *Handler) handleHelp(_ context.Context, chat string, _ *service.User, topic string) {
	var message strings.Builder
	if topic != "" {
		command, ok := h.findCommand(topic)
		if !ok {
			h.messenger.Send(chat, "Não encontrei ajuda para esse comando. Digite /ajuda para ver as opções.")
			return
		}
		fmt.Fprintf(&message, "📖 %s\n\n%s\n", commandNames(command), command.description)
		if command.details != "" {
			fmt.Fprintf(&message, "\n%s\n", command.details)
		}
		fmt.Fprintf(&message, "\nExemplos:\n%s", strings.Join(command.examples, "\n"))
	} else {
		message.WriteString("📖 Comandos do LexBot\n\n")
		for _, command := range h.commands() {
			fmt.Fprintf(&message, "%s — %s\n", commandNames(command), command.description)
			fmt.Fprintf(&message, "Exemplos: %s\n\n", strings.Join(command.examples, "; "))
		}
		fmt.Fprintf(&message, "Para cadastrar: envie uma palavra ou expressão em inglês por texto (ex.: resilient ou look forward to). Para enviar várias, use uma por linha, até %d por mensagem.\n\n", maxWordsPerMessage)
		message.WriteString("Durante um quiz, suas mensagens de texto são respostas à pergunta atual. Os comandos continuam disponíveis.\n\nDigite /ajuda <comando> para saber mais.")
	}
	if err := h.messenger.Send(chat, message.String()); err != nil {
		slog.Error("failed to send help", "error", err)
	}
}
