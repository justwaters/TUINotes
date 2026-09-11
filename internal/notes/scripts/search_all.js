function run(argv) {
  const params = JSON.parse(argv[0]);
  const Notes = Application("Notes");
  const account = Notes.accounts.byId(params.accountId);
  const folders = account.folders();
  const result = [];
  for (let i = 0; i < folders.length; i++) {
    const folder = folders[i];
    const notes = folder.notes;
    const ids = notes.id();
    if (ids.length === 0) continue;
    const names = notes.name();
    const mods = notes.modificationDate();
    const plaintexts = notes.plaintext();
    const folderId = folder.id();
    const folderName = folder.name();
    for (let j = 0; j < ids.length; j++) {
      result.push({
        id: ids[j],
        name: names[j],
        plaintext: plaintexts[j],
        modificationDate: mods[j],
        folderId,
        folderName,
      });
    }
  }
  return JSON.stringify(result);
}
