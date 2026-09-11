function run(argv) {
  const params = JSON.parse(argv[0]);
  const Notes = Application("Notes");
  const note = Notes.notes.byId(params.noteId);
  const dest = Notes.folders.byId(params.destFolderId);
  Notes.move(note, { to: dest });
  return JSON.stringify({ ok: true });
}
