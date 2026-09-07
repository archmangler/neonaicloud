package site

// DownloadAsset is a public, read-only file offered from a download page.
type DownloadAsset struct {
	Number      string
	Title       string
	Description string
	Href        string
	Filename    string
}

// ArchitectureSolutionPack returns the eight ACME Asia architecture pack PDFs.
func ArchitectureSolutionPack() []DownloadAsset {
	return []DownloadAsset{
		{
			Number:      "01",
			Title:       "ACME Asia RFI",
			Description: "Representative customer request for information for a 1024 GPU B300 AI Factory.",
			Href:        "/media/blogs/go-live/01-acme-asia-rfi-1024-gpu-b300.pdf",
			Filename:    "01-acme-asia-rfi-1024-gpu-b300.pdf",
		},
		{
			Number:      "02",
			Title:       "ACME Asia RFP",
			Description: "Representative customer request for proposal used as the architecture starting point.",
			Href:        "/media/blogs/go-live/02-acme-asia-rfp-1024-gpu-b300.pdf",
			Filename:    "02-acme-asia-rfp-1024-gpu-b300.pdf",
		},
		{
			Number:      "03",
			Title:       "Executive Architecture Brief",
			Description: "Neon-generated executive summary of the solved AI Factory architecture.",
			Href:        "/media/blogs/go-live/03-acme-asia-executive-architecture-brief.pdf",
			Filename:    "03-acme-asia-executive-architecture-brief.pdf",
		},
		{
			Number:      "04",
			Title:       "Full AI Factory Architecture",
			Description: "Complete solution architecture proposal compiled from the governed Neon design.",
			Href:        "/media/blogs/go-live/04-acme-asia-full-ai-factory-architecture.pdf",
			Filename:    "04-acme-asia-full-ai-factory-architecture.pdf",
		},
		{
			Number:      "05",
			Title:       "Security Architecture Report",
			Description: "Architecture-level security assessment, trust boundaries, findings and unresolved decisions.",
			Href:        "/media/blogs/go-live/05-acme-asia-security-architecture-report.pdf",
			Filename:    "05-acme-asia-security-architecture-report.pdf",
		},
		{
			Number:      "06",
			Title:       "Project Delivery Plan",
			Description: "Architecture-derived project planning baseline for engineering and delivery teams.",
			Href:        "/media/blogs/go-live/06-acme-asia-project-delivery-plan.pdf",
			Filename:    "06-acme-asia-project-delivery-plan.pdf",
		},
		{
			Number:      "07",
			Title:       "Customer Proposal (v1.1)",
			Description: "Final solution architecture proposal presented by the RFP response team.",
			Href:        "/media/blogs/go-live/07-acme-asia-proposal-v1.1.pdf",
			Filename:    "07-acme-asia-proposal-v1.1.pdf",
		},
		{
			Number:      "08",
			Title:       "End-to-end GUI Walkthrough",
			Description: "Walkthrough of the Neon workflow from customer inputs to architecture evidence.",
			Href:        "/media/blogs/go-live/08-acme-asia-end2end-gui-walkthrough.pdf",
			Filename:    "08-acme-asia-end2end-gui-walkthrough.pdf",
		},
	}
}
