package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"

	"forge.isaac.zone/isaac/nucord"
)

type optionMap = map[string]*nucord.ApplicationCommandInteractionDataOption

func parseOptions(options []*nucord.ApplicationCommandInteractionDataOption) (om optionMap) {
	om = make(optionMap)
	for _, opt := range options {
		om[opt.Name] = opt
	}
	return
}

func interactionAuthor(i *nucord.Interaction) *nucord.User {
	if i.Member != nil {
		return i.Member.User
	}
	return i.User
}

func handleEcho(s *nucord.Session, i *nucord.InteractionCreate, opts optionMap) {
	builder := new(strings.Builder)
	if v, ok := opts["author"]; ok && v.BoolValue() {
		author := interactionAuthor(i.Interaction)
		builder.WriteString("**" + author.String() + "** says: ")
	}
	builder.WriteString(opts["message"].StringValue())

	err := s.InteractionRespond(i.Interaction, &nucord.InteractionResponse{
		Type: nucord.InteractionResponseChannelMessageWithSource,
		Data: &nucord.InteractionResponseData{
			Content: builder.String(),
		},
	})

	if err != nil {
		log.Panicf("could not respond to interaction: %s", err)
	}
}

var commands = []*nucord.ApplicationCommand{
	{
		Name:        "echo",
		Description: "Say something through a bot",
		Options: []*nucord.ApplicationCommandOption{
			{
				Name:        "message",
				Description: "Contents of the message",
				Type:        nucord.ApplicationCommandOptionString,
				Required:    true,
			},
			{
				Name:        "author",
				Description: "Whether to prepend message's author",
				Type:        nucord.ApplicationCommandOptionBoolean,
			},
		},
	},
}

var (
	Token = flag.String("token", "", "Bot authentication token")
	App   = flag.String("app", "", "Application ID")
	Guild = flag.String("guild", "", "Guild ID")
)

func main() {
	flag.Parse()
	if *App == "" {
		log.Fatal("application id is not set")
	}

	session, _ := nucord.New("Bot " + *Token)

	session.HandleInteractionCreate(func(s *nucord.Session, i *nucord.InteractionCreate) {
		if i.Type != nucord.InteractionApplicationCommand {
			return
		}

		data := i.ApplicationCommandData()
		if data.Name != "echo" {
			return
		}

		handleEcho(s, i, parseOptions(data.Options))
	})

	session.HandleReady(func(s *nucord.Session, r *nucord.Ready) {
		log.Printf("Logged in as %s", r.User.String())
	})

	_, err := session.ApplicationCommandBulkOverwrite(*App, *Guild, commands)
	if err != nil {
		log.Fatalf("could not register commands: %s", err)
	}

	err = session.Open()
	if err != nil {
		log.Fatalf("could not open session: %s", err)
	}

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	<-sigch

	err = session.Close()
	if err != nil {
		log.Printf("could not close session gracefully: %s", err)
	}
}
