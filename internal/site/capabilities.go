package site

// Capability is a code-owned service line shown on the public site.
type Capability struct {
	Slug        string
	Name        string
	Outcome     string
	Overview    string
	Deliverables []string
	Engagement  []string
}

var capabilities = []Capability{
	{
		Slug:    "premium-ai-application-development",
		Name:    "Premium AI Application Development",
		Outcome: "Production-grade AI applications engineered for reliability, operability, and long-term ownership.",
		Overview: "We design and build AI applications as durable systems—not demos. Engagements emphasise clear interfaces, " +
			"measurable acceptance criteria, and operational readiness so applications remain maintainable as models, data, and scale evolve.",
		Deliverables: []string{
			"Application architecture and interface contracts",
			"Model integration with controlled evaluation paths",
			"Service boundaries, APIs, and operational telemetry",
			"Deployment packaging and runbooks for production ownership",
		},
		Engagement: []string{
			"Define outcomes, constraints, and acceptance criteria",
			"Design the application and integration surface",
			"Implement against measurable milestones",
			"Validate behaviour, operability, and handover readiness",
		},
	},
	{
		Slug:    "platform-engineering",
		Name:    "Platform Engineering",
		Outcome: "Internal platforms that make delivery safer, faster, and consistent across teams.",
		Overview: "We build platforms that encode engineering standards into paved paths—CI/CD, environments, observability, " +
			"and developer workflows—so product teams ship on shared infrastructure rather than bespoke stacks.",
		Deliverables: []string{
			"Platform architecture and capability map",
			"Golden paths for build, deploy, and operate",
			"Self-service interfaces and policy guardrails",
			"Observability, identity, and environment baselines",
		},
		Engagement: []string{
			"Map current delivery friction and platform gaps",
			"Design the platform surface and ownership model",
			"Implement core paths and guardrails",
			"Validate adoption, reliability, and operational fit",
		},
	},
	{
		Slug:    "ai-infrastructure",
		Name:    "AI Infrastructure Design, Implementation & Validation",
		Outcome: "Infrastructure designed for AI workloads, implemented with discipline, and proven under load.",
		Overview: "We design, implement, and validate AI infrastructure end to end—compute, networking, storage, orchestration, " +
			"and operational controls—so training and inference platforms are dependable rather than improvised.",
		Deliverables: []string{
			"Infrastructure architecture and capacity model",
			"Cluster, networking, and storage implementation",
			"Workload placement, scheduling, and isolation patterns",
			"Validation plans, performance evidence, and operational baselines",
		},
		Engagement: []string{
			"Capture workload profiles and non-functional requirements",
			"Design the infrastructure and control plane",
			"Implement and harden the target environment",
			"Validate performance, resilience, and operability",
		},
	},
	{
		Slug:    "embedded-systems",
		Name:    "Embedded Systems",
		Outcome: "Edge and embedded systems that bring intelligence to constrained, real-world environments.",
		Overview: "We engineer embedded and edge systems where compute, power, latency, and reliability constraints are first-class. " +
			"Work spans device software, integration with cloud or on-prem control planes, and validation in representative conditions.",
		Deliverables: []string{
			"Device and edge software architecture",
			"Integration with sensors, actuators, and host systems",
			"Update, security, and lifecycle considerations",
			"Field-representative validation and evidence packs",
		},
		Engagement: []string{
			"Define device constraints and operating context",
			"Design the embedded software and integration path",
			"Implement and instrument the target system",
			"Validate under representative conditions",
		},
	},
	{
		Slug:    "cloud",
		Name:    "Cloud",
		Outcome: "Cloud architectures that are secure, operable, and aligned to how the organisation actually runs.",
		Overview: "We design and implement cloud foundations and application platforms with clear tenancy, identity, networking, " +
			"and operational models—suited to regulated and infrastructure-critical environments as well as product delivery.",
		Deliverables: []string{
			"Landing zones and account or project topology",
			"Identity, network, and security baselines",
			"Workload platforms and deployment patterns",
			"Cost, reliability, and operational guardrails",
		},
		Engagement: []string{
			"Establish requirements, constraints, and target operating model",
			"Design the cloud foundation and platform layers",
			"Implement and document the environment",
			"Validate security, operability, and handover readiness",
		},
	},
}

// Capabilities returns all service lines in display order.
func Capabilities() []Capability {
	out := make([]Capability, len(capabilities))
	copy(out, capabilities)
	return out
}

// CapabilityBySlug returns a capability or false if unknown.
func CapabilityBySlug(slug string) (Capability, bool) {
	for _, c := range capabilities {
		if c.Slug == slug {
			return c, true
		}
	}
	return Capability{}, false
}
