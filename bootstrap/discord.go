package bootstrap

import (
	"context"
	"log"

	discordmod "github.com/airdb/scout/modules/discord"
	openaimod "github.com/airdb/scout/modules/openai"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/fx"
)

type discordDeps struct {
	fx.In

	Cfg             *discordmod.Config
	Session         *discordgo.Session
	Commands        []*discordgo.ApplicationCommand
	CommandHandlers map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate)

	ChatGpt *openaimod.ChatGpt
}

type Discord struct {
	deps               *discordDeps
	registeredCommands []*discordgo.ApplicationCommand
}

func NewDiscord(deps discordDeps) *Discord {

	return &Discord{
		deps:               &deps,
		registeredCommands: make([]*discordgo.ApplicationCommand, 0),
	}
}

func (d *Discord) init() error {
	// Register the messageCreate func as a callback for MessageCreate events.
	d.deps.Session.AddHandler(d.messageCreate)

	d.deps.Session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := d.deps.CommandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	// In this example, we only care about receiving message events.
	d.deps.Session.Identify.Intents = discordgo.IntentsGuildMessages
	d.deps.Session.Identify.Intents |= discordgo.IntentMessageContent

	d.deps.Session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})

	return nil
}

func (d *Discord) Start() error {
	var err error
	err = d.init()
	if err != nil {
		log.Fatalf("Cannot init the session: %v", err)
		return err
	}

	err = d.deps.Session.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
		return err
	}

	log.Println("Adding commands...")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(d.deps.Commands))
	for i, v := range d.deps.Commands {
		cmd, err := d.deps.Session.ApplicationCommandCreate(
			d.deps.Session.State.User.ID, d.deps.Cfg.GuildId, v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
			return nil
		}
		registeredCommands[i] = cmd
	}
	d.registeredCommands = registeredCommands

	return nil
}

func (d *Discord) Stop() error {
	defer d.deps.Session.Close()

	if d.deps.Cfg.RemoveCommands {
		log.Println("Removing commands...")
		// // We need to fetch the commands, since deleting requires the command ID.
		// // We are doing this from the returned commands on line 375, because using
		// // this will delete all the commands, which might not be desirable, so we
		// // are deleting only the commands that we added.
		// registeredCommands, err := s.ApplicationCommands(s.State.User.ID, *GuildID)
		// if err != nil {
		// 	log.Fatalf("Could not fetch registered commands: %v", err)
		// }

		for _, v := range d.registeredCommands {
			err := d.deps.Session.ApplicationCommandDelete(
				d.deps.Session.State.User.ID, d.deps.Cfg.GuildId, v.ID)
			if err != nil {
				log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
				return err
			}
		}
	}

	log.Println("Gracefully shutting down.")

	return nil
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func (d *Discord) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	switch m.Content {
	case "ping":
		s.ChannelMessageSend(m.ChannelID, "Pong!")
	case "pong":
		s.ChannelMessageSend(m.ChannelID, "Ping!")
	default:
		msg, err := d.deps.ChatGpt.GetResponse(context.TODO(), m.Content)
		if err != nil {
			log.Panicf("%s", err)
		}
		if len(msg) > 0 {
			s.ChannelMessageSend(m.ChannelID, msg)
		}
	}
}
