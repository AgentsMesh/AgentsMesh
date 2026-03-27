package eval

// ApplyRemoves post-processes a BuildResult by removing items
// collected from REMOVE declarations and remove statements.
func ApplyRemoves(r *BuildResult) {
	if len(r.RemoveArgs) > 0 {
		r.LaunchArgs = removeFromSlice(r.LaunchArgs, r.RemoveArgs)
	}
	if len(r.RemoveFiles) > 0 {
		removeSet := toSet(r.RemoveFiles)
		filtered := r.FilesToCreate[:0]
		for _, f := range r.FilesToCreate {
			if !removeSet[f.Path] {
				filtered = append(filtered, f)
			}
		}
		r.FilesToCreate = filtered

		filteredDirs := r.Dirs[:0]
		for _, d := range r.Dirs {
			if !removeSet[d] {
				filteredDirs = append(filteredDirs, d)
			}
		}
		r.Dirs = filteredDirs
	}
	if len(r.RemoveEnvs) > 0 {
		for _, name := range r.RemoveEnvs {
			delete(r.EnvVars, name)
		}
	}
	if len(r.RemoveSkills) > 0 {
		removeSet := toSet(r.RemoveSkills)
		filtered := r.Skills[:0]
		for _, s := range r.Skills {
			if !removeSet[s] {
				filtered = append(filtered, s)
			}
		}
		r.Skills = filtered
	}
}

// removeFromSlice removes args, handling flag+value pairs.
// E.g., removing "--model" also removes the next arg ("opus").
func removeFromSlice(args []string, removes []string) []string {
	removeSet := toSet(removes)
	var result []string
	for i := 0; i < len(args); i++ {
		if removeSet[args[i]] {
			// If this looks like a flag and next arg is a value, skip both
			if i+1 < len(args) && len(args[i]) > 1 && args[i][0] == '-' &&
				(len(args[i+1]) == 0 || args[i+1][0] != '-') {
				i++ // skip the value too
			}
			continue
		}
		result = append(result, args[i])
	}
	return result
}

func toSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}
