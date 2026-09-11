function run(argv) {
  const params = JSON.parse(argv[0]);
  const Notes = Application("Notes");
  const note = Notes.notes.byId(params.noteId);
  const container = note.container();
  return JSON.stringify({
    id: note.id(),
    name: note.name(),
    body: note.body(),
    modificationDate: note.modificationDate(),
    attachmentCount: note.attachments().length,
    folderId: container.id(),
    folderName: container.name(),
  });
}
