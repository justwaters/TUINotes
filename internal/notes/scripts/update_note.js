function run(argv) {
  const params = JSON.parse(argv[0]);
  const Notes = Application("Notes");
  const note = Notes.notes.byId(params.noteId);
  // Same rule as create: only ever write `body`, never `name` directly.
  note.body = params.body;
  const container = note.container();
  return JSON.stringify({
    id: note.id(),
    name: note.name(),
    plaintext: note.plaintext(),
    modificationDate: note.modificationDate(),
    attachmentCount: note.attachments().length,
    folderId: container.id(),
    folderName: container.name(),
  });
}
