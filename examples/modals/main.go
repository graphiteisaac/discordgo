package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	"forge.isaac.zone/isaac/nucord"
)

// Bot parameters
var (
	GuildID        = flag.String("guild", "", "Test guild ID")
	BotToken       = flag.String("token", "", "Bot access token")
	AppID          = flag.String("app", "", "Application ID")
	Cleanup        = flag.Bool("cleanup", true, "Cleanup of commands")
	ResultsChannel = flag.String("results", "", "Channel where send survey results to")
)

var s *nucord.Session

func init() {
	flag.Parse()
}

func init() {
	var err error
	s, err = nucord.New("Bot " + *BotToken)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}
}

var required = true

var (
	commands = []nucord.ApplicationCommand{
		{
			Name:        "modals-survey",
			Description: "Take a survey about modals",
		},
	}
	commandsHandlers = map[string]func(s *nucord.Session, i *nucord.InteractionCreate){
		"modals-survey": func(s *nucord.Session, i *nucord.InteractionCreate) {
			err := s.InteractionRespond(i.Interaction, &nucord.InteractionResponse{
				Type: nucord.InteractionResponseModal,
				Data: &nucord.InteractionResponseData{
					CustomID: "modals_survey_" + i.Interaction.Member.User.ID,
					Title:    "Modals survey",
					Flags:    nucord.MessageFlagsIsComponentsV2,
					Components: []nucord.MessageComponent{
						nucord.Label{
							Label:       "How would you rate them?",
							Description: "On a scale from terrible to awesome",
							Component: nucord.SelectMenu{
								MenuType:    nucord.StringSelectMenu,
								CustomID:    "rating",
								Placeholder: "Your rating...",
								Options: []nucord.SelectMenuOption{
									{Label: "Terrible", Value: "terrible"},
									{Label: "Bad", Value: "bad"},
									{Label: "Neutral", Value: "neutral", Default: true},
									{Label: "Good", Value: "good"},
									{Label: "Awesome", Value: "awesome"},
								},
							},
						},
						nucord.Label{
							Label:       "What would you suggest to improve them?",
							Description: "Please provide as much info as possible!",
							Component: nucord.TextInput{
								CustomID:  "suggestions",
								Style:     nucord.TextInputParagraph,
								Required:  &required,
								MaxLength: 2000,
							},
						},
					},
				},
			})
			if err != nil {
				panic(err)
			}
		},
	}
)

func main() {
	s.AddHandler(func(s *nucord.Session, r *nucord.Ready) {
		log.Println("Bot is up!")
	})

	s.AddHandler(func(s *nucord.Session, i *nucord.InteractionCreate) {
		switch i.Type {
		case nucord.InteractionApplicationCommand:
			if h, ok := commandsHandlers[i.ApplicationCommandData().Name]; ok {
				h(s, i)
			}
		case nucord.InteractionModalSubmit:
			err := s.InteractionRespond(i.Interaction, &nucord.InteractionResponse{
				Type: nucord.InteractionResponseChannelMessageWithSource,
				Data: &nucord.InteractionResponseData{
					Content: "Thank you for taking your time to fill this survey",
					Flags:   nucord.MessageFlagsEphemeral,
				},
			})
			if err != nil {
				panic(err)
			}
			data := i.ModalSubmitData()

			if !strings.HasPrefix(data.CustomID, "modals_survey") {
				return
			}

			userid := strings.Split(data.CustomID, "_")[2]
			_, err = s.ChannelMessageSend(*ResultsChannel, fmt.Sprintf(
				"Feedback received. From <@%s>\n\n**Rating**:\n%s\n\n**Suggestions**:\n%s",
				userid,
				data.Components[0].(*nucord.Label).Component.(*nucord.SelectMenu).Values[0],
				data.Components[1].(*nucord.Label).Component.(*nucord.TextInput).Value,
			))
			if err != nil {
				panic(err)
			}
		}
	})

	cmdIDs := make(map[string]string, len(commands))

	for _, cmd := range commands {
		rcmd, err := s.ApplicationCommandCreate(*AppID, *GuildID, &cmd)
		if err != nil {
			log.Fatalf("Cannot create slash command %q: %v", cmd.Name, err)
		}

		cmdIDs[rcmd.ID] = rcmd.Name
	}

	err := s.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}
	defer s.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	log.Println("Graceful shutdown")

	if !*Cleanup {
		return
	}

	for id, name := range cmdIDs {
		err := s.ApplicationCommandDelete(*AppID, *GuildID, id)
		if err != nil {
			log.Fatalf("Cannot delete slash command %q: %v", name, err)
		}
	}

}
