function run(argv) {
  const params = JSON.parse(argv[0]);
  const Notes = Application("Notes");
  const note = Notes.notes.byId(params.noteId);
  note.delete();
  return JSON.stringify({ ok: true });
}
