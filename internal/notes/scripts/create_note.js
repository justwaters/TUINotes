function run(argv) {
  const params = JSON.parse(argv[0]);
  const Notes = Application("Notes");
  const folder = Notes.folders.byId(params.folderId);
  // Only `body` is set. Notes.app derives `name` and `plaintext` from the
  // first line of body itself; setting `name` separately causes it to be
  // duplicated as an extra leading line in plaintext.
  const note = Notes.Note({ body: params.body });
  folder.notes.push(note);
  return JSON.stringify({
    id: note.id(),
    name: note.name(),
    body: note.body(),
    modificationDate: note.modificationDate(),
    attachmentCount: 0,
    folderId: folder.id(),
    folderName: folder.name(),
  });
}
