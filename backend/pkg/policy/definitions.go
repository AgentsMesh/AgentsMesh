package policy

var (
	PodPolicy = ResourcePolicy{Read: ReadOwnerOnly, Write: WriteCreatorAdmin}

	RunnerPolicy = ResourcePolicy{Read: ReadVisibility, Write: WriteAdminOnly}

	RepositoryPolicy = ResourcePolicy{Read: ReadVisibility, Write: WriteAdminOnly}
)
