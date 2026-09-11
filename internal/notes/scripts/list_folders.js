function run(argv) {
  const params = JSON.parse(argv[0]);
  const Notes = Application("Notes");
  const account = Notes.accounts.byId(params.accountId);
  const folders = account.folders;
  const ids = folders.id();
  const names = folders.name();
  const result = ids.map((id, i) => ({ id, name: names[i] }));
  return JSON.stringify(result);
}
