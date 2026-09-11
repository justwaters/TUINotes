package ui

import "sort"

// orderFoldersAsTree reorders a flat folder list (as returned by
// notes.Client.ListFolders, where nested and top-level folders are mixed
// together) into parent-before-children order, setting each folder's
// Depth. Siblings are sorted alphabetically. Folders whose ParentID
// doesn't resolve to any account or folder in the input (shouldn't happen,
// but Notes.app's scripting dictionary is undocumented) are appended at
// depth 0 rather than silently dropped.
func orderFoldersAsTree(folders []folderWithAccount) []folderWithAccount {
	childrenOf := make(map[string][]folderWithAccount, len(folders))
	for _, f := range folders {
		childrenOf[f.ParentID] = append(childrenOf[f.ParentID], f)
	}
	for _, kids := range childrenOf {
		sort.Slice(kids, func(i, j int) bool { return kids[i].Name < kids[j].Name })
	}

	var out []folderWithAccount
	visited := make(map[string]bool, len(folders))

	var visit func(parentID string, depth int)
	visit = func(parentID string, depth int) {
		for _, f := range childrenOf[parentID] {
			f.Depth = depth
			out = append(out, f)
			visited[f.ID] = true
			visit(f.ID, depth+1)
		}
	}

	seenAccount := map[string]bool{}
	for _, f := range folders {
		if !seenAccount[f.AccountID] {
			seenAccount[f.AccountID] = true
			visit(f.AccountID, 0)
		}
	}

	for _, f := range folders {
		if !visited[f.ID] {
			f.Depth = 0
			out = append(out, f)
		}
	}

	return out
}
