package site

import "strings"

// TwinPersona describes a public digital twin offered on the website.
type TwinPersona struct {
	ID         string
	Name       string
	Label      string
	ShortLabel string
	Initials   string
	Intro      string
}

var publicTwinPersonas = []TwinPersona{
	{
		ID:         "ceo",
		Name:       "Darin Srijan",
		Label:      "Chief Executive Officer",
		ShortLabel: "CEO",
		Initials:   "CEO",
		Intro:      "I represent Neon AI Cloud on strategy, partnerships, and commercial direction. Ask about our vision, how we engage customers, or whether Neon AI Cloud is the right partner for your programme.",
	},
	{
		ID:         "cto",
		Name:       "Traiano Giuseppe Welcome",
		Label:      "Chief Technology Officer",
		ShortLabel: "CTO",
		Initials:   "CTO",
		Intro:      "I represent Neon AI Cloud on architecture, platforms, AI infrastructure, and engagement fit. Ask about our capabilities, how we work, or whether we are the right partner for your programme.",
	},
	{
		ID:         "engineering",
		Name:       "Troy Welcome",
		Label:      "Head of Engineering",
		ShortLabel: "Engineering",
		Initials:   "ENG",
		Intro:      "I represent Neon AI Cloud engineering on delivery, product build quality, and how we ship robust platforms. Ask about implementation approach, engineering practices, or product depth.",
	},
	{
		ID:         "sales",
		Name:       "Irana Welcome",
		Label:      "Head of Sales",
		ShortLabel: "Sales",
		Initials:   "SAL",
		Intro:      "I represent Neon AI Cloud on customer engagement, scope, and commercial fit. Ask about how we work with clients, typical engagements, or the best next step for your enquiry.",
	},
}

// PublicTwinPersonas returns all website-visible digital twins.
func PublicTwinPersonas() []TwinPersona {
	out := make([]TwinPersona, len(publicTwinPersonas))
	copy(out, publicTwinPersonas)
	return out
}

// DefaultTwinPersona is the twin selected when none is specified.
func DefaultTwinPersona() TwinPersona {
	return publicTwinPersonas[1] // cto
}

// TwinPersonaByID looks up a public twin by id.
func TwinPersonaByID(id string) (TwinPersona, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	for _, twin := range publicTwinPersonas {
		if twin.ID == id {
			return twin, true
		}
	}
	return TwinPersona{}, false
}
