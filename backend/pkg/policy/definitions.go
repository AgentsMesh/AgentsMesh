package policy

var (
	// PodPolicy: members see/manage their own pods; admins see all.
	PodPolicy = ResourcePolicy{
		Read:  ReadOwnerOnly,
		Write: WriteCreatorAdmin,
	}

	// RunnerPolicy: visibility-controlled read (private = truly private, no admin bypass);
	// only admins may create/update/delete runners.
	RunnerPolicy = ResourcePolicy{
		Read:  ReadVisibility,
		Write: WriteAdminOnly,
	}

	// RepositoryPolicy: visibility-controlled read; only admins may manage repos.
	RepositoryPolicy = ResourcePolicy{
		Read:  ReadVisibility,
		Write: WriteAdminOnly,
	}

	// Tickets and Loops are org-open by design; no policy enforcement needed.
)
