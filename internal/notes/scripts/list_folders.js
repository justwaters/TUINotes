function run(argv) {
  const params = JSON.parse(argv[0]);
  const Notes = Application("Notes");
  const account = Notes.accounts.byId(params.accountId);
  const folders = account.folders;
  const ids = folders.id();
  const names = folders.name();
  // container() is the account itself for a top-level folder, or the
  // parent Folder for a nested one — needed to reconstruct the real
  // folder tree, since account.folders() returns every folder flattened.
  const containers = folders.container();
  const result = ids.map((id, i) => ({
    id,
    name: names[i],
    parentId: containers[i].id(),
  }));
  return JSON.stringify(result);
}
