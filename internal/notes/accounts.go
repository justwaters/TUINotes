package notes

import "context"

// ListAccounts returns every Notes account (e.g. "iCloud", "On My Mac").
func (c *Client) ListAccounts(ctx context.Context) ([]Account, error) {
	var accounts []Account
	if err := c.run(ctx, "list_accounts.js", nil, &accounts); err != nil {
		return nil, err
	}
	return accounts, nil
}
