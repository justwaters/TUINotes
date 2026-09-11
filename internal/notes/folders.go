package notes

import "context"

// ListFolders returns every folder belonging to the given account.
func (c *Client) ListFolders(ctx context.Context, accountID string) ([]Folder, error) {
	var folders []Folder
	params := map[string]string{"accountId": accountID}
	if err := c.run(ctx, "list_folders.js", params, &folders); err != nil {
		return nil, err
	}
	return folders, nil
}
