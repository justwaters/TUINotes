function run(argv) {
  const params = JSON.parse(argv[0]);
  const Notes = Application("Notes");
  const folder = Notes.folders.byId(params.folderId);
  const notes = folder.notes;
  const ids = notes.id();
  const names = notes.name();
  const mods = notes.modificationDate();
  const attArrays = notes.attachments();
  const result = ids.map((id, i) => ({
    id,
    name: names[i],
    modificationDate: mods[i],
    attachmentCount: attArrays[i].length,
  }));
  return JSON.stringify(result);
}
