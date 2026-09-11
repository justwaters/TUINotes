function run(argv) {
  const Notes = Application("Notes");
  const accounts = Notes.accounts();
  const result = [];
  for (let i = 0; i < accounts.length; i++) {
    result.push({ id: accounts[i].id(), name: accounts[i].name() });
  }
  return JSON.stringify(result);
}
